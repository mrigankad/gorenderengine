package scene

import "render-engine/math"

// Plane represents a half-space: ax + by + cz + d = 0
// Normal (a, b, c) points into the "inside" of the frustum.
type Plane struct {
	Normal math.Vec3
	D      float32
}

// DistanceTo returns the signed distance from a point to the plane.
// Positive means on the "inside" (same side as Normal).
func (p Plane) DistanceTo(pt math.Vec3) float32 {
	return p.Normal.Dot(pt) + p.D
}

// Frustum holds the six clip planes of a view frustum.
type Frustum struct {
	Planes [6]Plane // Left, Right, Bottom, Top, Near, Far
}

// FrustumFromVP extracts the six frustum planes from a view-projection matrix.
// The planes are normalized so DistanceTo returns a true distance in world units.
//
// Convention: The Go engine stores matrices as [col][row] and passes them to
// GLSL with transpose=false. The GLSL shader multiplies as "mvp * col_vector".
// Because Go's Mul uses row-major semantics internally, the GLSL matrix is the
// transpose of the Go matrix. Gribb/Hartmann frustum extraction operates on the
// GLSL matrix rows, which correspond to Go matrix columns (vp[col][0..3]).
func FrustumFromVP(vp math.Mat4) Frustum {
	// Row i of the GLSL matrix = Column i of the Go matrix = vp[i][0..3]
	r0 := math.Vec4{X: vp[0][0], Y: vp[0][1], Z: vp[0][2], W: vp[0][3]}
	r1 := math.Vec4{X: vp[1][0], Y: vp[1][1], Z: vp[1][2], W: vp[1][3]}
	r2 := math.Vec4{X: vp[2][0], Y: vp[2][1], Z: vp[2][2], W: vp[2][3]}
	r3 := math.Vec4{X: vp[3][0], Y: vp[3][1], Z: vp[3][2], W: vp[3][3]}

	var f Frustum
	// Left:   r3 + r0
	f.Planes[0] = normalizePlane(r3.X+r0.X, r3.Y+r0.Y, r3.Z+r0.Z, r3.W+r0.W)
	// Right:  r3 - r0
	f.Planes[1] = normalizePlane(r3.X-r0.X, r3.Y-r0.Y, r3.Z-r0.Z, r3.W-r0.W)
	// Bottom: r3 + r1
	f.Planes[2] = normalizePlane(r3.X+r1.X, r3.Y+r1.Y, r3.Z+r1.Z, r3.W+r1.W)
	// Top:    r3 - r1
	f.Planes[3] = normalizePlane(r3.X-r1.X, r3.Y-r1.Y, r3.Z-r1.Z, r3.W-r1.W)
	// Near:   r3 + r2
	f.Planes[4] = normalizePlane(r3.X+r2.X, r3.Y+r2.Y, r3.Z+r2.Z, r3.W+r2.W)
	// Far:    r3 - r2
	f.Planes[5] = normalizePlane(r3.X-r2.X, r3.Y-r2.Y, r3.Z-r2.Z, r3.W-r2.W)
	return f
}

func normalizePlane(a, b, c, d float32) Plane {
	l := math.Vec3{X: a, Y: b, Z: c}.Length()
	if l == 0 {
		return Plane{}
	}
	return Plane{Normal: math.Vec3{X: a / l, Y: b / l, Z: c / l}, D: d / l}
}

// AABB is an axis-aligned bounding box.
type AABB struct {
	Min, Max math.Vec3
}

// IntersectsFrustum returns false if the AABB is completely outside the frustum.
// Uses the "n-vertex" test: for each plane, check if the "positive vertex"
// (the corner most aligned with the plane normal) is on the outside.
func (box AABB) IntersectsFrustum(f *Frustum) bool {
	for i := 0; i < 6; i++ {
		p := f.Planes[i]
		// Choose the positive vertex (most in the direction of the plane normal)
		px := box.Max.X
		if p.Normal.X < 0 {
			px = box.Min.X
		}
		py := box.Max.Y
		if p.Normal.Y < 0 {
			py = box.Min.Y
		}
		pz := box.Max.Z
		if p.Normal.Z < 0 {
			pz = box.Min.Z
		}
		if p.DistanceTo(math.Vec3{X: px, Y: py, Z: pz}) < 0 {
			return false // outside this plane
		}
	}
	return true
}

// ComputeAABB computes the world-space AABB for a mesh transformed by worldMatrix.
// If the mesh has a cached local AABB, it transforms the 8 corners (fast path).
// Otherwise it falls back to iterating all vertices.
func ComputeAABB(mesh *Mesh, worldMatrix math.Mat4) AABB {
	if mesh.HasLocalAABB {
		return transformAABB(mesh.LocalAABB, worldMatrix)
	}
	return computeAABBSlow(mesh, worldMatrix)
}

// transformAABB transforms a local AABB by a world matrix by testing all 8 corners.
func transformAABB(local AABB, m math.Mat4) AABB {
	mn, mx := local.Min, local.Max
	corners := [8]math.Vec3{
		{X: mn.X, Y: mn.Y, Z: mn.Z},
		{X: mx.X, Y: mn.Y, Z: mn.Z},
		{X: mn.X, Y: mx.Y, Z: mn.Z},
		{X: mx.X, Y: mx.Y, Z: mn.Z},
		{X: mn.X, Y: mn.Y, Z: mx.Z},
		{X: mx.X, Y: mn.Y, Z: mx.Z},
		{X: mn.X, Y: mx.Y, Z: mx.Z},
		{X: mx.X, Y: mx.Y, Z: mx.Z},
	}
	first := m.MulVec3(corners[0])
	out := AABB{Min: first, Max: first}
	for i := 1; i < 8; i++ {
		wp := m.MulVec3(corners[i])
		if wp.X < out.Min.X { out.Min.X = wp.X }
		if wp.Y < out.Min.Y { out.Min.Y = wp.Y }
		if wp.Z < out.Min.Z { out.Min.Z = wp.Z }
		if wp.X > out.Max.X { out.Max.X = wp.X }
		if wp.Y > out.Max.Y { out.Max.Y = wp.Y }
		if wp.Z > out.Max.Z { out.Max.Z = wp.Z }
	}
	return out
}

// computeAABBSlow is the fallback when no cached local AABB is available.
func computeAABBSlow(mesh *Mesh, worldMatrix math.Mat4) AABB {
	if len(mesh.Vertices) == 0 {
		return AABB{}
	}
	first := worldMatrix.MulVec3(mesh.Vertices[0].Position)
	out := AABB{Min: first, Max: first}
	for i := 1; i < len(mesh.Vertices); i++ {
		wp := worldMatrix.MulVec3(mesh.Vertices[i].Position)
		if wp.X < out.Min.X { out.Min.X = wp.X }
		if wp.Y < out.Min.Y { out.Min.Y = wp.Y }
		if wp.Z < out.Min.Z { out.Min.Z = wp.Z }
		if wp.X > out.Max.X { out.Max.X = wp.X }
		if wp.Y > out.Max.Y { out.Max.Y = wp.Y }
		if wp.Z > out.Max.Z { out.Max.Z = wp.Z }
	}
	return out
}
