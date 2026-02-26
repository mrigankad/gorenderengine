package main

import (
	"fmt"
	stdmath "math"
	"time"

	"render-engine/core"
	"render-engine/math"
	"render-engine/renderer"
	"render-engine/scene"
)

// CameraController handles keyboard and mouse input for camera movement
type CameraController struct {
	moveSpeed      float32
	lookSpeed      float32
	lastMouseX     float64
	lastMouseY     float64
	firstMouse     bool
	rightMouseDown bool
	yaw            float32
	pitch          float32
}

func NewCameraController() *CameraController {
	return &CameraController{
		moveSpeed:  5.0,
		lookSpeed:  0.005,
		firstMouse: true,
		yaw:        -90.0,
		pitch:      0.0,
	}
}

func (cc *CameraController) Update(window *core.Window, camera *scene.Camera, deltaTime float32) {
	// Check for right mouse button
	cc.rightMouseDown = window.IsMouseButtonPressed(1) // 1 = right mouse button

	// Mouse look (only when right mouse button is pressed)
	if cc.rightMouseDown {
		mouseX, mouseY := window.GetCursorPos()

		if cc.firstMouse {
			cc.lastMouseX = mouseX
			cc.lastMouseY = mouseY
			cc.firstMouse = false
		}

		offsetX := float32(mouseX - cc.lastMouseX)
		offsetY := float32(cc.lastMouseY - mouseY) // Inverted Y

		cc.yaw += offsetX * cc.lookSpeed
		cc.pitch += offsetY * cc.lookSpeed

		// Clamp pitch
		if cc.pitch > 89.0 {
			cc.pitch = 89.0
		}
		if cc.pitch < -89.0 {
			cc.pitch = -89.0
		}

		cc.lastMouseX = mouseX
		cc.lastMouseY = mouseY
	} else {
		cc.firstMouse = true
	}

	// Calculate forward and right vectors from yaw/pitch
	yawRad := cc.yaw * stdmath.Pi / 180.0
	pitchRad := cc.pitch * stdmath.Pi / 180.0

	forward := math.Vec3{
		X: float32(stdmath.Cos(float64(yawRad)) * stdmath.Cos(float64(pitchRad))),
		Y: float32(stdmath.Sin(float64(pitchRad))),
		Z: float32(stdmath.Sin(float64(yawRad)) * stdmath.Cos(float64(pitchRad))),
	}.Normalize()

	right := math.Vec3{X: 1, Y: 0, Z: 0}
	right = math.Vec3{
		X: float32(stdmath.Cos(float64(yawRad - stdmath.Pi/2))),
		Y: 0,
		Z: float32(stdmath.Sin(float64(yawRad - stdmath.Pi/2))),
	}.Normalize()

	up := forward.Cross(right).Normalize()

	// Keyboard movement
	movement := math.Vec3{}

	if window.IsKeyPressed(core.KeyW) {
		movement = movement.Add(forward.Mul(cc.moveSpeed * deltaTime))
	}
	if window.IsKeyPressed(core.KeyS) {
		movement = movement.Add(forward.Mul(-cc.moveSpeed * deltaTime))
	}
	if window.IsKeyPressed(core.KeyD) {
		movement = movement.Add(right.Mul(cc.moveSpeed * deltaTime))
	}
	if window.IsKeyPressed(core.KeyA) {
		movement = movement.Add(right.Mul(-cc.moveSpeed * deltaTime))
	}
	if window.IsKeyPressed(core.KeySpace) {
		movement = movement.Add(math.Vec3Up.Mul(cc.moveSpeed * deltaTime))
	}
	if window.IsKeyPressed(core.KeyLeftControl) {
		movement = movement.Add(math.Vec3Up.Mul(-cc.moveSpeed * deltaTime))
	}

	// Update camera position
	newPos := camera.Position.Add(movement)
	camera.SetPosition(newPos)

	// Update camera look-at target
	target := newPos.Add(forward)
	camera.LookAt(target, up)
}

