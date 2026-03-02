# Sonorlax Engine - Development Plan

## Completed Features

### Foundation
- ✅ OpenGL 4.1 backend (VAO/VBO/EBO, indexed drawing, depth test)
- ✅ Scene graph — Transform, Node hierarchy, world-matrix caching with dirty flags
- ✅ 7 primitive shapes: Cube, Sphere, Cylinder, Cone, Pyramid, Torus, Plane
- ✅ Camera system (perspective projection, LookAt, OrbitCamera, free-look controller)
- ✅ Editor framework (input, selection, undo/redo, raycast, command pattern)
- ✅ Windows build (CGO + GLFW, `build.bat` → `triangle_app.exe`)
- ✅ Free camera controller (WASD + right-mouse drag look)
- ✅ Debug overlay (FPS, position, yaw/pitch in console + window title)
- ✅ Custom math library (Vec2, Vec3, Vec4, Mat4, Quaternion + tests)

### Lighting & Shading
- ✅ Scene lights wired to shader — directional (dir, color, intensity, ambient)
- ✅ Correct world-space normals via `mat3(model) * inNormal` in vertex shader
- ✅ Phong shading — ambient + diffuse + specular (Blinn-Phong half-vector)
- ✅ Point lights — up to 8, quadratic attenuation
- ✅ Spot lights — up to 4, inner/outer cone angles, SpotAngle in degrees
- ✅ **PBR shading** — Cook-Torrance BRDF (GGX NDF + Smith geometry + Schlick Fresnel)
  - `usePBR` uniform branches between Phong and PBR paths in the fragment shader
  - F0 = mix(0.04, albedo, metallic) for correct dielectric/metal transition
  - `NewPBRMaterial(name, albedo, metallic, roughness)` constructor
  - Emissive channel: `matEmissive` vec3 + optional `emissiveTex` (unit 4), works with bloom

### Shadows
- ✅ Shadow mapping — directional light, 2048×2048 depth FBO, PCF 3×3 soft shadows
  - `opengl/shadow.go` — ShadowMap: `DEPTH_COMPONENT32F`, `COMPARE_REF_TO_TEXTURE`
  - Depth-only pass shader + `BeginShadowPass` / `DrawMeshShadow` / `EndShadowPass`
  - `lightViewProj` uniform in vertex shader → `fragLightSpacePos`
  - `sampler2DShadow` + `calcShadow()` with bias=0.002 in fragment shader
  - Only directional light is shadowed; ambient + point + spot lights are unshadowed

### Materials & Textures
- ✅ Material system — `scene/material.go`:
  - Phong: Albedo, Specular, Shininess, Unlit, AlbedoTexture, NormalTexture
  - PBR: UsePBR, Metallic, Roughness, EmissiveColor, MetallicRoughnessTexture (unit 3), EmissiveTexture (unit 4)
- ✅ Texture loading — PNG / JPEG via Go stdlib `image` package (`scene/texture.go`)
- ✅ GPU texture upload — `opengl/texture.go`: UploadTexture / DeleteTexture
- ✅ Albedo texture sampler — unit 0, `hasTexture` bool uniform
- ✅ Normal map support — tangent-space TBN in vertex shader, unit 2
  - `scene/tangents.go` — `ComputeTangents(m *Mesh)`: per-triangle accumulation, Gram-Schmidt ortho
- ✅ Metallic-roughness texture — unit 3 (G=roughness, B=metallic, glTF convention)
- ✅ Emissive texture — unit 4, multiplied with `matEmissive`

### Asset Loading
- ✅ OBJ loader — `scene/obj_loader.go` with fan triangulation, smooth normal generation, MTL + texture support
- ✅ glTF / GLB loader — `scene/gltf_loader.go`
  - `scene.LoadGLTF(path) (*GLTFResult, error)` returns Roots []*Node + Textures []*Texture
  - PBR materials (base color, metallic, roughness), embedded + external textures
  - Node hierarchy with TRS transforms, multi-primitive meshes
  - Dependency: `github.com/qmuntal/gltf v0.28.0`

### Post-Processing
- ✅ HDR off-screen FBO — RGBA16F render target, Reinhard tone mapping, gamma 2.2
  - `opengl/postprocess.go` — PostProcessFBO, fullscreen triangle via gl_VertexID
  - `[` / `]` keys adjust exposure (0.1–5.0)
- ✅ Bloom — bright-pass threshold + ping-pong Gaussian blur (5-tap) + additive composite
  - Half-res ping-pong blur FBOs, configurable passes
  - `B` toggle, `-` / `=` strength
