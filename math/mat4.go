package math

import "math"

type Mat4 [4][4]float32

func Mat4Identity() Mat4 {
	return Mat4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func Mat4Zero() Mat4 {
	return Mat4{
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 0, 0},
	}
}

func (m Mat4) Mul(other Mat4) Mat4 {
	result := Mat4Zero()
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			for k := 0; k < 4; k++ {
				result[i][j] += m[i][k] * other[k][j]
			}
		}
	}
	return result
}

func (m Mat4) MulVec(v Vec4) Vec4 {
	return v.MulMat(m)
}

func (m Mat4) MulVec3(v Vec3) Vec3 {
	v4 := v.ToVec4(1.0)
	result := m.MulVec(v4)
	return result.ToVec3DivW()
}

func (m Mat4) Transpose() Mat4 {
	return Mat4{
		{m[0][0], m[1][0], m[2][0], m[3][0]},
		{m[0][1], m[1][1], m[2][1], m[3][1]},
		{m[0][2], m[1][2], m[2][2], m[3][2]},
		{m[0][3], m[1][3], m[2][3], m[3][3]},
	}
}

func Mat4Translation(translation Vec3) Mat4 {
	m := Mat4Identity()
	m[3][0] = translation.X
	m[3][1] = translation.Y
	m[3][2] = translation.Z
	return m
}

func Mat4Scale(scale Vec3) Mat4 {
	m := Mat4Identity()
	m[0][0] = scale.X
	m[1][1] = scale.Y
	m[2][2] = scale.Z
	return m
}

func Mat4RotationX(angle float32) Mat4 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	return Mat4{
		{1, 0, 0, 0},
		{0, c, s, 0},
		{0, -s, c, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotationY(angle float32) Mat4 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	return Mat4{
		{c, 0, -s, 0},
		{0, 1, 0, 0},
		{s, 0, c, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotationZ(angle float32) Mat4 {
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	return Mat4{
		{c, s, 0, 0},
		{-s, c, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func Mat4RotationAxis(axis Vec3, angle float32) Mat4 {
	axis = axis.Normalize()
	c := float32(math.Cos(float64(angle)))
	s := float32(math.Sin(float64(angle)))
	t := 1 - c

	x, y, z := axis.X, axis.Y, axis.Z

	return Mat4{
		{t*x*x + c, t*x*y + s*z, t*x*z - s*y, 0},
		{t*x*y - s*z, t*y*y + c, t*y*z + s*x, 0},
		{t*x*z + s*y, t*y*z - s*x, t*z*z + c, 0},
		{0, 0, 0, 1},
	}
}

func Mat4Perspective(fovY, aspect, near, far float32) Mat4 {
	tanHalfFovy := float32(math.Tan(float64(fovY) / 2))
	
	m := Mat4Zero()
	m[0][0] = 1 / (aspect * tanHalfFovy)
	m[1][1] = 1 / tanHalfFovy
	m[2][2] = -(far + near) / (far - near)
	m[2][3] = -1
	m[3][2] = -(2 * far * near) / (far - near)
	return m
}

func Mat4Orthographic(left, right, bottom, top, near, far float32) Mat4 {
	m := Mat4Identity()
	m[0][0] = 2 / (right - left)
	m[1][1] = 2 / (top - bottom)
	m[2][2] = -2 / (far - near)
	m[3][0] = -(right + left) / (right - left)
	m[3][1] = -(top + bottom) / (top - bottom)
	m[3][2] = -(far + near) / (far - near)
	return m
}

func Mat4LookAt(eye, target, up Vec3) Mat4 {
	zAxis := eye.Sub(target).Normalize()
	xAxis := up.Cross(zAxis).Normalize()
	yAxis := zAxis.Cross(xAxis)

	return Mat4{
		{xAxis.X, yAxis.X, zAxis.X, 0},
		{xAxis.Y, yAxis.Y, zAxis.Y, 0},
		{xAxis.Z, yAxis.Z, zAxis.Z, 0},
		{-xAxis.Dot(eye), -yAxis.Dot(eye), -zAxis.Dot(eye), 1},
	}
}

func Mat4TRS(translation, rotation, scale Vec3) Mat4 {
	translationMat := Mat4Translation(translation)
	rotationMat := Mat4Rotation(rotation)
	scaleMat := Mat4Scale(scale)
	return translationMat.Mul(rotationMat).Mul(scaleMat)
}

func Mat4Rotation(euler Vec3) Mat4 {
	return Mat4RotationY(euler.Y).Mul(Mat4RotationX(euler.X)).Mul(Mat4RotationZ(euler.Z))
}

func (m Mat4) Inverse() Mat4 {
	inv := Mat4Zero()
	
	inv[0][0] = m[1][1]*m[2][2]*m[3][3] - m[1][1]*m[2][3]*m[3][2] - m[2][1]*m[1][2]*m[3][3] + m[2][1]*m[1][3]*m[3][2] + m[3][1]*m[1][2]*m[2][3] - m[3][1]*m[1][3]*m[2][2]
	inv[1][0] = -m[1][0]*m[2][2]*m[3][3] + m[1][0]*m[2][3]*m[3][2] + m[2][0]*m[1][2]*m[3][3] - m[2][0]*m[1][3]*m[3][2] - m[3][0]*m[1][2]*m[2][3] + m[3][0]*m[1][3]*m[2][2]
	inv[2][0] = m[1][0]*m[2][1]*m[3][3] - m[1][0]*m[2][3]*m[3][1] - m[2][0]*m[1][1]*m[3][3] + m[2][0]*m[1][3]*m[3][1] + m[3][0]*m[1][1]*m[2][3] - m[3][0]*m[1][3]*m[2][1]
	inv[3][0] = -m[1][0]*m[2][1]*m[3][2] + m[1][0]*m[2][2]*m[3][1] + m[2][0]*m[1][1]*m[3][2] - m[2][0]*m[1][2]*m[3][1] - m[3][0]*m[1][1]*m[2][2] + m[3][0]*m[1][2]*m[2][1]
	
	det := m[0][0]*inv[0][0] + m[0][1]*inv[1][0] + m[0][2]*inv[2][0] + m[0][3]*inv[3][0]
	
	if det == 0 {
		return Mat4Identity()
	}
	
	det = 1 / det
	
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			inv[i][j] *= det
		}
	}
	
	return inv
}
