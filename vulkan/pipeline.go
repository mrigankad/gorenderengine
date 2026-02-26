package vulkan

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>

VkShaderModule createShaderModule(VkDevice device, const uint32_t* code, size_t size) {
    VkShaderModuleCreateInfo createInfo = {0};
    createInfo.sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO;
    createInfo.codeSize = size;
    createInfo.pCode = code;
    
    VkShaderModule shaderModule;
    if (vkCreateShaderModule(device, &createInfo, NULL, &shaderModule) != VK_SUCCESS) {
        return NULL;
    }
    return shaderModule;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Pipeline struct {
	Handle       C.VkPipeline
	Layout       C.VkPipelineLayout
	RenderPass   C.VkRenderPass
	VertexShader C.VkShaderModule
	FragShader   C.VkShaderModule
	DescriptorSetLayout C.VkDescriptorSetLayout
}

type VertexInputDescription struct {
	BindingDescriptions   []C.VkVertexInputBindingDescription
	AttributeDescriptions []C.VkVertexInputAttributeDescription
}

type PipelineConfig struct {
	VertexShaderCode    []uint32
	FragmentShaderCode  []uint32
	VertexDescription   VertexInputDescription
	Topology            C.VkPrimitiveTopology
	PolygonMode         C.VkPolygonMode
	CullMode            C.VkCullModeFlags
	FrontFace           C.VkFrontFace
	DepthTestEnable     bool
	DepthWriteEnable    bool
	BlendEnable         bool
	ViewportWidth       float32
	ViewportHeight      float32
	DescriptorSetLayout C.VkDescriptorSetLayout
}

func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Topology:        C.VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
		PolygonMode:     C.VK_POLYGON_MODE_FILL,
		CullMode:        C.VK_CULL_MODE_BACK_BIT,
		FrontFace:       C.VK_FRONT_FACE_COUNTER_CLOCKWISE,
		DepthTestEnable: true,
		DepthWriteEnable: true,
		BlendEnable:     true,
	}
}

