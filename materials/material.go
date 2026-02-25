package materials

import (
	"render-engine/core"
	"render-engine/textures"
)

// Material represents a PBR-lite material for rendering
type Material struct {
	Name string

	// Color properties
	DiffuseColor  core.Color // Base color (used when DiffuseTexture is nil)
	SpecularColor core.Color
	EmissiveColor core.Color

	// PBR-lite parameters
	Roughness float32 // 0.0 = smooth/mirror, 1.0 = rough/diffuse
	Metallic  float32 // 0.0 = dielectric, 1.0 = metal
	Specular  float32 // Specular intensity
	Opacity   float32 // 1.0 = fully opaque

	// Textures (nil = use color)
	DiffuseTexture  *textures.Texture
	NormalTexture   *textures.Texture
	EmissiveTexture *textures.Texture

	// Rendering state
	DoubleSided bool
	Wireframe   bool
}

// MaterialUniform is the GPU-side representation of material properties
// Must be aligned to std140 layout rules
type MaterialUniform struct {
	DiffuseColor  [4]float32
	SpecularColor [4]float32
	EmissiveColor [4]float32
	Roughness     float32
	Metallic      float32
	Specular      float32
	Opacity       float32
	HasDiffuseTex int32
	_pad          [3]int32
}

// NewMaterial creates a new material with default values
func NewMaterial(name string) *Material {
	return &Material{
		Name:          name,
		DiffuseColor:  core.Color{R: 0.8, G: 0.8, B: 0.8, A: 1.0},
		SpecularColor: core.Color{R: 1.0, G: 1.0, B: 1.0, A: 1.0},
		EmissiveColor: core.Color{R: 0.0, G: 0.0, B: 0.0, A: 0.0},
		Roughness:     0.5,
		Metallic:      0.0,
		Specular:      0.5,
		Opacity:       1.0,
		DoubleSided:   false,
		Wireframe:     false,
	}
}

// ToUniform converts the material to its GPU representation
func (m *Material) ToUniform() MaterialUniform {
	u := MaterialUniform{
		DiffuseColor:  [4]float32{m.DiffuseColor.R, m.DiffuseColor.G, m.DiffuseColor.B, m.DiffuseColor.A},
		SpecularColor: [4]float32{m.SpecularColor.R, m.SpecularColor.G, m.SpecularColor.B, m.SpecularColor.A},
		EmissiveColor: [4]float32{m.EmissiveColor.R, m.EmissiveColor.G, m.EmissiveColor.B, m.EmissiveColor.A},
		Roughness:     m.Roughness,
		Metallic:      m.Metallic,
		Specular:      m.Specular,
		Opacity:       m.Opacity,
	}
	if m.DiffuseTexture != nil {
		u.HasDiffuseTex = 1
	}
	return u
}

// Clone creates a deep copy of the material (textures are shared)
func (m *Material) Clone(newName string) *Material {
	clone := *m
	clone.Name = newName
	return &clone
}

// --- Default Material Library ---

// DefaultMaterial creates a standard grey material
func DefaultMaterial() *Material {
	return NewMaterial("Default")
}

// RedMaterial creates a red diffuse material
func RedMaterial() *Material {
	m := NewMaterial("Red")
	m.DiffuseColor = core.ColorRed
	return m
}

// GreenMaterial creates a green diffuse material
func GreenMaterial() *Material {
	m := NewMaterial("Green")
	m.DiffuseColor = core.ColorGreen
	return m
}

// BlueMaterial creates a blue diffuse material
func BlueMaterial() *Material {
	m := NewMaterial("Blue")
	m.DiffuseColor = core.ColorBlue
	return m
}

// MetalMaterial creates a metallic material
func MetalMaterial() *Material {
	m := NewMaterial("Metal")
	m.DiffuseColor = core.Color{R: 0.9, G: 0.9, B: 0.9, A: 1.0}
	m.Metallic = 1.0
	m.Roughness = 0.2
	m.Specular = 1.0
	return m
}

// GlassMaterial creates a transparent glass material
func GlassMaterial() *Material {
	m := NewMaterial("Glass")
	m.DiffuseColor = core.Color{R: 0.9, G: 0.95, B: 1.0, A: 0.3}
	m.Roughness = 0.05
	m.Specular = 1.0
	m.Opacity = 0.3
	return m
}

// EmissiveMaterial creates a self-illuminating material
func EmissiveMaterial(r, g, b float32) *Material {
	m := NewMaterial("Emissive")
	m.DiffuseColor = core.Color{R: r, G: g, B: b, A: 1.0}
	m.EmissiveColor = core.Color{R: r, G: g, B: b, A: 1.0}
	return m
}
