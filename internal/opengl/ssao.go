package opengl

import (
	"fmt"
	"math/rand"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/math"
)

// SSAO implements screen-space ambient occlusion.
// It reads the scene depth texture (from PostProcessFBO.DepthTex), reconstructs
// view-space positions, and outputs a per-pixel occlusion factor blurred into
// BlurTex which the composite stage can multiply into the HDR image.
type SSAO struct {
	// aoFBO/aoTex — raw per-pixel occlusion (RGBA16F, full-res)
	aoFBO uint32
	aoTex uint32

	// blurFBO/BlurTex — 4-tap box-blurred occlusion (exported for composite)
	blurFBO uint32
	BlurTex uint32

	width, height int32

	// SSAO pass shader
	ssaoProg      uint32
	depthLocS     int32 // depthTex  unit 0
	noiseLocS     int32 // noiseTex  unit 1
	kernelLoc     int32 // kernel[0] base
	projLocS      int32
	invProjLocS   int32
	radiusLoc     int32
	biasLoc       int32
	noiseScaleLoc int32

	// Blur pass shader
	blurProg   uint32
	blurSrcLoc int32

	// 4×4 rotation noise texture
	noiseTex uint32

	// Fullscreen triangle VAO (no VBO needed)
	quadVAO uint32

	// Configuration (tweakable at runtime)
	Radius   float32 // hemisphere radius in view-space units (default 0.5)
	Bias     float32 // depth bias to prevent self-occlusion acne (default 0.025)
	Strength float32 // blend factor: 0 = no AO, 1 = full AO (default 1.0)
}

// ── Shaders ───────────────────────────────────────────────────────────────────

// ssaoFragSrc reconstructs view-space position from the depth buffer and
// accumulates hemisphere occlusion using a precomputed random kernel.
const ssaoFragSrc = `
#version 410 core
in  vec2 fragUV;
out vec4 outAO;

uniform sampler2D depthTex;   // unit 0 — scene depth [0,1]
uniform sampler2D noiseTex;   // unit 1 — 4×4 XY rotation noise
uniform vec3  kernel[64];
uniform mat4  proj;
uniform mat4  invProj;
uniform float radius;
uniform float bias;
uniform vec2  noiseScale;     // vec2(screenW/4, screenH/4) for tiling

// Reconstruct view-space position from a UV + depth sample.
vec3 viewPos(vec2 uv) {
    float d  = texture(depthTex, uv).r * 2.0 - 1.0; // [0,1] → NDC [-1,1]
    vec4 ndc = vec4(uv * 2.0 - 1.0, d, 1.0);
    vec4 vp  = invProj * ndc;
    return vp.xyz / vp.w;
}

void main() {
    // Skip background (depth at or beyond far plane)
    if (texture(depthTex, fragUV).r >= 0.9999) { outAO = vec4(1.0); return; }

    vec3 pos = viewPos(fragUV);

    // Build surface normal from depth derivatives (view-space)
    vec3 N = normalize(cross(dFdx(pos), dFdy(pos)));
    // Ensure N faces toward the camera (origin in view space)
    if (dot(N, -pos) < 0.0) N = -N;

    // Random tangent from the tiled noise texture (XY in [-1,1], Z=0)
    vec3 rnd = texture(noiseTex, fragUV * noiseScale).xyz;
    rnd.z = 0.0;

    // Gram-Schmidt TBN to rotate the kernel to the surface hemisphere
    vec3 T   = normalize(rnd - N * dot(rnd, N));
    vec3 B   = cross(N, T);
    mat3 TBN = mat3(T, B, N);

    float occ = 0.0;
    for (int i = 0; i < 64; i++) {
        // Rotate kernel sample into view space and offset from fragment position
        vec3 s = pos + TBN * kernel[i] * radius;

        // Project sample into NDC then screen UV
        vec4 off = proj * vec4(s, 1.0);
        off.xyz /= off.w;
        vec2 suv = clamp(off.xy * 0.5 + 0.5, 0.001, 0.999);

        // Get the actual geometry depth at the sample's screen position
        float geoZ = viewPos(suv).z;

        // Range check prevents occlusion from distant geometry
        float rng = smoothstep(0.0, 1.0, radius / max(abs(pos.z - geoZ), 0.0001));

        // Occluded when geometry is closer to camera than the sample point
        // (in view space: larger z = closer, so geoZ >= sampleZ means occluded)
        occ += (geoZ >= s.z + bias ? 1.0 : 0.0) * rng;
    }

    outAO = vec4(1.0 - occ / 64.0, 0.0, 0.0, 1.0);
}
` + "\x00"