- ✅ SSAO — screen-space ambient occlusion via depth buffer reconstruction
  - `opengl/ssao.go` — 64-sample hemisphere kernel, 4×4 noise texture, 5×5 box blur
  - Depth texture (DEPTH_COMPONENT32F) → view-space position via inverse projection
  - `O` key toggles strength

### Environment
- ✅ Procedural gradient skybox — inverted cube at depth=1.0 (xyww trick)
  - `opengl/skybox.go` — zenith/horizon/ground gradient, strip translation from view
  - `renderer/renderer.go` — SetSkyboxColors(zenith, horizon, ground)

### Scene & Performance
- ✅ Wireframe toggle — `renderEngine.SetWireframe()`, Z key
- ✅ DrawMode per mesh — DrawTriangles, DrawLines, DrawPoints
- ✅ Unlit flag on Material — skips Phong/PBR, outputs raw base color (used by grid, AABB boxes)
- ✅ Grid floor — `scene/grid.go`: `CreateGrid(size, divisions)`, DrawLines, red X-axis, blue Z-axis
- ✅ AABB debug visualization — unit-box wireframe scaled to each node; X key toggles
- ✅ Local AABB caching — computed once in `CreateMeshFromData`, 8-corner transform for culling
- ✅ Frustum culling — `scene/frustum.go`: Gribb/Hartmann plane extraction, AABB n-vertex test; enabled by default in demo
- ✅ `DrawStats()` — returns (objects, vertices, triangles, culled) per frame
- ✅ Instanced rendering — `glDrawElementsInstanced` / `glDrawArraysInstanced`
  - Instance buffer: 32 floats/instance = MVP (locs 6-9) + Model (locs 10-13)
  - Lazy VBO creation, BufferSubData reuse; `I` key → 400 cubes in 1 draw call
- ✅ Scene serialization — `scene/serialization.go`: JSON save/load of camera, lights, nodes, transforms, materials (`F5` save, `F9` load)

---

- ✅ **Particle system** — CPU-simulated camera-facing billboards, two blend modes
  - `scene/particles.go` — ParticleEmitter (rate, cone spread, color lerp, gravity, compact update loop)
  - `NewParticleEmitter(max)` fire defaults; `NewSmokeEmitter(max)` smoke defaults
  - `opengl/particles.go` — billboard shader, dynamic VBO (6 verts × 9 floats/particle), soft-circle procedural alpha
  - Additive blend (fire/glow): `SRC_ALPHA, ONE`; Alpha blend (smoke): `SRC_ALPHA, ONE_MINUS_SRC_ALPHA`
  - Depth test ON, depth write OFF — correct compositing against scene geometry
  - **Render loop fix**: `Render()` no longer calls BlitPostProcess/SwapBuffers; new `Present()` does
  - Particles + instanced renders land in HDR FBO → benefit from bloom and tone mapping
  - Demo: fire + smoke + magic (blue, inverted gravity) emitters at scene; `E` key toggle; counts in overlay

---

### Font / Text Rendering
- ✅ **On-screen HUD text** — embedded 8×8 bitmap font, 2D orthographic text renderer
  - `opengl/font.go` — `fontBitmap [96*8]byte` (ASCII 32–127), 768×8 GL_RED atlas
  - `buildFontAtlas()` — expands compact bitmap to pixel array
  - `TextRenderer` — text shader (ortho MVP), VAO/VBO (4 floats: pos.xy + uv.xy), 6 verts per char
  - Fragment shader samples GL_RED atlas, multiplies by `textColor` alpha
  - `Mat4Orthographic(0, W, H, 0, -1, 1)` — top-left origin, y-down screen space
  - Depth test disabled, SRC_ALPHA ONE_MINUS_SRC_ALPHA blend; wireframe-safe (forces FILL)
  - `opengl/renderer.go` — lazy `DrawText(text, x, y, scale, color, sw, sh)` method
  - `renderer/renderer.go` — `textQueue`, `DrawText(text, x, y, scale, color)` queues; `Present()` flushes after HDR blit
  - Demo: multi-line HUD overlay: FPS, camera pos, draw stats, all feature toggles; scale=2 (16×16 px glyphs)

---

