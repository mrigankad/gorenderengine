package opengl

import (
	"fmt"
	gomath "math"
	"strings"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
)

// GPUMesh holds the OpenGL buffer objects for an uploaded mesh.
type GPUMesh struct {
	VAO         uint32
	VBO         uint32
	EBO         uint32
	IndexCount  int32
	HasIndices  bool
	InstanceVBO uint32 // per-instance data VBO (0 = not yet allocated)
	InstanceCap int    // capacity of InstanceVBO in instances
}

// Renderer is the OpenGL rendering backend.
type Renderer struct {
	program uint32

	// Vertex transform uniforms
	mvpLoc          int32
	modelLoc        int32
	lightViewProjLoc int32 // per-frame light VP for shadow map

	// Lighting uniforms — directional
	lightDirLoc       int32
	lightColorLoc     int32
	lightIntensityLoc int32
	ambientColorLoc   int32

	// Lighting uniforms — point lights (up to 8)
	pointLightCountLoc     int32
	pointLightPosLoc       [8]int32
	pointLightColorLoc     [8]int32
	pointLightIntensityLoc [8]int32
	pointLightRangeLoc     [8]int32

	// Lighting uniforms — spot lights (up to 4)
	spotLightCountLoc     int32
	spotLightPosLoc       [4]int32
	spotLightDirLoc       [4]int32
	spotLightColorLoc     [4]int32
	spotLightIntensityLoc [4]int32
	spotLightRangeLoc     [4]int32
	spotLightInnerLoc     [4]int32
	spotLightOuterLoc     [4]int32

	// Camera uniform (for specular)
	cameraPosLoc int32

	// Material uniforms — Phong
	matAlbedoLoc    int32
	matSpecularLoc  int32
	matShininessLoc int32

	// Material uniforms — PBR
	usePBRLoc      int32
	matMetallicLoc int32
	matRoughnessLoc int32
	matEmissiveLoc  int32

	// Texture uniforms
	albedoTexLoc   int32
	hasTextureLoc  int32
	normalTexLoc   int32
	hasNormalTexLoc int32

	// PBR texture uniforms
	metallicRoughnessTexLoc    int32
	hasMetallicRoughnessTexLoc int32
	emissiveTexLoc             int32
	hasEmissiveTexLoc          int32

	// Fog
	fogEnabledLoc int32
	fogColorLoc   int32
	fogDensityLoc int32
	fogEnabled    bool
	fogColor      core.Color
	fogDensity    float32

	// IBL (sky-based irradiance)
	useIBLLoc    int32
	iblZenithLoc int32
	iblHorizonLoc int32
	iblGroundLoc  int32
	iblEnabled   bool
	iblZenith    core.Color
	iblHorizon   core.Color
	iblGround    core.Color

	// Instancing
	instancedLoc int32

	// Unlit mode
	unlitLoc int32

	// Shadow map uniforms (main shader)
	shadowMapLoc  int32
	hasShadowsLoc int32

	// Shadow depth shader
	shadowProg        uint32
	shadowLightMVPLoc int32

	// Shadow map FBO (nil if shadows not enabled)
	shadowMap *ShadowMap

	// Stored viewport for restoring after shadow pass
	viewportW int32
	viewportH int32

	// Post-processing FBO (nil if disabled)
	postProcess *PostProcessFBO

	// SSAO (nil if disabled; requires postProcess)
	ssao     *SSAO
	lastProj math.Mat4 // stored each frame for SSAO pass

	// Skybox (nil if disabled)
	skybox *Skybox

	// Particle renderer (nil until first DrawParticles call)
	particleRenderer *ParticleRenderer

	// Text renderer (nil until first DrawText call)
	textRenderer *TextRenderer

	// Render state
	wireframe bool

	gpuMeshes map[*scene.Mesh]*GPUMesh
}

// ── Shaders ───────────────────────────────────────────────────────────────────

// vertex shader: MVP + model transform, world-space position and normal to fragment.
// Also computes fragLightSpacePos for shadow map lookup.
const vertSrc = `
#version 410 core
layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inUV;
layout(location = 3) in vec4 inColor;
layout(location = 4) in vec3 inTangent;
layout(location = 5) in vec3 inBitangent;

// Per-instance data (active only when instanced == true)
// Each mat4 occupies 4 consecutive vec4 attribute slots (one per column).
layout(location = 6)  in vec4 instMVP0;
layout(location = 7)  in vec4 instMVP1;
layout(location = 8)  in vec4 instMVP2;
layout(location = 9)  in vec4 instMVP3;
layout(location = 10) in vec4 instModel0;
layout(location = 11) in vec4 instModel1;
layout(location = 12) in vec4 instModel2;
layout(location = 13) in vec4 instModel3;

uniform mat4 mvp;
uniform mat4 model;
uniform mat4 lightViewProj;
uniform bool instanced;

out vec4 fragColor;
out vec3 fragNormal;
out vec2 fragUV;
out vec3 fragWorldPos;
out vec4 fragLightSpacePos;
out vec3 fragTangent;
out vec3 fragBitangent;

void main() {
    mat4 effectiveMVP;
    mat3 normalMat;
    vec4 worldPos;

    if (instanced) {
        mat4 iMVP   = mat4(instMVP0,   instMVP1,   instMVP2,   instMVP3);
        mat4 iModel = mat4(instModel0, instModel1, instModel2, instModel3);
        effectiveMVP      = iMVP;
        normalMat         = mat3(iModel);
        worldPos          = iModel * vec4(inPosition, 1.0);
        fragLightSpacePos = lightViewProj * worldPos;
    } else {
        effectiveMVP      = mvp;
        normalMat         = mat3(model);
        worldPos          = model * vec4(inPosition, 1.0);
        fragLightSpacePos = lightViewProj * worldPos;
    }

    gl_Position   = effectiveMVP * vec4(inPosition, 1.0);
    fragColor     = inColor;
    fragNormal    = normalMat * inNormal;
    fragUV        = inUV;
    fragWorldPos  = worldPos.xyz;
    fragTangent   = normalMat * inTangent;
    fragBitangent = normalMat * inBitangent;
}
` + "\x00"

