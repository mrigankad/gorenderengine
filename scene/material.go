package scene

import "render-engine/core"

// Material describes surface appearance properties for a mesh.
// Supports both Phong shading and PBR (Cook-Torrance BRDF).
// Set UsePBR = true to use physically-based rendering.
type Material struct {
	Name      string
	Albedo    core.Color // base diffuse color (multiplied with albedo texture if set)
	Specular  core.Color // Phong specular highlight color (ignored when UsePBR = true)
	Shininess float32    // Phong shininess exponent (1–256+; ignored when UsePBR = true)
	Unlit     bool       // skip lighting calculation — output raw albedo/texture color

	// PBR parameters (used when UsePBR = true)
	UsePBR      bool       // switch to Cook-Torrance BRDF instead of Phong
	Metallic    float32    // 0 = dielectric, 1 = fully metallic
	Roughness   float32    // 0 = perfectly smooth, 1 = fully rough
	EmissiveColor core.Color // self-emitted radiance (additive; use bright values for HDR glow)

	// Optional albedo texture; if set, it is multiplied with Albedo.
	// Upload via opengl.UploadTexture before rendering.
	AlbedoTexture *Texture

	// Optional tangent-space normal map (RGB → XYZ normals, stored 0-1 → -1..1).
	// Upload via opengl.UploadTexture before rendering.
	NormalTexture *Texture

	// Optional PBR combined metallic-roughness texture (glTF convention):
	//   G channel = roughness, B channel = metallic.
	// Upload via opengl.UploadTexture before rendering.
	MetallicRoughnessTexture *Texture

	// Optional emissive texture; multiplied with EmissiveColor.
	// Upload via opengl.UploadTexture before rendering.
	EmissiveTexture *Texture
}

// DefaultMaterial returns a plain white matte Phong material.
func DefaultMaterial() *Material {
	return &Material{
		Name:      "Default",
		Albedo:    core.ColorWhite,
		Specular:  core.Color{R: 0.3, G: 0.3, B: 0.3, A: 1},
		Shininess: 32,
		Roughness: 0.5,
	}
}

// NewMaterial creates a Phong material with the given albedo color.
func NewMaterial(name string, albedo core.Color) *Material {
	return &Material{
		Name:      name,
		Albedo:    albedo,
		Specular:  core.Color{R: 0.5, G: 0.5, B: 0.5, A: 1},
		Shininess: 32,
		Roughness: 0.5,
	}
}

// NewPBRMaterial creates a PBR material with the given albedo, metallic, and roughness.
func NewPBRMaterial(name string, albedo core.Color, metallic, roughness float32) *Material {
	return &Material{
		Name:      name,
		Albedo:    albedo,
		Metallic:  metallic,
		Roughness: roughness,
		UsePBR:    true,
	}
}
