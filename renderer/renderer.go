package renderer

import (
	"fmt"
	gomath "math"

	"render-engine/core"
	"render-engine/math"
	"render-engine/internal/opengl"
	"render-engine/scene"
)

// textCmd is a queued DrawText call, flushed in Present().
type textCmd struct {
	text  string
	x, y  float32
	scale float32
	color core.Color
}

// RenderEngine is the high-level renderer that drives the OpenGL backend.
type RenderEngine struct {
	gl             *opengl.Renderer
	window         *core.Window
	Scene          *scene.Scene
	FrustumCulling     bool // disabled by default — verify matrix convention first
	ShadowsEnabled     bool // enable via EnableShadows()
	PostProcessEnabled bool // enable via EnablePostProcess()
	SkyboxEnabled      bool // enable via EnableSkybox()
	DrawAABBs          bool // draw debug wireframe boxes around every node's AABB

	shadowOrthoSize float32       // orthographic half-extent for the shadow volume
	aabbMesh        *scene.Mesh   // unit-cube wireframe, created on first AABB draw

	// Per-frame stats (populated during Render)
	lastObjects   int
	lastVertices  int
	lastTriangles int
	lastCulled    int

	// Queued text commands, flushed in Present() after the HDR blit
	textQueue []textCmd
}

func NewRenderEngine(window *core.Window) (*RenderEngine, error) {
	glRenderer, err := opengl.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenGL renderer: %w", err)
	}

	glRenderer.SetViewport(window.Width, window.Height)

	fmt.Println("Render engine initialized (OpenGL)")
	return &RenderEngine{
		gl:              glRenderer,
		window:          window,
		FrustumCulling:  false,
		ShadowsEnabled:  false,
		shadowOrthoSize: 30.0,
	}, nil
}

// EnableSkybox creates the procedural gradient skybox.
// Call once after NewRenderEngine, before the first Render.
func (re *RenderEngine) EnableSkybox() error {
	if err := re.gl.EnableSkybox(); err != nil {
		return fmt.Errorf("skybox: %w", err)
	}
	re.SkyboxEnabled = true
	return nil
}

// SetSkyboxColors adjusts the three gradient stops and syncs IBL colours.
// zenith = overhead, horizon = eye-level, ground = below the horizon.
func (re *RenderEngine) SetSkyboxColors(zenith, horizon, ground core.Color) {
	sb := re.gl.SkyboxRef()
	if sb != nil {
		sb.ZenithColor  = zenith
		sb.HorizonColor = horizon
		sb.GroundColor  = ground
	}
	// Keep IBL in sync with the skybox gradient
	re.gl.SetIBLColors(zenith, horizon, ground)
}

// SetFog configures exponential depth fog. density: 0.01=haze, 0.05=thick.
// color should match the horizon sky for natural blending.
func (re *RenderEngine) SetFog(enabled bool, density float32, color core.Color) {
	re.gl.SetFog(enabled, density, color)
}

// EnableIBL activates sky-based ambient irradiance for PBR and Phong shading.
// Call after NewRenderEngine; SetSkyboxColors must be called to supply colours.
func (re *RenderEngine) EnableIBL() {
	re.gl.EnableIBL()
}

// EnablePostProcess creates the HDR post-processing FBO at the current window size.
// Call once after NewRenderEngine, before the first Render.
func (re *RenderEngine) EnablePostProcess() error {
	if err := re.gl.EnablePostProcess(re.window.Width, re.window.Height); err != nil {
		return fmt.Errorf("post-process: %w", err)
	}
	re.PostProcessEnabled = true
	return nil
}

// SetExposure sets the HDR tone-mapping exposure (default 1.0).
func (re *RenderEngine) SetExposure(exp float32) {
	re.gl.SetExposure(exp)
}

// EnableBloom activates the bloom effect. EnablePostProcess must be called first.
func (re *RenderEngine) EnableBloom() error {
	return re.gl.EnableBloom()
}

// SetBloomThreshold sets the luminance cut-off for bloom (default 1.0).
func (re *RenderEngine) SetBloomThreshold(t float32) { re.gl.SetBloomThreshold(t) }

// SetBloomStrength sets the additive bloom multiplier (default 0.6).
func (re *RenderEngine) SetBloomStrength(s float32) { re.gl.SetBloomStrength(s) }

// EnableShadows creates the shadow map FBO (2048×2048).
// Call once after NewRenderEngine, before the first Render.
func (re *RenderEngine) EnableShadows() error {
	if err := re.gl.EnableShadows(2048); err != nil {
		return fmt.Errorf("shadows: %w", err)
	}
	re.ShadowsEnabled = true
	return nil
}

func (re *RenderEngine) SetScene(s *scene.Scene) {
	re.Scene = s
}