### Gameplay & Scene
- ✅ **Gravity + ground collision** — `CameraController` gains `velocityY`, `onGround`, `eyeHeight=1.7`; gravity=-18 m/s², Space=jump; horizontal move ignores pitch
- ✅ **Outdoor town-square demo** — replaces 6-shape tech demo
  - 4 buildings (stone/brick/plaster/stone) + roofs, low perimeter walls
  - Central marble fountain (base ring, bowl, water, pillar, top sphere)
  - 6 trees (cylinder trunk + cone canopy), 4 lamp posts (metal pole + emissive cap) each with a warm point light
  - Golden-hour skybox, torch fire+smoke particles, fountain water-spray particles

### IBL
- ✅ **Sky-based IBL** — procedural hemisphere irradiance derived from skybox gradient
  - `sampleSkyGradient(dir)` — lerp zenith/horizon/ground by direction.y
  - PBR: `FresnelSchlickRoughness` splits diffuse (kD × irradiance × albedo) + specular (reflected dir, roughness²-fade)
  - Phong: `sampleSkyGradient(N) × albedo × 0.35` replaces flat ambient
  - `renderEngine.EnableIBL()` activates; `SetSkyboxColors()` auto-syncs IBL

---

## Next Steps

### High Priority
1. **Cascaded shadow maps** — larger shadow coverage for outdoor scenes (multiple frustum splits)
2. **Scene editor tools** — transform gizmos, property inspector, object hierarchy panel

### Medium Priority
3. **Material cache / asset registry** — avoid duplicate GPU uploads, reference-counted textures
4. **Shader hot-reload** — file watcher, recompile + re-link without restart

---

## Phase 3: Performance & Optimization

### 3.1 Culling & LOD
- [x] Frustum culling (AABB vs frustum planes)
- [x] Instanced rendering (`glDrawElementsInstanced`)
- [ ] Occlusion culling
- [ ] LOD system (distance-based mesh swapping)

### 3.2 Debug Visualization
- [x] Wireframe mode (Z key)
- [x] Grid floor with axis colours
- [x] On-screen draw stats (objects, verts, tris, culled)
- [x] Bounding box display (AABB wireframe, X key)
- [ ] Normal visualization (geometry shader or CPU-generated line mesh)
- [ ] Light gizmo (billboard at light position)
- [ ] Performance overlay (draw calls, GPU time)

---

## Phase 4: Editor Tooling

### 4.1 Scene Editor
- [x] Scene save / load JSON (F5 / F9)
- [ ] Property inspector panel (on-screen text or Dear ImGui)
- [ ] Hierarchy / Outliner panel (node tree display)
- [ ] Translate / Rotate / Scale gizmos (rendered handles)
- [ ] Grid snapping

### 4.2 Asset Browser
- [ ] Directory browser (list textures, models)
- [ ] Texture / model preview
- [ ] Drag-and-drop asset import

---

## Phase 5: Animation & Physics (Lower Priority)

### 5.1 Skeletal Animation
- [ ] Skeletal mesh (joints, skin weights)
- [ ] Keyframe animation system
- [ ] Animation blending / interpolation
- [ ] glTF animation loader

### 5.2 Physics
- [ ] Physics world / simulation step
- [ ] Rigid body dynamics (AABB, sphere colliders)
- [ ] Physics-based character controller

### 5.3 Particle System
- [ ] CPU particle emitter (billboarded quads, additive/alpha blend)
- [ ] GPU particles (transform feedback or compute shader)

---

## Phase 6: Cross-Platform (Lower Priority)
- [ ] Linux support (X11 / Wayland)
- [ ] macOS support (Metal via MoltenVK or OpenGL fallback)
- [ ] Headless rendering mode

---

