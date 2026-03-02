package opengl

import (
	"fmt"

	gl "github.com/go-gl/gl/v4.1-core/gl"
)

// PostProcessFBO is an HDR off-screen render target with tone mapping and
// optional bloom (bright-pass → separable Gaussian blur → additive composite).
type PostProcessFBO struct {
	// Main HDR FBO (scene renders into this)
	FBO      uint32 // framebuffer object
	ColorTex uint32 // RGBA16F colour attachment
	DepthTex uint32 // DEPTH_COMPONENT32F depth texture (sampleable for SSAO)
	Width    int32
	Height   int32

	// Tone-map + bloom composite shader
	prog        uint32
	hdrLoc      int32 // sampler2D unit 0
	bloomTexLoc int32 // sampler2D unit 1
	expLoc      int32
	bloomStrLoc int32
	hasBloomLoc int32
	// AO composite (unit 2)
	aoTexLoc    int32
	hasAOLoc    int32
	aoStrLoc    int32

	quadVAO uint32 // empty VAO for the fullscreen triangle

	// Tone-mapping
	Exposure float32

	// Bloom ping-pong FBOs (created by EnableBloom)
	bloomFBO        [2]uint32
	bloomTex        [2]uint32
	bloomW          int32
	bloomH          int32
	brightProg      uint32 // bright-pass shader
	brightThreshLoc int32
	blurProg        uint32 // separable Gaussian shader
	blurTexLoc      int32
	blurDirLoc      int32

	BloomEnabled   bool
	BloomThreshold float32 // luminance cut-off (1.0 = only pixels brighter than white)
	BloomStrength  float32 // additive bloom multiplier
	BloomPasses    int     // number of H+V blur pairs (more = softer, more expensive)
}

// ── Shaders ───────────────────────────────────────────────────────────────────

// ppVertSrc — fullscreen triangle via gl_VertexID (no VBO needed).
const ppVertSrc = `
#version 410 core
out vec2 fragUV;
void main() {
    const vec2 pos[3] = vec2[3](
        vec2(-1.0, -1.0),
        vec2( 3.0, -1.0),
        vec2(-1.0,  3.0)
    );
    gl_Position = vec4(pos[gl_VertexID], 0.0, 1.0);
    fragUV      = pos[gl_VertexID] * 0.5 + 0.5;
}
` + "\x00"

// ppFragSrc — exposure, Reinhard tone mapping, gamma 2.2, optional bloom add, optional SSAO.
const ppFragSrc = `
#version 410 core
in  vec2 fragUV;
out vec4 outColor;

uniform sampler2D hdrBuffer;  // unit 0
uniform sampler2D bloomTex;   // unit 1
uniform sampler2D aoTex;      // unit 2 (SSAO)
uniform float     exposure;
uniform float     bloomStrength;
uniform bool      hasBloom;
uniform bool      hasAO;
uniform float     aoStrength;

void main() {
    vec3 hdr = texture(hdrBuffer, fragUV).rgb;

    if (hasBloom) {
        hdr += texture(bloomTex, fragUV).rgb * bloomStrength;
    }

    // Apply SSAO occlusion (modulates HDR before tone-mapping so it stays in linear space)
    if (hasAO) {
        float ao = texture(aoTex, fragUV).r;
        hdr *= mix(1.0, ao, aoStrength);
    }

    // Exposure → Reinhard → gamma 2.2
    vec3 mapped = vec3(1.0) - exp(-hdr * exposure);
    mapped = pow(mapped, vec3(1.0 / 2.2));

    outColor = vec4(mapped, 1.0);
}
` + "\x00"

// ppBrightFragSrc — extracts pixels whose luminance exceeds the threshold.
const ppBrightFragSrc = `
#version 410 core
in  vec2 fragUV;
out vec4 outColor;

uniform sampler2D hdrBuffer;
uniform float     threshold;

void main() {
    vec3  color = texture(hdrBuffer, fragUV).rgb;
    float luma  = dot(color, vec3(0.2126, 0.7152, 0.0722));
    outColor = vec4(color * step(threshold, luma), 1.0);
}
` + "\x00"