// fragment shader: dual-path Phong + PBR (Cook-Torrance) with directional + point + spot lights.
// Set usePBR=true to use GGX/Smith/Schlick BRDF instead of Phong.
// Directional light shadows via PCF sampler2DShadow.
const fragSrc = `
#version 410 core
in vec4 fragColor;
in vec3 fragNormal;
in vec2 fragUV;
in vec3 fragWorldPos;
in vec4 fragLightSpacePos;
in vec3 fragTangent;
in vec3 fragBitangent;

out vec4 outColor;

// Directional light
uniform vec3  lightDir;
uniform vec3  lightColor;
uniform float lightIntensity;
uniform vec3  ambientColor;

// Point lights (up to 8)
#define MAX_POINT_LIGHTS 8
uniform int   pointLightCount;
uniform vec3  pointLightPos[MAX_POINT_LIGHTS];
uniform vec3  pointLightColor[MAX_POINT_LIGHTS];
uniform float pointLightIntensity[MAX_POINT_LIGHTS];
uniform float pointLightRange[MAX_POINT_LIGHTS];

// Spot lights (up to 4)
#define MAX_SPOT_LIGHTS 4
uniform int   spotLightCount;
uniform vec3  spotLightPos[MAX_SPOT_LIGHTS];
uniform vec3  spotLightDir[MAX_SPOT_LIGHTS];
uniform vec3  spotLightColor[MAX_SPOT_LIGHTS];
uniform float spotLightIntensity[MAX_SPOT_LIGHTS];
uniform float spotLightRange[MAX_SPOT_LIGHTS];
uniform float spotLightInner[MAX_SPOT_LIGHTS];
uniform float spotLightOuter[MAX_SPOT_LIGHTS];

// Camera
uniform vec3 cameraPos;

// Phong material
uniform vec3  matAlbedo;
uniform vec3  matSpecular;
uniform float matShininess;

// PBR material
uniform bool  usePBR;
uniform float matMetallic;
uniform float matRoughness;
uniform vec3  matEmissive;

// Albedo texture (unit 0)
uniform sampler2D albedoTex;
uniform bool      hasTexture;

// Shadow map (unit 1) — sampler2DShadow enables hardware PCF comparison
uniform sampler2DShadow shadowMap;
uniform bool            hasShadows;

// Normal map (unit 2) — tangent-space RGB normal map
uniform sampler2D normalTex;
uniform bool      hasNormalTex;

// PBR metallic-roughness texture (unit 3): G=roughness, B=metallic (glTF convention)
uniform sampler2D metallicRoughnessTex;
uniform bool      hasMetallicRoughnessTex;

// Emissive texture (unit 4): multiplied with matEmissive
uniform sampler2D emissiveTex;
uniform bool      hasEmissiveTex;

// When true, skip all lighting and output raw base color
uniform bool unlit;

// Exponential depth fog
uniform bool  fogEnabled;
uniform vec3  fogColor;
uniform float fogDensity; // 0 = no fog; typical range 0.01–0.15

// Sky-based IBL: hemisphere gradient matching the procedural skybox
uniform bool useIBL;
uniform vec3 iblZenith;   // sky colour straight up
uniform vec3 iblHorizon;  // sky colour at eye level
uniform vec3 iblGround;   // sky colour below horizon

// ── Shadow ───────────────────────────────────────────────────────────────────

float calcShadow() {
    vec3 p = fragLightSpacePos.xyz / fragLightSpacePos.w;
    p = p * 0.5 + 0.5;
    if (p.z > 1.0) return 1.0;
    float shadow = 0.0;
    float ts = 1.0 / 2048.0;
    for (int x = -1; x <= 1; x++) {
        for (int y = -1; y <= 1; y++) {
            shadow += texture(shadowMap, vec3(p.xy + vec2(float(x), float(y)) * ts, p.z - 0.002));
        }
    }
    return shadow / 9.0;
}

// ── Phong helpers ────────────────────────────────────────────────────────────

vec3 calcSpecular(vec3 N, vec3 L, vec3 V) {
    vec3 H = normalize(L + V);
    return matSpecular * pow(max(dot(N, H), 0.0), matShininess);
}

// ── PBR helpers (Cook-Torrance BRDF) ─────────────────────────────────────────

const float PI = 3.14159265359;

float DistributionGGX(vec3 N, vec3 H, float roughness) {
    float a  = roughness * roughness;
    float a2 = a * a;
    float NdH = max(dot(N, H), 0.0);
    float d   = NdH * NdH * (a2 - 1.0) + 1.0;
    return a2 / (PI * d * d);
}

float GeometrySchlickGGX(float cosTheta, float roughness) {
    float r = roughness + 1.0;
    float k = (r * r) / 8.0;
    return cosTheta / (cosTheta * (1.0 - k) + k);
}

float GeometrySmith(float NdV, float NdL, float roughness) {
    return GeometrySchlickGGX(NdV, roughness) * GeometrySchlickGGX(NdL, roughness);
}

vec3 FresnelSchlick(float cosTheta, vec3 F0) {
    return F0 + (1.0 - F0) * pow(clamp(1.0 - cosTheta, 0.0, 1.0), 5.0);
}

// Fresnel with roughness remapping for IBL diffuse/specular split
vec3 FresnelSchlickRoughness(float cosTheta, vec3 F0, float roughness) {
    return F0 + (max(vec3(1.0 - roughness), F0) - F0) * pow(clamp(1.0 - cosTheta, 0.0, 1.0), 5.0);
}

// Sample the procedural sky gradient in direction dir (must be normalised).
// dir.y > 0 → lerp horizon→zenith; dir.y < 0 → lerp horizon→ground.
vec3 sampleSkyGradient(vec3 dir) {
    float y = clamp(dir.y, -1.0, 1.0);
    if (y >= 0.0) return mix(iblHorizon, iblZenith,  y);
    else          return mix(iblHorizon, iblGround,  -y);
}

// Evaluate one Cook-Torrance lobe. L = unit vector toward light, rad = light radiance.
vec3 evalPBR(vec3 N, vec3 V, vec3 L, vec3 rad, vec3 albedo, float metallic, float roughness, vec3 F0) {
    float NdL = max(dot(N, L), 0.0);
    if (NdL <= 0.0) return vec3(0.0);

    vec3  H   = normalize(V + L);
    float NdV = max(dot(N, V), 0.0);

    float D  = DistributionGGX(N, H, roughness);
    float G  = GeometrySmith(NdV, NdL, roughness);
    vec3  F  = FresnelSchlick(max(dot(H, V), 0.0), F0);

    vec3 kD       = (vec3(1.0) - F) * (1.0 - metallic);
    vec3 specular = D * G * F / max(4.0 * NdV * NdL, 0.001);

    return (kD * albedo / PI + specular) * rad * NdL;
}

// ── Main ─────────────────────────────────────────────────────────────────────

void main() {
    // World-space normal — from normal map (TBN) or interpolated vertex normal
    vec3 N;
    if (hasNormalTex) {
        vec3 T  = normalize(fragTangent);
        vec3 B  = normalize(fragBitangent);
        vec3 Nv = normalize(fragNormal);
        mat3 TBN = mat3(T, B, Nv);
        N = normalize(TBN * (texture(normalTex, fragUV).rgb * 2.0 - 1.0));
    } else {
        N = normalize(fragNormal);
    }
    vec3 V = normalize(cameraPos - fragWorldPos);

    // Base color: vertex color * material albedo (* texture if present)
    vec4 baseColor = fragColor * vec4(matAlbedo, 1.0);
    if (hasTexture) {
        baseColor *= texture(albedoTex, fragUV);
    }

    // Unlit: skip all lighting
    if (unlit) {
        outColor = baseColor;
        return;
    }

    float shadowFactor = hasShadows ? calcShadow() : 1.0;

    // ── PBR path ─────────────────────────────────────────────────────────────
    if (usePBR) {
        float metallic  = matMetallic;
        float roughness = clamp(matRoughness, 0.04, 1.0);
        if (hasMetallicRoughnessTex) {
            vec4 mr  = texture(metallicRoughnessTex, fragUV);
            roughness = clamp(mr.g, 0.04, 1.0);
            metallic  = mr.b;
        }

        vec3 albedo = baseColor.rgb;
        vec3 F0     = mix(vec3(0.04), albedo, metallic);

        // Ambient: sky-based IBL or flat fallback
        vec3 color;
        if (useIBL) {
            // Diffuse irradiance: sky gradient sampled at surface normal direction
            vec3 irradiance = sampleSkyGradient(N);
            vec3 F_ibl = FresnelSchlickRoughness(max(dot(N, V), 0.0), F0, roughness);
            vec3 kD    = (vec3(1.0) - F_ibl) * (1.0 - metallic);
            vec3 diffuseIBL = irradiance * albedo * kD;
            // Specular IBL: sky gradient in reflected direction, fades with roughness
            vec3 R = reflect(-V, N);
            vec3 specIrradiance = sampleSkyGradient(R);
            float specStrength  = (1.0 - roughness * roughness);
            vec3 specularIBL    = specIrradiance * F_ibl * specStrength;
            color = diffuseIBL + specularIBL;
        } else {
            color = ambientColor * albedo * (1.0 - 0.5 * metallic);
        }

        // Directional light
        vec3 L_dir = normalize(-lightDir);
        vec3 dirRad = lightColor * lightIntensity * shadowFactor;
        color += evalPBR(N, V, L_dir, dirRad, albedo, metallic, roughness, F0);

        // Point lights
        for (int i = 0; i < pointLightCount && i < MAX_POINT_LIGHTS; i++) {
            vec3  toLight = pointLightPos[i] - fragWorldPos;
            float dist    = length(toLight);
            float range   = max(pointLightRange[i], 0.001);
            float atten   = clamp(1.0 - (dist*dist)/(range*range), 0.0, 1.0);
            atten *= atten;
            vec3 ptRad = pointLightColor[i] * pointLightIntensity[i] * atten;
            color += evalPBR(N, V, normalize(toLight), ptRad, albedo, metallic, roughness, F0);
        }

        // Spot lights
        for (int i = 0; i < spotLightCount && i < MAX_SPOT_LIGHTS; i++) {
            vec3  toLight = spotLightPos[i] - fragWorldPos;
            float dist    = length(toLight);
            float range   = max(spotLightRange[i], 0.001);
            float atten   = clamp(1.0 - (dist*dist)/(range*range), 0.0, 1.0);
            atten *= atten;
            vec3  L     = normalize(toLight);
            float theta = dot(L, normalize(-spotLightDir[i]));
            float eps   = spotLightInner[i] - spotLightOuter[i];
            float cone  = clamp((theta - spotLightOuter[i]) / eps, 0.0, 1.0);
            vec3 spRad = spotLightColor[i] * spotLightIntensity[i] * atten * cone;
            color += evalPBR(N, V, L, spRad, albedo, metallic, roughness, F0);
        }

        // Emissive
        vec3 emissive = matEmissive;
        if (hasEmissiveTex) {
            emissive *= texture(emissiveTex, fragUV).rgb;
        }
        color += emissive;

        if (fogEnabled) {
            float fogDist = length(fragWorldPos - cameraPos);
            float fogF    = clamp(exp(-fogDensity * fogDist), 0.0, 1.0);
            color = mix(fogColor, color, fogF);
        }
        outColor = vec4(color, baseColor.a);
        return;
    }

    // ── Phong path ───────────────────────────────────────────────────────────
    vec3 color;
    if (useIBL) {
        color = sampleSkyGradient(N) * baseColor.rgb * 0.35;
    } else {
        color = ambientColor * baseColor.rgb;
    }

    // Directional light
    vec3 L_dir = normalize(-lightDir);
    float NdL  = max(dot(N, L_dir), 0.0);
    color += shadowFactor * lightColor * lightIntensity * NdL * baseColor.rgb;
    if (NdL > 0.0) {
        color += shadowFactor * lightColor * lightIntensity * calcSpecular(N, L_dir, V);
    }

    // Point lights
    for (int i = 0; i < pointLightCount && i < MAX_POINT_LIGHTS; i++) {
        vec3  toLight = pointLightPos[i] - fragWorldPos;
        float dist    = length(toLight);
        float range   = max(pointLightRange[i], 0.001);
        float atten   = clamp(1.0 - (dist * dist) / (range * range), 0.0, 1.0);
        atten *= atten;
        vec3  L_pt = normalize(toLight);
        float NdL2 = max(dot(N, L_pt), 0.0);
        color += pointLightColor[i] * pointLightIntensity[i] * atten * NdL2 * baseColor.rgb;
        if (NdL2 > 0.0) {
            color += pointLightColor[i] * pointLightIntensity[i] * atten * calcSpecular(N, L_pt, V);
        }
    }

    // Spot lights
    for (int i = 0; i < spotLightCount && i < MAX_SPOT_LIGHTS; i++) {
        vec3  toLight = spotLightPos[i] - fragWorldPos;
        float dist    = length(toLight);
        float range   = max(spotLightRange[i], 0.001);
        float atten   = clamp(1.0 - (dist * dist) / (range * range), 0.0, 1.0);
        atten *= atten;
        vec3  L     = normalize(toLight);
        float theta = dot(L, normalize(-spotLightDir[i]));
        float eps   = spotLightInner[i] - spotLightOuter[i];
        float cone  = clamp((theta - spotLightOuter[i]) / eps, 0.0, 1.0);
        float NdL3  = max(dot(N, L), 0.0);
        float contrib = atten * cone * spotLightIntensity[i];
        color += spotLightColor[i] * contrib * NdL3 * baseColor.rgb;
        if (NdL3 > 0.0) {
            color += spotLightColor[i] * contrib * calcSpecular(N, L, V);
        }
    }

    if (fogEnabled) {
        float fogDist = length(fragWorldPos - cameraPos);
        float fogF    = clamp(exp(-fogDensity * fogDist), 0.0, 1.0);
        color = mix(fogColor, color, fogF);
    }
    outColor = vec4(color, baseColor.a);
}
` + "\x00"

