package scene

import (
	"encoding/json"
	"fmt"
	"os"

	"render-engine/core"
	"render-engine/math"
)

// ── JSON data structures ──────────────────────────────────────────────────────

type vec3JSON struct {
	X, Y, Z float32
}

type colorJSON struct {
	R, G, B, A float32
}

type transformJSON struct {
	Position vec3JSON
	Scale    vec3JSON
	// Quaternion stored as (X, Y, Z, W)
	RotX, RotY, RotZ, RotW float32
}

type materialJSON struct {
	Name      string
	Albedo    colorJSON
	Specular  colorJSON
	Shininess float32
	Unlit     bool
}

type nodeJSON struct {
	ID        uint32
	Name      string
	Transform transformJSON
	Visible   bool
	MeshName  string // hint for re-attaching meshes; not used during load
	Material  *materialJSON
	Children  []nodeJSON
}

type lightJSON struct {
	Type      int
	Position  vec3JSON
	Direction vec3JSON
	Color     colorJSON
	Intensity float32
	Range     float32
	SpotAngle float32
}

type cameraJSON struct {
	Position    vec3JSON
	FOV         float32
	AspectRatio float32
	NearPlane   float32
	FarPlane    float32
}

type sceneJSON struct {
	Version  int
	SkyColor colorJSON
	Ambient  colorJSON
	Camera   *cameraJSON
	Lights   []lightJSON
	Nodes    []nodeJSON
}

// ── Save ──────────────────────────────────────────────────────────────────────

// SaveScene serialises the scene (transforms, lights, camera, materials)
// to a JSON file at path.  Mesh geometry is not stored — re-attach meshes
// after loading by matching NodeJSON.MeshName.
func SaveScene(s *Scene, path string) error {
	js := sceneJSON{
		Version:  1,
		SkyColor: colorToJSON(s.SkyColor),
		Ambient:  colorToJSON(s.Ambient),
	}

	if s.Camera != nil {
		js.Camera = &cameraJSON{
			Position:    vec3ToJSON(s.Camera.Position),
			FOV:         s.Camera.FOV,
			AspectRatio: s.Camera.AspectRatio,
			NearPlane:   s.Camera.NearPlane,
			FarPlane:    s.Camera.FarPlane,
		}
	}

	for _, l := range s.Lights {
		js.Lights = append(js.Lights, lightToJSON(l))
	}

	// Serialise the root's direct children (skip the root node itself)
	for _, child := range s.Root.Children {
		js.Nodes = append(js.Nodes, nodeToJSON(child))
	}

	data, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal scene: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write scene %q: %w", path, err)
	}
	return nil
}

// ── Load ──────────────────────────────────────────────────────────────────────

// SceneData is returned by LoadScene and contains all serialised state.
// Meshes are not stored; re-attach them by iterating Nodes and matching MeshName.
type SceneData struct {
	SkyColor core.Color
	Ambient  core.Color
	Camera   *Camera
	Lights   []*Light
	Nodes    []*Node // fully constructed node hierarchy (no meshes)
}

// LoadScene reads a JSON file saved by SaveScene and reconstructs the scene
// state (nodes, transforms, lights, camera).  Assign meshes afterward.
func LoadScene(path string) (*SceneData, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scene %q: %w", path, err)
	}
	var js sceneJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, fmt.Errorf("unmarshal scene: %w", err)
	}

	sd := &SceneData{
		SkyColor: jsonToColor(js.SkyColor),
		Ambient:  jsonToColor(js.Ambient),
	}

	if js.Camera != nil {
		cam := NewCamera(js.Camera.FOV, js.Camera.AspectRatio, js.Camera.NearPlane, js.Camera.FarPlane)
		cam.SetPosition(jsonToVec3(js.Camera.Position))
		sd.Camera = cam
	}

	for _, lj := range js.Lights {
		sd.Lights = append(sd.Lights, jsonToLight(lj))
	}

	for _, nj := range js.Nodes {
		sd.Nodes = append(sd.Nodes, jsonToNode(nj, nil))
	}

	return sd, nil
}

