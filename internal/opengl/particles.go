package opengl

import (
	"fmt"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/math"
	"render-engine/scene"
)

// ── Particle shaders ─────────────────────────────────────────────────────────

// Billboard vertex shader: receives pre-built world-space quad corners from CPU.
const particleVertSrc = `
#version 410 core
layout(location = 0) in vec3 inPos;
layout(location = 1) in vec2 inUV;
layout(location = 2) in vec4 inColor;

uniform mat4 vp;

out vec2  fragUV;
out vec4  fragColor;

void main() {
    gl_Position = vp * vec4(inPos, 1.0);
    fragUV      = inUV;
    fragColor   = inColor;
}
` + "\x00"

// Procedural soft-circle fragment shader (no texture required).
// UV (0,1)² mapped so centre=0.5; alpha rolls off quadratically at the edge.
const particleFragSrc = `
#version 410 core
in vec2 fragUV;
in vec4 fragColor;

out vec4 outColor;

uniform sampler2D particleTex;
uniform bool      hasParticleTex;

void main() {
    vec4 col = fragColor;
    if (hasParticleTex) {
        col *= texture(particleTex, fragUV);
    } else {
        // Soft-circle: squared distance from centre, fade at edge
        float d = length(fragUV - vec2(0.5)) * 2.0;
        col.a  *= clamp(1.0 - d * d, 0.0, 1.0);
    }
    outColor = col;
}
` + "\x00"

// ── ParticleRenderer ─────────────────────────────────────────────────────────

// ParticleRenderer owns the GPU resources for billboard particle rendering.
// It is created lazily by Renderer.DrawParticles on first use.
type ParticleRenderer struct {
	prog          uint32
	vao           uint32
	vbo           uint32
	vpLoc         int32
	hasParticleTexLoc int32
	particleTexLoc    int32
	vboCap        int // current VBO capacity in vertices
}

// newParticleRenderer compiles the particle shader and creates the dynamic VAO/VBO.
func newParticleRenderer() (*ParticleRenderer, error) {
	prog, err := newProgram(particleVertSrc, particleFragSrc)
	if err != nil {
		return nil, fmt.Errorf("particle shader: %w", err)
	}

	var vao, vbo uint32
	gl.GenVertexArrays(1, &vao)
	gl.GenBuffers(1, &vbo)

	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)

	const stride = int32(9 * 4) // pos(3) + uv(2) + color(4) = 9 float32 × 4 bytes
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(0))  // pos
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 2, gl.FLOAT, false, stride, gl.PtrOffset(12)) // uv
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, stride, gl.PtrOffset(20)) // color
	gl.BindVertexArray(0)

	pr := &ParticleRenderer{
		prog:              prog,
		vao:               vao,
		vbo:               vbo,
		vpLoc:             gl.GetUniformLocation(prog, gl.Str("vp\x00")),
		hasParticleTexLoc: gl.GetUniformLocation(prog, gl.Str("hasParticleTex\x00")),
		particleTexLoc:    gl.GetUniformLocation(prog, gl.Str("particleTex\x00")),
	}
	gl.UseProgram(prog)
	gl.Uniform1i(pr.particleTexLoc, 0)
	gl.Uniform1i(pr.hasParticleTexLoc, 0)
	return pr, nil
}

// draw renders all live particles in the emitter as camera-facing billboards.
//
// Camera right and up are extracted from the view matrix ([col][row] layout):
//
//	right = row 0 of view = (view[0][0], view[1][0], view[2][0])
//	up    = row 1 of view = (view[0][1], view[1][1], view[2][1])
func (pr *ParticleRenderer) draw(emitter *scene.ParticleEmitter, view, proj math.Mat4) {
	n := len(emitter.Particles)
	if n == 0 {
		return
	}

	// Camera axes from view matrix rows
	camRight := math.Vec3{X: view[0][0], Y: view[1][0], Z: view[2][0]}
	camUp    := math.Vec3{X: view[0][1], Y: view[1][1], Z: view[2][1]}

	// Build CPU-side quad buffer: 6 vertices (2 triangles) per particle.
	const vertsPerParticle = 6
	const floatsPerVert    = 9
	buf := make([]float32, n*vertsPerParticle*floatsPerVert)
	out := 0

	addVert := func(p math.Vec3, u, v float32, c [4]float32) {
		buf[out+0] = p.X; buf[out+1] = p.Y; buf[out+2] = p.Z
		buf[out+3] = u;   buf[out+4] = v
		buf[out+5] = c[0]; buf[out+6] = c[1]; buf[out+7] = c[2]; buf[out+8] = c[3]
		out += floatsPerVert
	}

	for i := range emitter.Particles {
		p  := &emitter.Particles[i]
		s  := p.Size
		c  := [4]float32{p.Color.R, p.Color.G, p.Color.B, p.Color.A}
		r  := camRight.Mul(s)
		u  := camUp.Mul(s)

		// Four corners of the billboard quad
		bl := p.Position.Sub(r).Sub(u)
		br := p.Position.Add(r).Sub(u)
		tl := p.Position.Sub(r).Add(u)
		tr := p.Position.Add(r).Add(u)

		// Triangle 1: tl, tr, br
		addVert(tl, 0, 1, c)
		addVert(tr, 1, 1, c)
		addVert(br, 1, 0, c)
		// Triangle 2: tl, br, bl
		addVert(tl, 0, 1, c)
		addVert(br, 1, 0, c)
		addVert(bl, 0, 0, c)
	}

	// Upload to GPU (grow VBO only when needed)
	gl.BindBuffer(gl.ARRAY_BUFFER, pr.vbo)
	byteSize := len(buf) * 4
	vertCount := n * vertsPerParticle
	if vertCount > pr.vboCap {
		gl.BufferData(gl.ARRAY_BUFFER, byteSize, gl.Ptr(buf), gl.DYNAMIC_DRAW)
		pr.vboCap = vertCount
	} else {
		gl.BufferSubData(gl.ARRAY_BUFFER, 0, byteSize, gl.Ptr(buf))
	}
	gl.BindBuffer(gl.ARRAY_BUFFER, 0)

	// Blending: additive (fire/glow) or standard alpha (smoke)
	gl.Enable(gl.BLEND)
	switch emitter.BlendMode {
	case scene.BlendAdditive:
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE)
	default:
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}

	// Depth: read (test against scene) but do NOT write (particles don't occlude)
	gl.DepthMask(false)

	vp := view.Mul(proj)
	gl.UseProgram(pr.prog)
	gl.UniformMatrix4fv(pr.vpLoc, 1, false, (*float32)(unsafe.Pointer(&vp[0][0])))
	gl.Uniform1i(pr.hasParticleTexLoc, 0) // procedural soft-circle

	gl.BindVertexArray(pr.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(vertCount))
	gl.BindVertexArray(0)

	// Restore render state
	gl.DepthMask(true)
	gl.Disable(gl.BLEND)
}

func (pr *ParticleRenderer) destroy() {
	gl.DeleteVertexArrays(1, &pr.vao)
	gl.DeleteBuffers(1, &pr.vbo)
	gl.DeleteProgram(pr.prog)
}