// depth-only vertex shader for the shadow map pass
const depthVertSrc = `
#version 410 core
layout(location = 0) in vec3 inPosition;
uniform mat4 lightMVP;
void main() {
    gl_Position = lightMVP * vec4(inPosition, 1.0);
}
` + "\x00"

// depth-only fragment shader (OpenGL writes depth implicitly)
const depthFragSrc = `
#version 410 core
void main() {}
` + "\x00"

// ── NewRenderer ───────────────────────────────────────────────────────────────

// NewRenderer initialises OpenGL.
// Must be called after the GLFW window context is made current.
func NewRenderer() (*Renderer, error) {
	if err := gl.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize OpenGL: %w", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Printf("OpenGL version: %s\n", version)

	prog, err := newProgram(vertSrc, fragSrc)
	if err != nil {
		return nil, fmt.Errorf("main shader compile: %w", err)
	}

	shadowProg, err := newProgram(depthVertSrc, depthFragSrc)
	if err != nil {
		return nil, fmt.Errorf("depth shader compile: %w", err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	r := &Renderer{
		program:    prog,
		shadowProg: shadowProg,

		mvpLoc:           gl.GetUniformLocation(prog, gl.Str("mvp\x00")),
		modelLoc:         gl.GetUniformLocation(prog, gl.Str("model\x00")),
		lightViewProjLoc: gl.GetUniformLocation(prog, gl.Str("lightViewProj\x00")),

		lightDirLoc:       gl.GetUniformLocation(prog, gl.Str("lightDir\x00")),
		lightColorLoc:     gl.GetUniformLocation(prog, gl.Str("lightColor\x00")),
		lightIntensityLoc: gl.GetUniformLocation(prog, gl.Str("lightIntensity\x00")),
		ambientColorLoc:   gl.GetUniformLocation(prog, gl.Str("ambientColor\x00")),

		pointLightCountLoc: gl.GetUniformLocation(prog, gl.Str("pointLightCount\x00")),
		cameraPosLoc:       gl.GetUniformLocation(prog, gl.Str("cameraPos\x00")),

		matAlbedoLoc:    gl.GetUniformLocation(prog, gl.Str("matAlbedo\x00")),
		matSpecularLoc:  gl.GetUniformLocation(prog, gl.Str("matSpecular\x00")),
		matShininessLoc: gl.GetUniformLocation(prog, gl.Str("matShininess\x00")),

		usePBRLoc:       gl.GetUniformLocation(prog, gl.Str("usePBR\x00")),
		matMetallicLoc:  gl.GetUniformLocation(prog, gl.Str("matMetallic\x00")),
		matRoughnessLoc: gl.GetUniformLocation(prog, gl.Str("matRoughness\x00")),
		matEmissiveLoc:  gl.GetUniformLocation(prog, gl.Str("matEmissive\x00")),

		albedoTexLoc:    gl.GetUniformLocation(prog, gl.Str("albedoTex\x00")),
		hasTextureLoc:   gl.GetUniformLocation(prog, gl.Str("hasTexture\x00")),
		normalTexLoc:    gl.GetUniformLocation(prog, gl.Str("normalTex\x00")),
		hasNormalTexLoc: gl.GetUniformLocation(prog, gl.Str("hasNormalTex\x00")),

		metallicRoughnessTexLoc:    gl.GetUniformLocation(prog, gl.Str("metallicRoughnessTex\x00")),
		hasMetallicRoughnessTexLoc: gl.GetUniformLocation(prog, gl.Str("hasMetallicRoughnessTex\x00")),
		emissiveTexLoc:             gl.GetUniformLocation(prog, gl.Str("emissiveTex\x00")),
		hasEmissiveTexLoc:          gl.GetUniformLocation(prog, gl.Str("hasEmissiveTex\x00")),

		instancedLoc: gl.GetUniformLocation(prog, gl.Str("instanced\x00")),
		unlitLoc:     gl.GetUniformLocation(prog, gl.Str("unlit\x00")),

		useIBLLoc:    gl.GetUniformLocation(prog, gl.Str("useIBL\x00")),
		iblZenithLoc:  gl.GetUniformLocation(prog, gl.Str("iblZenith\x00")),
		iblHorizonLoc: gl.GetUniformLocation(prog, gl.Str("iblHorizon\x00")),
		iblGroundLoc:  gl.GetUniformLocation(prog, gl.Str("iblGround\x00")),

		fogEnabledLoc: gl.GetUniformLocation(prog, gl.Str("fogEnabled\x00")),
		fogColorLoc:   gl.GetUniformLocation(prog, gl.Str("fogColor\x00")),
		fogDensityLoc: gl.GetUniformLocation(prog, gl.Str("fogDensity\x00")),
		fogDensity:    0.03,
		fogColor:      core.Color{R: 0.7, G: 0.7, B: 0.75, A: 1},

		shadowMapLoc:  gl.GetUniformLocation(prog, gl.Str("shadowMap\x00")),
		hasShadowsLoc: gl.GetUniformLocation(prog, gl.Str("hasShadows\x00")),

		shadowLightMVPLoc: gl.GetUniformLocation(shadowProg, gl.Str("lightMVP\x00")),

		gpuMeshes: make(map[*scene.Mesh]*GPUMesh),
	}

	// Resolve per-element point light uniform locations
	for i := 0; i < 8; i++ {
		r.pointLightPosLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("pointLightPos[%d]\x00", i)))
		r.pointLightColorLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("pointLightColor[%d]\x00", i)))
		r.pointLightIntensityLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("pointLightIntensity[%d]\x00", i)))
		r.pointLightRangeLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("pointLightRange[%d]\x00", i)))
	}

	// Spot light locations
	r.spotLightCountLoc = gl.GetUniformLocation(prog, gl.Str("spotLightCount\x00"))
	for i := 0; i < 4; i++ {
		r.spotLightPosLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightPos[%d]\x00", i)))
		r.spotLightDirLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightDir[%d]\x00", i)))
		r.spotLightColorLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightColor[%d]\x00", i)))
		r.spotLightIntensityLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightIntensity[%d]\x00", i)))
		r.spotLightRangeLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightRange[%d]\x00", i)))
		r.spotLightInnerLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightInner[%d]\x00", i)))
		r.spotLightOuterLoc[i] = gl.GetUniformLocation(prog,
			gl.Str(fmt.Sprintf("spotLightOuter[%d]\x00", i)))
	}

	// Bind texture units: albedo=0, shadowMap=1, normalMap=2, metallicRoughness=3, emissive=4
	gl.UseProgram(prog)
	gl.Uniform1i(r.albedoTexLoc, 0)
	gl.Uniform1i(r.shadowMapLoc, 1)
	gl.Uniform1i(r.normalTexLoc, 2)
	gl.Uniform1i(r.metallicRoughnessTexLoc, 3)
	gl.Uniform1i(r.emissiveTexLoc, 4)

	// Initialise lightViewProj to identity so the shadow computation is safe
	// even when shadows are disabled
	ident := math.Mat4Identity()
	gl.UniformMatrix4fv(r.lightViewProjLoc, 1, false, (*float32)(unsafe.Pointer(&ident[0][0])))

	return r, nil
}