// ssaoBlurFragSrc applies a 5×5 box blur to reduce SSAO noise.
const ssaoBlurFragSrc = `
#version 410 core
in  vec2 fragUV;
out vec4 outAO;

uniform sampler2D ssaoTex;

void main() {
    vec2 texel  = 1.0 / vec2(textureSize(ssaoTex, 0));
    float result = 0.0;
    for (int x = -2; x <= 2; x++) {
        for (int y = -2; y <= 2; y++) {
            result += texture(ssaoTex, fragUV + vec2(x, y) * texel).r;
        }
    }
    outAO = vec4(result / 25.0, 0.0, 0.0, 1.0);
}
` + "\x00"

// ── Constructor ───────────────────────────────────────────────────────────────

// NewSSAO creates the SSAO shaders, kernel, noise texture, and output FBOs.
func NewSSAO(width, height int) (*SSAO, error) {
	s := &SSAO{
		width:    int32(width),
		height:   int32(height),
		Radius:   0.5,
		Bias:     0.025,
		Strength: 1.0,
	}

	// Compile SSAO pass shader (reuses ppVertSrc from postprocess.go)
	ssaoProg, err := newProgram(ppVertSrc, ssaoFragSrc)
	if err != nil {
		return nil, fmt.Errorf("ssao shader: %w", err)
	}
	s.ssaoProg     = ssaoProg
	s.depthLocS    = gl.GetUniformLocation(ssaoProg, gl.Str("depthTex\x00"))
	s.noiseLocS    = gl.GetUniformLocation(ssaoProg, gl.Str("noiseTex\x00"))
	s.kernelLoc    = gl.GetUniformLocation(ssaoProg, gl.Str("kernel\x00"))
	s.projLocS     = gl.GetUniformLocation(ssaoProg, gl.Str("proj\x00"))
	s.invProjLocS  = gl.GetUniformLocation(ssaoProg, gl.Str("invProj\x00"))
	s.radiusLoc    = gl.GetUniformLocation(ssaoProg, gl.Str("radius\x00"))
	s.biasLoc      = gl.GetUniformLocation(ssaoProg, gl.Str("bias\x00"))
	s.noiseScaleLoc = gl.GetUniformLocation(ssaoProg, gl.Str("noiseScale\x00"))

	gl.UseProgram(ssaoProg)
	gl.Uniform1i(s.depthLocS, 0)
	gl.Uniform1i(s.noiseLocS, 1)

	// Compile blur pass shader
	blurProg, err := newProgram(ppVertSrc, ssaoBlurFragSrc)
	if err != nil {
		gl.DeleteProgram(ssaoProg)
		return nil, fmt.Errorf("ssao blur shader: %w", err)
	}
	s.blurProg   = blurProg
	s.blurSrcLoc = gl.GetUniformLocation(blurProg, gl.Str("ssaoTex\x00"))

	gl.UseProgram(blurProg)
	gl.Uniform1i(s.blurSrcLoc, 0)

	// Fullscreen-triangle VAO (no vertex data, uses gl_VertexID)
	gl.GenVertexArrays(1, &s.quadVAO)

	s.generateKernel()
	s.generateNoise()
	s.allocFBOs(width, height)

	return s, nil
}

// ── Kernel & noise ────────────────────────────────────────────────────────────

// generateKernel creates 64 hemisphere sample points distributed with
// importance sampling (more samples near the origin for better contact shadows).
func (s *SSAO) generateKernel() {
	rng := rand.New(rand.NewSource(42)) // deterministic seed for reproducibility

	kernel := make([]float32, 64*3)
	for i := 0; i < 64; i++ {
		v := math.Vec3{
			X: rng.Float32()*2 - 1,
			Y: rng.Float32()*2 - 1,
			Z: rng.Float32(), // only positive Z → hemisphere facing +Z (surface normal)
		}.Normalize()

		// Accelerating lerp: cluster more samples close to the origin
		t := float32(i) / 64.0
		scale := 0.1 + 0.9*t*t // lerp(0.1, 1.0, t²)
		v = v.Mul(scale)

		kernel[i*3+0] = v.X
		kernel[i*3+1] = v.Y
		kernel[i*3+2] = v.Z
	}

	gl.UseProgram(s.ssaoProg)
	gl.Uniform3fv(s.kernelLoc, 64, &kernel[0])
}

// generateNoise creates a 4×4 texture of random XY tangent-space rotation
// vectors (Z=0).  The texture tiles over the screen to rotate the kernel
// per-fragment without a per-fragment random number.
func (s *SSAO) generateNoise() {
	rng := rand.New(rand.NewSource(123))

	noise := make([]float32, 4*4*3) // RGB32F
	for i := 0; i < 16; i++ {
		noise[i*3+0] = rng.Float32()*2 - 1 // X in [-1,1]
		noise[i*3+1] = rng.Float32()*2 - 1 // Y in [-1,1]
		noise[i*3+2] = 0                   // Z = 0 (rotate only in XY plane)
	}

	gl.GenTextures(1, &s.noiseTex)
	gl.BindTexture(gl.TEXTURE_2D, s.noiseTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGB32F, 4, 4, 0, gl.RGB, gl.FLOAT, gl.Ptr(noise))
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.BindTexture(gl.TEXTURE_2D, 0)
}