// ApplyToScene applies SceneData to an existing Scene, replacing camera /
// lights / nodes.  Existing nodes in the scene are removed first.
func (sd *SceneData) ApplyToScene(s *Scene) {
	s.SkyColor = sd.SkyColor
	s.Ambient = sd.Ambient

	if sd.Camera != nil {
		s.Camera = sd.Camera
	}

	s.Lights = sd.Lights

	// Clear existing children and re-add
	s.Root.Children = s.Root.Children[:0]
	for _, n := range sd.Nodes {
		s.AddNode(n)
	}
}

// ── conversion helpers ────────────────────────────────────────────────────────

func vec3ToJSON(v math.Vec3) vec3JSON        { return vec3JSON{v.X, v.Y, v.Z} }
func jsonToVec3(v vec3JSON) math.Vec3        { return math.Vec3{X: v.X, Y: v.Y, Z: v.Z} }
func colorToJSON(c core.Color) colorJSON     { return colorJSON{c.R, c.G, c.B, c.A} }
func jsonToColor(c colorJSON) core.Color     { return core.Color{R: c.R, G: c.G, B: c.B, A: c.A} }

func transformToJSON(t core.Transform) transformJSON {
	return transformJSON{
		Position: vec3ToJSON(t.Position),
		Scale:    vec3ToJSON(t.Scale),
		RotX:     t.Rotation.X,
		RotY:     t.Rotation.Y,
		RotZ:     t.Rotation.Z,
		RotW:     t.Rotation.W,
	}
}

func jsonToTransform(tj transformJSON) core.Transform {
	t := core.NewTransform()
	t.Position = jsonToVec3(tj.Position)
	t.Scale = jsonToVec3(tj.Scale)
	t.Rotation = math.Quaternion{X: tj.RotX, Y: tj.RotY, Z: tj.RotZ, W: tj.RotW}
	return t
}

func lightToJSON(l *Light) lightJSON {
	return lightJSON{
		Type:      l.Type,
		Position:  vec3ToJSON(l.Position),
		Direction: vec3ToJSON(l.Direction),
		Color:     colorToJSON(l.Color),
		Intensity: l.Intensity,
		Range:     l.Range,
		SpotAngle: l.SpotAngle,
	}
}

func jsonToLight(lj lightJSON) *Light {
	return &Light{
		Type:      lj.Type,
		Position:  jsonToVec3(lj.Position),
		Direction: jsonToVec3(lj.Direction),
		Color:     jsonToColor(lj.Color),
		Intensity: lj.Intensity,
		Range:     lj.Range,
		SpotAngle: lj.SpotAngle,
	}
}

func matToJSON(m *Material) *materialJSON {
	if m == nil {
		return nil
	}
	return &materialJSON{
		Name:      m.Name,
		Albedo:    colorToJSON(m.Albedo),
		Specular:  colorToJSON(m.Specular),
		Shininess: m.Shininess,
		Unlit:     m.Unlit,
	}
}

func jsonToMat(mj *materialJSON) *Material {
	if mj == nil {
		return nil
	}
	return &Material{
		Name:      mj.Name,
		Albedo:    jsonToColor(mj.Albedo),
		Specular:  jsonToColor(mj.Specular),
		Shininess: mj.Shininess,
		Unlit:     mj.Unlit,
	}
}

func nodeToJSON(n *Node) nodeJSON {
	nj := nodeJSON{
		ID:        n.Id,
		Name:      n.Name,
		Transform: transformToJSON(n.Transform),
		Visible:   n.Visible,
	}
	if n.Mesh != nil {
		nj.MeshName = n.Mesh.Name
		nj.Material = matToJSON(n.Mesh.Material)
	}
	for _, child := range n.Children {
		nj.Children = append(nj.Children, nodeToJSON(child))
	}
	return nj
}

func jsonToNode(nj nodeJSON, parent *Node) *Node {
	n := NewNode(nj.Name)
	n.Transform = jsonToTransform(nj.Transform)
	n.Visible = nj.Visible
	n.MarkWorldMatrixDirty()

	// Meshes are not serialised — the caller must re-attach them.
	// We store MeshName as a hint on a transient Mesh placeholder.
	if nj.MeshName != "" {
		placeholder := NewMesh(nj.MeshName)
		placeholder.Material = jsonToMat(nj.Material)
		n.Mesh = placeholder
	}

	for _, childJSON := range nj.Children {
		child := jsonToNode(childJSON, n)
		n.AddChild(child)
	}
	return n
}