// ── Viewport ──────────────────────────────────────────────────────────────────

// SetViewport resizes the OpenGL viewport and stores the dimensions for
// restoring after the shadow pass.
func (r *Renderer) SetViewport(width, height int) {
	r.viewportW = int32(width)
	r.viewportH = int32(height)
	gl.Viewport(0, 0, int32(width), int32(height))
}

// ── Skybox ────────────────────────────────────────────────────────────────────

// EnableSkybox compiles the gradient sky shader and uploads the cube geometry.
// Call once after NewRenderer, before the first Render.
func (r *Renderer) EnableSkybox() error {
	if r.skybox != nil {
		r.skybox.Destroy()
	}
	sb, err := NewSkybox()
	if err != nil {
		return err
	}
	r.skybox = sb
	return nil
}

// HasSkybox reports whether a skybox has been created.
func (r *Renderer) HasSkybox() bool { return r.skybox != nil }

// SkyboxRef returns the underlying Skybox so the caller can adjust colours.
// Returns nil when no skybox is active.
func (r *Renderer) SkyboxRef() *Skybox { return r.skybox }

// DrawSkybox renders the gradient sky.  It strips the translation column from
// view so the sky appears infinitely far away, then draws before scene geometry.
// No-op when no skybox has been enabled.
func (r *Renderer) DrawSkybox(view, proj math.Mat4) {
	if r.skybox == nil {
		return
	}
	// Strip translation: column 3, rows 0-2 in [col][row] layout
	skyView := view
	skyView[3][0] = 0
	skyView[3][1] = 0
	skyView[3][2] = 0
	r.skybox.Draw(skyView.Mul(proj))
}

