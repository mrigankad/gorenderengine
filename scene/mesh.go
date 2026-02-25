package scene

/*
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"unsafe"

	"render-engine/core"
	"render-engine/math"
	"render-engine/vulkan"
)

// Mesh represents a renderable mesh
type Mesh struct {
	Name     string
	Vertices []core.Vertex
	Indices  []uint32

	// GPU resources
	VertexBuffer *vulkan.Buffer
	IndexBuffer  *vulkan.Buffer
	IndexCount   uint32
	MaterialName string // reference to material by name
}

func NewMesh(name string) *Mesh {
	return &Mesh{
		Name:     name,
		Vertices: make([]core.Vertex, 0),
		Indices:  make([]uint32, 0),
	}
}

func CreateMeshFromData(device *vulkan.Device, name string, vertices []core.Vertex, indices []uint32) (*Mesh, error) {
	mesh := &Mesh{
		Name:       name,
		Vertices:   vertices,
		Indices:    indices,
		IndexCount: uint32(len(indices)),
	}

	// Create vertex buffer
	vertexBufferSize := uint64(len(vertices) * int(unsafe.Sizeof(core.Vertex{})))

	// Create staging buffer
	stagingBuffer, err := vulkan.CreateBuffer(device, vertexBufferSize,
		C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
		C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)
	if err != nil {
		return nil, err
	}
	defer stagingBuffer.Destroy(device)

	stagingBuffer.Map(device)
	stagingBuffer.CopyData(unsafe.Pointer(&vertices[0]), vertexBufferSize)
	stagingBuffer.Unmap(device)

	// Create device local vertex buffer
	mesh.VertexBuffer, err = vulkan.CreateBuffer(device, vertexBufferSize,
		C.VK_BUFFER_USAGE_TRANSFER_DST_BIT|C.VK_BUFFER_USAGE_VERTEX_BUFFER_BIT,
		C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
	if err != nil {
		return nil, err
	}

	vulkan.CopyBuffer(device, stagingBuffer.Handle, mesh.VertexBuffer.Handle, vertexBufferSize, device.CommandPool, device.GraphicsQueue)

	// Create index buffer
	if len(indices) > 0 {
		indexBufferSize := uint64(len(indices) * 4) // uint32 = 4 bytes

		stagingBuffer2, err := vulkan.CreateBuffer(device, indexBufferSize,
			C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
			C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)
		if err != nil {
			return nil, err
		}

		stagingBuffer2.Map(device)
		stagingBuffer2.CopyData(unsafe.Pointer(&indices[0]), indexBufferSize)
		stagingBuffer2.Unmap(device)

		mesh.IndexBuffer, err = vulkan.CreateBuffer(device, indexBufferSize,
			C.VK_BUFFER_USAGE_TRANSFER_DST_BIT|C.VK_BUFFER_USAGE_INDEX_BUFFER_BIT,
			C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT)
		if err != nil {
			stagingBuffer2.Destroy(device)
			return nil, err
		}

		vulkan.CopyBuffer(device, stagingBuffer2.Handle, mesh.IndexBuffer.Handle, indexBufferSize, device.CommandPool, device.GraphicsQueue)
		stagingBuffer2.Destroy(device)
	}

	return mesh, nil
}

func (m *Mesh) Update(deltaTime float32) {
	// Can be used for vertex animation, skinning, etc.
}

func (m *Mesh) Destroy(device *vulkan.Device) {
	if m.VertexBuffer != nil {
		m.VertexBuffer.Destroy(device)
	}
	if m.IndexBuffer != nil {
		m.IndexBuffer.Destroy(device)
	}
}

// Primitive generation helpers

func CreateTriangle(device *vulkan.Device) (*Mesh, error) {
	vertices := []core.Vertex{
		{
			Position: math.Vec3{X: 0, Y: -0.5, Z: 0},
			Color:    core.Color{R: 1, G: 0, B: 0, A: 1},
			UV:       math.Vec2{X: 0.5, Y: 0},
		},
		{
			Position: math.Vec3{X: 0.5, Y: 0.5, Z: 0},
			Color:    core.Color{R: 0, G: 1, B: 0, A: 1},
			UV:       math.Vec2{X: 1, Y: 1},
		},
		{
			Position: math.Vec3{X: -0.5, Y: 0.5, Z: 0},
			Color:    core.Color{R: 0, G: 0, B: 1, A: 1},
			UV:       math.Vec2{X: 0, Y: 1},
		},
	}

	indices := []uint32{0, 1, 2}

	return CreateMeshFromData(device, "Triangle", vertices, indices)
}

func CreateQuad(device *vulkan.Device) (*Mesh, error) {
	vertices := []core.Vertex{
		{
			Position: math.Vec3{X: -0.5, Y: -0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 0, Y: 0},
			Color:    core.ColorWhite,
		},
		{
			Position: math.Vec3{X: 0.5, Y: -0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 1, Y: 0},
			Color:    core.ColorWhite,
		},
		{
			Position: math.Vec3{X: 0.5, Y: 0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 1, Y: 1},
			Color:    core.ColorWhite,
		},
		{
			Position: math.Vec3{X: -0.5, Y: 0.5, Z: 0},
			Normal:   math.Vec3{X: 0, Y: 0, Z: 1},
			UV:       math.Vec2{X: 0, Y: 1},
			Color:    core.ColorWhite,
		},
	}

	indices := []uint32{0, 1, 2, 2, 3, 0}

	return CreateMeshFromData(device, "Quad", vertices, indices)
}

func CreateCube(device *vulkan.Device, size float32) (*Mesh, error) {
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
		0, 1, 2, 2, 3, 0, // Front
		4, 5, 6, 6, 7, 4, // Back
		8, 9, 10, 10, 11, 8, // Top
		12, 13, 14, 14, 15, 12, // Bottom
		16, 17, 18, 18, 19, 16, // Right
		20, 21, 22, 22, 23, 20, // Left
	}

	return CreateMeshFromData(device, "Cube", vertices, indices)
}
