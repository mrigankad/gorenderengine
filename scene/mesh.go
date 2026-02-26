package scene

import (
	"render-engine/core"
	"render-engine/math"
)

// Mesh holds CPU-side vertex/index data.
// GPU upload is managed by the renderer backend.
type Mesh struct {
	Name         string
	Vertices     []core.Vertex
	Indices      []uint32
	IndexCount   uint32
	MaterialName string

	// GPUData is set by the renderer backend (e.g. *opengl.GPUMesh).
	// Do not access directly; use the renderer's API.
	GPUData interface{}
}

func NewMesh(name string) *Mesh {
	return &Mesh{
		Name:     name,
		Vertices: make([]core.Vertex, 0),
		Indices:  make([]uint32, 0),
	}
}

func CreateMeshFromData(name string, vertices []core.Vertex, indices []uint32) *Mesh {
	return &Mesh{
		Name:       name,
		Vertices:   vertices,
		Indices:    indices,
		IndexCount: uint32(len(indices)),
	}
}

func (m *Mesh) Update(deltaTime float32) {}

func (m *Mesh) Destroy() {
	// GPU resources are freed by the renderer backend.
	// CPU data is garbage-collected automatically.
}

// Primitive generation helpers

func CreateTriangle() *Mesh {
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
	return CreateMeshFromData("Triangle", vertices, indices)
}

func CreateQuad() *Mesh {
	vertices := []core.Vertex{
		{Position: math.Vec3{X: -0.5, Y: -0.5, Z: 0}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: 0.5, Y: -0.5, Z: 0}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: 0.5, Y: 0.5, Z: 0}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -0.5, Y: 0.5, Z: 0}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
	}
	indices := []uint32{0, 1, 2, 2, 3, 0}
	return CreateMeshFromData("Quad", vertices, indices)
}

func CreateCube(size float32) *Mesh {
	s := size / 2

	vertices := []core.Vertex{
		// Front face
		{Position: math.Vec3{X: -s, Y: -s, Z: s}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: -s, Z: s}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: s}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: s, Z: s}, Normal: math.Vec3{X: 0, Y: 0, Z: 1}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		// Back face
		{Position: math.Vec3{X: -s, Y: -s, Z: -s}, Normal: math.Vec3{X: 0, Y: 0, Z: -1}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: -s, Z: -s}, Normal: math.Vec3{X: 0, Y: 0, Z: -1}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: -s}, Normal: math.Vec3{X: 0, Y: 0, Z: -1}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: s, Z: -s}, Normal: math.Vec3{X: 0, Y: 0, Z: -1}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		// Top face
		{Position: math.Vec3{X: -s, Y: s, Z: -s}, Normal: math.Vec3{X: 0, Y: 1, Z: 0}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: -s}, Normal: math.Vec3{X: 0, Y: 1, Z: 0}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: s}, Normal: math.Vec3{X: 0, Y: 1, Z: 0}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: s, Z: s}, Normal: math.Vec3{X: 0, Y: 1, Z: 0}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		// Bottom face
		{Position: math.Vec3{X: -s, Y: -s, Z: -s}, Normal: math.Vec3{X: 0, Y: -1, Z: 0}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: -s, Z: -s}, Normal: math.Vec3{X: 0, Y: -1, Z: 0}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: -s, Z: s}, Normal: math.Vec3{X: 0, Y: -1, Z: 0}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: -s, Z: s}, Normal: math.Vec3{X: 0, Y: -1, Z: 0}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		// Right face
		{Position: math.Vec3{X: s, Y: -s, Z: -s}, Normal: math.Vec3{X: 1, Y: 0, Z: 0}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: -s, Z: s}, Normal: math.Vec3{X: 1, Y: 0, Z: 0}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: s}, Normal: math.Vec3{X: 1, Y: 0, Z: 0}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: s, Y: s, Z: -s}, Normal: math.Vec3{X: 1, Y: 0, Z: 0}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		// Left face
		{Position: math.Vec3{X: -s, Y: -s, Z: -s}, Normal: math.Vec3{X: -1, Y: 0, Z: 0}, UV: math.Vec2{X: 1, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: -s, Z: s}, Normal: math.Vec3{X: -1, Y: 0, Z: 0}, UV: math.Vec2{X: 0, Y: 0}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: s, Z: s}, Normal: math.Vec3{X: -1, Y: 0, Z: 0}, UV: math.Vec2{X: 0, Y: 1}, Color: core.ColorWhite},
		{Position: math.Vec3{X: -s, Y: s, Z: -s}, Normal: math.Vec3{X: -1, Y: 0, Z: 0}, UV: math.Vec2{X: 1, Y: 1}, Color: core.ColorWhite},
	}

	indices := []uint32{
		0, 1, 2, 2, 3, 0,
		4, 5, 6, 6, 7, 4,
		8, 9, 10, 10, 11, 8,
		12, 13, 14, 14, 15, 12,
		16, 17, 18, 18, 19, 16,
		20, 21, 22, 22, 23, 20,
	}

	return CreateMeshFromData("Cube", vertices, indices)
}
