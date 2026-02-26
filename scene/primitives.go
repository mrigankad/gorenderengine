package scene

import (
	stdmath "math"

	"render-engine/core"
	"render-engine/math"
)

// CreateSphere generates a UV-sphere mesh
func CreateSphere(radius float32, segments, rings int) *Mesh {
	if segments < 3 {
		segments = 3
	}
	if rings < 2 {
		rings = 2
	}

	var vertices []core.Vertex
	var indices []uint32

	for ring := 0; ring <= rings; ring++ {
		phi := float64(ring) * stdmath.Pi / float64(rings)
		sinPhi := float32(stdmath.Sin(phi))
		cosPhi := float32(stdmath.Cos(phi))

		for seg := 0; seg <= segments; seg++ {
			theta := float64(seg) * 2.0 * stdmath.Pi / float64(segments)
			sinTheta := float32(stdmath.Sin(theta))
			cosTheta := float32(stdmath.Cos(theta))

			normal := math.Vec3{X: sinPhi * cosTheta, Y: cosPhi, Z: sinPhi * sinTheta}
			position := normal.Mul(radius)
			uv := math.Vec2{X: float32(seg) / float32(segments), Y: float32(ring) / float32(rings)}

			vertices = append(vertices, core.Vertex{
				Position: position,
				Normal:   normal,
				UV:       uv,
				Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
			})
		}
	}

	for ring := 0; ring < rings; ring++ {
		for seg := 0; seg < segments; seg++ {
			current := uint32(ring*(segments+1) + seg)
			next := current + uint32(segments+1)

			indices = append(indices, current, next, current+1)
			indices = append(indices, current+1, next, next+1)
		}
	}

	return CreateMeshFromData("Sphere", vertices, indices)
}

// CreateCylinder generates a cylinder mesh
func CreateCylinder(radius, height float32, segments int) *Mesh {
	if segments < 3 {
		segments = 3
	}

	var vertices []core.Vertex
	var indices []uint32
	halfHeight := height / 2.0

	for i := 0; i <= segments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(segments)
		cosT := float32(stdmath.Cos(theta))
		sinT := float32(stdmath.Sin(theta))
		normal := math.Vec3{X: cosT, Y: 0, Z: sinT}
		u := float32(i) / float32(segments)

		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: -halfHeight, Z: sinT * radius},
			Normal:   normal,
			UV:       math.Vec2{X: u, Y: 0},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: halfHeight, Z: sinT * radius},
			Normal:   normal,
			UV:       math.Vec2{X: u, Y: 1},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
	}

	for i := 0; i < segments; i++ {
		base := uint32(i * 2)
		indices = append(indices, base, base+1, base+2)
		indices = append(indices, base+2, base+1, base+3)
	}

	topCenter := uint32(len(vertices))
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfHeight, Z: 0},
		Normal:   math.Vec3Up,
		UV:       math.Vec2{X: 0.5, Y: 0.5},
		Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
	})

	for i := 0; i < segments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(segments)
		nextTheta := float64(i+1) * 2.0 * stdmath.Pi / float64(segments)
		cosT := float32(stdmath.Cos(theta))
		sinT := float32(stdmath.Sin(theta))
		cosN := float32(stdmath.Cos(nextTheta))
		sinN := float32(stdmath.Sin(nextTheta))

		v1 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: halfHeight, Z: sinT * radius},
			Normal:   math.Vec3Up,
			UV:       math.Vec2{X: cosT*0.5 + 0.5, Y: sinT*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		v2 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosN * radius, Y: halfHeight, Z: sinN * radius},
			Normal:   math.Vec3Up,
			UV:       math.Vec2{X: cosN*0.5 + 0.5, Y: sinN*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		indices = append(indices, topCenter, v1, v2)
	}

	botCenter := uint32(len(vertices))
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: -halfHeight, Z: 0},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 0.5, Y: 0.5},
		Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
	})

	for i := 0; i < segments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(segments)
		nextTheta := float64(i+1) * 2.0 * stdmath.Pi / float64(segments)
		cosT := float32(stdmath.Cos(theta))
		sinT := float32(stdmath.Sin(theta))
		cosN := float32(stdmath.Cos(nextTheta))
		sinN := float32(stdmath.Sin(nextTheta))

		v1 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: -halfHeight, Z: sinT * radius},
			Normal:   math.Vec3Down,
			UV:       math.Vec2{X: cosT*0.5 + 0.5, Y: sinT*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		v2 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosN * radius, Y: -halfHeight, Z: sinN * radius},
			Normal:   math.Vec3Down,
			UV:       math.Vec2{X: cosN*0.5 + 0.5, Y: sinN*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		indices = append(indices, botCenter, v2, v1)
	}

	return CreateMeshFromData("Cylinder", vertices, indices)
}

