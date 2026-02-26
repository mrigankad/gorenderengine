package renderer

import (
	"fmt"

	"render-engine/core"
	"render-engine/opengl"
	"render-engine/scene"
)

// RenderEngine is the high-level renderer that drives the OpenGL backend.
type RenderEngine struct {
	gl     *opengl.Renderer
	window *core.Window
	Scene  *scene.Scene
}

func NewRenderEngine(window *core.Window) (*RenderEngine, error) {
	glRenderer, err := opengl.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenGL renderer: %w", err)
	}

	glRenderer.SetViewport(window.Width, window.Height)

	fmt.Println("Render engine initialized (OpenGL)")
	return &RenderEngine{gl: glRenderer, window: window}, nil
}

func (re *RenderEngine) SetScene(s *scene.Scene) {
	re.Scene = s
}

func (re *RenderEngine) Render() error {
	if re.Scene == nil || re.Scene.Camera == nil {
		return fmt.Errorf("no scene or camera")
	}

	re.gl.BeginFrame(re.Scene.SkyColor)

	view := re.Scene.Camera.GetViewMatrix()
	proj := re.Scene.Camera.GetProjectionMatrix()

	for _, node := range re.Scene.GetVisibleNodes() {
		if node.Mesh == nil {
			continue
		}
		model := node.GetWorldMatrix()
		// Mul semantics: A.Mul(B) = B_gl * A_gl, so to get P*V*M use M.Mul(V).Mul(P)
		mvp := model.Mul(view).Mul(proj)
		re.gl.DrawMesh(node.Mesh, mvp)
	}

	re.window.SwapBuffers()
	return nil
}

func (re *RenderEngine) Resize(width, height uint32) {
	re.gl.SetViewport(int(width), int(height))
	if re.Scene != nil && re.Scene.Camera != nil {
		re.Scene.Camera.UpdateAspectRatio(float32(width), float32(height))
	}
}

func (re *RenderEngine) Destroy() {
	re.gl.Destroy()
}

func (re *RenderEngine) WaitIdle() {
	// No-op for OpenGL; synchronous by nature.
}
