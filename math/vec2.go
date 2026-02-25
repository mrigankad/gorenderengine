package math

import "math"

type Vec2 struct {
	X, Y float32
}

func NewVec2(x, y float32) Vec2 {
	return Vec2{X: x, Y: y}
}

func (v Vec2) Add(other Vec2) Vec2 {
	return Vec2{X: v.X + other.X, Y: v.Y + other.Y}
}

func (v Vec2) Sub(other Vec2) Vec2 {
	return Vec2{X: v.X - other.X, Y: v.Y - other.Y}
}

func (v Vec2) Mul(scalar float32) Vec2 {
	return Vec2{X: v.X * scalar, Y: v.Y * scalar}
}

func (v Vec2) Dot(other Vec2) float32 {
	return v.X*other.X + v.Y*other.Y
}

func (v Vec2) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y)))
}

func (v Vec2) Normalize() Vec2 {
	length := v.Length()
	if length > 0 {
		return v.Mul(1.0 / length)
	}
	return v
}

func (v Vec2) Lerp(other Vec2, t float32) Vec2 {
	return v.Add(other.Sub(v).Mul(t))
}
