package core

import (
	"render-engine/math"
)

type Color struct {
	R, G, B, A float32
}

var (
	ColorWhite  = Color{1, 1, 1, 1}
	ColorBlack  = Color{0, 0, 0, 1}
	ColorRed    = Color{1, 0, 0, 1}
	ColorGreen  = Color{0, 1, 0, 1}
	ColorBlue   = Color{0, 0, 1, 1}
	ColorYellow = Color{1, 1, 0, 1}
)

type Vertex struct {
	Position  math.Vec3
	Normal    math.Vec3
	UV        math.Vec2
	Color     Color
	Tangent   math.Vec3
	Bitangent math.Vec3
}

type MeshData struct {
	Vertices []Vertex
	Indices  []uint32
}

type Transform struct {
	Position math.Vec3
	Rotation math.Quaternion
	Scale    math.Vec3
}

func NewTransform() Transform {
	return Transform{
		Position: math.Vec3Zero,
		Rotation: math.QuaternionIdentity(),
		Scale:    math.Vec3One,
	}
}

func (t Transform) GetMatrix() math.Mat4 {
	translation := math.Mat4Translation(t.Position)
	rotation := t.Rotation.ToMat4()
	scale := math.Mat4Scale(t.Scale)
	return translation.Mul(rotation).Mul(scale)
}

func (t Transform) GetForward() math.Vec3 {
	return t.Rotation.RotateVector(math.Vec3Front)
}

func (t Transform) GetRight() math.Vec3 {
	return t.Rotation.RotateVector(math.Vec3Right)
}

func (t Transform) GetUp() math.Vec3 {
	return t.Rotation.RotateVector(math.Vec3Up)
}

type Rect struct {
	X, Y, Width, Height float32
}

type Viewport struct {
	X, Y, Width, Height float32
	MinDepth, MaxDepth  float32
}

type Scissor struct {
	X, Y, Width, Height int32
}

type ClearValue struct {
	Color   Color
	Depth   float32
	Stencil uint32
}
