package scene

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"render-engine/core"
	remath "render-engine/math"
)

// objFace is an already-triangulated face (three vertex references).
type objFace struct {
	vIdx, vtIdx, vnIdx [3]int // 0-based position / UV / normal indices (-1 = absent)
}

// LoadOBJ parses a Wavefront .obj file and returns one Mesh per object/group.
// A companion .mtl file is loaded automatically if referenced via "mtllib".
// The returned meshes are CPU-side only; upload GPU resources via the renderer.
func LoadOBJ(path string) ([]*Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open obj %q: %w", path, err)
	}
	defer f.Close()

	dir := filepath.Dir(path)

	// Indexed OBJ data pools
	var positions []remath.Vec3
	var normals   []remath.Vec3
	var uvs       []remath.Vec2

	// MTL materials (loaded on first "mtllib" directive)
	materials := map[string]*Material{}

	// Accumulate per-object data
	type objObject struct {
		name    string
		matName string
		faces   []objFace
	}

	var objects []objObject
	cur := &objObject{name: "default"}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "v":
			if len(fields) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 32)
			y, _ := strconv.ParseFloat(fields[2], 32)
			z, _ := strconv.ParseFloat(fields[3], 32)
			positions = append(positions, remath.Vec3{X: float32(x), Y: float32(y), Z: float32(z)})

		case "vn":
			if len(fields) < 4 {
				continue
			}
			x, _ := strconv.ParseFloat(fields[1], 32)
			y, _ := strconv.ParseFloat(fields[2], 32)
			z, _ := strconv.ParseFloat(fields[3], 32)
			normals = append(normals, remath.Vec3{X: float32(x), Y: float32(y), Z: float32(z)})

		case "vt":
			if len(fields) < 3 {
				continue
			}
			u, _ := strconv.ParseFloat(fields[1], 32)
			v, _ := strconv.ParseFloat(fields[2], 32)
			uvs = append(uvs, remath.Vec2{X: float32(u), Y: float32(v)})

		case "o", "g":
			// Push the current object if it has faces, then start a new one
			if len(cur.faces) > 0 {
				objects = append(objects, *cur)
			}
			name := "default"
			if len(fields) > 1 {
				name = fields[1]
			}
			cur = &objObject{name: name, matName: cur.matName}

		case "usemtl":
			if len(fields) > 1 {
				cur.matName = fields[1]
			}

		case "mtllib":
			if len(fields) > 1 {
				mtlPath := filepath.Join(dir, fields[1])
				loaded, err := loadMTL(mtlPath, dir)
				if err == nil {
					for k, v := range loaded {
						materials[k] = v
					}
				}
			}

		case "f":
			// Fan-triangulate polygon (handles 3+ vertices)
			if len(fields) < 4 {
				continue
			}
			type fv struct{ v, vt, vn int }
			var fverts []fv
			for _, tok := range fields[1:] {
				fverts = append(fverts, parseFaceVertex(tok))
			}
			// Fan triangulation: 0-1-2, 0-2-3, 0-3-4, ...
			for i := 1; i+1 < len(fverts); i++ {
				f0, f1, f2 := fverts[0], fverts[i], fverts[i+1]
				cur.faces = append(cur.faces, objFace{
					vIdx:  [3]int{f0.v, f1.v, f2.v},
					vtIdx: [3]int{f0.vt, f1.vt, f2.vt},
					vnIdx: [3]int{f0.vn, f1.vn, f2.vn},
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan obj: %w", err)
	}

	// Push final object
	if len(cur.faces) > 0 {
		objects = append(objects, *cur)
	}
	if len(objects) == 0 {
		return nil, fmt.Errorf("no geometry found in %q", path)
	}

	// Convert each OBJ object to a scene.Mesh
	meshes := make([]*Mesh, 0, len(objects))
	for _, obj := range objects {
		mesh := buildMeshFromOBJ(obj.name, obj.faces, positions, normals, uvs)

		if mat, ok := materials[obj.matName]; ok {
			mesh.Material = mat
		} else {
			mesh.Material = DefaultMaterial()
		}
		mesh.MaterialName = obj.matName
		meshes = append(meshes, mesh)
	}

	return meshes, nil
}

// parseFaceVertex parses one face vertex token: "v", "v/vt", "v//vn", "v/vt/vn".
// Returns 0-based indices (-1 if absent). OBJ is 1-based.
func parseFaceVertex(tok string) struct{ v, vt, vn int } {
	parseIdx := func(s string) int {
		if s == "" {
			return -1
		}
		n, _ := strconv.Atoi(s)
		if n > 0 {
			return n - 1
		}
		return n
	}
	parts := strings.Split(tok, "/")
	res := struct{ v, vt, vn int }{v: -1, vt: -1, vn: -1}
	if len(parts) > 0 {
		res.v = parseIdx(parts[0])
	}
	if len(parts) > 1 {
		res.vt = parseIdx(parts[1])
	}
	if len(parts) > 2 {
		res.vn = parseIdx(parts[2])
	}
	return res
}

// buildMeshFromOBJ converts parsed face data into a deduplicated Mesh.
func buildMeshFromOBJ(
	name string,
	faces []objFace,
	positions []remath.Vec3,
	normals   []remath.Vec3,
	uvs       []remath.Vec2,
) *Mesh {
	type key struct{ v, vt, vn int }
	vertMap := map[key]uint32{}
	var vertices []core.Vertex
	var indices  []uint32

	safePos := func(i int) remath.Vec3 {
		if i >= 0 && i < len(positions) {
			return positions[i]
		}
		return remath.Vec3Zero
	}
	safeNorm := func(i int) remath.Vec3 {
		if i >= 0 && i < len(normals) {
			return normals[i]
		}
		return remath.Vec3{X: 0, Y: 1, Z: 0}
	}
	safeUV := func(i int) remath.Vec2 {
		if i >= 0 && i < len(uvs) {
			return uvs[i]
		}
		return remath.Vec2{}
	}

	hasNormals := len(normals) > 0

	for _, face := range faces {
		for c := 0; c < 3; c++ {
			k := key{face.vIdx[c], face.vtIdx[c], face.vnIdx[c]}
			if idx, ok := vertMap[k]; ok {
				indices = append(indices, idx)
			} else {
				v := core.Vertex{
					Position: safePos(k.v),
					Normal:   safeNorm(k.vn),
					UV:       safeUV(k.vt),
					Color:    core.ColorWhite,
				}
				idx = uint32(len(vertices))
				vertices = append(vertices, v)
				vertMap[k] = idx
				indices = append(indices, idx)
			}
		}
	}

	// Generate smooth normals if the OBJ had no normals
	if !hasNormals {
		generateFlatNormals(vertices, indices)
	}

	return CreateMeshFromData(name, vertices, indices)
}

// generateFlatNormals computes area-weighted normals and writes them to the vertex slice.
func generateFlatNormals(vertices []core.Vertex, indices []uint32) {
	accum  := make([]remath.Vec3, len(vertices))
	counts := make([]int, len(vertices))

	for i := 0; i+2 < len(indices); i += 3 {
		i0, i1, i2 := indices[i], indices[i+1], indices[i+2]
		v0 := vertices[i0].Position
		v1 := vertices[i1].Position
		v2 := vertices[i2].Position
		n := v1.Sub(v0).Cross(v2.Sub(v0)) // area-weighted normal
		accum[i0] = accum[i0].Add(n)
		accum[i1] = accum[i1].Add(n)
		accum[i2] = accum[i2].Add(n)
		counts[i0]++
		counts[i1]++
		counts[i2]++
	}
	for i := range vertices {
		if counts[i] > 0 {
			vertices[i].Normal = accum[i].Normalize()
		}
	}
}

// ── MTL loader ───────────────────────────────────────────────────────────────

func loadMTL(path, dir string) (map[string]*Material, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mats := map[string]*Material{}
	var cur *Material

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "newmtl":
			if len(fields) > 1 {
				m := DefaultMaterial()
				m.Name = fields[1]
				mats[fields[1]] = m
				cur = m
			}
		case "Kd":
			if cur != nil && len(fields) >= 4 {
				r, _ := strconv.ParseFloat(fields[1], 32)
				g, _ := strconv.ParseFloat(fields[2], 32)
				b, _ := strconv.ParseFloat(fields[3], 32)
				cur.Albedo = core.Color{R: float32(r), G: float32(g), B: float32(b), A: 1}
			}
		case "Ks":
			if cur != nil && len(fields) >= 4 {
				r, _ := strconv.ParseFloat(fields[1], 32)
				g, _ := strconv.ParseFloat(fields[2], 32)
				b, _ := strconv.ParseFloat(fields[3], 32)
				cur.Specular = core.Color{R: float32(r), G: float32(g), B: float32(b), A: 1}
			}
		case "Ns":
			if cur != nil && len(fields) >= 2 {
				ns, _ := strconv.ParseFloat(fields[1], 32)
				cur.Shininess = float32(math.Max(1, ns))
			}
		case "map_Kd":
			if cur != nil && len(fields) >= 2 {
				texPath := filepath.Join(dir, fields[1])
				tex, err := LoadTexture(texPath)
				if err == nil {
					cur.AlbedoTexture = tex
				}
			}
		}
	}

	return mats, scanner.Err()
}