func CreateGraphicsPipeline(device *Device, config PipelineConfig) (*Pipeline, error) {
	p := &Pipeline{}
	
	// Create shader modules
	if len(config.VertexShaderCode) > 0 {
		p.VertexShader = C.createShaderModule(device.Device, (*C.uint32_t)(unsafe.Pointer(&config.VertexShaderCode[0])), C.size_t(len(config.VertexShaderCode)*4))
		if p.VertexShader == nil {
			return nil, fmt.Errorf("failed to create vertex shader module")
		}
	}
	
	if len(config.FragmentShaderCode) > 0 {
		p.FragShader = C.createShaderModule(device.Device, (*C.uint32_t)(unsafe.Pointer(&config.FragmentShaderCode[0])), C.size_t(len(config.FragmentShaderCode)*4))
		if p.FragShader == nil {
			return nil, fmt.Errorf("failed to create fragment shader module")
		}
	}
	
	// Shader stages
	shaderStages := []C.VkPipelineShaderStageCreateInfo{
		{
			sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
			stage:  C.VK_SHADER_STAGE_VERTEX_BIT,
			module: p.VertexShader,
			pName:  C.CString("main"),
		},
		{
			sType:  C.VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
			stage:  C.VK_SHADER_STAGE_FRAGMENT_BIT,
			module: p.FragShader,
			pName:  C.CString("main"),
		},
	}
	defer C.free(unsafe.Pointer(shaderStages[0].pName))
	defer C.free(unsafe.Pointer(shaderStages[1].pName))
	
	// Vertex input
	var vertexInputInfo C.VkPipelineVertexInputStateCreateInfo
	if len(config.VertexDescription.BindingDescriptions) > 0 {
		vertexInputInfo = C.VkPipelineVertexInputStateCreateInfo{
			sType:                           C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
			vertexBindingDescriptionCount:   C.uint32_t(len(config.VertexDescription.BindingDescriptions)),
			pVertexBindingDescriptions:      &config.VertexDescription.BindingDescriptions[0],
			vertexAttributeDescriptionCount: C.uint32_t(len(config.VertexDescription.AttributeDescriptions)),
			pVertexAttributeDescriptions:    &config.VertexDescription.AttributeDescriptions[0],
		}
	} else {
		vertexInputInfo = C.VkPipelineVertexInputStateCreateInfo{
			sType: C.VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
		}
	}
	
	// Input assembly
	inputAssembly := C.VkPipelineInputAssemblyStateCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
		topology: config.Topology,
	}
	
	// Viewport
	viewport := C.VkViewport{
		x:        0,
		y:        0,
		width:    C.float(config.ViewportWidth),
		height:   C.float(config.ViewportHeight),
		minDepth: 0,
		maxDepth: 1,
	}
	
	scissor := C.VkRect2D{
		offset: C.VkOffset2D{x: 0, y: 0},
		extent: C.VkExtent2D{width: C.uint32_t(config.ViewportWidth), height: C.uint32_t(config.ViewportHeight)},
	}
	
	viewportState := C.VkPipelineViewportStateCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
		viewportCount: 1,
		pViewports:    &viewport,
		scissorCount:  1,
		pScissors:     &scissor,
	}
	
	// Rasterizer
	rasterizer := C.VkPipelineRasterizationStateCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
		depthClampEnable:        C.VK_FALSE,
		rasterizerDiscardEnable: C.VK_FALSE,
		polygonMode:             config.PolygonMode,
		cullMode:                config.CullMode,
		frontFace:               config.FrontFace,
		depthBiasEnable:         C.VK_FALSE,
		lineWidth:               1.0,
	}
	
	// Multisampling
	multisampling := C.VkPipelineMultisampleStateCreateInfo{
		sType:                C.VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
		rasterizationSamples: C.VK_SAMPLE_COUNT_1_BIT,
	}
	
	// Depth stencil
	depthStencil := C.VkPipelineDepthStencilStateCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
		depthTestEnable:  C.VK_FALSE,
		depthWriteEnable: C.VK_FALSE,
	}
	if config.DepthTestEnable {
		depthStencil.depthTestEnable = C.VK_TRUE
		depthStencil.depthWriteEnable = C.VK_TRUE
		depthStencil.depthCompareOp = C.VK_COMPARE_OP_LESS
		depthStencil.depthBoundsTestEnable = C.VK_FALSE
		depthStencil.stencilTestEnable = C.VK_FALSE
	}
	
	// Color blending
	colorBlendAttachment := C.VkPipelineColorBlendAttachmentState{
		colorWriteMask: C.VK_COLOR_COMPONENT_R_BIT | C.VK_COLOR_COMPONENT_G_BIT | C.VK_COLOR_COMPONENT_B_BIT | C.VK_COLOR_COMPONENT_A_BIT,
	}
	if config.BlendEnable {
		colorBlendAttachment.blendEnable = C.VK_TRUE
		colorBlendAttachment.srcColorBlendFactor = C.VK_BLEND_FACTOR_SRC_ALPHA
		colorBlendAttachment.dstColorBlendFactor = C.VK_BLEND_FACTOR_ONE_MINUS_SRC_ALPHA
		colorBlendAttachment.colorBlendOp = C.VK_BLEND_OP_ADD
		colorBlendAttachment.srcAlphaBlendFactor = C.VK_BLEND_FACTOR_ONE
		colorBlendAttachment.dstAlphaBlendFactor = C.VK_BLEND_FACTOR_ZERO
		colorBlendAttachment.alphaBlendOp = C.VK_BLEND_OP_ADD
	}
	
	colorBlending := C.VkPipelineColorBlendStateCreateInfo{
		sType:           C.VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
		logicOpEnable:   C.VK_FALSE,
		attachmentCount: 1,
		pAttachments:    &colorBlendAttachment,
	}
	
	// Pipeline layout
	layoutInfo := C.VkPipelineLayoutCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
	}

	if config.DescriptorSetLayout != nil {
		p.DescriptorSetLayout = config.DescriptorSetLayout
		layoutInfo.setLayoutCount = 1
		layoutInfo.pSetLayouts = &p.DescriptorSetLayout
	}
	
	result := C.vkCreatePipelineLayout(device.Device, &layoutInfo, nil, &p.Layout)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create pipeline layout: %d", result)
	}
	
	// Create pipeline
	pipelineInfo := C.VkGraphicsPipelineCreateInfo{
		sType:               C.VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
		stageCount:          2,
		pStages:             &shaderStages[0],
		pVertexInputState:   &vertexInputInfo,
		pInputAssemblyState: &inputAssembly,
		pViewportState:      &viewportState,
		pRasterizationState: &rasterizer,
		pMultisampleState:   &multisampling,
		pDepthStencilState:  &depthStencil,
		pColorBlendState:    &colorBlending,
		layout:              p.Layout,
		renderPass:          p.RenderPass,
		subpass:             0,
	}
	
	result = C.vkCreateGraphicsPipelines(device.Device, nil, 1, &pipelineInfo, nil, &p.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create graphics pipeline: %d", result)
	}
	
	return p, nil
}

func (p *Pipeline) Destroy(device *Device) {
	if p.Handle != nil {
		C.vkDestroyPipeline(device.Device, p.Handle, nil)
	}
	if p.Layout != nil {
		C.vkDestroyPipelineLayout(device.Device, p.Layout, nil)
	}
	if p.VertexShader != nil {
		C.vkDestroyShaderModule(device.Device, p.VertexShader, nil)
	}
	if p.FragShader != nil {
		C.vkDestroyShaderModule(device.Device, p.FragShader, nil)
	}
	if p.DescriptorSetLayout != nil {
		C.vkDestroyDescriptorSetLayout(device.Device, p.DescriptorSetLayout, nil)
	}
}