// CreateCone generates a cone mesh
func CreateCone(radius, height float32, segments int) *Mesh {
	if segments < 3 {
		segments = 3
	}

	var vertices []core.Vertex
	var indices []uint32
	halfHeight := height / 2.0

	tipIdx := uint32(0)
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfHeight, Z: 0},
		Normal:   math.Vec3Up,
		UV:       math.Vec2{X: 0.5, Y: 0},
		Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
	})

	for i := 0; i <= segments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(segments)
		cosT := float32(stdmath.Cos(theta))
		sinT := float32(stdmath.Sin(theta))

		slopeAngle := float32(stdmath.Atan2(float64(radius), float64(height)))
		ny := float32(stdmath.Cos(float64(slopeAngle)))
		nr := float32(stdmath.Sin(float64(slopeAngle)))
		normal := math.Vec3{X: cosT * nr, Y: ny, Z: sinT * nr}.Normalize()

		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: -halfHeight, Z: sinT * radius},
			Normal:   normal,
			UV:       math.Vec2{X: float32(i) / float32(segments), Y: 1},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
	}

	for i := 0; i < segments; i++ {
		indices = append(indices, tipIdx, uint32(i+1), uint32(i+2))
	}

	botCenter := uint32(len(vertices))
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: -halfHeight, Z: 0},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 0.5, Y: 0.5},
		Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
	})

	for i := 0; i < segments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(segments)
		nextTheta := float64(i+1) * 2.0 * stdmath.Pi / float64(segments)
		cosT := float32(stdmath.Cos(theta))
		sinT := float32(stdmath.Sin(theta))
		cosN := float32(stdmath.Cos(nextTheta))
		sinN := float32(stdmath.Sin(nextTheta))

		v1 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosT * radius, Y: -halfHeight, Z: sinT * radius},
			Normal:   math.Vec3Down,
			UV:       math.Vec2{X: cosT*0.5 + 0.5, Y: sinT*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		v2 := uint32(len(vertices))
		vertices = append(vertices, core.Vertex{
			Position: math.Vec3{X: cosN * radius, Y: -halfHeight, Z: sinN * radius},
			Normal:   math.Vec3Down,
			UV:       math.Vec2{X: cosN*0.5 + 0.5, Y: sinN*0.5 + 0.5},
			Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		})
		indices = append(indices, botCenter, v2, v1)
	}

	return CreateMeshFromData("Cone", vertices, indices)
}

// CreateTorus generates a torus mesh
func CreateTorus(majorRadius, minorRadius float32, majorSegments, minorSegments int) *Mesh {
	if majorSegments < 3 {
		majorSegments = 3
	}
	if minorSegments < 3 {
		minorSegments = 3
	}

	var vertices []core.Vertex
	var indices []uint32

	for i := 0; i <= majorSegments; i++ {
		theta := float64(i) * 2.0 * stdmath.Pi / float64(majorSegments)
		cosTheta := float32(stdmath.Cos(theta))
		sinTheta := float32(stdmath.Sin(theta))

		for j := 0; j <= minorSegments; j++ {
			phi := float64(j) * 2.0 * stdmath.Pi / float64(minorSegments)
			cosPhi := float32(stdmath.Cos(phi))
			sinPhi := float32(stdmath.Sin(phi))

			x := (majorRadius + minorRadius*cosPhi) * cosTheta
			y := minorRadius * sinPhi
			z := (majorRadius + minorRadius*cosPhi) * sinTheta

			nx := cosPhi * cosTheta
			ny := sinPhi
			nz := cosPhi * sinTheta

			vertices = append(vertices, core.Vertex{
				Position: math.Vec3{X: x, Y: y, Z: z},
				Normal:   math.Vec3{X: nx, Y: ny, Z: nz}.Normalize(),
				UV:       math.Vec2{X: float32(i) / float32(majorSegments), Y: float32(j) / float32(minorSegments)},
				Color:    core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
			})
		}
	}

	for i := 0; i < majorSegments; i++ {
		for j := 0; j < minorSegments; j++ {
			current := uint32(i*(minorSegments+1) + j)
			next := uint32((i+1)*(minorSegments+1) + j)

			indices = append(indices, current, next, current+1)
			indices = append(indices, current+1, next, next+1)
		}
	}

	return CreateMeshFromData("Torus", vertices, indices)
}

// CreatePlane generates a flat plane mesh
func CreatePlane(width, depth float32, subdivisions int) *Mesh {
	if subdivisions < 1 {
		subdivisions = 1
	}

	var vertices []core.Vertex
	var indices []uint32

	halfW := width / 2.0
	halfD := depth / 2.0

	for z := 0; z <= subdivisions; z++ {
		for x := 0; x <= subdivisions; x++ {
			u := float32(x) / float32(subdivisions)
			v := float32(z) / float32(subdivisions)

			vertices = append(vertices, core.Vertex{
				Position: math.Vec3{
					X: -halfW + u*width,
					Y: 0,
					Z: -halfD + v*depth,
				},
				Normal: math.Vec3Up,
				UV:     math.Vec2{X: u, Y: v},
				Color:  core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
			})
		}
	}

	for z := 0; z < subdivisions; z++ {
		for x := 0; x < subdivisions; x++ {
			topLeft := uint32(z*(subdivisions+1) + x)
			topRight := topLeft + 1
			bottomLeft := topLeft + uint32(subdivisions+1)
			bottomRight := bottomLeft + 1

			indices = append(indices, topLeft, bottomLeft, topRight)
			indices = append(indices, topRight, bottomLeft, bottomRight)
		}
	}

	return CreateMeshFromData("Plane", vertices, indices)
}