func (re *RenderEngine) Render() error {
	if re.Scene == nil || re.Scene.Camera == nil {
		return fmt.Errorf("no scene or camera")
	}

	// ── Find directional light (first one wins) ───────────────────────────────
	var dirLight *scene.Light
	for _, l := range re.Scene.Lights {
		if l != nil && l.Type == scene.LightTypeDirectional {
			dirLight = l
			break
		}
	}

	// ── Shadow pass ───────────────────────────────────────────────────────────
	doShadows := re.ShadowsEnabled && re.gl.HasShadowMap() && dirLight != nil
	lightVP := math.Mat4Identity()

	if doShadows {
		ortho := re.shadowOrthoSize
		camPos := re.Scene.Camera.Position
		lightDir := dirLight.Direction.Normalize()

		// Guard: degenerate direction (zero vector)
		if lightDir.LengthSqr() < 0.001 {
			doShadows = false
		} else {
			// Place shadow camera behind the scene along the light direction
			lightEye := camPos.Sub(lightDir.Mul(ortho))

			// Choose an up vector that is not parallel to the light direction
			upVec := math.Vec3Up
			if gomath.Abs(float64(lightDir.Dot(math.Vec3Up))) > 0.999 {
				upVec = math.Vec3{X: 0, Y: 0, Z: 1}
			}

			lightView := math.Mat4LookAt(lightEye, camPos, upVec)
			lightProj := math.Mat4Orthographic(
				-ortho, ortho, -ortho, ortho,
				-ortho, ortho*3,
			)
			lightVP = lightView.Mul(lightProj)

			re.gl.BeginShadowPass()
			for _, node := range re.Scene.GetVisibleNodes() {
				if node.Mesh == nil || node.Mesh.DrawMode != scene.DrawTriangles {
					continue
				}
				model := node.GetWorldMatrix()
				lightMVP := model.Mul(lightView).Mul(lightProj)
				re.gl.DrawMeshShadow(node.Mesh, lightMVP)
			}
			re.gl.EndShadowPass()
		}
	}

	// ── Main render pass ──────────────────────────────────────────────────────
	// Compute proj before BeginFrame so it can be stored for the SSAO pass.
	proj := re.Scene.Camera.GetProjectionMatrix()
	re.gl.BeginFrame(
		re.Scene.SkyColor,
		re.Scene.Lights,
		re.Scene.Ambient,
		re.Scene.Camera.Position,
		lightVP,
		doShadows,
		proj,
	)

	view := re.Scene.Camera.GetViewMatrix()

	// Draw skybox first (depth=1.0 via xyww, before all scene geometry)
	re.gl.DrawSkybox(view, proj)

	// Build view-projection matrix for frustum culling
	vp := view.Mul(proj)
	frustum := scene.FrustumFromVP(vp)

	objects, vertices, triangles, culled := 0, 0, 0, 0

	for _, node := range re.Scene.GetVisibleNodes() {
		if node.Mesh == nil {
			continue
		}

		model := node.GetWorldMatrix()

		// Frustum culling: skip draw if AABB is completely outside the frustum
		if re.FrustumCulling {
			aabb := scene.ComputeAABB(node.Mesh, model)
			if !aabb.IntersectsFrustum(&frustum) {
				culled++
				continue
			}
		}

		mvp := model.Mul(view).Mul(proj)
		re.gl.DrawMesh(node.Mesh, mvp, model)

		objects++
		vertices += len(node.Mesh.Vertices)
		triangles += len(node.Mesh.Indices) / 3
	}

	re.lastObjects = objects
	re.lastVertices = vertices
	re.lastTriangles = triangles
	re.lastCulled = culled

	// ── AABB debug visualization ───────────────────────────────────────────
	if re.DrawAABBs {
		re.drawAABBs(view, proj)
	}

	return nil
}

// Present resolves the HDR FBO (tone mapping, bloom, SSAO) to the default
// framebuffer, flushes queued text (drawn on top of the HDR blit), and swaps
// buffers. Call after Render() and any additional draw passes.
func (re *RenderEngine) Present() {
	re.gl.BlitPostProcess()
	// Flush text queue — drawn to the default framebuffer, always on top
	if len(re.textQueue) > 0 {
		sw := float32(re.window.Width)
		sh := float32(re.window.Height)
		for _, cmd := range re.textQueue {
			re.gl.DrawText(cmd.text, cmd.x, cmd.y, cmd.scale, cmd.color, sw, sh)
		}
		re.textQueue = re.textQueue[:0]
	}
	re.window.SwapBuffers()
}

// DrawText queues a text string to be drawn at screen position (x, y) in the
// next Present() call. scale=1 → 8×8 px glyphs, scale=2 → 16×16 px, etc.
// Text is drawn after tone mapping, so it bypasses HDR and is always readable.
func (re *RenderEngine) DrawText(text string, x, y int, scale float32, color core.Color) {
	re.textQueue = append(re.textQueue, textCmd{
		text:  text,
		x:     float32(x),
		y:     float32(y),
		scale: scale,
		color: color,
	})
}

func (re *RenderEngine) Resize(width, height uint32) {
	re.gl.SetViewport(int(width), int(height))
	if re.PostProcessEnabled {
		re.gl.ResizePostProcess(int(width), int(height))
	}
	if re.Scene != nil && re.Scene.Camera != nil {
		re.Scene.Camera.UpdateAspectRatio(float32(width), float32(height))
	}
}