// ppBlurFragSrc — single-axis 5-tap Gaussian blur.
// texelDir = (1/w, 0) for horizontal, (0, 1/h) for vertical.
const ppBlurFragSrc = `
#version 410 core
in  vec2 fragUV;
out vec4 outColor;

uniform sampler2D blurTex;
uniform vec2      texelDir;

void main() {
    const float w[5] = float[](0.0625, 0.25, 0.375, 0.25, 0.0625);
    vec3 result = vec3(0.0);
    for (int i = -2; i <= 2; i++) {
        result += texture(blurTex, fragUV + float(i) * texelDir).rgb * w[i + 2];
    }
    outColor = vec4(result, 1.0);
}
` + "\x00"

// ── Constructor ───────────────────────────────────────────────────────────────

func NewPostProcessFBO(width, height int) (*PostProcessFBO, error) {
	pp := &PostProcessFBO{Exposure: 1.0}

	prog, err := newProgram(ppVertSrc, ppFragSrc)
	if err != nil {
		return nil, fmt.Errorf("post-process shader: %w", err)
	}
	pp.prog        = prog
	pp.hdrLoc      = gl.GetUniformLocation(prog, gl.Str("hdrBuffer\x00"))
	pp.bloomTexLoc = gl.GetUniformLocation(prog, gl.Str("bloomTex\x00"))
	pp.expLoc      = gl.GetUniformLocation(prog, gl.Str("exposure\x00"))
	pp.bloomStrLoc = gl.GetUniformLocation(prog, gl.Str("bloomStrength\x00"))
	pp.hasBloomLoc = gl.GetUniformLocation(prog, gl.Str("hasBloom\x00"))
	pp.aoTexLoc    = gl.GetUniformLocation(prog, gl.Str("aoTex\x00"))
	pp.hasAOLoc    = gl.GetUniformLocation(prog, gl.Str("hasAO\x00"))
	pp.aoStrLoc    = gl.GetUniformLocation(prog, gl.Str("aoStrength\x00"))

	gl.UseProgram(prog)
	gl.Uniform1i(pp.hdrLoc, 0)
	gl.Uniform1i(pp.bloomTexLoc, 1)
	gl.Uniform1i(pp.aoTexLoc, 2)

	gl.GenVertexArrays(1, &pp.quadVAO)

	pp.allocFBO(width, height)
	return pp, nil
}

// ── Bloom ─────────────────────────────────────────────────────────────────────

// EnableBloom compiles the bright-pass and blur shaders, and creates the
// half-resolution ping-pong FBOs used for the bloom effect.
func (pp *PostProcessFBO) EnableBloom() error {
	if pp.brightProg != 0 {
		return nil // already enabled
	}

	// Bright-pass shader
	bp, err := newProgram(ppVertSrc, ppBrightFragSrc)
	if err != nil {
		return fmt.Errorf("bright-pass shader: %w", err)
	}
	pp.brightProg      = bp
	pp.brightThreshLoc = gl.GetUniformLocation(bp, gl.Str("threshold\x00"))
	gl.UseProgram(bp)
	gl.Uniform1i(gl.GetUniformLocation(bp, gl.Str("hdrBuffer\x00")), 0)

	// Blur shader
	blp, err := newProgram(ppVertSrc, ppBlurFragSrc)
	if err != nil {
		gl.DeleteProgram(bp)
		pp.brightProg = 0
		return fmt.Errorf("blur shader: %w", err)
	}
	pp.blurProg    = blp
	pp.blurTexLoc  = gl.GetUniformLocation(blp, gl.Str("blurTex\x00"))
	pp.blurDirLoc  = gl.GetUniformLocation(blp, gl.Str("texelDir\x00"))
	gl.UseProgram(blp)
	gl.Uniform1i(pp.blurTexLoc, 0)

	// Half-resolution bloom FBOs
	pp.bloomW = pp.Width / 2
	if pp.bloomW < 1 {
		pp.bloomW = 1
	}
	pp.bloomH = pp.Height / 2
	if pp.bloomH < 1 {
		pp.bloomH = 1
	}
	pp.allocBloomFBOs()

	pp.BloomEnabled   = true
	pp.BloomThreshold = 1.0 // only HDR-bright pixels
	pp.BloomStrength  = 0.6
	pp.BloomPasses    = 4   // 4 H+V pairs = decent soft glow

	return nil
}

