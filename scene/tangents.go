package scene

import "render-engine/math"

// ComputeTangents generates per-vertex tangent and bitangent vectors for a Mesh.
// These are required for tangent-space normal mapping. The mesh must have UV
// coordinates; triangles with a degenerate UV area are skipped.
//
// Call after CreateMeshFromData, before uploading the mesh to the GPU.
func ComputeTangents(m *Mesh) {
	// Zero any existing tangents/bitangents
	for i := range m.Vertices {
		m.Vertices[i].Tangent   = math.Vec3{}
		m.Vertices[i].Bitangent = math.Vec3{}
	}

	// accum adds the tangent/bitangent contribution of one triangle to its vertices.
	accum := func(i0, i1, i2 uint32) {
		v0 := m.Vertices[i0]
		v1 := m.Vertices[i1]
		v2 := m.Vertices[i2]

		e1 := v1.Position.Sub(v0.Position)
		e2 := v2.Position.Sub(v0.Position)

		du1 := v1.UV.X - v0.UV.X
		dv1 := v1.UV.Y - v0.UV.Y
		du2 := v2.UV.X - v0.UV.X
		dv2 := v2.UV.Y - v0.UV.Y

		denom := du1*dv2 - du2*dv1
		if denom == 0 {
			return // degenerate UV triangle
		}
		r := 1.0 / denom

		t := e1.Mul(dv2 * r).Sub(e2.Mul(dv1 * r))
		b := e2.Mul(du1 * r).Sub(e1.Mul(du2 * r))

		m.Vertices[i0].Tangent = m.Vertices[i0].Tangent.Add(t)
		m.Vertices[i1].Tangent = m.Vertices[i1].Tangent.Add(t)
		m.Vertices[i2].Tangent = m.Vertices[i2].Tangent.Add(t)

		m.Vertices[i0].Bitangent = m.Vertices[i0].Bitangent.Add(b)
		m.Vertices[i1].Bitangent = m.Vertices[i1].Bitangent.Add(b)
		m.Vertices[i2].Bitangent = m.Vertices[i2].Bitangent.Add(b)
	}

	if len(m.Indices) > 0 {
		for i := 0; i+2 < len(m.Indices); i += 3 {
			accum(m.Indices[i], m.Indices[i+1], m.Indices[i+2])
		}
	} else {
		for i := 0; i+2 < len(m.Vertices); i += 3 {
			accum(uint32(i), uint32(i+1), uint32(i+2))
		}
	}

	// Gram-Schmidt orthogonalize and normalize each vertex tangent frame.
	for i := range m.Vertices {
		n := m.Vertices[i].Normal
		t := m.Vertices[i].Tangent
		b := m.Vertices[i].Bitangent

		// T = normalize(T - N*(N·T))
		t = t.Sub(n.Mul(n.Dot(t)))
		if t.LengthSqr() < 1e-8 {
			// Degenerate: choose an arbitrary tangent perpendicular to N.
			if tangentAbs(n.X) < 0.9 {
				t = math.Vec3{X: 1}.Sub(n.Mul(n.X))
			} else {
				t = math.Vec3{Y: 1}.Sub(n.Mul(n.Y))
			}
		}
		m.Vertices[i].Tangent = t.Normalize()

		if b.LengthSqr() < 1e-8 {
			b = n.Cross(m.Vertices[i].Tangent)
		}
		m.Vertices[i].Bitangent = b.Normalize()
	}
}

func tangentAbs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
