package scene

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/modeler"

	"render-engine/core"
	"render-engine/math"
)

// GLTFResult holds the nodes and textures loaded from a .glb / .gltf file.
// Before the first Render call, upload every texture in the Textures slice:
//
//	for _, tex := range result.Textures {
//	    renderEngine.UploadTexture(tex)
//	}
type GLTFResult struct {
	Roots    []*Node    // top-level nodes; add each with scene.AddNode(n)
	Textures []*Texture // textures that need GPU upload
}

// LoadGLTF opens a .glb or .gltf file and returns a ready-to-use scene graph.
// Mesh geometry, materials, base-colour textures, and the node hierarchy are
// all populated.  PBR metallic-roughness is approximated to Blinn-Phong.
func LoadGLTF(path string) (*GLTFResult, error) {
	doc, err := gltf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gltf open %q: %w", path, err)
	}
	dir := filepath.Dir(path)
	result := &GLTFResult{}

	// ── 1. Textures ───────────────────────────────────────────────────────────
	texCache := make([]*Texture, len(doc.Textures))
	for i, gt := range doc.Textures {
		if gt.Source == nil {
			continue
		}
		img := doc.Images[*gt.Source]

		var tex *Texture
		if img.BufferView != nil {
			// Binary GLB: image data lives in a buffer view
			raw, err := modeler.ReadBufferView(doc, doc.BufferViews[*img.BufferView])
			if err != nil {
				fmt.Printf("gltf: image %d bufferview: %v\n", *gt.Source, err)
				continue
			}
			name := img.Name
			if name == "" {
				name = fmt.Sprintf("gltf_img_%d", *gt.Source)
			}
			tex, err = decodeImageBytes(name, raw)
			if err != nil {
				fmt.Printf("gltf: image %d decode: %v\n", *gt.Source, err)
				continue
			}
		} else if img.URI != "" && !img.IsEmbeddedResource() {
			// External file referenced by relative URI
			tex, err = LoadTexture(filepath.Join(dir, img.URI))
			if err != nil {
				fmt.Printf("gltf: image %d (%s): %v\n", *gt.Source, img.URI, err)
				continue
			}
		}

		if tex != nil {
			texCache[i] = tex
			result.Textures = append(result.Textures, tex)
		}
	}

	// ── 2. Materials ─────────────────────────────────────────────────────────
	matCache := make([]*Material, len(doc.Materials))
	for i, gm := range doc.Materials {
		mat := DefaultMaterial()
		mat.Name = gm.Name

		if pbr := gm.PBRMetallicRoughness; pbr != nil {
			cf := pbr.BaseColorFactorOrDefault()
			mat.Albedo = core.Color{
				R: float32(cf[0]), G: float32(cf[1]),
				B: float32(cf[2]), A: float32(cf[3]),
			}
			if pbr.BaseColorTexture != nil {
				idx := pbr.BaseColorTexture.Index
				if idx < len(texCache) && texCache[idx] != nil {
					mat.AlbedoTexture = texCache[idx]
				}
			}
			// PBR → Phong approximation:
			//   roughness → shininess (smooth surface = high shininess)
			//   metallic  → specular intensity
			roughness := float32(pbr.RoughnessFactorOrDefault())
			metallic  := float32(pbr.MetallicFactorOrDefault())
			mat.Shininess = (1.0-roughness)*(1.0-roughness)*128.0 + 1.0
			s := metallic * 0.7
			mat.Specular = core.Color{R: s, G: s, B: s, A: 1}
		}

		// Normal map texture
		if gm.NormalTexture != nil && gm.NormalTexture.Index != nil {
			idx := *gm.NormalTexture.Index
			if idx >= 0 && idx < len(texCache) && texCache[idx] != nil {
				mat.NormalTexture = texCache[idx]
			}
		}
		matCache[i] = mat
	}

	// ── 3. Mesh primitives ────────────────────────────────────────────────────
	// meshPrims[meshIdx] = []*Mesh (one entry per primitive)
	meshPrims := make([][]*Mesh, len(doc.Meshes))
	for mi, gm := range doc.Meshes {
		for pi, prim := range gm.Primitives {
			m, err := loadGLTFPrimitive(doc, gm.Name, pi, *prim)
			if err != nil {
				fmt.Printf("gltf: mesh %d prim %d: %v\n", mi, pi, err)
				continue
			}
			ComputeTangents(m)
			if prim.Material != nil && *prim.Material < len(matCache) {
				m.Material = matCache[*prim.Material]
			}
			meshPrims[mi] = append(meshPrims[mi], m)
		}
	}

	// ── 4. Nodes ──────────────────────────────────────────────────────────────
	nodes := make([]*Node, len(doc.Nodes))
	for i, gn := range doc.Nodes {
		name := gn.Name
		if name == "" {
			name = fmt.Sprintf("node_%d", i)
		}
		n := NewNode(name)

		t := gn.TranslationOrDefault()
		n.SetPosition(math.Vec3{X: float32(t[0]), Y: float32(t[1]), Z: float32(t[2])})

		sc := gn.ScaleOrDefault()
		n.SetScale(math.Vec3{X: float32(sc[0]), Y: float32(sc[1]), Z: float32(sc[2])})

		r := gn.RotationOrDefault() // [x, y, z, w]
		n.SetRotation(math.Quaternion{
			X: float32(r[0]), Y: float32(r[1]),
			Z: float32(r[2]), W: float32(r[3]),
		})

		if gn.Mesh != nil && *gn.Mesh < len(meshPrims) {
			prims := meshPrims[*gn.Mesh]
			switch len(prims) {
			case 0:
				// no geometry
			case 1:
				n.Mesh = prims[0]
			default:
				// Multiple primitives → one child node per primitive
				for pi, p := range prims {
					child := NewNode(fmt.Sprintf("%s_prim%d", name, pi))
					child.Mesh = p
					n.AddChild(child)
				}
			}
		}
		nodes[i] = n
	}

	// Wire up parent-child relationships
	for i, gn := range doc.Nodes {
		if nodes[i] == nil {
			continue
		}
		for _, childIdx := range gn.Children {
			if childIdx < len(nodes) && nodes[childIdx] != nil {
				nodes[i].AddChild(nodes[childIdx])
			}
		}
	}

	// ── 5. Root nodes ─────────────────────────────────────────────────────────
	if doc.Scene != nil && *doc.Scene < len(doc.Scenes) {
		for _, rootIdx := range doc.Scenes[*doc.Scene].Nodes {
			if rootIdx < len(nodes) && nodes[rootIdx] != nil {
				result.Roots = append(result.Roots, nodes[rootIdx])
			}
		}
	} else {
		// No default scene: collect all parentless nodes
		hasParent := make([]bool, len(nodes))
		for _, gn := range doc.Nodes {
			for _, c := range gn.Children {
				if c < len(hasParent) {
					hasParent[c] = true
				}
			}
		}
		for i, n := range nodes {
			if n != nil && !hasParent[i] {
				result.Roots = append(result.Roots, n)
			}
		}
	}

	return result, nil
}