// DrawParticles renders a ParticleEmitter's live particles as camera-facing
// billboards.  Call between Render() and Present() so particles are included
// in the HDR FBO and benefit from tone mapping and bloom.
func (re *RenderEngine) DrawParticles(emitter *scene.ParticleEmitter) {
	if re.Scene == nil || re.Scene.Camera == nil || emitter == nil {
		return
	}
	view := re.Scene.Camera.GetViewMatrix()
	proj := re.Scene.Camera.GetProjectionMatrix()
	re.gl.DrawParticles(emitter, view, proj)
}

// DrawMeshInstanced renders mesh at every transform in models using a single
// GPU draw call. This is orders of magnitude faster than individual AddNode
// calls for large repeated geometry (grass, trees, rocks, crowds).
//
// The mesh must not be part of the scene graph — call this every frame from
// outside the normal Render() loop:
//
//	renderEngine.Render()                              // normal scene pass
//	renderEngine.DrawMeshInstanced(treeMesh, matrices) // instanced overlay
func (re *RenderEngine) DrawMeshInstanced(mesh *scene.Mesh, models []math.Mat4) {
	if re.Scene == nil || re.Scene.Camera == nil || len(models) == 0 {
		return
	}
	view := re.Scene.Camera.GetViewMatrix()
	proj := re.Scene.Camera.GetProjectionMatrix()
	re.gl.DrawMeshInstanced(mesh, view, proj, models)
}

// EnableSSAO creates the SSAO pipeline.  EnablePostProcess must be called first.
func (re *RenderEngine) EnableSSAO() error {
	if err := re.gl.EnableSSAO(); err != nil {
		return fmt.Errorf("ssao: %w", err)
	}
	return nil
}

// SetSSAORadius sets the SSAO hemisphere radius in view-space units (default 0.5).
func (re *RenderEngine) SetSSAORadius(v float32) { re.gl.SetSSAORadius(v) }

// SetSSAOBias sets the depth bias to prevent self-occlusion acne (default 0.025).
func (re *RenderEngine) SetSSAOBias(v float32) { re.gl.SetSSAOBias(v) }

// SetSSAOStrength sets the AO blend factor: 0 = no AO, 1 = full AO (default 1.0).
func (re *RenderEngine) SetSSAOStrength(v float32) { re.gl.SetSSAOStrength(v) }

// SetWireframe toggles wireframe rendering mode on/off.
func (re *RenderEngine) SetWireframe(enabled bool) {
	re.gl.SetWireframe(enabled)
}

// IsWireframe returns whether wireframe mode is currently active.
func (re *RenderEngine) IsWireframe() bool {
	return re.gl.IsWireframe()
}

// UploadTexture uploads a texture to the GPU. Must be called from the main thread.
func (re *RenderEngine) UploadTexture(tex *scene.Texture) error {
	return opengl.UploadTexture(tex)
}

// DeleteTexture frees a previously uploaded GPU texture.
func (re *RenderEngine) DeleteTexture(tex *scene.Texture) {
	opengl.DeleteTexture(tex)
}

func (re *RenderEngine) Destroy() {
	re.gl.Destroy()
}

func (re *RenderEngine) WaitIdle() {
	// No-op for OpenGL; synchronous by nature.
}

// DrawStats returns stats from the most recent Render call.
func (re *RenderEngine) DrawStats() (objects, vertices, triangles, culled int) {
	return re.lastObjects, re.lastVertices, re.lastTriangles, re.lastCulled
}

// drawAABBs draws a wireframe unit-cube scaled/translated to each visible node's
// world-space AABB.  The unit-box mesh is created lazily on first call.
func (re *RenderEngine) drawAABBs(view, proj math.Mat4) {
	if re.aabbMesh == nil {
		re.aabbMesh = scene.CreateUnitBoxWireframe()
	}

	identity := math.Mat4Identity()

	for _, node := range re.Scene.GetVisibleNodes() {
		if node.Mesh == nil {
			continue
		}
		worldMat := node.GetWorldMatrix()
		aabb := scene.ComputeAABB(node.Mesh, worldMat)

		// Build a scale+translate matrix that maps the unit cube (±1) to the AABB.
		// In [col][row] (column-major) layout:
		//   scale:     [col][col] diagonal
		//   translate: column 3, rows 0-2
		cx := (aabb.Min.X + aabb.Max.X) * 0.5
		cy := (aabb.Min.Y + aabb.Max.Y) * 0.5
		cz := (aabb.Min.Z + aabb.Max.Z) * 0.5
		hx := (aabb.Max.X - aabb.Min.X) * 0.5
		hy := (aabb.Max.Y - aabb.Min.Y) * 0.5
		hz := (aabb.Max.Z - aabb.Min.Z) * 0.5

		aabbModel := math.Mat4Identity()
		aabbModel[0][0] = hx
		aabbModel[1][1] = hy
		aabbModel[2][2] = hz
		aabbModel[3][0] = cx
		aabbModel[3][1] = cy
		aabbModel[3][2] = cz

		mvp := aabbModel.Mul(view).Mul(proj)
		re.gl.DrawMesh(re.aabbMesh, mvp, identity)
	}
}
