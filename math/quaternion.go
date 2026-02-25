package math

import "math"

type Quaternion struct {
	X, Y, Z, W float32
}

func QuaternionIdentity() Quaternion {
	return Quaternion{X: 0, Y: 0, Z: 0, W: 1}
}

func NewQuaternion(x, y, z, w float32) Quaternion {
	return Quaternion{X: x, Y: y, Z: z, W: w}
}

func QuaternionFromAxisAngle(axis Vec3, angle float32) Quaternion {
	halfAngle := angle / 2
	s := float32(math.Sin(float64(halfAngle)))
	c := float32(math.Cos(float64(halfAngle)))
	
	axis = axis.Normalize()
	return Quaternion{
		X: axis.X * s,
		Y: axis.Y * s,
		Z: axis.Z * s,
		W: c,
	}
}

func QuaternionFromEuler(euler Vec3) Quaternion {
	cx := float32(math.Cos(float64(euler.X) / 2))
	sx := float32(math.Sin(float64(euler.X) / 2))
	cy := float32(math.Cos(float64(euler.Y) / 2))
	sy := float32(math.Sin(float64(euler.Y) / 2))
	cz := float32(math.Cos(float64(euler.Z) / 2))
	sz := float32(math.Sin(float64(euler.Z) / 2))
	
	return Quaternion{
		X: sx*cy*cz - cx*sy*sz,
		Y: cx*sy*cz + sx*cy*sz,
		Z: cx*cy*sz - sx*sy*cz,
		W: cx*cy*cz + sx*sy*sz,
	}
}

func (q Quaternion) Mul(other Quaternion) Quaternion {
	return Quaternion{
		X: q.W*other.X + q.X*other.W + q.Y*other.Z - q.Z*other.Y,
		Y: q.W*other.Y - q.X*other.Z + q.Y*other.W + q.Z*other.X,
		Z: q.W*other.Z + q.X*other.Y - q.Y*other.X + q.Z*other.W,
		W: q.W*other.W - q.X*other.X - q.Y*other.Y - q.Z*other.Z,
	}
}

func (q Quaternion) Normalize() Quaternion {
	length := float32(math.Sqrt(float64(q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W)))
	if length > 0 {
		invLength := 1 / length
		return Quaternion{
			X: q.X * invLength,
			Y: q.Y * invLength,
			Z: q.Z * invLength,
			W: q.W * invLength,
		}
	}
	return q
}

func (q Quaternion) Conjugate() Quaternion {
	return Quaternion{X: -q.X, Y: -q.Y, Z: -q.Z, W: q.W}
}

func (q Quaternion) Inverse() Quaternion {
	conjugate := q.Conjugate()
	lengthSqr := q.X*q.X + q.Y*q.Y + q.Z*q.Z + q.W*q.W
	if lengthSqr > 0 {
		invLengthSqr := 1 / lengthSqr
		return Quaternion{
			X: conjugate.X * invLengthSqr,
			Y: conjugate.Y * invLengthSqr,
			Z: conjugate.Z * invLengthSqr,
			W: conjugate.W * invLengthSqr,
		}
	}
	return q
}

func (q Quaternion) RotateVector(v Vec3) Vec3 {
	qVec := Vec3{X: q.X, Y: q.Y, Z: q.Z}
	t := qVec.Cross(v).Mul(2)
	return v.Add(t.Mul(q.W)).Add(qVec.Cross(t))
}

func (q Quaternion) ToMat4() Mat4 {
	xx := q.X * q.X
	yy := q.Y * q.Y
	zz := q.Z * q.Z
	xy := q.X * q.Y
	xz := q.X * q.Z
	yz := q.Y * q.Z
	wx := q.W * q.X
	wy := q.W * q.Y
	wz := q.W * q.Z
	
	return Mat4{
		{1 - 2*(yy+zz), 2 * (xy + wz), 2 * (xz - wy), 0},
		{2 * (xy - wz), 1 - 2*(xx+zz), 2 * (yz + wx), 0},
		{2 * (xz + wy), 2 * (yz - wx), 1 - 2*(xx+yy), 0},
		{0, 0, 0, 1},
	}
}

func (q Quaternion) ToEuler() Vec3 {
	sinRCosP := 2 * (q.W*q.X + q.Y*q.Z)
	cosRCosP := 1 - 2*(q.X*q.X+q.Y*q.Y)
	roll := float32(math.Atan2(float64(sinRCosP), float64(cosRCosP)))
	
	sinP := 2 * (q.W*q.Y - q.Z*q.X)
	var pitch float32
	if math.Abs(float64(sinP)) >= 1 {
		pitch = float32(math.Copysign(math.Pi/2, float64(sinP)))
	} else {
		pitch = float32(math.Asin(float64(sinP)))
	}
	
	sinYCosR := 2 * (q.W*q.Z + q.X*q.Y)
	cosYCosR := 1 - 2*(q.Y*q.Y+q.Z*q.Z)
	yaw := float32(math.Atan2(float64(sinYCosR), float64(cosYCosR)))
	
	return Vec3{X: pitch, Y: yaw, Z: roll}
}

func (q Quaternion) Lerp(other Quaternion, t float32) Quaternion {
	return Quaternion{
		X: q.X + (other.X-q.X)*t,
		Y: q.Y + (other.Y-q.Y)*t,
		Z: q.Z + (other.Z-q.Z)*t,
		W: q.W + (other.W-q.W)*t,
	}.Normalize()
}

func (q Quaternion) Slerp(other Quaternion, t float32) Quaternion {
	dot := q.X*other.X + q.Y*other.Y + q.Z*other.Z + q.W*other.W
	
	if dot < 0 {
		dot = -dot
		other = Quaternion{-other.X, -other.Y, -other.Z, -other.W}
	}
	
	if dot > 0.9995 {
		return q.Lerp(other, t)
	}
	
	theta0 := math.Acos(float64(dot))
	theta := theta0 * float64(t)
	sinTheta := math.Sin(theta)
	sinTheta0 := math.Sin(theta0)
	
	s0 := float32(math.Cos(theta) - float64(dot)*sinTheta/sinTheta0)
	s1 := float32(sinTheta / sinTheta0)
	
	return Quaternion{
		X: q.X*s0 + other.X*s1,
		Y: q.Y*s0 + other.Y*s1,
		Z: q.Z*s0 + other.Z*s1,
		W: q.W*s0 + other.W*s1,
	}
}