// CreatePyramid generates a pyramid mesh with a square base
func CreatePyramid(width, height float32) *Mesh {
	var vertices []core.Vertex
	var indices []uint32

	halfW := width / 2.0
	halfH := height / 2.0

	// Base vertices (square at y = -height/2)
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: -halfW},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 0, Y: 0},
		Color:    core.Color{R: 1.0, G: 0.5, B: 0.2, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: -halfW},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 1, Y: 0},
		Color:    core.Color{R: 1.0, G: 0.5, B: 0.2, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: halfW},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 1, Y: 1},
		Color:    core.Color{R: 1.0, G: 0.5, B: 0.2, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: halfW},
		Normal:   math.Vec3Down,
		UV:       math.Vec2{X: 0, Y: 1},
		Color:    core.Color{R: 1.0, G: 0.5, B: 0.2, A: 1.0},
	})

	// Base face (2 triangles)
	indices = append(indices, 0, 2, 1)
	indices = append(indices, 0, 3, 2)

	// Tip vertex (top at y = height/2)
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfH, Z: 0},
		Normal:   math.Vec3Up,
		UV:       math.Vec2{X: 0.5, Y: 0.5},
		Color:    core.Color{R: 1.0, G: 0.2, B: 0.2, A: 1.0},
	})

	// Front face (0-1-tip)
	frontNorm := math.Vec3{X: 0, Y: 0.5, Z: -1}.Normalize()
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: -halfW},
		Normal:   frontNorm,
		UV:       math.Vec2{X: 0, Y: 0},
		Color:    core.Color{R: 1.0, G: 0.3, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: -halfW},
		Normal:   frontNorm,
		UV:       math.Vec2{X: 1, Y: 0},
		Color:    core.Color{R: 1.0, G: 0.3, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfH, Z: 0},
		Normal:   frontNorm,
		UV:       math.Vec2{X: 0.5, Y: 1},
		Color:    core.Color{R: 1.0, G: 0.3, B: 0.3, A: 1.0},
	})
	indices = append(indices, 5, 7, 6)

	// Right face (1-2-tip)
	rightNorm := math.Vec3{X: 1, Y: 0.5, Z: 0}.Normalize()
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: -halfW},
		Normal:   rightNorm,
		UV:       math.Vec2{X: 0, Y: 0},
		Color:    core.Color{R: 0.3, G: 1.0, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: halfW},
		Normal:   rightNorm,
		UV:       math.Vec2{X: 1, Y: 0},
		Color:    core.Color{R: 0.3, G: 1.0, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfH, Z: 0},
		Normal:   rightNorm,
		UV:       math.Vec2{X: 0.5, Y: 1},
		Color:    core.Color{R: 0.3, G: 1.0, B: 0.3, A: 1.0},
	})
	indices = append(indices, 8, 10, 9)

	// Back face (2-3-tip)
	backNorm := math.Vec3{X: 0, Y: 0.5, Z: 1}.Normalize()
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: halfW, Y: -halfH, Z: halfW},
		Normal:   backNorm,
		UV:       math.Vec2{X: 0, Y: 0},
		Color:    core.Color{R: 0.3, G: 0.3, B: 1.0, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: halfW},
		Normal:   backNorm,
		UV:       math.Vec2{X: 1, Y: 0},
		Color:    core.Color{R: 0.3, G: 0.3, B: 1.0, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfH, Z: 0},
		Normal:   backNorm,
		UV:       math.Vec2{X: 0.5, Y: 1},
		Color:    core.Color{R: 0.3, G: 0.3, B: 1.0, A: 1.0},
	})
	indices = append(indices, 11, 13, 12)

	// Left face (3-0-tip)
	leftNorm := math.Vec3{X: -1, Y: 0.5, Z: 0}.Normalize()
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: halfW},
		Normal:   leftNorm,
		UV:       math.Vec2{X: 0, Y: 0},
		Color:    core.Color{R: 1.0, G: 1.0, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: -halfW, Y: -halfH, Z: -halfW},
		Normal:   leftNorm,
		UV:       math.Vec2{X: 1, Y: 0},
		Color:    core.Color{R: 1.0, G: 1.0, B: 0.3, A: 1.0},
	})
	vertices = append(vertices, core.Vertex{
		Position: math.Vec3{X: 0, Y: halfH, Z: 0},
		Normal:   leftNorm,
		UV:       math.Vec2{X: 0.5, Y: 1},
		Color:    core.Color{R: 1.0, G: 1.0, B: 0.3, A: 1.0},
	})
	indices = append(indices, 14, 16, 15)

	return CreateMeshFromData("Pyramid", vertices, indices)
}
