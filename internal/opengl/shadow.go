package opengl

import (
	"fmt"

	gl "github.com/go-gl/gl/v4.1-core/gl"
)

// ShadowMap wraps a depth-only framebuffer used for shadow mapping.
type ShadowMap struct {
	FBO      uint32
	DepthTex uint32
	Size     int32
}

// NewShadowMap creates a depth-only FBO of size×size resolution.
// Uses a 32-bit float depth texture with hardware PCF (COMPARE_REF_TO_TEXTURE).
func NewShadowMap(size int) (*ShadowMap, error) {
	sm := &ShadowMap{Size: int32(size)}

	// Depth texture
	gl.GenTextures(1, &sm.DepthTex)
	gl.BindTexture(gl.TEXTURE_2D, sm.DepthTex)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.DEPTH_COMPONENT32F,
		int32(size), int32(size), 0, gl.DEPTH_COMPONENT, gl.FLOAT, nil)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_BORDER)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_BORDER)
	// Fragments outside the shadow map are lit (border depth = 1.0)
	border := [4]float32{1, 1, 1, 1}
	gl.TexParameterfv(gl.TEXTURE_2D, gl.TEXTURE_BORDER_COLOR, &border[0])
	// Enable hardware PCF: texture() returns 0.0 or 1.0 based on depth comparison
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_MODE, gl.COMPARE_REF_TO_TEXTURE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_COMPARE_FUNC, gl.LEQUAL)

	// Depth-only framebuffer
	gl.GenFramebuffers(1, &sm.FBO)
	gl.BindFramebuffer(gl.FRAMEBUFFER, sm.FBO)
	gl.FramebufferTexture2D(gl.FRAMEBUFFER, gl.DEPTH_ATTACHMENT, gl.TEXTURE_2D, sm.DepthTex, 0)
	gl.DrawBuffer(gl.NONE)
	gl.ReadBuffer(gl.NONE)

	status := gl.CheckFramebufferStatus(gl.FRAMEBUFFER)
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	if status != gl.FRAMEBUFFER_COMPLETE {
		gl.DeleteTextures(1, &sm.DepthTex)
		gl.DeleteFramebuffers(1, &sm.FBO)
		return nil, fmt.Errorf("shadow FBO incomplete: status=0x%X", status)
	}

	return sm, nil
}

// Destroy frees GPU resources.
func (sm *ShadowMap) Destroy() {
	if sm.FBO != 0 {
		gl.DeleteFramebuffers(1, &sm.FBO)
		sm.FBO = 0
	}
	if sm.DepthTex != 0 {
		gl.DeleteTextures(1, &sm.DepthTex)
		sm.DepthTex = 0
	}
}
