package scene

import (
	"render-engine/core"
	"render-engine/math"
)

// Scene manages a collection of nodes and the active camera
type Scene struct {
	Root     *Node
	Camera   *Camera
	Lights   []*Light
	Ambient  core.Color
	SkyColor core.Color
}

// Light types
const (
	LightTypeDirectional = iota
	LightTypePoint
	LightTypeSpot
)

// Light represents a light source
type Light struct {
	Type       int
	Position   math.Vec3
	Direction  math.Vec3
	Color      core.Color
	Intensity  float32
	Range      float32
	SpotAngle  float32
}

func NewScene() *Scene {
	return &Scene{
		Root:     NewNode("Root"),
		Lights:   make([]*Light, 0),
		Ambient:  core.Color{R: 0.2, G: 0.2, B: 0.2, A: 1.0},
		SkyColor: core.Color{R: 0.5, G: 0.7, B: 1.0, A: 1.0},
	}
}

func (s *Scene) SetCamera(camera *Camera) {
	s.Camera = camera
}

func (s *Scene) AddNode(node *Node) {
	s.Root.AddChild(node)
}

func (s *Scene) RemoveNode(node *Node) {
	s.Root.RemoveChild(node)
}

func (s *Scene) AddLight(light *Light) {
	s.Lights = append(s.Lights, light)
}

func (s *Scene) RemoveLight(light *Light) {
	for i, l := range s.Lights {
		if l == light {
			s.Lights = append(s.Lights[:i], s.Lights[i+1:]...)
			return
		}
	}
}

func (s *Scene) Update(deltaTime float32) {
	if s.Root != nil {
		s.Root.Update(deltaTime)
	}
}

// GetVisibleNodes returns all nodes with meshes that are visible
func (s *Scene) GetVisibleNodes() []*Node {
	var visible []*Node
	
	s.Root.Traverse(func(node *Node) {
		if node.Visible && node.Mesh != nil {
			visible = append(visible, node)
		}
	})
	
	return visible
}

// Create a default scene with some objects
func CreateDefaultScene(device interface{}) (*Scene, error) {
	scene := NewScene()
	
	// Create camera
	camera := NewCamera(1.0472, 16.0/9.0, 0.1, 1000.0) // 60 degrees FOV
	camera.SetPosition(math.Vec3{X: 0, Y: 2, Z: 5})
	camera.LookAt(math.Vec3Zero, math.Vec3Up)
	scene.SetCamera(camera)
	
	// Add ambient light
	ambient := &Light{
		Type:      LightTypeDirectional,
		Direction: math.Vec3{X: 0.5, Y: -1, Z: -0.5}.Normalize(),
		Color:     core.ColorWhite,
		Intensity: 0.8,
	}
	scene.AddLight(ambient)
	
	return scene, nil
}

// CreateDemoScene creates a scene with demo objects
func CreateDemoScene(device interface{}) (*Scene, error) {
	scene, _ := CreateDefaultScene(device)
	
	// Add a rotating cube
	cubeNode := NewNode("Cube")
	scene.AddNode(cubeNode)
	
	return scene, nil
}
