package math

import "math"

type Vec3 struct {
	X, Y, Z float32
}

var (
	Vec3Zero  = Vec3{0, 0, 0}
	Vec3One   = Vec3{1, 1, 1}
	Vec3Up    = Vec3{0, 1, 0}
	Vec3Down  = Vec3{0, -1, 0}
	Vec3Right = Vec3{1, 0, 0}
	Vec3Left  = Vec3{-1, 0, 0}
	Vec3Front = Vec3{0, 0, 1}
	Vec3Back  = Vec3{0, 0, -1}
)

func NewVec3(x, y, z float32) Vec3 {
	return Vec3{X: x, Y: y, Z: z}
}

func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z}
}

func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z}
}

func (v Vec3) Mul(scalar float32) Vec3 {
	return Vec3{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar}
}

func (v Vec3) MulVec(other Vec3) Vec3 {
	return Vec3{X: v.X * other.X, Y: v.Y * other.Y, Z: v.Z * other.Z}
}

func (v Vec3) Div(scalar float32) Vec3 {
	return v.Mul(1.0 / scalar)
}

func (v Vec3) Dot(other Vec3) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vec3) Cross(other Vec3) Vec3 {
	return Vec3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

func (v Vec3) Length() float32 {
	return float32(math.Sqrt(float64(v.X*v.X + v.Y*v.Y + v.Z*v.Z)))
}

func (v Vec3) LengthSqr() float32 {
	return v.X*v.X + v.Y*v.Y + v.Z*v.Z
}

func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length > 0 {
		return v.Mul(1.0 / length)
	}
	return v
}

func (v Vec3) Distance(other Vec3) float32 {
	return v.Sub(other).Length()
}

func (v Vec3) Lerp(other Vec3, t float32) Vec3 {
	return v.Add(other.Sub(v).Mul(t))
}

func (v Vec3) Negate() Vec3 {
	return Vec3{-v.X, -v.Y, -v.Z}
}

func (v Vec3) ToVec4(w float32) Vec4 {
	return Vec4{X: v.X, Y: v.Y, Z: v.Z, W: w}
}