// ── FBO management ────────────────────────────────────────────────────────────

func (s *SSAO) allocFBOs(width, height int) {
	s.width  = int32(width)
	s.height = int32(height)

	for _, pair := range []struct {
		fbo *uint32
		tex *uint32
		tag string
	}{
		{&s.aoFBO, &s.aoTex, "SSAO"},
		{&s.blurFBO, &s.BlurTex, "SSAO-blur"},
	} {
		gl.GenTextures(1, pair.tex)
		gl.BindTexture(gl.TEXTURE_2D, *pair.tex)
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F,
			int32(width), int32(height), 0, gl.RGBA, gl.FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		gl.GenFramebuffers(1, pair.fbo)
		gl.BindFramebuffer(gl.FRAMEBUFFER, *pair.fbo)
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0,
			gl.TEXTURE_2D, *pair.tex, 0)
		if st := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); st != gl.FRAMEBUFFER_COMPLETE {
			fmt.Printf("WARNING: %s FBO incomplete (0x%X)\n", pair.tag, st)
		}
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	}
}

func (s *SSAO) freeFBOs() {
	if s.aoFBO != 0 {
		gl.DeleteFramebuffers(1, &s.aoFBO)
		s.aoFBO = 0
	}
	if s.aoTex != 0 {
		gl.DeleteTextures(1, &s.aoTex)
		s.aoTex = 0
	}
	if s.blurFBO != 0 {
		gl.DeleteFramebuffers(1, &s.blurFBO)
		s.blurFBO = 0
	}
	if s.BlurTex != 0 {
		gl.DeleteTextures(1, &s.BlurTex)
		s.BlurTex = 0
	}
}

// Resize recreates the AO and blur FBOs at the new pixel dimensions.
func (s *SSAO) Resize(width, height int) {
	s.freeFBOs()
	s.allocFBOs(width, height)
}

// Destroy frees all GPU resources.
func (s *SSAO) Destroy() {
	s.freeFBOs()
	if s.noiseTex != 0 {
		gl.DeleteTextures(1, &s.noiseTex)
		s.noiseTex = 0
	}
	if s.ssaoProg != 0 {
		gl.DeleteProgram(s.ssaoProg)
		s.ssaoProg = 0
	}
	if s.blurProg != 0 {
		gl.DeleteProgram(s.blurProg)
		s.blurProg = 0
	}
	if s.quadVAO != 0 {
		gl.DeleteVertexArrays(1, &s.quadVAO)
		s.quadVAO = 0
	}
}

// ── Render passes ─────────────────────────────────────────────────────────────

// RunPasses executes the SSAO and blur passes.
// depthTex must be the scene depth texture (PostProcessFBO.DepthTex).
// proj must be the camera projection matrix (used to project kernel samples).
// On return, BlurTex contains the blurred AO factor ready for compositing.
func (s *SSAO) RunPasses(depthTex uint32, proj math.Mat4) {
	invProj := proj.Inverse()

	gl.Disable(gl.DEPTH_TEST)
	gl.BindVertexArray(s.quadVAO)

	// ── Pass 1: SSAO ──────────────────────────────────────────────────────────
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.aoFBO)
	gl.Viewport(0, 0, s.width, s.height)
	gl.UseProgram(s.ssaoProg)

	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, depthTex)
	gl.ActiveTexture(gl.TEXTURE1)
	gl.BindTexture(gl.TEXTURE_2D, s.noiseTex)

	gl.UniformMatrix4fv(s.projLocS, 1, false,
		(*float32)(unsafe.Pointer(&proj[0][0])))
	gl.UniformMatrix4fv(s.invProjLocS, 1, false,
		(*float32)(unsafe.Pointer(&invProj[0][0])))
	gl.Uniform1f(s.radiusLoc, s.Radius)
	gl.Uniform1f(s.biasLoc, s.Bias)
	gl.Uniform2f(s.noiseScaleLoc,
		float32(s.width)/4.0,
		float32(s.height)/4.0)

	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	// ── Pass 2: Blur ──────────────────────────────────────────────────────────
	gl.BindFramebuffer(gl.FRAMEBUFFER, s.blurFBO)
	gl.UseProgram(s.blurProg)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, s.aoTex)
	gl.DrawArrays(gl.TRIANGLES, 0, 3)

	gl.BindVertexArray(0)
	gl.Enable(gl.DEPTH_TEST)
}
