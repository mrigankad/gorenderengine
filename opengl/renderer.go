package opengl

import (
	"fmt"
	"strings"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
)

// GPUMesh holds the OpenGL buffer objects for an uploaded mesh.
type GPUMesh struct {
	VAO        uint32
	VBO        uint32
	EBO        uint32
	IndexCount int32
	HasIndices bool
}

// Renderer is the OpenGL rendering backend.
type Renderer struct {
	program   uint32
	mvpLoc    int32
	gpuMeshes map[*scene.Mesh]*GPUMesh
}

// vertex shader: MVP transform + per-vertex colour passthrough
const vertSrc = `
#version 410 core
layout(location = 0) in vec3 inPosition;
layout(location = 1) in vec3 inNormal;
layout(location = 2) in vec2 inUV;
layout(location = 3) in vec4 inColor;

uniform mat4 mvp;

out vec4 fragColor;
out vec3 fragNormal;

void main() {
    gl_Position = mvp * vec4(inPosition, 1.0);
    fragColor   = inColor;
    fragNormal  = inNormal;
}
` + "\x00"

// fragment shader: colour with simple directional shading
const fragSrc = `
#version 410 core
in vec4 fragColor;
in vec3 fragNormal;

out vec4 outColor;

void main() {
    vec3  lightDir = normalize(vec3(0.5, -1.0, -0.5));
    float diff     = max(dot(normalize(fragNormal), -lightDir), 0.0);
    vec3  lit      = fragColor.rgb * (0.3 + 0.7 * diff);
    outColor = vec4(lit, fragColor.a);
}
` + "\x00"

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
		return nil, fmt.Errorf("shader compile: %w", err)
	}

	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)

	r := &Renderer{
		program:   prog,
		mvpLoc:    gl.GetUniformLocation(prog, gl.Str("mvp\x00")),
		gpuMeshes: make(map[*scene.Mesh]*GPUMesh),
	}
	return r, nil
}

// SetViewport resizes the OpenGL viewport.
func (r *Renderer) SetViewport(width, height int) {
	gl.Viewport(0, 0, int32(width), int32(height))
}

// BeginFrame clears the framebuffer with the given colour.
func (r *Renderer) BeginFrame(sky core.Color) {
	gl.ClearColor(sky.R, sky.G, sky.B, sky.A)
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
}

// DrawMesh uploads mesh data on first use, then issues a draw call with the
// given MVP matrix.
func (r *Renderer) DrawMesh(mesh *scene.Mesh, mvp math.Mat4) {
	gpu := r.ensureUploaded(mesh)
	if gpu == nil {
		return
	}

	gl.UseProgram(r.program)
	// Mat4 is [4][4]float32 stored column-major — pass directly (transpose=false).
	gl.UniformMatrix4fv(r.mvpLoc, 1, false, (*float32)(unsafe.Pointer(&mvp[0][0])))

	gl.BindVertexArray(gpu.VAO)
	if gpu.HasIndices {
		gl.DrawElements(gl.TRIANGLES, gpu.IndexCount, gl.UNSIGNED_INT, nil)
	} else {
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(mesh.Vertices)))
	}
	gl.BindVertexArray(0)
}

// ReleaseMesh frees GPU buffers for the given mesh.
func (r *Renderer) ReleaseMesh(mesh *scene.Mesh) {
	if gpu, ok := r.gpuMeshes[mesh]; ok {
		gl.DeleteVertexArrays(1, &gpu.VAO)
		gl.DeleteBuffers(1, &gpu.VBO)
		if gpu.HasIndices {
			gl.DeleteBuffers(1, &gpu.EBO)
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
	gl.DeleteProgram(r.program)
}

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

	// Upload vertex data
	gl.BindBuffer(gl.ARRAY_BUFFER, gpu.VBO)
	gl.BufferData(gl.ARRAY_BUFFER,
		len(mesh.Vertices)*int(stride),
		gl.Ptr(mesh.Vertices),
		gl.STATIC_DRAW)

	// Compute field offsets from an empty Vertex
	var v core.Vertex
	posOff := int(unsafe.Offsetof(v.Position))
	normOff := int(unsafe.Offsetof(v.Normal))
	uvOff := int(unsafe.Offsetof(v.UV))
	colorOff := int(unsafe.Offsetof(v.Color))

	// location 0: Position (vec3)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, stride, gl.PtrOffset(posOff))

	// location 1: Normal (vec3)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 3, gl.FLOAT, false, stride, gl.PtrOffset(normOff))

	// location 2: UV (vec2)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 2, gl.FLOAT, false, stride, gl.PtrOffset(uvOff))

	// location 3: Color (vec4 RGBA float32)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(3, 4, gl.FLOAT, false, stride, gl.PtrOffset(colorOff))

	// Upload index data
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

// ── shader helpers ────────────────────────────────────────────────────────────

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
