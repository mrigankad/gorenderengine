# Sonorlax Engine - Development Plan

## Current State (Completed)
- ✅ OpenGL 4.1 backend functional
- ✅ Scene graph system with Transform, Node hierarchy
- ✅ 7 primitive shapes (Cube, Sphere, Cylinder, Cone, Pyramid, Torus, Plane)
- ✅ Camera system with perspective projection and LookAt
- ✅ Basic directional lighting
- ✅ Editor framework (input, selection, undo/redo, raycast)
- ✅ Windows-only build (CGO + GLFW)
- ✅ Free camera controller (WASD + mouse look)
- ✅ Debug overlay (FPS, position, angles in console/title)
- ✅ On-screen control hints and information

---

## Immediate Next Steps (Ready to Start)
1. **ImGui Integration** - Add Dear ImGui for professional debug UI
2. **Texture System** - Load PNG/JPG textures and apply to shapes
3. **Model Importer** - glTF format loader for 3D models
4. **Material System** - Basic Phong/PBR material properties
5. **Performance Stats** - Memory usage, draw call count display

---

## Phase 1: Asset Pipeline & Materials (High Priority)
### 1.1 Texture System
- [ ] Texture loading (PNG, JPG via image libraries)
- [ ] Texture sampler management in OpenGL
- [ ] UV coordinate support in vertex layout
- [ ] Texture binding in shader system
- [ ] Normal map support

### 1.2 Material System
- [ ] Material definition (albedo, normal, roughness, metallic)
- [ ] PBR shader implementation
- [ ] Material cache/asset manager
- [ ] Phong fallback for basic materials

### 1.3 Model Loading
- [ ] glTF/glb format loader
- [ ] OBJ format loader (simpler)
- [ ] Automatic mesh import to scene
- [ ] Embedded material loading from model files

---

## Phase 2: Advanced Rendering (Medium Priority)
### 2.1 Shadows
- [ ] Shadow mapping (directional light shadows)
- [ ] Shadow map generation pass
- [ ] PCF filtering for soft shadows
- [ ] Cascaded shadow maps for large scenes

### 2.2 Post-Processing
- [ ] Post-process pipeline architecture
- [ ] Bloom effect
- [ ] Tone mapping (HDR support)
- [ ] SSAO (screen-space ambient occlusion)
- [ ] Motion blur

### 2.3 Advanced Lighting
- [ ] Point lights with attenuation
- [ ] Spot lights with cone angles
- [ ] Light batching/culling
- [ ] Environment mapping (cubemaps)

---

## Phase 3: Performance & Optimization (Medium Priority)
### 3.1 Culling & LOD
- [ ] Frustum culling for visible objects
- [ ] Occlusion culling system
- [ ] LOD (level of detail) system for distant objects
- [ ] Instanced rendering for repeated meshes

### 3.2 Memory & GPU Optimization
- [ ] Mesh/buffer pooling
- [ ] Batch rendering optimization
- [ ] Reduce draw calls via batching
- [ ] GPU memory profiling

### 3.3 Multi-threading
- [ ] Asset loading on background thread
- [ ] Physics simulation thread (prep for Phase 4)
- [ ] Command buffer generation parallel

---

## Phase 4: Animation & Physics (Lower Priority)
### 4.1 Skeletal Animation
- [ ] Skeletal mesh data structure
- [ ] Bone hierarchy and transforms
- [ ] Keyframe animation system
- [ ] Animation blending/interpolation
- [ ] glTF animation loader

### 4.2 Physics Engine
- [ ] Physics world/simulation step
- [ ] Rigid body dynamics (colliders, mass, velocity)
- [ ] Basic collision detection (AABB, sphere, mesh)
- [ ] Physics-based character controller

### 4.3 Particle System
- [ ] Particle emitter architecture
- [ ] GPU-based particle simulation
- [ ] Various emitter shapes
- [ ] Particle effects (sparks, smoke, explosions)

---

## Phase 5: Editor & Tooling (Ongoing)
### 5.1 Scene Editor Enhancements
- [ ] Property inspector panel
- [ ] Hierarchy/Outliner panel
- [ ] Gizmos (translate, rotate, scale)
- [ ] Grid/snapping system
- [ ] Scene save/load (JSON/binary format)

### 5.2 Debug Visualization
- [ ] Wireframe mode toggle
- [ ] Light gizmo visualization
- [ ] Bounding box display
- [ ] Normal visualization
- [ ] Performance overlay (FPS, draw calls, memory)

### 5.3 Asset Browser
- [ ] Asset directory browser
- [ ] Texture preview
- [ ] Model preview
- [ ] Drag-and-drop asset import

---

## Phase 6: Cross-Platform & Distribution (Lower Priority)
### 6.1 Cross-Platform Build
- [ ] Linux support (X11/Wayland)
- [ ] macOS support (Cocoa)
- [ ] Platform-agnostic window abstraction
- [ ] Headless rendering mode

### 6.2 Packaging
- [ ] Executable distribution
- [ ] Runtime asset bundling
- [ ] Shader compilation pipeline
- [ ] Release build optimization

---

## Phase 7: Advanced Features (Nice-to-Have)
### 7.1 Advanced Shaders
- [ ] Parallax mapping
- [ ] Displacement mapping
- [ ] Subsurface scattering
- [ ] Hair/fur shading

### 7.2 Advanced Graphics
- [ ] Deferred rendering pipeline
- [ ] Screen-space reflections
- [ ] Global illumination (baked + real-time)
- [ ] Volumetric rendering

### 7.3 VR/Multi-Viewport
- [ ] Multi-camera rendering
- [ ] Stereo rendering for VR
- [ ] Mirror/reflection surfaces
- [ ] Split-screen support

---

## Recommended Short-Term Tasks (Next 1-2 weeks)
1. **Texture System** - Load and display textures on primitives
2. **Model Loader** - Import a simple glTF model
3. **Mesh Inspector** - Visualize mesh data in editor
4. **Grid & Snapping** - Editor usability improvements
5. **Material Editor** - UI for tweaking material properties

---

## Technical Debt & Known Issues
- [ ] Vulkan backend kept for reference; consider if should be removed
- [ ] Memory leak testing on long-running sessions
- [ ] Shader hot-reload capability
- [ ] Error recovery in renderer
- [ ] Console/logging system improvements

---

## Dependencies to Evaluate
- **Asset Loading**: `go-gl/gltf`, `image` stdlib (textures)
- **Physics**: `go-echarts` or custom simple solver
- **Math**: Current `math/` package sufficient; may need quaternion improvements
- **Serialization**: Consider `encoding/json` or `gob` for scene saves
- **UI Framework**: Consider `imgui-go` for debug UI if needed