## Technical Debt & Known Issues
- [ ] SSAO normals reconstructed from depth derivatives (dFdx/dFdy) — less accurate at silhouettes vs G-buffer normals
- [ ] No IBL — PBR ambient is a flat approximation; needs diffuse irradiance + specular prefilter cubemaps
- [ ] `ComputeAABB` re-transforms 8 corners every call for AABB debug draw (local AABB cached on Mesh, world AABB recomputed)
- [ ] Shader hot-reload not implemented — must restart to see shader changes
- [ ] Error recovery in GL renderer (currently panics on GL errors)
- [ ] Memory leak audit on long-running sessions
- [ ] ARCHITECTURE.md describes Vulkan backend that no longer exists — update docs
- [ ] Vulkan stub in `renderer/shaders.go` — remove or implement

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `opengl/renderer.go` | Shaders (Phong+PBR dual-path), uniform locations, draw calls, `applyMaterial()` |
| `opengl/shadow.go` | ShadowMap FBO (depth-only, PCF hardware) |
| `opengl/postprocess.go` | HDR FBO, Reinhard tone map, bloom ping-pong, SSAO composite |
| `opengl/ssao.go` | SSAO: 64-sample kernel, noise, SSAO+blur shaders |
| `opengl/skybox.go` | Procedural gradient skybox (inverted cube, xyww depth trick) |
| `opengl/texture.go` | GPU texture upload / delete |
| `renderer/renderer.go` | High-level RenderEngine: shadow pass, scene loop, frustum culling, AABB draw |
| `scene/mesh.go` | Mesh struct, DrawMode, AABB caching, CreateMeshFromData |
| `scene/material.go` | Material (Phong + PBR fields), NewMaterial, NewPBRMaterial, DefaultMaterial |
| `scene/texture.go` | CPU Texture type, PNG/JPEG load |
| `scene/frustum.go` | Frustum planes, AABB, ComputeAABB, IntersectsFrustum |
| `scene/tangents.go` | ComputeTangents (Gram-Schmidt, per-triangle accumulation) |
| `scene/grid.go` | CreateGrid, CreateUnitBoxWireframe (AABB debug) |
| `scene/obj_loader.go` | Wavefront OBJ + MTL loader |
| `scene/gltf_loader.go` | glTF / GLB loader (PBR materials, node hierarchy, tangents) |
| `scene/serialization.go` | JSON scene save/load (SaveScene, LoadScene, ApplyToScene) |
| `scene/primitives.go` | Cube, Sphere, Cylinder, Cone, Pyramid, Torus, Plane |
| `scene/particles.go` | ParticleEmitter, Particle, BlendMode, Update, NewParticleEmitter, NewSmokeEmitter |
| `opengl/particles.go` | ParticleRenderer: billboard shader, dynamic VBO, soft-circle alpha |
| `examples/basic_triangle/main.go` | Demo: 6 shapes (3 Phong + 3 PBR), free camera, all feature toggles |

---

## Texture Unit Layout

| Unit | Sampler | Content |
|------|---------|---------|
| 0 | `albedoTex` | Base colour / albedo (RGB) |
| 1 | `shadowMap` | Directional shadow depth (sampler2DShadow) |
| 2 | `normalTex` | Tangent-space normal map (RGB) |
| 3 | `metallicRoughnessTex` | PBR: G=roughness, B=metallic (glTF) |
| 4 | `emissiveTex` | Emissive colour map (RGB) |

---

## Demo Key Bindings

| Key | Action |
|-----|--------|
| W/A/S/D | Move camera |
| Space / LCtrl | Move up / down |
| Right mouse drag | Look around |
| Z | Toggle wireframe |
| X | Toggle AABB debug boxes |
| P | Toggle PBR ↔ Phong on bottom row of shapes |
| E | Toggle particle emitters (fire / smoke / magic) |
| I | Toggle 400-cube instanced grid (1 draw call) |
| O | Toggle SSAO |
| B | Toggle bloom |
| `[` / `]` | Decrease / increase HDR exposure |
| `-` / `=` | Decrease / increase bloom strength |
| F5 | Save scene to scene.json |
| F9 | Load scene from scene.json |
| ESC | Quit |

---

## Dependencies

| Package | Status | Purpose |
|---------|--------|---------|
| `github.com/go-gl/gl/v4.1-core/gl` | ✅ In use | OpenGL 4.1 bindings |
| `github.com/go-gl/glfw/v3.3/glfw` | ✅ In use | Window + input |
| `github.com/qmuntal/gltf v0.28.0` | ✅ In use | glTF / GLB loader |
| `encoding/json` | ✅ In use | Scene serialisation |
| `image/png`, `image/jpeg` | ✅ In use | Texture loading (stdlib) |
| Physics library | ❌ Not yet | TBD (`jakecoffman/cp` or custom) |

---

## Build Requirements (Windows)
- **Go** — `C:\Program Files\Go\bin\go.exe`; must be on PATH
- **GCC** — `C:\Users\mriga_ijtdono\mingw64\bin\gcc.exe` (WinLibs MinGW-w64 13.2.0); must be on PATH
- MSVC does NOT work — CGO requires GCC-compatible flags
- Build: `go build -o triangle_app.exe ./examples/basic_triangle/`
- New bash session: `export PATH="/c/Users/mriga_ijtdono/mingw64/bin:/c/Program Files/Go/bin:$PATH"`
