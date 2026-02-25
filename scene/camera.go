package scene

import (
	"math"
	
	reMath "render-engine/math"
)

// Camera represents a view camera
type Camera struct {
	Position    reMath.Vec3
	Rotation    reMath.Quaternion
	FOV         float32
	AspectRatio float32
	NearPlane   float32
	FarPlane    float32
	
	// Cached matrices
	viewMatrix       reMath.Mat4
	projectionMatrix reMath.Mat4
	viewProjMatrix   reMath.Mat4
	dirty            bool
}

func NewCamera(fov, aspectRatio, nearPlane, farPlane float32) *Camera {
	return &Camera{
		Position:    reMath.Vec3Zero,
		Rotation:    reMath.QuaternionIdentity(),
		FOV:         fov,
		AspectRatio: aspectRatio,
		NearPlane:   nearPlane,
		FarPlane:    farPlane,
		dirty:       true,
	}
}

func (c *Camera) UpdateAspectRatio(width, height float32) {
	if height > 0 {
		c.AspectRatio = width / height
		c.dirty = true
	}
}

func (c *Camera) SetPosition(pos reMath.Vec3) {
	c.Position = pos
	c.dirty = true
}

func (c *Camera) SetRotation(rot reMath.Quaternion) {
	c.Rotation = rot
	c.dirty = true
}

func (c *Camera) Translate(delta reMath.Vec3) {
	c.Position = c.Position.Add(delta)
	c.dirty = true
}

func (c *Camera) Rotate(axis reMath.Vec3, angle float32) {
	rotation := reMath.QuaternionFromAxisAngle(axis, angle)
	c.Rotation = c.Rotation.Mul(rotation).Normalize()
	c.dirty = true
}

func (c *Camera) LookAt(target, up reMath.Vec3) {
	c.viewMatrix = reMath.Mat4LookAt(c.Position, target, up)
	c.Rotation = c.QuaternionFromLookAt(target, up)
	c.dirty = true
}

func (c *Camera) GetViewMatrix() reMath.Mat4 {
	if c.dirty {
		c.updateMatrices()
	}
	return c.viewMatrix
}

func (c *Camera) GetProjectionMatrix() reMath.Mat4 {
	if c.dirty {
		c.updateMatrices()
	}
	return c.projectionMatrix
}

func (c *Camera) GetViewProjectionMatrix() reMath.Mat4 {
	if c.dirty {
		c.updateMatrices()
	}
	return c.viewProjMatrix
}

func (c *Camera) GetForward() reMath.Vec3 {
	return c.Rotation.RotateVector(reMath.Vec3Front)
}

func (c *Camera) GetRight() reMath.Vec3 {
	return c.Rotation.RotateVector(reMath.Vec3Right)
}

func (c *Camera) GetUp() reMath.Vec3 {
	return c.Rotation.RotateVector(reMath.Vec3Up)
}

func (c *Camera) updateMatrices() {
	// Create view matrix from position and rotation
	rotationMatrix := c.Rotation.ToMat4()
	translationMatrix := reMath.Mat4Translation(c.Position.Negate())
	c.viewMatrix = rotationMatrix.Mul(translationMatrix)
	
	// Create projection matrix
	c.projectionMatrix = reMath.Mat4Perspective(c.FOV, c.AspectRatio, c.NearPlane, c.FarPlane)
	
	// View projection matrix
	c.viewProjMatrix = c.projectionMatrix.Mul(c.viewMatrix)
	
	c.dirty = false
}

func (c *Camera) QuaternionFromLookAt(target, up reMath.Vec3) reMath.Quaternion {
	forward := target.Sub(c.Position).Normalize()
	right := up.Cross(forward).Normalize()
	upNew := forward.Cross(right)
	
	// Convert rotation matrix to quaternion
	m := reMath.Mat4{
		{right.X, upNew.X, -forward.X, 0},
		{right.Y, upNew.Y, -forward.Y, 0},
		{right.Z, upNew.Z, -forward.Z, 0},
		{0, 0, 0, 1},
	}
	
	trace := m[0][0] + m[1][1] + m[2][2]
	
	var q reMath.Quaternion
	if trace > 0 {
		s := float32(0.5 / math.Sqrt(float64(trace+1)))
		q.W = 0.25 / s
		q.X = (m[2][1] - m[1][2]) * s
		q.Y = (m[0][2] - m[2][0]) * s
		q.Z = (m[1][0] - m[0][1]) * s
	} else if m[0][0] > m[1][1] && m[0][0] > m[2][2] {
		s := 2 * float32(math.Sqrt(float64(1+m[0][0]-m[1][1]-m[2][2])))
		q.W = (m[2][1] - m[1][2]) / s
		q.X = 0.25 * s
		q.Y = (m[0][1] + m[1][0]) / s
		q.Z = (m[0][2] + m[2][0]) / s
	} else if m[1][1] > m[2][2] {
		s := 2 * float32(math.Sqrt(float64(1+m[1][1]-m[0][0]-m[2][2])))
		q.W = (m[0][2] - m[2][0]) / s
		q.X = (m[0][1] + m[1][0]) / s
		q.Y = 0.25 * s
		q.Z = (m[1][2] + m[2][1]) / s
	} else {
		s := 2 * float32(math.Sqrt(float64(1+m[2][2]-m[0][0]-m[1][1])))
		q.W = (m[1][0] - m[0][1]) / s
		q.X = (m[0][2] + m[2][0]) / s
		q.Y = (m[1][2] + m[2][1]) / s
		q.Z = 0.25 * s
	}
	
	return q.Normalize()
}

// OrbitCamera is a specialized camera for orbiting around a target
type OrbitCamera struct {
	Camera
	Target   reMath.Vec3
	Distance float32
	Yaw      float32
	Pitch    float32
}

func NewOrbitCamera(target reMath.Vec3, distance, fov, aspectRatio float32) *OrbitCamera {
	c := &OrbitCamera{
		Target:   target,
		Distance: distance,
		Yaw:      0,
		Pitch:    0.3,
	}
	c.Camera = *NewCamera(fov, aspectRatio, 0.1, 1000.0)
	c.UpdatePosition()
	return c
}

func (c *OrbitCamera) UpdatePosition() {
	// Clamp pitch
	if c.Pitch > 1.5 {
		c.Pitch = 1.5
	}
	if c.Pitch < -1.5 {
		c.Pitch = -1.5
	}
	
	// Calculate position from spherical coordinates
	cosPitch := float32(math.Cos(float64(c.Pitch)))
	sinPitch := float32(math.Sin(float64(c.Pitch)))
	cosYaw := float32(math.Cos(float64(c.Yaw)))
	sinYaw := float32(math.Sin(float64(c.Yaw)))
	
	offset := reMath.Vec3{
		X: c.Distance * cosPitch * sinYaw,
		Y: c.Distance * sinPitch,
		Z: c.Distance * cosPitch * cosYaw,
	}
	
	c.Position = c.Target.Add(offset)
	c.LookAt(c.Target, reMath.Vec3Up)
}

func (c *OrbitCamera) Orbit(deltaYaw, deltaPitch float32) {
	c.Yaw += deltaYaw
	c.Pitch += deltaPitch
	c.UpdatePosition()
}

func (c *OrbitCamera) Zoom(delta float32) {
	c.Distance += delta
	if c.Distance < 0.1 {
		c.Distance = 0.1
	}
	c.UpdatePosition()
}