// allocBloomFBOs creates the two ping-pong colour-only FBOs for bloom.
func (pp *PostProcessFBO) allocBloomFBOs() {
	for i := 0; i < 2; i++ {
		gl.GenTextures(1, &pp.bloomTex[i])
		gl.BindTexture(gl.TEXTURE_2D, pp.bloomTex[i])
		gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F,
			pp.bloomW, pp.bloomH, 0, gl.RGBA, gl.HALF_FLOAT, nil)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
		gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		gl.GenFramebuffers(1, &pp.bloomFBO[i])
		gl.BindFramebuffer(gl.FRAMEBUFFER, pp.bloomFBO[i])
		gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0,
			gl.TEXTURE_2D, pp.bloomTex[i], 0)
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	}
}

// freeBloomFBOs deletes the bloom ping-pong textures and FBOs.
func (pp *PostProcessFBO) freeBloomFBOs() {
	for i := 0; i < 2; i++ {
		if pp.bloomFBO[i] != 0 {
			gl.DeleteFramebuffers(1, &pp.bloomFBO[i])
			pp.bloomFBO[i] = 0
		}
		if pp.bloomTex[i] != 0 {
			gl.DeleteTextures(1, &pp.bloomTex[i])
			pp.bloomTex[i] = 0
		}
	}
}

// ── Main FBO lifecycle ────────────────────────────────────────────────────────

func (pp *PostProcessFBO) allocFBO(width, height int) {
	pp.Width  = int32(width)
	pp.Height = int32(height)

	gl.GenTextures(1, &pp.ColorTex)
	gl.BindTexture(gl.TEXTURE_2D, pp.ColorTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA16F,
		int32(width), int32(height), 0, gl.RGBA, gl.HALF_FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	// Depth as a sampleable texture (required by SSAO pass)
	gl.GenTextures(1, &pp.DepthTex)
	gl.BindTexture(gl.TEXTURE_2D, pp.DepthTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT32F,
		int32(width), int32(height), 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	gl.GenFramebuffers(1, &pp.FBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, pp.FBO)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0,
		gl.TEXTURE_2D, pp.ColorTex, 0)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT,
		gl.TEXTURE_2D, pp.DepthTex, 0)
	if s := gl.CheckFramebufferStatus(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
		fmt.Printf("WARNING: HDR FBO incomplete (0x%X)\n", s)
	}
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
}

func (pp *PostProcessFBO) freeFBO() {
	if pp.FBO != 0 {
		gl.DeleteFramebuffers(1, &pp.FBO)
		pp.FBO = 0
	}
	if pp.ColorTex != 0 {
		gl.DeleteTextures(1, &pp.ColorTex)
		pp.ColorTex = 0
	}
	if pp.DepthTex != 0 {
		gl.DeleteTextures(1, &pp.DepthTex)
		pp.DepthTex = 0
	}
}

// Resize recreates the main HDR FBO and (if bloom is active) the bloom FBOs
// at the new pixel dimensions.
func (pp *PostProcessFBO) Resize(width, height int) {
	pp.freeFBO()
	pp.allocFBO(width, height)

	if pp.BloomEnabled {
		pp.freeBloomFBOs()
		pp.bloomW = int32(width) / 2
		if pp.bloomW < 1 {
			pp.bloomW = 1
		}
		pp.bloomH = int32(height) / 2
		if pp.bloomH < 1 {
			pp.bloomH = 1
		}
		pp.allocBloomFBOs()
	}
}

// Destroy frees all GPU resources owned by this object.
func (pp *PostProcessFBO) Destroy() {
	pp.freeFBO()
	pp.freeBloomFBOs()
	if pp.brightProg != 0 {
		gl.DeleteProgram(pp.brightProg)
		pp.brightProg = 0
	}
	if pp.blurProg != 0 {
		gl.DeleteProgram(pp.blurProg)
		pp.blurProg = 0
	}
	if pp.prog != 0 {
		gl.DeleteProgram(pp.prog)
		pp.prog = 0
	}
	if pp.quadVAO != 0 {
		gl.DeleteVertexArrays(1, &pp.quadVAO)
		pp.quadVAO = 0
	}
}

