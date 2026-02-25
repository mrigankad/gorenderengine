package vulkan

/*
#include <vulkan/vulkan.h>
#include <windows.h>

VkResult CreateWin32Surface(VkInstance instance, HINSTANCE hinstance, HWND hwnd, VkSurfaceKHR* surface) {
    VkWin32SurfaceCreateInfoKHR createInfo = {0};
    createInfo.sType = VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR;
    createInfo.hinstance = hinstance;
    createInfo.hwnd = hwnd;
    return vkCreateWin32SurfaceKHR(instance, &createInfo, NULL, surface);
}

HWND GetActiveWindowHandle() {
    return GetActiveWindow();
}

HINSTANCE GetModuleHandleWin() {
    return GetModuleHandle(NULL);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
	
	"render-engine/core"
)

const MaxFramesInFlight = 2

type Renderer struct {
	Instance     *Instance
	Surface      C.VkSurfaceKHR
	Device       *Device
	SwapChain    *SwapChain
	RenderPass   C.VkRenderPass
	DepthImage   *Image
	
	// Per-frame resources
	CommandBuffers   []CommandBuffer
	ImageAvailable   []*Semaphore
	RenderFinished   []*Semaphore
	InFlightFences   []*Fence
	ImagesInFlight   []C.VkFence
	CurrentFrame     uint32
	
	// Default pipeline
	DefaultPipeline  *Pipeline
	
	// Descriptor resources
	DescriptorPool   *DescriptorPool
	DescriptorSetLayout C.VkDescriptorSetLayout
	
	// Uniform buffers (per frame)
	UniformBuffers   []*Buffer
	UniformBufferSize uint64
}

func NewRenderer(window *core.Window) (*Renderer, error) {
	r := &Renderer{}
	
	// Create Vulkan instance
	config := DefaultInstanceConfig()
	config.RequiredExtensions = window.GetRequiredInstanceExtensions()
	
	instance, err := NewInstance(config)
	if err != nil {
		return nil, err
	}
	r.Instance = instance
	
	// Create surface
	if err := r.createSurface(window); err != nil {
		return nil, err
	}
	
	// Select physical device and create logical device
	device, err := PickPhysicalDevice(instance, r.Surface)
	if err != nil {
		return nil, err
	}
	r.Device = device
	
	if err := device.CreateLogicalDevice(r.Surface); err != nil {
		return nil, err
	}
	
	fmt.Printf("Selected GPU: %s (%s)\n", device.GetGPUName(), device.GetDeviceType())
	
	// Create swapchain
	swapConfig := SwapChainConfig{
		Width:  uint32(window.Width),
		Height: uint32(window.Height),
		VSync:  true,
	}
	
	swapChain, err := CreateSwapChain(device, r.Surface, swapConfig)
	if err != nil {
		return nil, err
	}
	r.SwapChain = swapChain
	
	// Create depth buffer
	depthImage, err := CreateDepthBuffer(device, swapChain.Extent.width, swapChain.Extent.height)
	if err != nil {
		return nil, err
	}
	r.DepthImage = depthImage
	
	// Create render pass
	depthFormat := FindDepthFormat(device)
	renderPass, err := CreateRenderPass(device, swapChain.Format, depthFormat)
	if err != nil {
		return nil, err
	}
	r.RenderPass = renderPass
	
	// Create framebuffers
	if err := swapChain.CreateFramebuffers(device, renderPass, depthImage.View); err != nil {
		return nil, err
	}
	
	// Create command buffers
	r.CommandBuffers, err = AllocateCommandBuffers(device, device.CommandPool, MaxFramesInFlight)
	if err != nil {
		return nil, err
	}
	
	// Create synchronization objects
	r.ImageAvailable = make([]*Semaphore, MaxFramesInFlight)
	r.RenderFinished = make([]*Semaphore, MaxFramesInFlight)
	r.InFlightFences = make([]*Fence, MaxFramesInFlight)
	
	for i := 0; i < MaxFramesInFlight; i++ {
		r.ImageAvailable[i], err = CreateSemaphore(device)
		if err != nil {
			return nil, err
		}
		r.RenderFinished[i], err = CreateSemaphore(device)
		if err != nil {
			return nil, err
		}
		r.InFlightFences[i], err = CreateFence(device, true)
		if err != nil {
			return nil, err
		}
	}
	
	r.ImagesInFlight = make([]C.VkFence, swapChain.ImageCount)
	
	return r, nil
}

func (r *Renderer) createSurface(window *core.Window) error {
	// For Windows
	hwnd := C.GetActiveWindowHandle()
	hinstance := C.GetModuleHandleWin()
	
	result := C.CreateWin32Surface(r.Instance.Handle, hinstance, hwnd, &r.Surface)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to create window surface: %d", result)
	}
	
	return nil
}

func (r *Renderer) CreateDefaultPipeline(vertexShader, fragmentShader []uint32) error {
	// Create descriptor set layout
	bindings := []C.VkDescriptorSetLayoutBinding{
		UniformBufferBinding(0, C.VK_SHADER_STAGE_VERTEX_BIT),
	}
	
	layout, err := CreateDescriptorSetLayout(r.Device, bindings)
	if err != nil {
		return err
	}
	r.DescriptorSetLayout = layout
	
	// Create descriptor pool
	poolSizes := []C.VkDescriptorPoolSize{
		{
			type_:            C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
			descriptorCount:  MaxFramesInFlight,
		},
	}
	
	r.DescriptorPool, err = CreateDescriptorPool(r.Device, poolSizes, MaxFramesInFlight)
	if err != nil {
		return err
	}
	
	// Create uniform buffers
	r.UniformBufferSize = 256 // sizeof(mat4) * 2 aligned
	r.UniformBuffers = make([]*Buffer, MaxFramesInFlight)
	for i := 0; i < MaxFramesInFlight; i++ {
		buffer, err := CreateBuffer(r.Device, r.UniformBufferSize, 
			C.VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
			C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)
		if err != nil {
			return err
		}
		buffer.Map(r.Device)
		r.UniformBuffers[i] = buffer
	}
	
	// Create pipeline
	config := DefaultPipelineConfig()
	config.VertexShaderCode = vertexShader
	config.FragmentShaderCode = fragmentShader
	config.ViewportWidth = float32(r.SwapChain.Extent.width)
	config.ViewportHeight = float32(r.SwapChain.Extent.height)
	
	r.DefaultPipeline, err = CreateGraphicsPipeline(r.Device, config)
	if err != nil {
		return err
	}
	r.DefaultPipeline.RenderPass = r.RenderPass
	r.DefaultPipeline.DescriptorSetLayout = r.DescriptorSetLayout
	
	return nil
}

func (r *Renderer) BeginFrame() (uint32, error) {
	fence := r.InFlightFences[r.CurrentFrame]
	fence.Wait(r.Device, ^uint64(0))
	
	imageIndex, err := r.SwapChain.AcquireNextImage(r.Device, r.ImageAvailable[r.CurrentFrame].Handle, ^uint64(0))
	if err != nil {
		return 0, err
	}
	
	// Check if a previous frame is using this image
	if r.ImagesInFlight[imageIndex] != nil {
		fence.Handle = r.ImagesInFlight[imageIndex]
		fence.Wait(r.Device, ^uint64(0))
	}
	
	// Mark the image as now being in use by this frame
	r.ImagesInFlight[imageIndex] = r.InFlightFences[r.CurrentFrame].Handle
	
	return imageIndex, nil
}

func (r *Renderer) BeginCommandBuffer(imageIndex uint32, clearColor core.Color) {
	cmdBuffer := r.CommandBuffers[r.CurrentFrame]
	
	// Reset and begin recording
	C.vkResetCommandBuffer(cmdBuffer.Handle, 0)
	cmdBuffer.Begin(false)
	
	// Begin render pass
	clearValues := []C.VkClearValue{
		{
			color: C.VkClearColorValue{
				float32: [4]C.float{C.float(clearColor.R), C.float(clearColor.G), C.float(clearColor.B), C.float(clearColor.A)},
			},
		},
		{
			depthStencil: C.VkClearDepthStencilValue{
				depth:   1.0,
				stencil: 0,
			},
		},
	}
	
	renderArea := C.VkRect2D{
		offset: C.VkOffset2D{x: 0, y: 0},
		extent: r.SwapChain.Extent,
	}
	
	cmdBuffer.BeginRenderPass(r.RenderPass, r.SwapChain.Framebuffers[imageIndex], renderArea, clearValues)
}

func (r *Renderer) EndCommandBuffer() {
	cmdBuffer := r.CommandBuffers[r.CurrentFrame]
	cmdBuffer.EndRenderPass()
	cmdBuffer.End()
}

func (r *Renderer) SubmitAndPresent(imageIndex uint32) error {
	cmdBuffer := r.CommandBuffers[r.CurrentFrame]
	
	err := SubmitQueue(
		r.Device.GraphicsQueue,
		[]CommandBuffer{cmdBuffer},
		[]C.VkSemaphore{r.ImageAvailable[r.CurrentFrame].Handle},
		[]C.VkSemaphore{r.RenderFinished[r.CurrentFrame].Handle},
		r.InFlightFences[r.CurrentFrame],
	)
	if err != nil {
		return err
	}
	
	err = PresentQueue(
		r.Device.PresentQueue,
		[]C.VkSwapchainKHR{r.SwapChain.Handle},
		[]uint32{imageIndex},
		[]C.VkSemaphore{r.RenderFinished[r.CurrentFrame].Handle},
	)
	
	r.CurrentFrame = (r.CurrentFrame + 1) % MaxFramesInFlight
	
	return err
}

func (r *Renderer) Resize(width, height uint32) {
	r.Device.WaitIdle()
	
	// Recreate swapchain
	oldSwapChain := r.SwapChain
	
	swapConfig := SwapChainConfig{
		Width:  width,
		Height: height,
		VSync:  true,
	}
	
	newSwapChain, err := CreateSwapChain(r.Device, r.Surface, swapConfig)
	if err != nil {
		fmt.Printf("Failed to recreate swapchain: %v\n", err)
		return
	}
	
	// Destroy old swapchain
	for _, framebuffer := range r.SwapChain.Framebuffers {
		C.vkDestroyFramebuffer(r.Device.Device, framebuffer, nil)
	}
	oldSwapChain.Destroy(r.Device)
	
	r.SwapChain = newSwapChain
	
	// Recreate depth buffer
	r.DepthImage.Destroy(r.Device)
	depthImage, err := CreateDepthBuffer(r.Device, width, height)
	if err != nil {
		fmt.Printf("Failed to recreate depth buffer: %v\n", err)
		return
	}
	r.DepthImage = depthImage
	
	// Recreate framebuffers
	r.SwapChain.CreateFramebuffers(r.Device, r.RenderPass, r.DepthImage.View)
}

func (r *Renderer) Destroy() {
	r.Device.WaitIdle()
	
	// Cleanup resources
	for _, buffer := range r.UniformBuffers {
		buffer.Destroy(r.Device)
	}
	
	if r.DescriptorPool != nil {
		r.DescriptorPool.Destroy(r.Device)
	}
	
	if r.DescriptorSetLayout != nil {
		C.vkDestroyDescriptorSetLayout(r.Device.Device, r.DescriptorSetLayout, nil)
	}
	
	if r.DefaultPipeline != nil {
		r.DefaultPipeline.Destroy(r.Device)
	}
	
	for i := 0; i < MaxFramesInFlight; i++ {
		r.ImageAvailable[i].Destroy(r.Device)
		r.RenderFinished[i].Destroy(r.Device)
		r.InFlightFences[i].Destroy(r.Device)
	}
	
	r.DepthImage.Destroy(r.Device)
	
	for _, framebuffer := range r.SwapChain.Framebuffers {
		C.vkDestroyFramebuffer(r.Device.Device, framebuffer, nil)
	}
	
	DestroyRenderPass(r.Device, r.RenderPass)
	r.SwapChain.Destroy(r.Device)
	C.vkDestroySurfaceKHR(r.Instance.Handle, r.Surface, nil)
	r.Device.Destroy()
	r.Instance.Destroy()
}

func (r *Renderer) GetCurrentCommandBuffer() *CommandBuffer {
	return &r.CommandBuffers[r.CurrentFrame]
}

func (r *Renderer) GetCurrentUniformBuffer() *Buffer {
	return r.UniformBuffers[r.CurrentFrame]
}

func (r *Renderer) UpdateUniformBuffer(data unsafe.Pointer, size uint64) {
	r.UniformBuffers[r.CurrentFrame].CopyData(data, size)
}