func main() {
	fmt.Println("Starting shapes showcase...")

	windowConfig := core.DefaultWindowConfig()
	windowConfig.Title = "Render Engine - Shapes"
	windowConfig.Width = 1280
	windowConfig.Height = 720

	window, err := core.NewWindow(windowConfig)
	if err != nil {
		fmt.Printf("Failed to create window: %v\n", err)
		return
	}
	defer window.Destroy()

	renderEngine, err := renderer.NewRenderEngine(window)
	if err != nil {
		fmt.Printf("Failed to create render engine: %v\n", err)
		return
	}
	defer renderEngine.Destroy()

	// Scene setup
	s := scene.NewScene()

	camera := scene.NewCamera(float32(stdmath.Pi)/3, 16.0/9.0, 0.1, 1000.0)
	camera.SetPosition(math.Vec3{X: 0, Y: 3, Z: 8})
	camera.LookAt(math.Vec3{X: 0, Y: 0, Z: 0}, math.Vec3Up)
	s.SetCamera(camera)

	// Create and position shapes in a grid pattern around the center
	shapes := []struct {
		name     string
		mesh     *scene.Mesh
		position math.Vec3
	}{
		// Top row
		{"Cube", scene.CreateCube(1.0), math.Vec3{X: -4, Y: 1, Z: 0}},
		{"Sphere", scene.CreateSphere(0.8, 24, 12), math.Vec3{X: 0, Y: 1, Z: 0}},
		{"Cylinder", scene.CreateCylinder(0.6, 1.5, 16), math.Vec3{X: 4, Y: 1, Z: 0}},

		// Bottom row
		{"Cone", scene.CreateCone(0.8, 1.5, 16), math.Vec3{X: -4, Y: -1.5, Z: 0}},
		{"Pyramid", scene.CreatePyramid(1.5, 1.5), math.Vec3{X: 0, Y: -1.5, Z: 0}},
		{"Torus", scene.CreateTorus(1.0, 0.3, 16, 8), math.Vec3{X: 4, Y: -1.5, Z: 0}},
	}

	shapeNodes := make([]*scene.Node, len(shapes))
	for i, shape := range shapes {
		defer shape.mesh.Destroy()

		node := scene.NewNode(shape.name)
		node.Mesh = shape.mesh
		node.SetPosition(shape.position)
		s.AddNode(node)
		shapeNodes[i] = node

		fmt.Printf("Added %s at position (%.1f, %.1f, %.1f)\n",
			shape.name, shape.position.X, shape.position.Y, shape.position.Z)
	}

	light := &scene.Light{
		Type:      scene.LightTypeDirectional,
		Direction: math.Vec3{X: 0.5, Y: -1, Z: -0.5}.Normalize(),
		Color:     core.ColorWhite,
		Intensity: 1.0,
	}
	s.AddLight(light)

	renderEngine.SetScene(s)

	// Initialize camera controller
	camController := NewCameraController()

	frameCount := 0
	lastTime := time.Now()
	deltaTime := float32(0.016) // 60 FPS default

	fmt.Println("Starting render loop... (ESC to exit)")
	fmt.Println("Camera Controls:")
	fmt.Println("  WASD - Move forward/backward/strafe")
	fmt.Println("  Space/Ctrl - Move up/down")
	fmt.Println("  Right Mouse Drag - Look around")

	for !window.ShouldClose() {
		window.PollEvents()

		if window.IsKeyPressed(core.KeyEscape) {
			break
		}

		// Update camera with controller
		camController.Update(window, camera, deltaTime)

		// Spin all shapes
		for _, node := range shapeNodes {
			node.Rotate(math.Vec3Up, 0.016)
		}

		if err := renderEngine.Render(); err != nil {
			width, height := window.GetFramebufferSize()
			if width > 0 && height > 0 {
				renderEngine.Resize(uint32(width), uint32(height))
			}
		}

		frameCount++
		now := time.Now()
		elapsed := now.Sub(lastTime)
		if elapsed.Seconds() >= 1.0 {
			window.SetTitle(fmt.Sprintf("%s - FPS: %d", windowConfig.Title, frameCount))
			frameCount = 0
			lastTime = now
		}
		deltaTime = float32(elapsed.Seconds())
	}

	renderEngine.WaitIdle()
	fmt.Println("Exiting...")
}
