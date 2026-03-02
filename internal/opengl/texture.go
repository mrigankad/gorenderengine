package opengl

import (
	"fmt"
	"unsafe"

	gl "github.com/go-gl/gl/v4.1-core/gl"

	"render-engine/scene"
)

// UploadTexture uploads a scene.Texture to the GPU and sets its GLID field.
// Call this from the main goroutine (OpenGL context must be current).
// The texture can then be assigned to a Mesh.Texture and will be sampled
// automatically during DrawMesh.
func UploadTexture(tex *scene.Texture) error {
	if tex == nil {
		return fmt.Errorf("nil texture")
	}
	if len(tex.Pixels) == 0 {
		return fmt.Errorf("texture %q has no pixel data", tex.Name)
	}

	var id uint32
	gl.GenTextures(1, &id)
	gl.BindTexture(gl.TEXTURE_2D, id)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)

	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(tex.Width),
		int32(tex.Height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		unsafe.Pointer(&tex.Pixels[0]),
	)
	gl.GenerateMipmap(gl.TEXTURE_2D)

	gl.BindTexture(gl.TEXTURE_2D, 0)

	tex.GLID = id
	return nil
}

// DeleteTexture frees a previously uploaded GPU texture and zeroes its GLID.
func DeleteTexture(tex *scene.Texture) {
	if tex == nil || tex.GLID == 0 {
		return
	}
	gl.DeleteTextures(1, &tex.GLID)
	tex.GLID = 0
}