// ── Post-processing ───────────────────────────────────────────────────────────

// EnablePostProcess creates the HDR FBO at the current viewport size.
// Call once after NewRenderer; re-create on resize via ResizePostProcess.
func (r *Renderer) EnablePostProcess(width, height int) error {
	if r.postProcess != nil {
		r.postProcess.Destroy()
	}
	pp, err := NewPostProcessFBO(width, height)
	if err != nil {
		return err
	}
	r.postProcess = pp
	return nil
}

// HasPostProcess reports whether the HDR FBO is active.
func (r *Renderer) HasPostProcess() bool {
	return r.postProcess != nil
}

// ResizePostProcess recreates the HDR FBO (and SSAO buffers if active) at new dimensions.
func (r *Renderer) ResizePostProcess(width, height int) {
	if r.postProcess != nil {
		r.postProcess.Resize(width, height)
	}
	if r.ssao != nil {
		r.ssao.Resize(width, height)
	}
}

// EnableSSAO creates the SSAO pipeline.  EnablePostProcess must be called first.
func (r *Renderer) EnableSSAO() error {
	if r.postProcess == nil {
		return fmt.Errorf("EnableSSAO: EnablePostProcess must be called first")
	}
	if r.ssao != nil {
		r.ssao.Destroy()
	}
	s, err := NewSSAO(int(r.viewportW), int(r.viewportH))
	if err != nil {
		return fmt.Errorf("ssao: %w", err)
	}
	r.ssao = s
	return nil
}

// SetSSAORadius sets the SSAO hemisphere sampling radius (default 0.5).
func (r *Renderer) SetSSAORadius(v float32) {
	if r.ssao != nil {
		r.ssao.Radius = v
	}
}

// SetSSAOBias sets the depth bias preventing self-occlusion (default 0.025).
func (r *Renderer) SetSSAOBias(v float32) {
	if r.ssao != nil {
		r.ssao.Bias = v
	}
}

// SetSSAOStrength sets the AO blend factor: 0 = no AO, 1 = full AO (default 1.0).
func (r *Renderer) SetSSAOStrength(v float32) {
	if r.ssao != nil {
		r.ssao.Strength = v
	}
}

// SetExposure sets the tone-mapping exposure value (default 1.0).
func (r *Renderer) SetExposure(exp float32) {
	if r.postProcess != nil {
		r.postProcess.Exposure = exp
	}
}

// EnableBloom compiles the bloom shaders and creates the blur FBOs.
// Requires post-processing to be enabled first.
func (r *Renderer) EnableBloom() error {
	if r.postProcess == nil {
		return fmt.Errorf("EnableBloom: post-processing must be enabled first")
	}
	return r.postProcess.EnableBloom()
}

// SetBloomThreshold sets the luminance cut-off for the bright-pass (default 1.0).
func (r *Renderer) SetBloomThreshold(t float32) {
	if r.postProcess != nil {
		r.postProcess.BloomThreshold = t
	}
}

// SetBloomStrength sets the bloom additive multiplier (default 0.6).
func (r *Renderer) SetBloomStrength(s float32) {
	if r.postProcess != nil {
		r.postProcess.BloomStrength = s
	}
}

// BlitPostProcess runs the optional SSAO pass then resolves the HDR FBO to
// the default framebuffer with tone mapping.  A no-op when post-processing is
// disabled.
func (r *Renderer) BlitPostProcess() {
	if r.postProcess == nil {
		return
	}
	// All fullscreen passes (SSAO, bloom, tone-map) draw a single triangle.
	// gl.PolygonMode LINE would rasterize it as 3 edges and leave the screen
	// mostly empty, so temporarily force FILL for the entire post-process blit.
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}

	// Run SSAO passes (depth → AO → blur) if enabled
	var aoTex uint32
	var aoStr float32
	if r.ssao != nil {
		r.ssao.RunPasses(r.postProcess.DepthTex, r.lastProj)
		aoTex = r.ssao.BlurTex
		aoStr = r.ssao.Strength
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, r.viewportW, r.viewportH)
	r.postProcess.Blit(aoTex, aoStr)

	// Restore wireframe so the next frame's geometry draws correctly.
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}

// ── Particles ─────────────────────────────────────────────────────────────────

// DrawParticles renders emitter.Particles as camera-facing billboards.
// Must be called after BeginFrame (so the correct FBO is bound) and before
// BlitPostProcess (so particles are tone-mapped and may catch bloom).
// Lazily creates the particle renderer on first call.
func (r *Renderer) DrawParticles(emitter *scene.ParticleEmitter, view, proj math.Mat4) {
	if emitter == nil || len(emitter.Particles) == 0 {
		return
	}
	if r.particleRenderer == nil {
		pr, err := newParticleRenderer()
		if err != nil {
			fmt.Printf("particle renderer init: %v\n", err)
			return
		}
		r.particleRenderer = pr
	}
	// Particle billboards are always solid; wireframe mode would render them
	// as triangle outlines (invisible soft-circles) so force FILL temporarily.
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	r.particleRenderer.draw(emitter, view, proj)
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}

// ── Shadow map ────────────────────────────────────────────────────────────────

// EnableShadows creates the depth FBO.  Call once after NewRenderer.
func (r *Renderer) EnableShadows(size int) error {
	if r.shadowMap != nil {
		r.shadowMap.Destroy()
	}
	sm, err := NewShadowMap(size)
	if err != nil {
		return err
	}
	r.shadowMap = sm
	return nil
}

// HasShadowMap reports whether the shadow FBO has been created.
func (r *Renderer) HasShadowMap() bool {
	return r.shadowMap != nil
}

// BeginShadowPass binds the depth FBO and sets up for the shadow pass.
// The wireframe polygon mode is temporarily set to fill.
func (r *Renderer) BeginShadowPass() {
	if r.shadowMap == nil {
		return
	}
	// Shadow pass always renders filled triangles regardless of wireframe mode
	gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	gl.BindFramebuffer(gl.FRAMEBUFFER, r.shadowMap.FBO)
	gl.Viewport(0, 0, r.shadowMap.Size, r.shadowMap.Size)
	gl.Clear(gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(r.shadowProg)
}

// DrawMeshShadow draws a mesh into the depth buffer using the depth-only shader.
// Only triangle meshes cast shadows.
func (r *Renderer) DrawMeshShadow(mesh *scene.Mesh, lightMVP math.Mat4) {
	if r.shadowMap == nil || r.shadowProg == 0 {
		return
	}
	gpu := r.ensureUploaded(mesh)
	if gpu == nil {
		return
	}
	gl.UniformMatrix4fv(r.shadowLightMVPLoc, 1, false,
		(*float32)(unsafe.Pointer(&lightMVP[0][0])))
	gl.BindVertexArray(gpu.VAO)
	if gpu.HasIndices {
		gl.DrawElements(gl.TRIANGLES, gpu.IndexCount, gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(mesh.Vertices)))
	}
	gl.BindVertexArray(0)
}

