package editor

import (
	stdmath "math"

	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
)

// Ray represents a ray in 3D space
type Ray struct {
	Origin    math.Vec3
	Direction math.Vec3
}

// AABB represents an axis-aligned bounding box
type AABB struct {
	Min math.Vec3
	Max math.Vec3
}

// HitResult stores the result of a ray intersection test
type HitResult struct {
	Hit      bool
	Distance float32
	Point    math.Vec3
	Normal   math.Vec3
	Node     *scene.Node
	FaceIdx  int // triangle index in the mesh
}

// ScreenToRay converts a screen-space mouse position to a world-space ray
func ScreenToRay(mouseX, mouseY float32, screenWidth, screenHeight float32, camera *scene.Camera) Ray {
	// Convert to normalized device coordinates (-1 to 1)
	ndcX := (2.0*mouseX)/screenWidth - 1.0
	ndcY := 1.0 - (2.0*mouseY)/screenHeight // flip Y

	// Create clip-space coordinates
	clipNear := math.Vec4{X: ndcX, Y: ndcY, Z: 0.0, W: 1.0}

	// Inverse projection and view matrices
	invProj := camera.GetProjectionMatrix().Inverse()
	invView := camera.GetViewMatrix().Inverse()

	// Transform to view space
	viewNear := invProj.MulVec(clipNear)
	viewNear = viewNear.Div(viewNear.W)

	// Transform to world space
	worldNear := invView.MulVec(math.Vec4{X: viewNear.X, Y: viewNear.Y, Z: viewNear.Z, W: 1.0})

	direction := math.Vec3{
		X: worldNear.X - camera.Position.X,
		Y: worldNear.Y - camera.Position.Y,
		Z: worldNear.Z - camera.Position.Z,
	}.Normalize()

	return Ray{
		Origin:    camera.Position,
		Direction: direction,
	}
}

// RaycastScene tests a ray against all visible meshes in the scene, returns closest hit
func RaycastScene(ray Ray, s *scene.Scene) HitResult {
	closestHit := HitResult{Distance: float32(stdmath.MaxFloat32)}

	nodes := s.GetVisibleNodes()
	for _, node := range nodes {
		if node.Mesh == nil {
			continue
		}

		// Build AABB from mesh data in world space
		worldMatrix := node.GetWorldMatrix()
		aabb := computeAABB(node.Mesh.Vertices, worldMatrix)

		// Broad phase: AABB test
		t, hit := rayAABBIntersect(ray, aabb)
		if !hit || t > closestHit.Distance {
			continue
		}

		// Narrow phase: triangle test
		result := rayMeshIntersect(ray, node)
		if result.Hit && result.Distance < closestHit.Distance {
			closestHit = result
		}
	}

	return closestHit
}

// computeAABB calculates the AABB for a set of vertices transformed by a world matrix
func computeAABB(vertices []core.Vertex, worldMatrix math.Mat4) AABB {
	if len(vertices) == 0 {
		return AABB{}
	}

	maxFloat := float32(stdmath.MaxFloat32)
	aabb := AABB{
		Min: math.Vec3{X: maxFloat, Y: maxFloat, Z: maxFloat},
		Max: math.Vec3{X: -maxFloat, Y: -maxFloat, Z: -maxFloat},
	}

	for _, v := range vertices {
		worldPos := worldMatrix.MulVec3(v.Position)
		if worldPos.X < aabb.Min.X {
			aabb.Min.X = worldPos.X
		}
		if worldPos.Y < aabb.Min.Y {
			aabb.Min.Y = worldPos.Y
		}
		if worldPos.Z < aabb.Min.Z {
			aabb.Min.Z = worldPos.Z
		}
		if worldPos.X > aabb.Max.X {
			aabb.Max.X = worldPos.X
		}
		if worldPos.Y > aabb.Max.Y {
			aabb.Max.Y = worldPos.Y
		}
		if worldPos.Z > aabb.Max.Z {
			aabb.Max.Z = worldPos.Z
		}
	}

	return aabb
}

// rayAABBIntersect tests ray-AABB intersection
func rayAABBIntersect(ray Ray, aabb AABB) (float32, bool) {
	invDir := math.Vec3{
		X: 1.0 / ray.Direction.X,
		Y: 1.0 / ray.Direction.Y,
		Z: 1.0 / ray.Direction.Z,
	}

	t1 := (aabb.Min.X - ray.Origin.X) * invDir.X
	t2 := (aabb.Max.X - ray.Origin.X) * invDir.X
	t3 := (aabb.Min.Y - ray.Origin.Y) * invDir.Y
	t4 := (aabb.Max.Y - ray.Origin.Y) * invDir.Y
	t5 := (aabb.Min.Z - ray.Origin.Z) * invDir.Z
	t6 := (aabb.Max.Z - ray.Origin.Z) * invDir.Z

	tmin := max32(max32(min32(t1, t2), min32(t3, t4)), min32(t5, t6))
	tmax := min32(min32(max32(t1, t2), max32(t3, t4)), max32(t5, t6))

	if tmax < 0 || tmin > tmax {
		return 0, false
	}

	return tmin, true
}

// rayMeshIntersect performs per-triangle intersection using Möller–Trumbore algorithm
func rayMeshIntersect(ray Ray, node *scene.Node) HitResult {
	mesh := node.Mesh
	worldMatrix := node.GetWorldMatrix()
	closest := HitResult{Distance: float32(stdmath.MaxFloat32)}

	for i := 0; i < len(mesh.Indices); i += 3 {
		i0, i1, i2 := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
		v0 := worldMatrix.MulVec3(mesh.Vertices[i0].Position)
		v1 := worldMatrix.MulVec3(mesh.Vertices[i1].Position)
		v2 := worldMatrix.MulVec3(mesh.Vertices[i2].Position)

		t, hit := mollerTrumbore(ray, v0, v1, v2)
		if hit && t > 0 && t < closest.Distance {
			closest.Hit = true
			closest.Distance = t
			closest.Point = ray.Origin.Add(ray.Direction.Mul(t))
			closest.Normal = v1.Sub(v0).Cross(v2.Sub(v0)).Normalize()
			closest.Node = node
			closest.FaceIdx = i / 3
		}
	}

	return closest
}

// mollerTrumbore implements the Möller–Trumbore ray-triangle intersection algorithm
func mollerTrumbore(ray Ray, v0, v1, v2 math.Vec3) (float32, bool) {
	const epsilon = 0.0000001

	edge1 := v1.Sub(v0)
	edge2 := v2.Sub(v0)
	h := ray.Direction.Cross(edge2)
	a := edge1.Dot(h)

	if a > -epsilon && a < epsilon {
		return 0, false // parallel
	}

	f := 1.0 / a
	s := ray.Origin.Sub(v0)
	u := f * s.Dot(h)

	if u < 0.0 || u > 1.0 {
		return 0, false
	}

	q := s.Cross(edge1)
	v := f * ray.Direction.Dot(q)

	if v < 0.0 || u+v > 1.0 {
		return 0, false
	}

	t := f * edge2.Dot(q)
	return t, t > epsilon
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
