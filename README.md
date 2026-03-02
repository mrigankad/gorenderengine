<div align="center">
  <img src="./Chibilax.png" alt="Snorlax Engine Logo" width="200"/>

  # 🚀 Snorlax Engine
  **A Go-based 3D render engine and game development framework built from scratch.**

  [![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
  [![OpenGL](https://img.shields.io/badge/OpenGL-4.1-5586A4?logo=opengl)](https://www.opengl.org/)
  [![Platform](https://img.shields.io/badge/Platform-Windows-0078D6?logo=windows)](https://microsoft.com/)
  [![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
</div>

<br/>

## 📖 Table of Contents

- [About Snorlax Engine](#-about)
- [Key Features](#-key-features)
- [Getting Started](#-getting-started)
- [Usage Example](#-usage-example)
- [Project Architecture](#-project-architecture)
- [Roadmap & Current State](#-roadmap--current-state)
- [License](#-license)

---

## 🧐 About

**Snorlax Engine** is a custom-built 3D rendering engine and editor toolkit written entirely in Go. Leveraging **OpenGL 4.1**, it aims to provide a reliable, from-scratch foundation for building games. It focuses on physically based rendering (PBR), flexible scene graphs, custom math libraries (zero-dependency Quaternions/Vectors/Matrices), and an extensible architectural design.

---

## ✨ Key Features

### 🎨 Rendering & Materials
* **OpenGL 4.1 Backend**: Fast, low-level rendering loop powered by `go-gl/gl` + `GLFW` windowing.
* **Dual-Path Shading Pipeline**: Supports both legacy **Phong shading** and modern **Cook-Torrance PBR** (Metallic/Roughness, Schlick Fresnel, Smith geometry, GGX NDF).
* **Dynamic Lighting**: Directional lights with PCF 3x3 soft shadows, configurable point lights (up to 8, quadratic attenuation), and spot lights (up to 4).
* **Image-Based Lighting (IBL)**: Procedural sky-gradient irradiance for dynamic ambient environment lighting without external HDR files.
* **Advanced Texturing**: GPU-uploaded normal mapping (Gram-Schmidt Tangent Space), and dedicated emissive/metallic/roughness maps.

### 🎥 Post-Processing & Visual FX
* **HDR Pipeline**: RGBA16F off-screen FBO with Reinhard tone mapping and Gamma 2.2 correction routines.
* **Bloom**: Ping-pong Gaussian blur (half-res) additive composite driven by bright-pass thresholds.
* **SSAO**: Screen-Space Ambient Occlusion with 64-sample hemisphere kernels, 4x4 noise, and 5x5 box blur smoothing.
* **Dynamic Environments**: Procedural Day/Night cycle driving zenith/horizon gradients, exponential depth fog, and sun positioning.
* **Particle System**: CPU-simulated billboard particles featuring alpha/additive blend modes, depth testing, gravity, and lifetime lerping.

### 🏗️ Scene Graph & Optimizations
* **Hierarchical Nodes**: Comprehensive scene graph (`scene.Node`) managing parent/child transforms, rotations (Quaternions), and scale.
* **Frustum Culling**: Gribb/Hartmann plane extraction paired with AABB intersection filtering.
* **Instanced Rendering**: `glDrawElementsInstanced` implementations using CPU-computed VBO instances for massive draw call reduction.
* **Asset Loaders**: Built-in support for Wavefront `.obj` (with `.mtl`) and `.gltf / .glb` with embedded textures and full hierarchy preservation.
* **Scene Serialization**: Save and load full scene states (Nodes, Lights, Materials) via JSON.

### 🕹️ Gameplay & Tooling 
* **Built-in HUD text rendering** utilizing an embedded 8x8 ASCII bitmap font atlas.
* **Player Controller** with physics-aware gravity (-18 m/s²), jump momentum, and building-pushout collision detection.
* **Debug Visualizations**: Wireframe mode (Z), AABB bounding boxes (X), draw stats overlay, and real-time PBR/Phong toggles.

---

## 🚀 Getting Started

### Prerequisites

To compile the Snorlax Engine natively, you will need:
- **Go 1.21** or newer
- **C compiler** (`MinGW-w64` or Visual Studio on Windows) for CGO support
- **GLFW3** development libraries

### Windows Setup

The engine is currently highly optimized for Windows development using MSVC or MinGW.

1. Ensure Go is installed and present in your system PATH.
2. Clone the repository to your local machine.
3. Open a developer command prompt and use the provided batch script:

```batch
build.bat
```

> **Note:** Alternatively, you can build the main executable directly using Go:
> ```bash
> go build -o triangle_app.exe ./cmd/demo/
> ```

---

## 💻 Usage Example

Creating a basic window, setting up a camera, and rendering a rotating 3D object is simple:

```go
package main

import (
    "render-engine/core"
    "render-engine/opengl"
    "render-engine/renderer"
    "render-engine/scene"
    "render-engine/math"
)

func main() {
    // 1. Initialize Window
    window, _ := core.NewWindow(core.DefaultWindowConfig())
    defer window.Destroy()
    
    // 2. Initialize Render Engine
    backend := opengl.NewBackend()
    engine, _ := renderer.NewRenderEngine(window, backend)
    defer engine.Destroy()
    
    // 3. Setup Scene & Camera
    s := scene.NewScene()
    camera := scene.NewCamera(1.0472, 16.0/9.0, 0.1, 1000.0)
    s.SetCamera(camera)
    
    // 4. Create and Add Objects
    cube, _ := scene.CreateCube()
    node := scene.NewNode("MyCube")
    node.Mesh = cube
    s.AddNode(node)
    
    engine.SetScene(s)
    
    // 5. Main Loop
    for !window.ShouldClose() {
        window.PollEvents()
        
        // Rotate the cube slowly each frame
        node.Rotate(math.Vec3Up, 0.01)
        
        engine.Render()
        engine.Present()
    }
}
```

---

## 📂 Project Architecture

```text
├── cmd/demo/          # Runnable application entrypoints (main.go, demo logic)
├── internal/opengl/   # Core GPU backend & native GL logic (Go-enforced private)
├── core/              # Foundational types (Color, Vertex, Window interface)
├── math/              # High-performance Vec2/3/4, Mat4, Quaternion library
├── scene/             # Scene graph, Primitives, Camera, Lights, Loaders
├── renderer/          # High-level public RenderEngine API
├── editor/            # Interactive editor tools (raycast, undo/redo)
├── assets/            # Static assets (textures, objects, fonts)
└── docs/              # Development plans and architectural maps
```

---

## 🚧 Roadmap & Current State

While the rendering core is highly capable, several massive systems are currently missing before the engine can be used for full production games.

### High-Priority Implementations

Based on our current development cycle, this is our targeted roadmap:

| Priority | Feature / Module | Purpose |
|:---:|:---|:---|
| **1** | **Game States & Main Menu** | Implements a primary application state machine (Menu → Init → Play → Pause). |
| **2** | **Player Interaction** | Raycasting and trigger-volumes to allow picking up, opening, or activating objects. |
| **3** | **Scene Editor Tooling** | Wiring up the existing editor backend to UI Gizmos (Translate/Rotate/Scale objects). |
| **4** | **Audio Subsystem** | Integrating a spatial audio mixer (OpenAL/miniaudio) for SFX and Music. |
| **5** | **Mesh-based Collision** | Replacing simple box-AABB collisions with precise triangle mesh physics. |
| **6** | **NPCs & Simple AI** | NavMeshes or A* pathfinding for rudimentary agent behavior and town population. |
| **7** | **Skeletal Animation** | Skinned mesh rendering and bone matrix calculations for animated characters. |
| **8** | **Terrain Generation** | Heightmap chunking and LOD (Level of Detail) systems for large outdoor environments. |

### Technical Debt / Missing Features
* **Rendering Deficits:** Point/Spot lights lack shadow map support. No volumetric lighting, true reflections, or distinct water shaders.
* **System Deficits:** No real physics bodies (Rigidbodies), asset hot-reloading is absent, and the module name still defaults to `render-engine`.

---

## 📄 License

This project is licensed under the **MIT License**.

- Graphics API bindings via [go-gl/gl](https://github.com/go-gl/gl)
- Windowing via [go-gl/glfw](https://github.com/go-gl/glfw)