// EndShadowPass restores the default framebuffer and viewport.
func (r *Renderer) EndShadowPass() {
	if r.shadowMap == nil {
		return
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.Viewport(0, 0, r.viewportW, r.viewportH)
	// Restore wireframe mode if it was active
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}

// ── BeginFrame ────────────────────────────────────────────────────────────────

// BeginFrame clears the framebuffer and sets per-frame lighting, camera, and
// shadow uniforms.  lightVP is the light view-projection matrix (used for
// shadow map lookup); hasShadows should be true when a populated shadow map
// is available.  proj is stored internally for the SSAO pass.
func (r *Renderer) BeginFrame(sky core.Color, lights []*scene.Light, ambient core.Color, camPos math.Vec3, lightVP math.Mat4, hasShadows bool, proj math.Mat4) {
	r.lastProj = proj
	// Render into the HDR FBO when post-processing is active.
	if r.postProcess != nil {
		gl.BindFramebuffer(gl.FRAMEBUFFER, r.postProcess.FBO)
		gl.Viewport(0, 0, r.postProcess.Width, r.postProcess.Height)
	} else {
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	}
	gl.ClearColor(sky.R, sky.G, sky.B, sky.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	gl.UseProgram(r.program)

	// Ambient + camera
	gl.Uniform3f(r.ambientColorLoc, ambient.R, ambient.G, ambient.B)
	gl.Uniform3f(r.cameraPosLoc, camPos.X, camPos.Y, camPos.Z)

	// IBL
	if r.iblEnabled {
		gl.Uniform1i(r.useIBLLoc, 1)
		gl.Uniform3f(r.iblZenithLoc,  r.iblZenith.R,  r.iblZenith.G,  r.iblZenith.B)
		gl.Uniform3f(r.iblHorizonLoc, r.iblHorizon.R, r.iblHorizon.G, r.iblHorizon.B)
		gl.Uniform3f(r.iblGroundLoc,  r.iblGround.R,  r.iblGround.G,  r.iblGround.B)
	} else {
		gl.Uniform1i(r.useIBLLoc, 0)
	}

	// Fog
	if r.fogEnabled {
		gl.Uniform1i(r.fogEnabledLoc, 1)
		gl.Uniform3f(r.fogColorLoc, r.fogColor.R, r.fogColor.G, r.fogColor.B)
		gl.Uniform1f(r.fogDensityLoc, r.fogDensity)
	} else {
		gl.Uniform1i(r.fogEnabledLoc, 0)
	}

	// Light-space VP matrix for shadow lookup in vertex shader
	gl.UniformMatrix4fv(r.lightViewProjLoc, 1, false,
		(*float32)(unsafe.Pointer(&lightVP[0][0])))

	// Shadow map: bind depth texture to unit 1
	if hasShadows && r.shadowMap != nil {
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, r.shadowMap.DepthTex)
		gl.Uniform1i(r.hasShadowsLoc, 1)
	} else {
		gl.Uniform1i(r.hasShadowsLoc, 0)
	}

	// Defaults for directional light
	dirLight := math.Vec3{X: 0.5, Y: -1, Z: -0.5}.Normalize()
	dirColor := core.ColorWhite
	dirIntensity := float32(0.8)

	pointIdx := 0
	for _, l := range lights {
		if l == nil {
			continue
		}
		switch l.Type {
		case scene.LightTypeDirectional:
			dirLight = l.Direction.Normalize()
			dirColor = l.Color
			dirIntensity = l.Intensity
		case scene.LightTypePoint:
			if pointIdx < 8 {
				gl.Uniform3f(r.pointLightPosLoc[pointIdx], l.Position.X, l.Position.Y, l.Position.Z)
				gl.Uniform3f(r.pointLightColorLoc[pointIdx], l.Color.R, l.Color.G, l.Color.B)
				gl.Uniform1f(r.pointLightIntensityLoc[pointIdx], l.Intensity)
				gl.Uniform1f(r.pointLightRangeLoc[pointIdx], l.Range)
				pointIdx++
			}
		}
	}

	spotIdx := 0
	for _, l := range lights {
		if l == nil || l.Type != scene.LightTypeSpot || spotIdx >= 4 {
			continue
		}
		dir := l.Direction.Normalize()
		outerCos := cosAngleDeg(l.SpotAngle)
		innerCos := cosAngleDeg(l.SpotAngle * 0.8)
		gl.Uniform3f(r.spotLightPosLoc[spotIdx], l.Position.X, l.Position.Y, l.Position.Z)
		gl.Uniform3f(r.spotLightDirLoc[spotIdx], dir.X, dir.Y, dir.Z)
		gl.Uniform3f(r.spotLightColorLoc[spotIdx], l.Color.R, l.Color.G, l.Color.B)
		gl.Uniform1f(r.spotLightIntensityLoc[spotIdx], l.Intensity)
		gl.Uniform1f(r.spotLightRangeLoc[spotIdx], l.Range)
		gl.Uniform1f(r.spotLightInnerLoc[spotIdx], innerCos)
		gl.Uniform1f(r.spotLightOuterLoc[spotIdx], outerCos)
		spotIdx++
	}

	gl.Uniform3f(r.lightDirLoc, dirLight.X, dirLight.Y, dirLight.Z)
	gl.Uniform3f(r.lightColorLoc, dirColor.R, dirColor.G, dirColor.B)
	gl.Uniform1f(r.lightIntensityLoc, dirIntensity)
	gl.Uniform1i(r.pointLightCountLoc, int32(pointIdx))
	gl.Uniform1i(r.spotLightCountLoc, int32(spotIdx))
}

// ── Wireframe ─────────────────────────────────────────────────────────────────

// SetWireframe toggles wireframe rendering mode.
func (r *Renderer) SetWireframe(enabled bool) {
	r.wireframe = enabled
	if enabled {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	} else {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
}

// IsWireframe returns whether wireframe mode is active.
func (r *Renderer) IsWireframe() bool {
	return r.wireframe
}

// ── DrawMesh ──────────────────────────────────────────────────────────────────

// DrawMesh draws a mesh with the given MVP and model matrices.
// Material properties (albedo, specular, shininess, texture) are read from mesh.Material.
func (r *Renderer) DrawMesh(mesh *scene.Mesh, mvp, model math.Mat4) {
	gpu := r.ensureUploaded(mesh)
	if gpu == nil {
		return
	}

	gl.UseProgram(r.program)
	gl.Uniform1i(r.instancedLoc, 0) // non-instanced path
	gl.UniformMatrix4fv(r.mvpLoc, 1, false, (*float32)(unsafe.Pointer(&mvp[0][0])))
	gl.UniformMatrix4fv(r.modelLoc, 1, false, (*float32)(unsafe.Pointer(&model[0][0])))

	// Material
	mat := mesh.Material
	if mat == nil {
		mat = scene.DefaultMaterial()
	}
	r.applyMaterial(mat)

	// Resolve draw primitive from mesh.DrawMode
	primitive := uint32(gl.TRIANGLES)
	switch mesh.DrawMode {
	case scene.DrawLines:
		primitive = gl.LINES
	case scene.DrawPoints:
		primitive = gl.POINTS
	}

	gl.BindVertexArray(gpu.VAO)
	if gpu.HasIndices {
		gl.DrawElements(primitive, gpu.IndexCount, gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(primitive, 0, int32(len(mesh.Vertices)))
	}
	gl.BindVertexArray(0)
}

// ── Instanced rendering ───────────────────────────────────────────────────────

// DrawMeshInstanced renders mesh len(models) times in a single GPU draw call.
// models contains one world-space transform per instance.
// MVPs are computed on the CPU (same convention as DrawMesh) and streamed to
// the GPU via a dynamic per-instance VBO bound to attrib locations 6-13.
func (r *Renderer) DrawMeshInstanced(mesh *scene.Mesh, view, proj math.Mat4, models []math.Mat4) {
	if len(models) == 0 {
		return
	}
	gpu := r.ensureUploaded(mesh)
	if gpu == nil {
		return
	}

	// Build flat instance buffer: 32 float32 per instance (MVP mat4 + Model mat4).
	// Layout (column-major to match OpenGL expectation):
	//   [0..15]  MVP   = models[i].Mul(view).Mul(proj)
	//   [16..31] Model = models[i]
	n := len(models)
	buf := make([]float32, n*32)
	for i, m := range models {
		mvp := m.Mul(view).Mul(proj)
		base := i * 32
		for col := 0; col < 4; col++ {
			for row := 0; row < 4; row++ {
				buf[base+col*4+row]    = mvp[col][row]
				buf[base+16+col*4+row] = m[col][row]
			}
		}
	}

	// Upload instance data to the per-mesh VBO (lazy create + attrib setup).
	r.uploadInstanceVBO(gpu, buf, n)

	// Material uniforms — identical to DrawMesh.
	gl.UseProgram(r.program)
	gl.Uniform1i(r.instancedLoc, 1)

	mat := mesh.Material
	if mat == nil {
		mat = scene.DefaultMaterial()
	}
	r.applyMaterial(mat)

	primitive := uint32(gl.TRIANGLES)
	switch mesh.DrawMode {
	case scene.DrawLines:
		primitive = gl.LINES
	case scene.DrawPoints:
		primitive = gl.POINTS
	}

	gl.BindVertexArray(gpu.VAO)
	if gpu.HasIndices {
		gl.DrawElementsInstanced(primitive, gpu.IndexCount, gl.UNSIGNED_INT, nil, int32(n))
	} else {
		gl.DrawArraysInstanced(primitive, 0, int32(len(mesh.Vertices)), int32(n))
	}
	gl.BindVertexArray(0)

	// Reset instanced flag so subsequent DrawMesh calls are unaffected.
	gl.Uniform1i(r.instancedLoc, 0)
}

// applyMaterial sets all material-related shader uniforms and binds textures.
// Must be called while r.program is active (UseProgram already called by DrawMesh/DrawMeshInstanced).
func (r *Renderer) applyMaterial(mat *scene.Material) {
	// Phong params (always set so the Phong path has valid values)
	gl.Uniform3f(r.matAlbedoLoc, mat.Albedo.R, mat.Albedo.G, mat.Albedo.B)
	gl.Uniform3f(r.matSpecularLoc, mat.Specular.R, mat.Specular.G, mat.Specular.B)
	gl.Uniform1f(r.matShininessLoc, mat.Shininess)

	// PBR params
	if mat.UsePBR {
		gl.Uniform1i(r.usePBRLoc, 1)
	} else {
		gl.Uniform1i(r.usePBRLoc, 0)
	}
	gl.Uniform1f(r.matMetallicLoc, mat.Metallic)
	gl.Uniform1f(r.matRoughnessLoc, mat.Roughness)
	gl.Uniform3f(r.matEmissiveLoc, mat.EmissiveColor.R, mat.EmissiveColor.G, mat.EmissiveColor.B)

	// Unlit flag
	if mat.Unlit {
		gl.Uniform1i(r.unlitLoc, 1)
	} else {
		gl.Uniform1i(r.unlitLoc, 0)
	}

	// Albedo texture (unit 0)
	if tex := mat.AlbedoTexture; tex != nil && tex.GLID != 0 {
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex.GLID)
		gl.Uniform1i(r.hasTextureLoc, 1)
	} else {
		gl.Uniform1i(r.hasTextureLoc, 0)
	}

	// Normal map (unit 2)
	if nrm := mat.NormalTexture; nrm != nil && nrm.GLID != 0 {
		gl.ActiveTexture(gl.TEXTURE2)
		gl.BindTexture(gl.TEXTURE_2D, nrm.GLID)
		gl.Uniform1i(r.hasNormalTexLoc, 1)
	} else {
		gl.Uniform1i(r.hasNormalTexLoc, 0)
	}

	// Metallic-roughness texture (unit 3)
	if mr := mat.MetallicRoughnessTexture; mr != nil && mr.GLID != 0 {
		gl.ActiveTexture(gl.TEXTURE3)
		gl.BindTexture(gl.TEXTURE_2D, mr.GLID)
		gl.Uniform1i(r.hasMetallicRoughnessTexLoc, 1)
	} else {
		gl.Uniform1i(r.hasMetallicRoughnessTexLoc, 0)
	}

	// Emissive texture (unit 4)
	if em := mat.EmissiveTexture; em != nil && em.GLID != 0 {
		gl.ActiveTexture(gl.TEXTURE4)
		gl.BindTexture(gl.TEXTURE_2D, em.GLID)
		gl.Uniform1i(r.hasEmissiveTexLoc, 1)
	} else {
		gl.Uniform1i(r.hasEmissiveTexLoc, 0)
	}
}

// uploadInstanceVBO uploads buf to the per-mesh instance VBO, creating it
// and wiring attrib locations 6-13 into the VAO on first call.
func (r *Renderer) uploadInstanceVBO(gpu *GPUMesh, buf []float32, count int) {
	const stride = int32(32 * 4) // 32 float32 * 4 bytes = 128 bytes

	if gpu.InstanceVBO == 0 {
		gl.GenBuffers(1, &gpu.InstanceVBO)
		gl.BindVertexArray(gpu.VAO)
		gl.BindBuffer(gl.ARRAY_BUFFER, gpu.InstanceVBO)

		// MVP columns at locations 6-9
		for i := uint32(0); i < 4; i++ {
			gl.EnableVertexAttribArray(6 + i)
			gl.VertexAttribPointer(6+i, 4, gl.FLOAT, false, stride, gl.PtrOffset(int(i)*16))
			gl.VertexAttribDivisor(6+i, 1)
		}
		// Model columns at locations 10-13 (offset = 16 * 4 bytes = 64 bytes past MVP)
		for i := uint32(0); i < 4; i++ {
			gl.EnableVertexAttribArray(10 + i)
			gl.VertexAttribPointer(10+i, 4, gl.FLOAT, false, stride, gl.PtrOffset(64+int(i)*16))
			gl.VertexAttribDivisor(10+i, 1)
		}
		gl.BindVertexArray(0)
	}

	byteSize := len(buf) * 4
	gl.BindBuffer(gl.ARRAY_BUFFER, gpu.InstanceVBO)
	if count > gpu.InstanceCap {
		gl.BufferData(gl.ARRAY_BUFFER, byteSize, gl.Ptr(buf), gl.DYNAMIC_DRAW)
		gpu.InstanceCap = count
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteSize, gl.Ptr(buf))
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
}

// ── Resource management ───────────────────────────────────────────────────────

// ReleaseMesh frees GPU buffers for the given mesh.
func (r *Renderer) ReleaseMesh(mesh *scene.Mesh) {
	if gpu, ok := r.gpuMeshes[mesh]; ok {
		gl.DeleteVertexArrays(1, &gpu.VAO)
		gl.DeleteBuffers(1, &gpu.VBO)
		if gpu.HasIndices {
			gl.DeleteBuffers(1, &gpu.EBO)
		}
		if gpu.InstanceVBO != 0 {
			gl.DeleteBuffers(1, &gpu.InstanceVBO)
		}
		delete(r.gpuMeshes, mesh)
		mesh.GPUData = nil
	}
}

// Destroy releases all GPU resources.
func (r *Renderer) Destroy() {
	for mesh := range r.gpuMeshes {
		r.ReleaseMesh(mesh)
	}
	if r.shadowMap != nil {
		r.shadowMap.Destroy()
	}
	if r.shadowProg != 0 {
		gl.DeleteProgram(r.shadowProg)
	}
	if r.ssao != nil {
		r.ssao.Destroy()
	}
	if r.postProcess != nil {
		r.postProcess.Destroy()
	}
	if r.skybox != nil {
		r.skybox.Destroy()
	}
	if r.particleRenderer != nil {
		r.particleRenderer.destroy()
	}
	if r.textRenderer != nil {
		r.textRenderer.destroy()
	}
	gl.DeleteProgram(r.program)
}

// SetFog configures and enables exponential depth fog.
// density: 0.01 = light haze, 0.05 = thick fog. color should match the horizon sky.
func (r *Renderer) SetFog(enabled bool, density float32, color core.Color) {
	r.fogEnabled = enabled
	r.fogDensity = density
	r.fogColor   = color
}

// EnableIBL activates sky-based image-based lighting in the PBR and Phong shaders.
func (r *Renderer) EnableIBL() {
	r.iblEnabled = true
}

// SetIBLColors updates the sky gradient colours used for ambient irradiance.
func (r *Renderer) SetIBLColors(zenith, horizon, ground core.Color) {
	r.iblZenith  = zenith
	r.iblHorizon = horizon
	r.iblGround  = ground
}

// DrawText renders a string at screen-space position (x, y) with pixel scale.
// Must be called after BlitPostProcess so text lands on the default framebuffer.
// Lazily creates the TextRenderer on first call.
func (r *Renderer) DrawText(text string, x, y, scale float32, color core.Color, screenW, screenH float32) {
	if r.textRenderer == nil {
		tr, err := newTextRenderer()
		if err != nil {
			fmt.Printf("text renderer init: %v\n", err)
			return
		}
		r.textRenderer = tr
	}
	// Text is always solid; wireframe would show triangle outlines instead of glyphs.
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
	}
	r.textRenderer.draw(text, x, y, scale, color, screenW, screenH)
	if r.wireframe {
		gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
	}
}

// ── Internal helpers ──────────────────────────────────────────────────────────

// ensureUploaded uploads vertex/index data if not already done.
func (r *Renderer) ensureUploaded(mesh *scene.Mesh) *GPUMesh {
	if gpu, ok := r.gpuMeshes[mesh]; ok {
		return gpu
	}
	if len(mesh.Vertices) == 0 {
		return nil
	}

	stride := int32(unsafe.Sizeof(core.Vertex{}))

	gpu := &GPUMesh{
		IndexCount: int32(len(mesh.Indices)),
		HasIndices: len(mesh.Indices) > 0,
	}

	gl.GenVertexArrays(1, &gpu.VAO)
	gl.GenBuffers(1, &gpu.VBO)
	gl.BindVertexArray(gpu.VAO)

	gl.BindBuffer(gl.ARRAY_BUFFER, gpu.VBO)
	gl.BufferData(gl.ARRAY_BUFFER,
		len(mesh.Vertices)*int(stride),
		gl.Ptr(mesh.Vertices),
		gl.STATIC_DRAW)

	var v core.Vertex
	posOff       := int(unsafe.Offsetof(v.Position))
	normOff      := int(unsafe.Offsetof(v.Normal))
	uvOff        := int(unsafe.Offsetof(v.UV))
	colorOff     := int(unsafe.Offsetof(v.Color))
	tangentOff   := int(unsafe.Offsetof(v.Tangent))
	bitangentOff := int(unsafe.Offsetof(v.Bitangent))

	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(posOff))

	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(normOff))

	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, stride, gl.PtrOffset(uvOff))

	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(3, 4, gl.FLOAT, false, stride, gl.PtrOffset(colorOff))

	gl.EnableVertexAttribArray(4)
	gl.VertexAttribPointer(4, 3, gl.FLOAT, false, stride, gl.PtrOffset(tangentOff))

	gl.EnableVertexAttribArray(5)
	gl.VertexAttribPointer(5, 3, gl.FLOAT, false, stride, gl.PtrOffset(bitangentOff))

	if gpu.HasIndices {
		gl.GenBuffers(1, &gpu.EBO)
		gl.BindBuffer(gl.ELEMENT_ARRAY_BUFFER, gpu.EBO)
		gl.BufferData(gl.ELEMENT_ARRAY_BUFFER,
			len(mesh.Indices)*4,
			gl.Ptr(mesh.Indices),
			gl.STATIC_DRAW)
	}

	gl.BindVertexArray(0)

	r.gpuMeshes[mesh] = gpu
	mesh.GPUData = gpu
	return gpu
}

// ── Shader helpers ────────────────────────────────────────────────────────────

func newProgram(vertSrc, fragSrc string) (uint32, error) {
	vert, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, fmt.Errorf("vertex: %w", err)
	}
	frag, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, fmt.Errorf("fragment: %w", err)
	}

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vert)
	gl.AttachShader(prog, frag)
	gl.LinkProgram(prog)

	var status int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(prog, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(prog, logLen, nil, gl.Str(log))
		return 0, fmt.Errorf("link failed: %v", log)
	}

	gl.DeleteShader(vert)
	gl.DeleteShader(frag)
	return prog, nil
}

// cosAngleDeg converts an angle in degrees to its cosine (for spot light cutoffs).
func cosAngleDeg(deg float32) float32 {
	return float32(gomath.Cos(float64(deg) * gomath.Pi / 180.0))
}

func compileShader(src string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csrc, free := gl.Strs(src)
	gl.ShaderSource(shader, 1, csrc, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		log := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(log))
		return 0, fmt.Errorf("compile failed: %v", log)
	}
	return shader, nil
}
