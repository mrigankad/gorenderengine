package math

type Vec4 struct {
	X, Y, Z, W float32
}

func NewVec4(x, y, z, w float32) Vec4 {
	return Vec4{X: x, Y: y, Z: z, W: w}
}

func (v Vec4) Add(other Vec4) Vec4 {
	return Vec4{X: v.X + other.X, Y: v.Y + other.Y, Z: v.Z + other.Z, W: v.W + other.W}
}

func (v Vec4) Sub(other Vec4) Vec4 {
	return Vec4{X: v.X - other.X, Y: v.Y - other.Y, Z: v.Z - other.Z, W: v.W - other.W}
}

func (v Vec4) Mul(scalar float32) Vec4 {
	return Vec4{X: v.X * scalar, Y: v.Y * scalar, Z: v.Z * scalar, W: v.W * scalar}
}

func (v Vec4) MulMat(m Mat4) Vec4 {
	return Vec4{
		X: v.X*m[0][0] + v.Y*m[1][0] + v.Z*m[2][0] + v.W*m[3][0],
		Y: v.X*m[0][1] + v.Y*m[1][1] + v.Z*m[2][1] + v.W*m[3][1],
		Z: v.X*m[0][2] + v.Y*m[1][2] + v.Z*m[2][2] + v.W*m[3][2],
		W: v.X*m[0][3] + v.Y*m[1][3] + v.Z*m[2][3] + v.W*m[3][3],
	}
}

func (v Vec4) Dot(other Vec4) float32 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z + v.W*other.W
}

func (v Vec4) ToVec3() Vec3 {
	return Vec3{X: v.X, Y: v.Y, Z: v.Z}
}

func (v Vec4) ToVec3DivW() Vec3 {
	if v.W != 0 {
		return Vec3{X: v.X / v.W, Y: v.Y / v.W, Z: v.Z / v.W}
	}
	return Vec3{X: v.X, Y: v.Y, Z: v.Z}
}