// ── Blit ──────────────────────────────────────────────────────────────────────

// Blit resolves the HDR FBO to the currently bound framebuffer (FBO 0).
// When bloom is enabled it runs: bright-pass → ping-pong blur → composite.
// aoTex = SSAO blur texture (0 = disabled), aoStrength = blend factor [0,1].
func (pp *PostProcessFBO) Blit(aoTex uint32, aoStrength float32) {
	gl.Disable(gl.DEPTH_TEST)
	gl.BindVertexArray(pp.quadVAO)

	if pp.BloomEnabled && pp.brightProg != 0 {
		// ── Step 1: bright-pass → bloomFBO[0] ─────────────────────────────
		gl.BindFramebuffer(gl.FRAMEBUFFER, pp.bloomFBO[0])
		gl.Viewport(0, 0, pp.bloomW, pp.bloomH)
		gl.UseProgram(pp.brightProg)
		gl.Uniform1f(pp.brightThreshLoc, pp.BloomThreshold)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, pp.ColorTex)
		gl.DrawArrays(gl.TRIANGLES, 0, 3)

		// ── Step 2: ping-pong Gaussian blur ───────────────────────────────
		// Trace: bright-pass is in bloomTex[0].
		// Each pair does H (src→dst) then V (dst→src), so after BloomPasses
		// pairs the result always ends up back in bloomTex[0].
		src, dst := 0, 1
		gl.UseProgram(pp.blurProg)
		for i := 0; i < pp.BloomPasses*2; i++ {
			gl.BindFramebuffer(gl.FRAMEBUFFER, pp.bloomFBO[dst])
			if i%2 == 0 { // horizontal
				gl.Uniform2f(pp.blurDirLoc, 1.0/float32(pp.bloomW), 0)
			} else { // vertical
				gl.Uniform2f(pp.blurDirLoc, 0, 1.0/float32(pp.bloomH))
			}
			gl.BindTexture(gl.TEXTURE_2D, pp.bloomTex[src])
			gl.DrawArrays(gl.TRIANGLES, 0, 3)
			src, dst = dst, src
		}
		// After an even number of total iterations the result is in bloomTex[0].
		// (each pair restores src=0; BloomPasses pairs = BloomPasses*2 iters)

		// ── Step 3: composite → default FBO ───────────────────────────────
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, pp.Width, pp.Height)
		gl.UseProgram(pp.prog)
		gl.Uniform1f(pp.expLoc, pp.Exposure)
		gl.Uniform1f(pp.bloomStrLoc, pp.BloomStrength)
		gl.Uniform1i(pp.hasBloomLoc, 1)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, pp.ColorTex)
		gl.ActiveTexture(gl.TEXTURE1)
		gl.BindTexture(gl.TEXTURE_2D, pp.bloomTex[0])
		if aoTex != 0 {
			gl.ActiveTexture(gl.TEXTURE2)
			gl.BindTexture(gl.TEXTURE_2D, aoTex)
			gl.Uniform1i(pp.hasAOLoc, 1)
			gl.Uniform1f(pp.aoStrLoc, aoStrength)
		} else {
			gl.Uniform1i(pp.hasAOLoc, 0)
		}
		gl.DrawArrays(gl.TRIANGLES, 0, 3)

	} else {
		// ── No bloom: just tone-map ────────────────────────────────────────
		gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
		gl.Viewport(0, 0, pp.Width, pp.Height)
		gl.UseProgram(pp.prog)
		gl.Uniform1f(pp.expLoc, pp.Exposure)
		gl.Uniform1i(pp.hasBloomLoc, 0)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, pp.ColorTex)
		if aoTex != 0 {
			gl.ActiveTexture(gl.TEXTURE2)
			gl.BindTexture(gl.TEXTURE_2D, aoTex)
			gl.Uniform1i(pp.hasAOLoc, 1)
			gl.Uniform1f(pp.aoStrLoc, aoStrength)
		} else {
			gl.Uniform1i(pp.hasAOLoc, 0)
		}
		gl.DrawArrays(gl.TRIANGLES, 0, 3)
	}

	gl.BindVertexArray(0)
	gl.Enable(gl.DEPTH_TEST)
}
