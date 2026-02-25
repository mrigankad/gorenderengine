package renderer

/*
#include <vulkan/vulkan.h>
*/
import "C"

import (
	"fmt"
	"unsafe"

	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
	"render-engine/vulkan"
)

// Simple uniform buffer structure
type SimpleMVP struct {
	MVP math.Mat4
}

// Full uniform buffer structure
type UniformBufferObject struct {
	Model math.Mat4
	View  math.Mat4
	Proj  math.Mat4
}

type RenderEngine struct {
	Renderer *vulkan.Renderer
	Scene    *scene.Scene

	// Shader modules
	VertexShader   []uint32
	FragmentShader []uint32
}

func NewRenderEngine(window *core.Window) (*RenderEngine, error) {
	re := &RenderEngine{}

	// Create Vulkan renderer
	renderer, err := vulkan.NewRenderer(window)
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}
	re.Renderer = renderer

	// Use default shaders
	re.VertexShader = SimpleVertexShaderSPIRV
	re.FragmentShader = SimpleFragmentShaderSPIRV

	// Create default pipeline
	if err := renderer.CreateDefaultPipeline(re.VertexShader, re.FragmentShader); err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	fmt.Println("Render engine initialized successfully")

	return re, nil
}

func (re *RenderEngine) SetScene(s *scene.Scene) {
	re.Scene = s
}

func (re *RenderEngine) Render() error {
	if re.Scene == nil || re.Scene.Camera == nil {
		return fmt.Errorf("no scene or camera set")
	}

	// Acquire next image
	imageIndex, err := re.Renderer.BeginFrame()
	if err != nil {
		return err
	}

	// Begin command buffer recording
	re.Renderer.BeginCommandBuffer(imageIndex, re.Scene.SkyColor)

	// Get command buffer and bind pipeline
	cmdBuffer := re.Renderer.GetCurrentCommandBuffer()
	cmdBuffer.BindPipeline(re.Renderer.DefaultPipeline.Handle)

	// Set dynamic state
	cmdBuffer.SetViewport(0, 0, float32(re.Renderer.SwapChain.Width), float32(re.Renderer.SwapChain.Height))
	cmdBuffer.SetScissor(0, 0, re.Renderer.SwapChain.Width, re.Renderer.SwapChain.Height)

	// Get camera matrices
	view := re.Scene.Camera.GetViewMatrix()
	proj := re.Scene.Camera.GetProjectionMatrix()

	// Render all visible nodes
	visibleNodes := re.Scene.GetVisibleNodes()
	for _, node := range visibleNodes {
		if node.Mesh == nil {
			continue
		}

		// Calculate MVP matrix
		model := node.GetWorldMatrix()
		mvp := proj.Mul(view).Mul(model)

		// Update uniform buffer
		ubo := SimpleMVP{MVP: mvp}
		re.Renderer.UpdateUniformBuffer(unsafe.Pointer(&ubo), uint64(unsafe.Sizeof(ubo)))

		// Bind vertex buffer
		if node.Mesh.VertexBuffer != nil {
			cmdBuffer.BindVertexBuffer(node.Mesh.VertexBuffer.Handle, 0)
		}

		// Draw
		if node.Mesh.IndexBuffer != nil && node.Mesh.IndexCount > 0 {
			cmdBuffer.BindIndexBuffer(node.Mesh.IndexBuffer.Handle, 0, C.VK_INDEX_TYPE_UINT32)
			cmdBuffer.DrawIndexed(node.Mesh.IndexCount, 1, 0, 0, 0)
		} else if node.Mesh.VertexBuffer != nil {
			cmdBuffer.Draw(uint32(len(node.Mesh.Vertices)), 1, 0, 0)
		}
	}

	// End command buffer
	re.Renderer.EndCommandBuffer()

	// Submit and present
	return re.Renderer.SubmitAndPresent(imageIndex)
}

func (re *RenderEngine) Resize(width, height uint32) {
	re.Renderer.Resize(width, height)
	if re.Scene != nil && re.Scene.Camera != nil {
		re.Scene.Camera.UpdateAspectRatio(float32(width), float32(height))
	}
}

func (re *RenderEngine) Destroy() {
	re.Renderer.Destroy()
}

func (re *RenderEngine) WaitIdle() {
	re.Renderer.Device.WaitIdle()
}
