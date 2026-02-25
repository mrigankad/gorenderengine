package main

import (
	"fmt"
	"math"
	"time"
	
	"render-engine/core"
	"render-engine/math"
	"render-engine/renderer"
	"render-engine/scene"
	"render-engine/vulkan"
)

func main() {
	// Create window
	windowConfig := core.DefaultWindowConfig()
	windowConfig.Title = "Render Engine - Basic Triangle"
	windowConfig.Width = 1280
	windowConfig.Height = 720
	
	window, err := core.NewWindow(windowConfig)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}
	defer window.Destroy()
	
	// Create render engine
	renderEngine, err := renderer.NewRenderEngine(window)
	if err != nil {
		fmt.Printf("Failed to create render engine: %v\n", err)
		return
	}
	defer renderEngine.Destroy()
	
	// Create scene
	s := scene.NewScene()
	
	// Create camera
	camera := scene.NewCamera(float32(math.Pi)/3, 16.0/9.0, 0.1, 1000.0)
	camera.SetPosition(math.Vec3{X: 0, Y: 0, Z: 3})
	s.SetCamera(camera)
	
	// Create a triangle mesh
	device := renderEngine.Renderer.Device
	
	// Create triangle
	vertices := []core.Vertex{
		{
			Position: math.Vec3{X: 0, Y: -0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 0.5, Y: 0},
			Color:    core.Color{R: 1, G: 0, B: 0, A: 1},
		},
		{
			Position: math.Vec3{X: 0.5, Y: 0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 1, Y: 1},
			Color:    core.Color{R: 0, G: 1, B: 0, A: 1},
		},
		{
			Position: math.Vec3{X: -0.5, Y: 0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 0, Y: 1},
			Color:    core.Color{R: 0, G: 0, B: 1, A: 1},
		},
	}
	indices := []uint32{0, 1, 2}
	
	triangleMesh, err := scene.CreateMeshFromData(device, "Triangle", vertices, indices)
	if err != nil {
		fmt.Printf("Failed to create triangle mesh: %v\n", err)
		return
	}
	defer triangleMesh.Destroy(device)
	
	// Create triangle node
	triangleNode := scene.NewNode("Triangle")
	triangleNode.Mesh = triangleMesh
	s.AddNode(triangleNode)
	
	// Add light
	light := &scene.Light{
		Type:      scene.LightTypeDirectional,
		Direction: math.Vec3{X: 0.5, Y: -1, Z: -0.5}.Normalize(),
		Color:     core.ColorWhite,
		Intensity: 1.0,
	}
	s.AddLight(light)
	
	renderEngine.SetScene(s)
	
	// Main loop
	frameCount := 0
	lastTime := time.Now()
	
	fmt.Println("Starting render loop...")
	fmt.Println("Controls:")
	fmt.Println("  ESC - Exit")
	
	for !window.ShouldClose() {
		window.PollEvents()
		
		// Handle input
		if window.IsKeyPressed(core.KeyEscape) {
			break
		}
		
		// Rotate triangle
		rotationSpeed := float32(1.0)
		triangleNode.Rotate(math.Vec3Up, rotationSpeed*0.016)
		
		// Render
		if err := renderEngine.Render(); err != nil {
			// Handle resize
			width, height := window.GetFramebufferSize()
			if width > 0 && height > 0 {
				renderEngine.Resize(uint32(width), uint32(height))
			}
		}
		
		// Calculate FPS
		frameCount++
		now := time.Now()
		if now.Sub(lastTime).Seconds() >= 1.0 {
			window.SetTitle(fmt.Sprintf("%s - FPS: %d", windowConfig.Title, frameCount))
			frameCount = 0
			lastTime = now
		}
	}
	
	renderEngine.WaitIdle()
	fmt.Println("Exiting...")
}

// Helper to create a cube for more complex demo
func createCubeDemo(renderEngine *renderer.RenderEngine, s *scene.Scene) error {
	device := renderEngine.Renderer.Device
	
	// Create multiple cubes
	for i := 0; i < 5; i++ {
		cubeMesh, err := scene.CreateCube(device, 0.5)
		if err != nil {
			return err
		}
		
		cubeNode := scene.NewNode(fmt.Sprintf("Cube%d", i))
		cubeNode.Mesh = cubeMesh
		cubeNode.SetPosition(math.Vec3{X: float32(i-2) * 1.5, Y: 0, Z: 0})
		s.AddNode(cubeNode)
	}
	
	return nil
}
