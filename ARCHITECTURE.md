# 3D Render Engine Architecture

## Overview

This is a 3D render engine built from scratch in Go with Vulkan bindings for cross-platform GPU support. The engine supports AMD, Intel, and NVIDIA GPUs through the Vulkan API.

## Architecture Layers

### 1. Math Library (`math/`)
Core mathematical types and operations:
- **Vec2, Vec3, Vec4**: 2D/3D/4D vectors with full arithmetic operations
- **Mat4**: 4x4 matrices for transformations, projection, and view
- **Quaternion**: Rotation representation with slerp interpolation

Key features:
- SIMD-friendly layout
- Full test coverage
- Common operations: dot, cross, normalize, lerp, slerp
- Matrix operations: multiplication, inversion, look-at, perspective, orthographic

### 2. Core Types (`core/`)
Fundamental data structures:
- **Vertex**: Position, normal, UV, color, tangent, bitangent
- **Transform**: Position, rotation (quaternion), scale with matrix caching
- **Window**: GLFW-based windowing with input handling
- **Color**: RGBA color representation

### 3. Vulkan Backend (`vulkan/`)
Low-level Vulkan API wrapper using CGO:

#### Instance Management (`instance.go`)
- Vulkan instance creation
- Validation layers and debug callbacks
- Extension enumeration

#### Device Management (`device.go`)
- Physical device enumeration and selection
- Queue family detection (graphics, present, compute)
- Logical device creation
- Command pool management

#### Swapchain (`swapchain.go`)
- Swapchain creation and recreation
- Surface format selection
- Present mode configuration (FIFO, Mailbox, etc.)
- Framebuffer management

#### Pipeline (`pipeline.go`)
- Graphics pipeline creation
- Shader module loading (SPIR-V)
- Vertex input configuration
- Rasterization state
- Depth/stencil configuration
- Blend state

#### Memory Management (`buffer.go`)
- Buffer creation and allocation
- Device local vs host visible memory
- Image creation and views
- Staging buffers for GPU upload
- Memory type selection

#### Command Recording (`command.go`)
- Command buffer allocation
- Render pass recording
- Pipeline binding
- Vertex/Index buffer binding
- Draw commands
- Image layout transitions

#### Synchronization (`synchronization.go`)
- Semaphores for GPU-GPU synchronization
- Fences for CPU-GPU synchronization
- Queue submission

#### Descriptor Management (`descriptors.go`)
- Descriptor set layouts
- Descriptor pools
- Uniform buffer binding
- Image/sampler binding

#### High-Level Renderer (`renderer.go`)
- Frame management (acquire, render, present)
- Per-frame resource management
- Viewport and scissor configuration
- Window resize handling

### 4. Scene Management (`scene/`)

#### Node (`node.go`)
- Hierarchical scene graph
- Transform hierarchy with world matrix caching
- Visibility control
- Mesh attachment

#### Mesh (`mesh.go`)
- Vertex and index data
- GPU buffer management
- Primitive generation (triangle, quad, cube)

#### Camera (`camera.go`)
- Perspective and orthographic projection
- Look-at functionality
- Orbit camera for inspection
- View frustum culling support

#### Scene (`scene.go`)
- Root node management
- Active camera
- Light management (directional, point, spot)
- Scene traversal for rendering

### 5. Renderer (`renderer/`)

#### Shaders (`shaders.go`)
- GLSL to SPIR-V compilation support
- Default shader implementations
- Hardcoded SPIR-V for fallback

#### Render Engine (`renderer.go`)
- High-level rendering API
- Scene rendering
- Camera matrix updates
- Object submission and drawing

## GPU Support

The engine uses Vulkan, providing support for:

| Vendor | Support | Notes |
|--------|---------|-------|
| NVIDIA | Full | GeForce GTX 600+, RTX series |
| AMD | Full | Radeon HD 7000+, RX series |
| Intel | Full | HD 5000+, UHD, Xe Graphics |
| Apple | Via MoltenVK | Translates Vulkan to Metal |

## Coordinate System

Right-handed coordinate system:
- **+X**: Right
- **+Y**: Up
- **+Z**: Forward (into screen)

## Rendering Pipeline

1. **Application** → Scene update, input handling
2. **Culling** → Frustum and occlusion culling
3. **Render Queue** → Sort by material/shader
4. **Command Recording** → Record draw commands
5. **Submission** → Submit to graphics queue
6. **Presentation** → Present to swapchain

## Performance Considerations

- **Persistent mapped buffers** for uniform data
- **Command buffer reuse** when possible
- **Descriptor set caching**
- **Frustum culling** to reduce draw calls
- **Instancing** support for repeated geometry

## Building

### Requirements
- Go 1.21+
- Vulkan SDK 1.2+
- C compiler (GCC/MSVC/Clang)
- GLFW3

### Windows
```batch
# Install Vulkan SDK from https://vulkan.lunarg.com/
# Run from Visual Studio Developer Command Prompt
build.bat
```

### Linux
```bash
sudo apt-get install libvulkan-dev libglfw3-dev
go build ./...
```

### macOS
```bash
brew install glfw
go build ./...
```

## Future Enhancements

- [ ] Compute shaders
- [ ] Deferred rendering
- [ ] PBR materials
- [ ] Shadow mapping
- [ ] Post-processing effects
- [ ] Multi-threaded command buffer recording
- [ ] Texture loading (PNG, JPG, etc.)
- [ ] Model loading (GLTF, OBJ)
- [ ] Animation system
- [ ] Particle system
- [ ] GUI integration
