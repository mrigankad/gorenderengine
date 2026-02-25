package math

import (
	"math"
	"testing"
)

func TestVec3Operations(t *testing.T) {
	v1 := NewVec3(1, 2, 3)
	v2 := NewVec3(4, 5, 6)
	
	// Addition
	result := v1.Add(v2)
	expected := NewVec3(5, 7, 9)
	if result != expected {
		t.Errorf("Add: expected %v, got %v", expected, result)
	}
	
	// Subtraction
	result = v2.Sub(v1)
	expected = NewVec3(3, 3, 3)
	if result != expected {
		t.Errorf("Sub: expected %v, got %v", expected, result)
	}
	
	// Scalar multiplication
	result = v1.Mul(2)
	expected = NewVec3(2, 4, 6)
	if result != expected {
		t.Errorf("Mul: expected %v, got %v", expected, result)
	}
	
	// Dot product
	dot := v1.Dot(v2)
	expectedDot := float32(32) // 1*4 + 2*5 + 3*6
	if dot != expectedDot {
		t.Errorf("Dot: expected %v, got %v", expectedDot, dot)
	}
	
	// Cross product (Right x Up = Front in right-handed system)
	cross := Vec3Right.Cross(Vec3Up)
	if cross != Vec3Front {
		t.Errorf("Cross: expected %v, got %v", Vec3Front, cross)
	}
}

func TestVec3Normalize(t *testing.T) {
	v := NewVec3(3, 0, 0)
	normalized := v.Normalize()
	expected := NewVec3(1, 0, 0)
	
	if normalized != expected {
		t.Errorf("Normalize: expected %v, got %v", expected, normalized)
	}
	
	// Check length is 1
	length := normalized.Length()
	if math.Abs(float64(length-1)) > 0.0001 {
		t.Errorf("Normalize: expected length 1, got %v", length)
	}
}

func TestMat4Identity(t *testing.T) {
	m := Mat4Identity()
	
	// Check diagonal is 1
	for i := 0; i < 4; i++ {
		if m[i][i] != 1 {
			t.Errorf("Identity: expected diagonal to be 1, got %v", m[i][i])
		}
	}
	
	// Check non-diagonal is 0
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			if i != j && m[i][j] != 0 {
				t.Errorf("Identity: expected non-diagonal to be 0, got %v", m[i][j])
			}
		}
	}
}

func TestMat4Multiplication(t *testing.T) {
	m1 := Mat4Identity()
	m2 := Mat4Identity()
	
	result := m1.Mul(m2)
	
	// Identity * Identity = Identity
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			expected := float32(0)
			if i == j {
				expected = 1
			}
			if result[i][j] != expected {
				t.Errorf("Mul: expected [%d][%d] = %v, got %v", i, j, expected, result[i][j])
			}
		}
	}
}

func TestMat4Translation(t *testing.T) {
	translation := NewVec3(1, 2, 3)
	m := Mat4Translation(translation)
	
	// Check translation components
	if m[3][0] != 1 || m[3][1] != 2 || m[3][2] != 3 {
		t.Errorf("Translation: expected (1,2,3), got (%v,%v,%v)", m[3][0], m[3][1], m[3][2])
	}
	
	// Test transforming a point
	point := NewVec4(0, 0, 0, 1)
	result := point.MulMat(m)
	
	if result.ToVec3() != translation {
		t.Errorf("Translation: expected %v, got %v", translation, result.ToVec3())
	}
}

func TestQuaternionIdentity(t *testing.T) {
	q := QuaternionIdentity()
	
	if q.X != 0 || q.Y != 0 || q.Z != 0 || q.W != 1 {
		t.Errorf("QuaternionIdentity: expected (0,0,0,1), got (%v,%v,%v,%v)", q.X, q.Y, q.Z, q.W)
	}
}

func TestQuaternionRotation(t *testing.T) {
	// 90 degree rotation around Y axis
	q := QuaternionFromAxisAngle(Vec3Up, float32(math.Pi/2))
	
	// Rotate the X unit vector 90 degrees around Y should give Z
	result := q.RotateVector(Vec3Right)
	
	// Check that result is approximately -Z (due to coordinate system)
	tolerance := float32(0.001)
	if math.Abs(float64(result.X-0)) > float64(tolerance) ||
		math.Abs(float64(result.Y-0)) > float64(tolerance) ||
		math.Abs(float64(result.Z+1)) > float64(tolerance) {
		t.Errorf("Quaternion rotation: expected approximately (0,0,-1), got (%v,%v,%v)", result.X, result.Y, result.Z)
	}
}

func TestMat4Perspective(t *testing.T) {
	fov := float32(math.Pi / 4) // 45 degrees
	aspect := float32(16.0 / 9.0)
	near := float32(0.1)
	far := float32(100.0)
	
	m := Mat4Perspective(fov, aspect, near, far)
	
	// Check aspect ratio affects the matrix
	if m[0][0] == 0 {
		t.Error("Perspective: expected non-zero X scale")
	}
	if m[1][1] == 0 {
		t.Error("Perspective: expected non-zero Y scale")
	}
}

func TestMat4LookAt(t *testing.T) {
eye := NewVec3(0, 0, 5)
	target := NewVec3(0, 0, 0)
	up := Vec3Up
	
	m := Mat4LookAt(eye, target, up)
	
	// The view matrix should transform the eye position to origin
	point := eye.ToVec4(1)
	result := m.MulVec(point)
	
	tolerance := float32(0.001)
	if math.Abs(float64(result.X)) > float64(tolerance) ||
		math.Abs(float64(result.Y)) > float64(tolerance) ||
		math.Abs(float64(result.Z)) > float64(tolerance) {
		t.Errorf("LookAt: expected eye to transform to origin, got (%v,%v,%v)", result.X, result.Y, result.Z)
	}
}

func BenchmarkVec3Add(b *testing.B) {
	v1 := NewVec3(1, 2, 3)
	v2 := NewVec3(4, 5, 6)
	
	for i := 0; i < b.N; i++ {
		_ = v1.Add(v2)
	}
}

func BenchmarkMat4Mul(b *testing.B) {
	m1 := Mat4Identity()
	m2 := Mat4Identity()
	
	for i := 0; i < b.N; i++ {
		_ = m1.Mul(m2)
	}
}
