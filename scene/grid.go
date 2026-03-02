package scene

import "render-engine/core"
import "render-engine/math"

// CreateGrid builds a flat grid mesh rendered as GL_LINES.
//
//   size      — total world-space extent (grid goes from -size/2 to +size/2)
//   divisions — number of cells along each axis
//
// The X-axis centre line is red, the Z-axis centre line is blue,
// and all other lines are dark gray.
func CreateGrid(size float32, divisions int) *Mesh {
	if divisions < 1 {
		divisions = 1
	}

	half := size / 2.0
	step := size / float32(divisions)

	gray   := core.Color{R: 0.35, G: 0.35, B: 0.35, A: 1}
	red    := core.Color{R: 0.8, G: 0.15, B: 0.15, A: 1}  // X axis
	blue   := core.Color{R: 0.15, G: 0.35, B: 0.9, A: 1}  // Z axis

	var vertices []core.Vertex
	var indices  []uint32

	addLine := func(a, b math.Vec3, c core.Color) {
		base := uint32(len(vertices))
		vertices = append(vertices,
			core.Vertex{Position: a, Normal: math.Vec3Up, Color: c},
			core.Vertex{Position: b, Normal: math.Vec3Up, Color: c},
		)
		indices = append(indices, base, base+1)
	}

	// Lines parallel to Z (vary X)
	for i := 0; i <= divisions; i++ {
		x := -half + float32(i)*step
		c := gray
		if i == divisions/2 {
			c = blue // Z axis at x=0
		}
		addLine(
			math.Vec3{X: x, Y: 0, Z: -half},
			math.Vec3{X: x, Y: 0, Z: half},
			c,
		)
	}

	// Lines parallel to X (vary Z)
	for i := 0; i <= divisions; i++ {
		z := -half + float32(i)*step
		c := gray
		if i == divisions/2 {
			c = red // X axis at z=0
		}
		addLine(
			math.Vec3{X: -half, Y: 0, Z: z},
			math.Vec3{X: half, Y: 0, Z: z},
			c,
		)
	}

	m := CreateMeshFromData("Grid", vertices, indices)
	m.DrawMode = DrawLines

	unlitMat := DefaultMaterial()
	unlitMat.Name = "GridMaterial"
	unlitMat.Unlit = true
	m.Material = unlitMat

	return m
}

// CreateUnitBoxWireframe creates a unit cube wireframe mesh (corners at ±1, all axes).
// Used as a reusable AABB visualizer: supply a model matrix that scales and
// translates the cube to match the desired bounding box.
func CreateUnitBoxWireframe() *Mesh {
	white  := core.Color{R: 1, G: 1, B: 1, A: 1}
	normal := math.Vec3Up

	var vertices []core.Vertex
	var indices  []uint32

	addLine := func(a, b math.Vec3) {
		base := uint32(len(vertices))
		vertices = append(vertices,
			core.Vertex{Position: a, Normal: normal, Color: white},
			core.Vertex{Position: b, Normal: normal, Color: white},
		)
		indices = append(indices, base, base+1)
	}

	// Bottom face (y = -1)
	addLine(math.Vec3{X: -1, Y: -1, Z: -1}, math.Vec3{X: 1, Y: -1, Z: -1})
	addLine(math.Vec3{X: 1, Y: -1, Z: -1}, math.Vec3{X: 1, Y: -1, Z: 1})
	addLine(math.Vec3{X: 1, Y: -1, Z: 1}, math.Vec3{X: -1, Y: -1, Z: 1})
	addLine(math.Vec3{X: -1, Y: -1, Z: 1}, math.Vec3{X: -1, Y: -1, Z: -1})
	// Top face (y = +1)
	addLine(math.Vec3{X: -1, Y: 1, Z: -1}, math.Vec3{X: 1, Y: 1, Z: -1})
	addLine(math.Vec3{X: 1, Y: 1, Z: -1}, math.Vec3{X: 1, Y: 1, Z: 1})
	addLine(math.Vec3{X: 1, Y: 1, Z: 1}, math.Vec3{X: -1, Y: 1, Z: 1})
	addLine(math.Vec3{X: -1, Y: 1, Z: 1}, math.Vec3{X: -1, Y: 1, Z: -1})
	// Vertical edges
	addLine(math.Vec3{X: -1, Y: -1, Z: -1}, math.Vec3{X: -1, Y: 1, Z: -1})
	addLine(math.Vec3{X: 1, Y: -1, Z: -1}, math.Vec3{X: 1, Y: 1, Z: -1})
	addLine(math.Vec3{X: 1, Y: -1, Z: 1}, math.Vec3{X: 1, Y: 1, Z: 1})
	addLine(math.Vec3{X: -1, Y: -1, Z: 1}, math.Vec3{X: -1, Y: 1, Z: 1})

	m := CreateMeshFromData("UnitBoxWireframe", vertices, indices)
	m.DrawMode = DrawLines

	mat := DefaultMaterial()
	mat.Name  = "AABBMaterial"
	mat.Albedo = core.Color{R: 0.1, G: 0.95, B: 0.1, A: 1} // bright green
	mat.Unlit  = true
	m.Material = mat

	return m
}