func CreateRenderPass(device *Device, swapchainFormat C.VkFormat, depthFormat C.VkFormat) (C.VkRenderPass, error) {
	// Count attachments
	attachmentCount := 1
	if depthFormat != 0 {
		attachmentCount = 2
	}
	
	// Allocate attachments in C memory
	attachments := C.malloc(C.size_t(attachmentCount) * C.size_t(unsafe.Sizeof(C.VkAttachmentDescription{})))
	defer C.free(attachments)
	
	// Color attachment
	colorAttach := (*C.VkAttachmentDescription)(attachments)
	colorAttach.format = swapchainFormat
	colorAttach.samples = C.VK_SAMPLE_COUNT_1_BIT
	colorAttach.loadOp = C.VK_ATTACHMENT_LOAD_OP_CLEAR
	colorAttach.storeOp = C.VK_ATTACHMENT_STORE_OP_STORE
	colorAttach.stencilLoadOp = C.VK_ATTACHMENT_LOAD_OP_DONT_CARE
	colorAttach.stencilStoreOp = C.VK_ATTACHMENT_STORE_OP_DONT_CARE
	colorAttach.initialLayout = C.VK_IMAGE_LAYOUT_UNDEFINED
	colorAttach.finalLayout = C.VK_IMAGE_LAYOUT_PRESENT_SRC_KHR
	
	// Color attachment reference - in C memory
	colorAttachRef := (*C.VkAttachmentReference)(C.malloc(C.size_t(unsafe.Sizeof(C.VkAttachmentReference{}))))
	defer C.free(unsafe.Pointer(colorAttachRef))
	colorAttachRef.attachment = 0
	colorAttachRef.layout = C.VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL
	
	// Subpass description - in C memory
	subpass := (*C.VkSubpassDescription)(C.malloc(C.size_t(unsafe.Sizeof(C.VkSubpassDescription{}))))
	defer C.free(unsafe.Pointer(subpass))
	C.memset(unsafe.Pointer(subpass), 0, C.size_t(unsafe.Sizeof(C.VkSubpassDescription{})))
	subpass.pipelineBindPoint = C.VK_PIPELINE_BIND_POINT_GRAPHICS
	subpass.colorAttachmentCount = 1
	subpass.pColorAttachments = colorAttachRef
	
	// Depth attachment if provided
	var depthAttachRef *C.VkAttachmentReference
	if depthFormat != 0 {
		depthAttach := (*C.VkAttachmentDescription)(unsafe.Pointer(uintptr(attachments) + unsafe.Sizeof(C.VkAttachmentDescription{})))
		depthAttach.format = depthFormat
		depthAttach.samples = C.VK_SAMPLE_COUNT_1_BIT
		depthAttach.loadOp = C.VK_ATTACHMENT_LOAD_OP_CLEAR
		depthAttach.storeOp = C.VK_ATTACHMENT_STORE_OP_DONT_CARE
		depthAttach.stencilLoadOp = C.VK_ATTACHMENT_LOAD_OP_DONT_CARE
		depthAttach.stencilStoreOp = C.VK_ATTACHMENT_STORE_OP_DONT_CARE
		depthAttach.initialLayout = C.VK_IMAGE_LAYOUT_UNDEFINED
		depthAttach.finalLayout = C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		
		depthAttachRef = (*C.VkAttachmentReference)(C.malloc(C.size_t(unsafe.Sizeof(C.VkAttachmentReference{}))))
		defer C.free(unsafe.Pointer(depthAttachRef))
		depthAttachRef.attachment = 1
		depthAttachRef.layout = C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL
		subpass.pDepthStencilAttachment = depthAttachRef
	}
	
	// Subpass dependency - in C memory
	dependency := (*C.VkSubpassDependency)(C.malloc(C.size_t(unsafe.Sizeof(C.VkSubpassDependency{}))))
	defer C.free(unsafe.Pointer(dependency))
	dependency.srcSubpass = C.VK_SUBPASS_EXTERNAL
	dependency.dstSubpass = 0
	dependency.srcStageMask = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | C.VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT
	dependency.srcAccessMask = 0
	dependency.dstStageMask = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT | C.VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT
	dependency.dstAccessMask = C.VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | C.VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT
	
	// Render pass create info - in C memory
	renderPassInfo := (*C.VkRenderPassCreateInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkRenderPassCreateInfo{}))))
	defer C.free(unsafe.Pointer(renderPassInfo))
	C.memset(unsafe.Pointer(renderPassInfo), 0, C.size_t(unsafe.Sizeof(C.VkRenderPassCreateInfo{})))
	renderPassInfo.sType = C.VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO
	renderPassInfo.attachmentCount = C.uint32_t(attachmentCount)
	renderPassInfo.pAttachments = (*C.VkAttachmentDescription)(attachments)
	renderPassInfo.subpassCount = 1
	renderPassInfo.pSubpasses = subpass
	renderPassInfo.dependencyCount = 1
	renderPassInfo.pDependencies = dependency
	
	var renderPass C.VkRenderPass
	result := C.vkCreateRenderPass(device.Device, renderPassInfo, nil, &renderPass)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create render pass: %d", result)
	}
	
	return renderPass, nil
}

func DestroyRenderPass(device *Device, renderPass C.VkRenderPass) {
	C.vkDestroyRenderPass(device.Device, renderPass, nil)
}
