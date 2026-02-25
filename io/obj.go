package io

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"render-engine/core"
	"render-engine/materials"
	"render-engine/math"
)

// OBJData holds parsed OBJ file data before GPU upload
type OBJData struct {
	Name      string
	Meshes    []OBJMesh
	Materials map[string]*materials.Material
}

// OBJMesh is a single mesh group from an OBJ file
type OBJMesh struct {
	Name     string
	Vertices []core.Vertex
	Indices  []uint32
	Material string // material name reference
}

// LoadOBJ parses a Wavefront .obj file and returns mesh data
func LoadOBJ(path string) (*OBJData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open OBJ file: %w", err)
	}
	defer f.Close()

	data := &OBJData{
		Name:      filepath.Base(path),
		Materials: make(map[string]*materials.Material),
	}

	var positions []math.Vec3
	var normals []math.Vec3
	var uvs []math.Vec2

	// Current mesh state
	currentMesh := OBJMesh{Name: "default"}
	currentMaterial := ""
	vertexMap := make(map[string]uint32) // "v/vt/vn" -> vertex index

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "v":
			if len(parts) >= 4 {
				x, _ := strconv.ParseFloat(parts[1], 32)
				y, _ := strconv.ParseFloat(parts[2], 32)
				z, _ := strconv.ParseFloat(parts[3], 32)
				positions = append(positions, math.Vec3{X: float32(x), Y: float32(y), Z: float32(z)})
			}
		case "vn":
			if len(parts) >= 4 {
				x, _ := strconv.ParseFloat(parts[1], 32)
				y, _ := strconv.ParseFloat(parts[2], 32)
				z, _ := strconv.ParseFloat(parts[3], 32)
				normals = append(normals, math.Vec3{X: float32(x), Y: float32(y), Z: float32(z)})
			}
		case "vt":
			if len(parts) >= 3 {
				u, _ := strconv.ParseFloat(parts[1], 32)
				v, _ := strconv.ParseFloat(parts[2], 32)
				uvs = append(uvs, math.Vec2{X: float32(u), Y: float32(v)})
			}
		case "f":
			// Triangulate faces (fan triangulation for n-gons)
			faceVerts := make([]uint32, 0, len(parts)-1)
			for _, faceStr := range parts[1:] {
				key := faceStr
				if idx, ok := vertexMap[key]; ok {
					faceVerts = append(faceVerts, idx)
					continue
				}

				vertex := parseFaceVertex(faceStr, positions, normals, uvs)
				newIdx := uint32(len(currentMesh.Vertices))
				currentMesh.Vertices = append(currentMesh.Vertices, vertex)
				vertexMap[key] = newIdx
				faceVerts = append(faceVerts, newIdx)
			}

			// Fan triangulation
			for i := 2; i < len(faceVerts); i++ {
				currentMesh.Indices = append(currentMesh.Indices,
					faceVerts[0], faceVerts[i-1], faceVerts[i])
			}

		case "o", "g":
			// New object/group — flush current mesh
			if len(currentMesh.Vertices) > 0 {
				data.Meshes = append(data.Meshes, currentMesh)
			}
			name := "unnamed"
			if len(parts) > 1 {
				name = parts[1]
			}
			currentMesh = OBJMesh{Name: name, Material: currentMaterial}
			vertexMap = make(map[string]uint32)

		case "usemtl":
			if len(parts) > 1 {
				currentMaterial = parts[1]
				currentMesh.Material = currentMaterial
			}

		case "mtllib":
			if len(parts) > 1 {
				mtlPath := filepath.Join(filepath.Dir(path), parts[1])
				mtls, err := LoadMTL(mtlPath)
				if err != nil {
					fmt.Printf("Warning: failed to load MTL file %s: %v\n", mtlPath, err)
				} else {
					for k, v := range mtls {
						data.Materials[k] = v
					}
				}
			}
		}
	}

	// Flush last mesh
	if len(currentMesh.Vertices) > 0 {
		data.Meshes = append(data.Meshes, currentMesh)
	}

	if len(data.Meshes) == 0 {
		return nil, fmt.Errorf("no mesh data found in OBJ file")
	}

	return data, scanner.Err()
}

