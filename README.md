<div align="center">

<img src="./Chibilax.png" alt="Snorlax Engine Logo" width="200"/>

# 🚀 Snorlax Engine
**A Go-based 3D render engine and editor built from scratch.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![OpenGL](https://img.shields.io/badge/OpenGL-4.1-5586A4?logo=opengl)](https://www.opengl.org/)
[![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?logo=windows)](https://microsoft.com/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## 🌟 Snorlax Engine — Full Feature Summary

### Rendering Pipeline

**Core Renderer**
- OpenGL 4.1 backend via `go-gl/gl` + GLFW windowing
- MVP matrix pipeline — `model.Mul(view).Mul(proj)` column-major convention, correct for GLSL
- Model matrix passed separately so normals are transformed in world space (not just MVP)

**Shading**
- Phong shading — diffuse + specular (shininess), per-material properties
- PBR shading (Cook-Torrance GGX) — metallic/roughness workflow, Schlick Fresnel, Smith geometry, GGX NDF
  - Toggle per-material with `mat.UsePBR = true`; `P` key switches at runtime
- Dual-path fragment shader — single shader branches between Phong and PBR

**Lighting**
- Directional light — direction, color, intensity; used for sun + shadow casting
- Point lights (up to 8) — position, color, intensity, range; quadratic attenuation
- Spot lights (up to 4) — inner/outer cone angle, quadratic attenuation
- All lights wired from `scene.Scene.Lights` to shader uniforms each frame via `BeginFrame`

**IBL (Image-Based Lighting)**
- Sky-gradient irradiance — `sampleSkyGradient(dir)` blends zenith/horizon/ground by direction
- PBR ambient: diffuse IBL (kD × irradiance × albedo) + specular IBL (reflected dir, roughness-fade)
- Phong ambient: `sampleSkyGradient(N)` × albedo × 0.35
- Auto-synced when `SetSkyboxColors()` is called; no HDR image file needed

**Textures & Materials**
- Material system — `scene.Material`: Albedo, Specular, Shininess, EmissiveColor
- PBR material — Metallic, Roughness, MetallicRoughnessTexture (unit 3), EmissiveTexture (unit 4)
- Texture loading — PNG/JPEG via stdlib `image`, GPU upload via `UploadTexture()`
- Normal maps — tangent-space TBN in vertex shader, `normalTex` at unit 2
- Tangent/bitangent computed via `ComputeTangents()` with Gram-Schmidt orthogonalization
- Unlit flag — `mat.Unlit = true` skips all lighting (used for grid, AABB boxes)

---

### Post-Processing

**HDR Pipeline**
- HDR FBO — RGBA16F off-screen render target
- Reinhard tone mapping + gamma 2.2 correction on blit
- `[ / ]` — adjust exposure (0.1–5.0)

**Bloom**
- Bright-pass threshold → ping-pong Gaussian blur (half-res) → additive composite
- `B` — toggle; `- / =` — adjust strength

**SSAO**
- 64-sample hemisphere kernel, 4×4 noise texture, depth reconstruction to view-space
- 5×5 box blur pass on raw AO output
- `O` — toggle; strength adjustable via API

**Fog**
- Exponential depth fog: `exp(-density × dist)` blended toward fog color
- Density and color driven by the day/night cycle automatically

---

### Skybox

- Procedural gradient skybox — inverted cube rendered at depth=1.0 (xyww trick)
- Three gradient stops: zenith (overhead) / horizon (eye level) / ground (below)
- Animated by the day/night cycle each frame

---

### Shadow Mapping

- Directional light shadow map — 2048×2048 depth FBO
- PCF 3×3 soft shadows via `sampler2DShadow`
- Orthographic light camera follows the scene camera
- Only directional light is shadowed; point/spot are unshadowed

---

### Scene Graph

**Nodes & Transforms**
- `scene.Node` — position, rotation (quaternion), scale; hierarchical parent/child
- `GetWorldMatrix()` — concatenates transform chain up to root
- `scene.NewScene()` — holds nodes, lights, camera, ambient color

**Primitives**
- `CreateCube`, `CreateSphere`, `CreateCylinder`, `CreateCone`, `CreatePlane`
- `CreateGrid(size, divisions)` — line-mode grid, red X-axis, blue Z-axis, unlit

**Loaders**
- OBJ loader — `scene.LoadOBJ(path)` → `[]*Mesh` with MTL material support
- glTF/GLB loader — `scene.LoadGLTF(path)` → nodes + textures; PBR materials, embedded textures, node hierarchy, TRS transforms

**Frustum Culling**
- `scene.FrustumFromVP(vp)` — Gribb/Hartmann plane extraction
- `AABB.IntersectsFrustum()` — skips draw if fully outside any plane
- `X` key — toggle green wireframe AABB debug visualization

**Scene Serialization**
- `scene.SaveScene(s, path)` / `scene.LoadScene(path)` → JSON
- `F5` — save; `F9` — load

---

### Instanced Rendering

- `DrawMeshInstanced(mesh, []Mat4)` — single `glDrawElementsInstanced` call
- MVP + Model matrices computed on CPU, uploaded as per-instance VBO (32 floats/instance)
- Instance VBO reused with `BufferSubData` if count ≤ capacity
- `I` key — toggles 400-cube demo grid (20×20, one draw call)

---

### Particle System

- CPU-simulated billboarded particles, camera-facing via view matrix rows
- Two blend modes: Alpha (smoke) and Additive (fire, magic)
- Per-emitter: spawn rate, spread cone, color lerp over lifetime, gravity, min/max life/speed/size
- `NewParticleEmitter(max)` — fire defaults; `NewSmokeEmitter(max)` — smoke defaults
- Depth test ON, depth write OFF; rendered into HDR FBO before tone-map/bloom
- `E` key — toggle all emitters

---

### Day/Night Cycle

- 6 palette keyframes: noon → golden hour → dusk → midnight → pre-dawn → sunrise
- Each keyframe holds: zenith/horizon/ground colors, fog color/density, sun color/intensity, ambient
- Linear interpolation between adjacent keyframes with wrap-around
- Sun direction animates as a full arc: `(sin(t·2π), -cos(t·2π), 0.35).Normalize()`
- Apply() pushes everything — sky, fog, sun, ambient — to the renderer each frame
- `N` — pause/resume; `, / .` — slow down / speed up (10–600s per cycle)
- HUD shows current time of day (e.g. 06:30 AM)

---

### Player Controller

- Gravity (-18 m/s²) + jump (7 m/s initial velocity, Space key, debounced)
- Eye height 1.7m above ground plane
- Horizontal movement decoupled from pitch (level strafing regardless of look angle)
- Right-mouse-drag look with yaw/pitch clamped to ±88°

**Building Collision**
- `collBox` (XZ AABB) + `resolvePlayerCollision()` — push-out along axis of minimum penetration
- 7 registered boxes: 4 buildings, 2 walls, fountain bowl
- Applied every frame after gravity/movement, before rendering

---

### HUD & Text Rendering

- Embedded 8×8 bitmap font (96 ASCII chars), uploaded as `GL_RED` atlas texture
- Rendered to the default framebuffer after HDR tone-mapping blit — always readable
- `renderEngine.DrawText(text, x, y, scale, color)` — queued, flushed in `Present()`
- On-screen overlay: FPS, position, draw stats, exposure, bloom, SSAO, PBR, particles, day/night time

---

### Wireframe Mode

- `renderEngine.SetWireframe(true/false)` — toggles `GL_LINE` / `GL_FILL`
- Text and particle passes force `GL_FILL` temporarily so they're unaffected
- `Z` key — toggle

---

### Project Structure

```text
cmd/demo/          ← runnable app (main.go, daynight.go, hud.go)
internal/opengl/   ← GPU backend (Go-enforced private)
core/              ← Color, Vertex, Window, key constants
math/              ← Vec2/3/4, Mat4, Quaternion
scene/             ← Node, Mesh, Camera, lights, loaders, particles
renderer/          ← public RenderEngine API
editor/            ← selection, undo/redo, raycast (unused in demo)
assets/            ← images and future assets
docs/              ← ARCHITECTURE.md, plan.md
```

---

### Build & Run

**Build:** `go build -o triangle_app.exe ./cmd/demo/`
**Module:** `render-engine` | **Platform:** Windows (CGO + GCC + GLFW)