// loadGLTFPrimitive converts one glTF mesh primitive into a scene.Mesh.
func loadGLTFPrimitive(doc *gltf.Document, meshName string, primIdx int, prim gltf.Primitive) (*Mesh, error) {
	name := fmt.Sprintf("%s_p%d", meshName, primIdx)
	if meshName == "" {
		name = fmt.Sprintf("prim_%d", primIdx)
	}

	// Positions are required
	posIdx, ok := prim.Attributes["POSITION"]
	if !ok {
		return nil, fmt.Errorf("no POSITION attribute")
	}
	positions, err := modeler.ReadPosition(doc, doc.Accessors[posIdx], nil)
	if err != nil {
		return nil, fmt.Errorf("positions: %w", err)
	}

	var normals [][3]float32
	var uvs     [][2]float32

	if idx, ok := prim.Attributes["NORMAL"]; ok {
		normals, _ = modeler.ReadNormal(doc, doc.Accessors[idx], nil)
	}
	if idx, ok := prim.Attributes["TEXCOORD_0"]; ok {
		uvs, _ = modeler.ReadTextureCoord(doc, doc.Accessors[idx], nil)
	}

	verts := make([]core.Vertex, len(positions))
	for i, p := range positions {
		v := core.Vertex{
			Position: math.Vec3{X: p[0], Y: p[1], Z: p[2]},
			Normal:   math.Vec3{X: 0, Y: 1, Z: 0},
			Color:    core.ColorWhite,
		}
		if i < len(normals) {
			n := normals[i]
			v.Normal = math.Vec3{X: n[0], Y: n[1], Z: n[2]}
		}
		if i < len(uvs) {
			v.UV = math.Vec2{X: uvs[i][0], Y: uvs[i][1]}
		}
		verts[i] = v
	}

	var indices []uint32
	if prim.Indices != nil {
		indices, err = modeler.ReadIndices(doc, doc.Accessors[*prim.Indices], nil)
		if err != nil {
			return nil, fmt.Errorf("indices: %w", err)
		}
	}

	return CreateMeshFromData(name, verts, indices), nil
}

// decodeImageBytes decodes a PNG or JPEG byte slice into an RGBA8 scene.Texture.
func decodeImageBytes(name string, data []byte) (*Texture, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	return &Texture{
		Name:   name,
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Pixels: rgba.Pix,
	}, nil
}
