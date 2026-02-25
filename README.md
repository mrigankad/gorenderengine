# 3D Render Engine in Go

A cross-platform 3D rendering engine built from scratch in Go with Vulkan support for AMD, Intel, and NVIDIA GPUs.

## Features

- **Cross-Platform GPU Support**: Works with AMD, Intel, and NVIDIA GPUs via Vulkan
- **Modern Graphics API**: Uses Vulkan for high-performance, low-overhead rendering
- **Math Library**: Complete vector, matrix, and quaternion math library
- **Scene Graph**: Hierarchical scene management with transform hierarchy
- **Camera System**: Perspective/orthographic cameras with orbit controls
- **Mesh Rendering**: Vertex and index buffer management with GPU upload
- **Shader Support**: SPIR-V shader loading with pipeline state management

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                   APPLICATION LAYER                          │
├──────────────────────────────────────────────────────────────┤
│                   SCENE GRAPH (Go)                           │
│         (Entities, Transforms, Cameras, Lights)             │
├──────────────────────────────────────────────────────────────┤
│              RENDERER (Go)                                   │
│    (Materials, Meshes, Textures, Render Queue)              │
├──────────────────────────────────────────────────────────────┤
│              VULKAN BACKEND (Go + CGO)                       │
│    (Instance, Device, Swapchain, Pipeline, Commands)        │
├──────────────────────────────────────────────────────────────┤
│                    VULKAN DRIVER                             │
│       (AMD, Intel, NVIDIA drivers via Vulkan)               │
└──────────────────────────────────────────────────────────────┘
```

## Project Structure

```
├── math/           # Vector, Matrix, Quaternion math library
├── core/           # Core types (Vertex, Color, Transform, Window)
├── vulkan/         # Vulkan graphics API wrapper
├── renderer/       # High-level rendering interface
├── scene/          # Scene graph and camera system
└── examples/       # Example applications
```

## Requirements

- Go 1.21+
- Vulkan SDK (1.2+)
- C compiler (GCC on Linux/Windows, Xcode on macOS)
- GLFW3

### Platform-Specific Setup

#### Windows
1. Install [Vulkan SDK](https://vulkan.lunarg.com/)
2. Install MinGW-w64 or use Visual Studio
3. Run from Visual Studio Developer Command Prompt:
   ```batch
   build.bat
   ```

#### Linux
```bash
# Ubuntu/Debian
sudo apt-get install libvulkan-dev libglfw3-dev

# Build
go build ./...
```

#### macOS
```bash
# Install MoltenVK (usually comes with Vulkan SDK)
brew install glfw

# Build
go build ./...
```

## Usage

### Basic Example

```go
package main

import (
    "render-engine/core"
    "render-engine/renderer"
    "render-engine/scene"
)

func main() {
    // Create window
    window, _ := core.NewWindow(core.DefaultWindowConfig())
    defer window.Destroy()
    
    // Create render engine
    engine, _ := renderer.NewRenderEngine(window)
    defer engine.Destroy()
    
    // Create scene
    s := scene.NewScene()
    camera := scene.NewCamera(1.0472, 16.0/9.0, 0.1, 1000.0)
    s.SetCamera(camera)
    
    // Add objects
    triangle, _ := scene.CreateTriangle(engine.Renderer.Device)
    node := scene.NewNode("Triangle")
    node.Mesh = triangle
    s.AddNode(node)
    
    engine.SetScene(s)
    
    // Main loop
    for !window.ShouldClose() {
        window.PollEvents()
        node.Rotate(math.Vec3Up, 0.01)
        engine.Render()
    }
}
```

## GPU Support

The engine uses Vulkan, which provides native support for:
- **AMD**: Radeon GPUs (GCN and RDNA architectures)
- **NVIDIA**: GeForce/Quadro/RTX GPUs (Kepler and newer)
- **Intel**: HD/UHD/Xe Graphics (Broadwell and newer)
- **Apple**: Via MoltenVK (translates Vulkan to Metal)

## Math Library

The math library provides:
- `Vec2`, `Vec3`, `Vec4`: 2D/3D/4D vectors
- `Mat4`: 4x4 transformation matrices
- `Quaternion`: Rotation representation
- Common operations: addition, subtraction, dot product, cross product, normalization
- Matrix operations: multiplication, inversion, look-at, perspective, orthographic

## License

MIT License - See LICENSE file for details.

## Acknowledgments

- [Vulkan](https://www.vulkan.org/) - Graphics API
- [GLFW](https://www.glfw.org/) - Windowing library
- [Go-GL](https://github.com/go-gl/glfw) - Go GLFW bindings
