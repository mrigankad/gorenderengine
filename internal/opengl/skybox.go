package opengl

import (
	"fmt"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/core"
	"render-engine/math"
)

// Skybox renders a procedural gradient sky using an inverted unit cube.
// The cube vertex shader uses the xyww trick (gl_Position.z = gl_Position.w)
// so every fragment lands at NDC depth 1.0 — always behind scene geometry.
type Skybox struct {
	vao  uint32
	vbo  uint32
	prog uint32

	vpLoc      int32
	zenithLoc  int32
	horizonLoc int32
	groundLoc  int32

	// ZenithColor is the sky colour directly overhead (Y = +1).
	ZenithColor core.Color
	// HorizonColor is the sky colour at the horizon (Y ≈ 0).
	HorizonColor core.Color
	// GroundColor is the colour below the horizon (Y = -1).
	GroundColor core.Color
}

// ── Shaders ───────────────────────────────────────────────────────────────────

// skyVertSrc — transforms cube vertices with a view matrix that has its
// translation stripped, then forces depth = 1.0 via the xyww trick.
const skyVertSrc = `
#version 410 core
layout(location = 0) in vec3 inPosition;

uniform mat4 skyVP;

out vec3 fragDir;

void main() {
    fragDir = inPosition;
    vec4 pos = skyVP * vec4(inPosition, 1.0);
    // xyww → after perspective divide: z/w = w/w = 1.0 (far plane)
    gl_Position = pos.xyww;
}
` + "\x00"

// skyFragSrc — gradient based on the fragment's vertical direction.
// Above the horizon: lerp horizon→zenith.  Below: lerp horizon→ground.
const skyFragSrc = `
#version 410 core
in vec3 fragDir;
out vec4 outColor;

uniform vec3 zenith;
uniform vec3 horizon;
uniform vec3 ground;

void main() {
    float t = normalize(fragDir).y;     // -1 (down) to +1 (up)

    vec3 color;
    if (t >= 0.0) {
        // Subtle power curve makes the zenith transition feel natural
        color = mix(horizon, zenith, pow(t, 0.4));
    } else {
        // Ground fades in quickly below the horizon
        color = mix(horizon, ground, min(-t * 3.0, 1.0));
    }
    outColor = vec4(color, 1.0);
}
` + "\x00"

// ── Cube geometry ─────────────────────────────────────────────────────────────

// 36 positions (xyz) for a unit cube — standard CCW winding from the outside.
// Face culling is disabled during draw so we see the inside faces.
var skyboxVerts = []float32{
	// -Z face
	-1, -1, -1, 1, 1, -1, 1, -1, -1,
	1, 1, -1, -1, -1, -1, -1, 1, -1,
	// +Z face
	-1, -1, 1, 1, -1, 1, 1, 1, 1,
	1, 1, 1, -1, 1, 1, -1, -1, 1,
	// -X face
	-1, 1, 1, -1, 1, -1, -1, -1, -1,
	-1, -1, -1, -1, -1, 1, -1, 1, 1,
	// +X face
	1, 1, 1, 1, -1, -1, 1, 1, -1,
	1, -1, -1, 1, 1, 1, 1, -1, 1,
	// -Y face
	-1, -1, -1, 1, -1, -1, 1, -1, 1,
	1, -1, 1, -1, -1, 1, -1, -1, -1,
	// +Y face
	-1, 1, -1, 1, 1, 1, 1, 1, -1,
	1, 1, 1, -1, 1, -1, -1, 1, 1,
}

// ── Constructor ───────────────────────────────────────────────────────────────

// NewSkybox compiles the gradient sky shader and uploads the cube geometry.
// Default colours give a pleasant blue sky with a warm brown ground.
func NewSkybox() (*Skybox, error) {
	prog, err := newProgram(skyVertSrc, skyFragSrc)
	if err != nil {
		return nil, fmt.Errorf("skybox shader: %w", err)
	}

	sb := &Skybox{
		prog:       prog,
		vpLoc:      gl.GetUniformLocation(prog, gl.Str("skyVP\x00")),
		zenithLoc:  gl.GetUniformLocation(prog, gl.Str("zenith\x00")),
		horizonLoc: gl.GetUniformLocation(prog, gl.Str("horizon\x00")),
		groundLoc:  gl.GetUniformLocation(prog, gl.Str("ground\x00")),

		// Deep blue zenith, pale blue horizon, warm brown ground
		ZenithColor:  core.Color{R: 0.10, G: 0.30, B: 0.70, A: 1},
		HorizonColor: core.Color{R: 0.60, G: 0.80, B: 1.00, A: 1},
		GroundColor:  core.Color{R: 0.30, G: 0.25, B: 0.20, A: 1},
	}

	gl.GenVertexArrays(1, &sb.vao)
	gl.GenBuffers(1, &sb.vbo)
	gl.BindVertexArray(sb.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, sb.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(skyboxVerts)*4, gl.Ptr(skyboxVerts), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 12, gl.PtrOffset(0))
	gl.BindVertexArray(0)

	return sb, nil
}

// ── Draw ──────────────────────────────────────────────────────────────────────

// Draw renders the sky.  skyVP must be the combined (view-without-translation)×proj
// matrix — the caller is responsible for stripping the translation column from view.
func (sb *Skybox) Draw(skyVP math.Mat4) {
	// Depth LEQUAL so depth=1.0 fragments pass against the cleared depth value (1.0).
	// Depth mask off — we don't want to write 1.0 into the depth buffer.
	gl.DepthFunc(gl.LEQUAL)
	gl.DepthMask(false)

	gl.UseProgram(sb.prog)
	gl.UniformMatrix4fv(sb.vpLoc, 1, false, (*float32)(unsafe.Pointer(&skyVP[0][0])))
	gl.Uniform3f(sb.zenithLoc, sb.ZenithColor.R, sb.ZenithColor.G, sb.ZenithColor.B)
	gl.Uniform3f(sb.horizonLoc, sb.HorizonColor.R, sb.HorizonColor.G, sb.HorizonColor.B)
	gl.Uniform3f(sb.groundLoc, sb.GroundColor.R, sb.GroundColor.G, sb.GroundColor.B)

	gl.BindVertexArray(sb.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, 36)
	gl.BindVertexArray(0)

	// Restore depth state for scene geometry
	gl.DepthMask(true)
	gl.DepthFunc(gl.LESS)
}

// Destroy frees all GPU resources owned by this skybox.
func (sb *Skybox) Destroy() {
	gl.DeleteVertexArrays(1, &sb.vao)
	gl.DeleteBuffers(1, &sb.vbo)
	gl.DeleteProgram(sb.prog)
}
