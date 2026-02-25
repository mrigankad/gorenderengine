package io

import (
	"encoding/json"
	"fmt"
	"os"

	"render-engine/core"
	"render-engine/math"
)

// SceneFile is the top-level structure for the .gorscene format
type SceneFile struct {
	Version  string        `json:"version"`
	Name     string        `json:"name"`
	Camera   CameraData    `json:"camera"`
	Lights   []LightData   `json:"lights"`
	Objects  []ObjectData  `json:"objects"`
	Settings SceneSettings `json:"settings"`
}

// CameraData stores camera state
type CameraData struct {
	Position [3]float32 `json:"position"`
	Target   [3]float32 `json:"target"`
	FOV      float32    `json:"fov"`
	Near     float32    `json:"near"`
	Far      float32    `json:"far"`
	Mode     string     `json:"mode"` // "orbit" or "fly"
	Distance float32    `json:"distance,omitempty"`
	Yaw      float32    `json:"yaw,omitempty"`
	Pitch    float32    `json:"pitch,omitempty"`
}

// LightData stores light state
type LightData struct {
	Type      string     `json:"type"` // "directional", "point", "spot"
	Position  [3]float32 `json:"position"`
	Direction [3]float32 `json:"direction"`
	Color     [4]float32 `json:"color"`
	Intensity float32    `json:"intensity"`
	Range     float32    `json:"range,omitempty"`
	SpotAngle float32    `json:"spot_angle,omitempty"`
}

// ObjectData stores a scene object
type ObjectData struct {
	Name     string       `json:"name"`
	Position [3]float32   `json:"position"`
	Rotation [4]float32   `json:"rotation"` // Quaternion (x,y,z,w)
	Scale    [3]float32   `json:"scale"`
	Visible  bool         `json:"visible"`
	MeshType string       `json:"mesh_type"`           // "cube", "sphere", "imported", etc.
	MeshFile string       `json:"mesh_file,omitempty"` // for imported meshes
	Material MaterialData `json:"material"`
	Children []ObjectData `json:"children,omitempty"`
}

// MaterialData stores material properties
type MaterialData struct {
	Name         string     `json:"name"`
	DiffuseColor [4]float32 `json:"diffuse_color"`
	Roughness    float32    `json:"roughness"`
	Metallic     float32    `json:"metallic"`
	Specular     float32    `json:"specular"`
	Opacity      float32    `json:"opacity"`
	TexturePath  string     `json:"texture_path,omitempty"`
}

// SceneSettings stores editor preferences
type SceneSettings struct {
	AmbientColor [4]float32 `json:"ambient_color"`
	SkyColor     [4]float32 `json:"sky_color"`
	ShowGrid     bool       `json:"show_grid"`
	GridSize     float32    `json:"grid_size"`
	SnapToGrid   bool       `json:"snap_to_grid"`
}

// SaveScene serializes scene data to a .gorscene JSON file
func SaveScene(path string, scene *SceneFile) error {
	data, err := json.MarshalIndent(scene, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scene: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// LoadScene deserializes a .gorscene JSON file
func LoadScene(path string) (*SceneFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read scene file: %w", err)
	}

	scene := &SceneFile{}
	if err := json.Unmarshal(data, scene); err != nil {
		return nil, fmt.Errorf("failed to parse scene file: %w", err)
	}
	return scene, nil
}

// NewDefaultSceneFile creates a new scene file with sensible defaults
func NewDefaultSceneFile(name string) *SceneFile {
	return &SceneFile{
		Version: "1.0",
		Name:    name,
		Camera: CameraData{
			Position: [3]float32{0, 2, 5},
			Target:   [3]float32{0, 0, 0},
			FOV:      60.0,
			Near:     0.1,
			Far:      1000.0,
			Mode:     "orbit",
			Distance: 5.0,
		},
		Lights: []LightData{
			{
				Type:      "directional",
				Direction: [3]float32{0.5, -1, -0.5},
				Color:     [4]float32{1, 1, 1, 1},
				Intensity: 0.8,
			},
		},
		Settings: SceneSettings{
			AmbientColor: [4]float32{0.2, 0.2, 0.2, 1},
			SkyColor:     [4]float32{0.5, 0.7, 1.0, 1},
			ShowGrid:     true,
			GridSize:     1.0,
			SnapToGrid:   false,
		},
	}
}

// --- Helper conversions ---

// Vec3ToArray converts a Vec3 to a [3]float32
func Vec3ToArray(v math.Vec3) [3]float32 {
	return [3]float32{v.X, v.Y, v.Z}
}

// ArrayToVec3 converts a [3]float32 to Vec3
func ArrayToVec3(a [3]float32) math.Vec3 {
	return math.Vec3{X: a[0], Y: a[1], Z: a[2]}
}

// ColorToArray converts a Color to [4]float32
func ColorToArray(c core.Color) [4]float32 {
	return [4]float32{c.R, c.G, c.B, c.A}
}

// ArrayToColor converts [4]float32 to Color
func ArrayToColor(a [4]float32) core.Color {
	return core.Color{R: a[0], G: a[1], B: a[2], A: a[3]}
}

// QuatToArray converts a Quaternion to [4]float32
func QuatToArray(q math.Quaternion) [4]float32 {
	return [4]float32{q.X, q.Y, q.Z, q.W}
}

// ArrayToQuat converts [4]float32 to Quaternion
func ArrayToQuat(a [4]float32) math.Quaternion {
	return math.Quaternion{X: a[0], Y: a[1], Z: a[2], W: a[3]}
}
