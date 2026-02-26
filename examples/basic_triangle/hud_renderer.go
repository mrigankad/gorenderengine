package main

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

// SimpleHUDRenderer renders basic 2D UI overlays
type SimpleHUDRenderer struct {
	vao    uint32
	vbo    uint32
	shader uint32
}

func NewSimpleHUDRenderer() *SimpleHUDRenderer {
	hr := &SimpleHUDRenderer{}
	hr.initShaders()
	hr.initGeometry()
	return hr
}

func (hr *SimpleHUDRenderer) initShaders() {
	// Simple 2D orthographic shader
	vertexShader := `
#version 410 core
layout(location = 0) in vec2 position;
layout(location = 1) in vec4 color;

out vec4 fragmentColor;

uniform mat4 projection;

void main() {
	gl_Position = projection * vec4(position, 0.0, 1.0);
	fragmentColor = color;
}
`

	fragmentShader := `
#version 410 core
in vec4 fragmentColor;
out vec4 outColor;

void main() {
	outColor = fragmentColor;
}
`

	vShader := gl.CreateShader(gl.VERTEX_SHADER)
	vSource, free := gl.Strs(vertexShader)
	defer free()
	gl.ShaderSource(vShader, 1, vSource, nil)
	gl.CompileShader(vShader)

	fShader := gl.CreateShader(gl.FRAGMENT_SHADER)
	fSource, free := gl.Strs(fragmentShader)
	defer free()
	gl.ShaderSource(fShader, 1, fSource, nil)
	gl.CompileShader(fShader)

	hr.shader = gl.CreateProgram()
	gl.AttachShader(hr.shader, vShader)
	gl.AttachShader(hr.shader, fShader)
	gl.LinkProgram(hr.shader)

	gl.DeleteShader(vShader)
	gl.DeleteShader(fShader)
}

func (hr *SimpleHUDRenderer) initGeometry() {
	gl.GenVertexArrays(1, &hr.vao)
	gl.GenBuffers(1, &hr.vbo)

	gl.BindVertexArray(hr.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, hr.vbo)

	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 24, gl.PtrOffset(0))
	gl.EnableVertexAttribArray(0)

	gl.VertexAttribPointer(1, 4, gl.FLOAT, false, 24, gl.PtrOffset(8))
	gl.EnableVertexAttribArray(1)

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}

// DrawPanel renders a colored panel at specified screen position
func (hr *SimpleHUDRenderer) DrawPanel(width, height float32, x, y float32, r, g, b, a float32) {
	// Simple quad vertices (screen-space coordinates normalized to -1..1)
	// This is a placeholder - actual implementation would need proper ortho projection
	vertices := []float32{
		// Position (x, y), Color (r, g, b, a)
		x, y, r, g, b, a,
		x + width, y, r, g, b, a,
		x + width, y + height, r, g, b, a,
		x, y + height, r, g, b, a,
	}

	gl.BindVertexArray(hr.vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, hr.vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(vertices)*4, gl.Ptr(vertices), gl.DYNAMIC_DRAW)

	gl.UseProgram(hr.shader)
	gl.DrawArrays(gl.TRIANGLE_FAN, 0, 4)

	gl.BindBuffer(gl.ARRAY_BUFFER, 0)
	gl.BindVertexArray(0)
}

func (hr *SimpleHUDRenderer) Destroy() {
	if hr.vao != 0 {
		gl.DeleteVertexArrays(1, &hr.vao)
	}
	if hr.vbo != 0 {
		gl.DeleteBuffers(1, &hr.vbo)
	}
	if hr.shader != 0 {
		gl.DeleteProgram(hr.shader)
	}
}