// LoadMTL parses a Wavefront .mtl material file
func LoadMTL(path string) (map[string]*materials.Material, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := make(map[string]*materials.Material)
	var current *materials.Material

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "newmtl":
			if len(parts) > 1 {
				current = materials.NewMaterial(parts[1])
				result[parts[1]] = current
			}
		case "Kd":
			if current != nil && len(parts) >= 4 {
				r, _ := strconv.ParseFloat(parts[1], 32)
				g, _ := strconv.ParseFloat(parts[2], 32)
				b, _ := strconv.ParseFloat(parts[3], 32)
				current.DiffuseColor = core.Color{R: float32(r), G: float32(g), B: float32(b), A: 1}
			}
		case "Ks":
			if current != nil && len(parts) >= 4 {
				r, _ := strconv.ParseFloat(parts[1], 32)
				g, _ := strconv.ParseFloat(parts[2], 32)
				b, _ := strconv.ParseFloat(parts[3], 32)
				current.SpecularColor = core.Color{R: float32(r), G: float32(g), B: float32(b), A: 1}
			}
		case "Ns":
			if current != nil && len(parts) >= 2 {
				ns, _ := strconv.ParseFloat(parts[1], 32)
				// Convert OBJ shininess (0-1000) to roughness (0-1)
				current.Roughness = 1.0 - float32(ns)/1000.0
				if current.Roughness < 0 {
					current.Roughness = 0
				}
			}
		case "d", "Tr":
			if current != nil && len(parts) >= 2 {
				d, _ := strconv.ParseFloat(parts[1], 32)
				if parts[0] == "Tr" {
					d = 1.0 - d // Tr is inverse of d
				}
				current.Opacity = float32(d)
			}
		}
	}

	return result, scanner.Err()
}

// ExportOBJ writes scene mesh data to a .obj file
func ExportOBJ(path string, meshes []OBJMesh) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create OBJ file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	fmt.Fprintln(w, "# Exported by Go Render Engine")
	fmt.Fprintln(w)

	vertexOffset := 0
	normalOffset := 0
	uvOffset := 0

	for _, mesh := range meshes {
		fmt.Fprintf(w, "o %s\n", mesh.Name)

		// Write vertices
		for _, v := range mesh.Vertices {
			fmt.Fprintf(w, "v %f %f %f\n", v.Position.X, v.Position.Y, v.Position.Z)
		}

		// Write normals
		for _, v := range mesh.Vertices {
			fmt.Fprintf(w, "vn %f %f %f\n", v.Normal.X, v.Normal.Y, v.Normal.Z)
		}

		// Write UVs
		for _, v := range mesh.Vertices {
			fmt.Fprintf(w, "vt %f %f\n", v.UV.X, v.UV.Y)
		}

		// Write faces (1-indexed in OBJ)
		if mesh.Material != "" {
			fmt.Fprintf(w, "usemtl %s\n", mesh.Material)
		}
		for i := 0; i < len(mesh.Indices); i += 3 {
			i0 := mesh.Indices[i] + 1 + uint32(vertexOffset)
			i1 := mesh.Indices[i+1] + 1 + uint32(vertexOffset)
			i2 := mesh.Indices[i+2] + 1 + uint32(vertexOffset)
			fmt.Fprintf(w, "f %d/%d/%d %d/%d/%d %d/%d/%d\n",
				i0, mesh.Indices[i]+1+uint32(uvOffset), mesh.Indices[i]+1+uint32(normalOffset),
				i1, mesh.Indices[i+1]+1+uint32(uvOffset), mesh.Indices[i+1]+1+uint32(normalOffset),
				i2, mesh.Indices[i+2]+1+uint32(uvOffset), mesh.Indices[i+2]+1+uint32(normalOffset))
		}

		vertexOffset += len(mesh.Vertices)
		normalOffset += len(mesh.Vertices)
		uvOffset += len(mesh.Vertices)
		fmt.Fprintln(w)
	}

	return nil
}

// parseFaceVertex parses an OBJ face vertex spec like "v/vt/vn"
func parseFaceVertex(spec string, positions []math.Vec3, normals []math.Vec3, uvs []math.Vec2) core.Vertex {
	v := core.Vertex{
		Color: core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
	}

	parts := strings.Split(spec, "/")

	// Position (required)
	if len(parts) >= 1 && parts[0] != "" {
		idx, _ := strconv.Atoi(parts[0])
		if idx < 0 {
			idx = len(positions) + idx + 1
		}
		if idx > 0 && idx <= len(positions) {
			v.Position = positions[idx-1]
		}
	}

	// UV (optional)
	if len(parts) >= 2 && parts[1] != "" {
		idx, _ := strconv.Atoi(parts[1])
		if idx < 0 {
			idx = len(uvs) + idx + 1
		}
		if idx > 0 && idx <= len(uvs) {
			v.UV = uvs[idx-1]
		}
	}

	// Normal (optional)
	if len(parts) >= 3 && parts[2] != "" {
		idx, _ := strconv.Atoi(parts[2])
		if idx < 0 {
			idx = len(normals) + idx + 1
		}
		if idx > 0 && idx <= len(normals) {
			v.Normal = normals[idx-1]
		}
	}

	return v
}
