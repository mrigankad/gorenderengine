package vulkan

/*
#include <vulkan/vulkan.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type CommandBuffer struct {
	Handle C.VkCommandBuffer
}

func AllocateCommandBuffers(device *Device, pool C.VkCommandPool, count uint32) ([]CommandBuffer, error) {
	allocInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		commandPool:        pool,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandBufferCount: C.uint32_t(count),
	}
	
	buffers := make([]CommandBuffer, count)
	handles := make([]C.VkCommandBuffer, count)
	
	result := C.vkAllocateCommandBuffers(device.Device, &allocInfo, &handles[0])
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to allocate command buffers: %d", result)
	}
	
	for i := range buffers {
		buffers[i].Handle = handles[i]
	}
	
	return buffers, nil
}

func (cb *CommandBuffer) Begin(oneTime bool) error {
	beginInfo := C.VkCommandBufferBeginInfo{
		sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
	}
	
	if oneTime {
		beginInfo.flags = C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT
	}
	
	result := C.vkBeginCommandBuffer(cb.Handle, &beginInfo)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to begin recording command buffer: %d", result)
	}
	return nil
}

func (cb *CommandBuffer) End() error {
	result := C.vkEndCommandBuffer(cb.Handle)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to end recording command buffer: %d", result)
	}
	return nil
}

func (cb *CommandBuffer) BeginRenderPass(renderPass C.VkRenderPass, framebuffer C.VkFramebuffer, renderArea C.VkRect2D, clearValues []C.VkClearValue) {
	renderPassInfo := C.VkRenderPassBeginInfo{
		sType:           C.VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
		renderPass:      renderPass,
		framebuffer:     framebuffer,
		renderArea:      renderArea,
		clearValueCount: C.uint32_t(len(clearValues)),
		pClearValues:    &clearValues[0],
	}
	
	C.vkCmdBeginRenderPass(cb.Handle, &renderPassInfo, C.VK_SUBPASS_CONTENTS_INLINE)
}

func (cb *CommandBuffer) EndRenderPass() {
	C.vkCmdEndRenderPass(cb.Handle)
}

func (cb *CommandBuffer) BindPipeline(pipeline C.VkPipeline) {
	C.vkCmdBindPipeline(cb.Handle, C.VK_PIPELINE_BIND_POINT_GRAPHICS, pipeline)
}

func (cb *CommandBuffer) BindVertexBuffer(buffer C.VkBuffer, offset uint64) {
	C.vkCmdBindVertexBuffers(cb.Handle, 0, 1, &buffer, (*C.VkDeviceSize)(&offset))
}

func (cb *CommandBuffer) BindIndexBuffer(buffer C.VkBuffer, offset uint64, indexType C.VkIndexType) {
	C.vkCmdBindIndexBuffer(cb.Handle, buffer, C.VkDeviceSize(offset), indexType)
}

func (cb *CommandBuffer) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	C.vkCmdDraw(cb.Handle, C.uint32_t(vertexCount), C.uint32_t(instanceCount), C.uint32_t(firstVertex), C.uint32_t(firstInstance))
}

func (cb *CommandBuffer) DrawIndexed(indexCount, instanceCount, firstIndex, vertexOffset, firstInstance uint32) {
	C.vkCmdDrawIndexed(cb.Handle, C.uint32_t(indexCount), C.uint32_t(instanceCount), C.uint32_t(firstIndex), C.int32_t(vertexOffset), C.uint32_t(firstInstance))
}

func (cb *CommandBuffer) SetViewport(viewport C.VkViewport) {
	C.vkCmdSetViewport(cb.Handle, 0, 1, &viewport)
}

func (cb *CommandBuffer) SetScissor(scissor C.VkRect2D) {
	C.vkCmdSetScissor(cb.Handle, 0, 1, &scissor)
}

func (cb *CommandBuffer) PushConstants(layout C.VkPipelineLayout, stageFlags C.VkShaderStageFlags, offset uint32, size uint32, values unsafe.Pointer) {
	C.vkCmdPushConstants(cb.Handle, layout, stageFlags, C.uint32_t(offset), C.uint32_t(size), values)
}

func (cb *CommandBuffer) BindDescriptorSets(layout C.VkPipelineLayout, firstSet uint32, descriptorSets []C.VkDescriptorSet) {
	C.vkCmdBindDescriptorSets(cb.Handle, C.VK_PIPELINE_BIND_POINT_GRAPHICS, layout, C.uint32_t(firstSet), C.uint32_t(len(descriptorSets)), &descriptorSets[0], 0, nil)
}

func TransitionImageLayout(cmdBuffer C.VkCommandBuffer, image C.VkImage, format C.VkFormat, oldLayout, newLayout C.VkImageLayout, mipLevels uint32) {
	var barrier C.VkImageMemoryBarrier
	barrier.sType = C.VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER
	barrier.oldLayout = oldLayout
	barrier.newLayout = newLayout
	barrier.srcQueueFamilyIndex = C.VK_QUEUE_FAMILY_IGNORED
	barrier.dstQueueFamilyIndex = C.VK_QUEUE_FAMILY_IGNORED
	barrier.image = image
	barrier.subresourceRange.aspectMask = C.VK_IMAGE_ASPECT_COLOR_BIT
	barrier.subresourceRange.baseMipLevel = 0
	barrier.subresourceRange.levelCount = C.uint32_t(mipLevels)
	barrier.subresourceRange.baseArrayLayer = 0
	barrier.subresourceRange.layerCount = 1
	
	var srcStage, dstStage C.VkPipelineStageFlags
	
	if oldLayout == C.VK_IMAGE_LAYOUT_UNDEFINED && newLayout == C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL {
		barrier.srcAccessMask = 0
		barrier.dstAccessMask = C.VK_ACCESS_TRANSFER_WRITE_BIT
		srcStage = C.VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT
		dstStage = C.VK_PIPELINE_STAGE_TRANSFER_BIT
	} else if oldLayout == C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL && newLayout == C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL {
		barrier.srcAccessMask = C.VK_ACCESS_TRANSFER_WRITE_BIT
		barrier.dstAccessMask = C.VK_ACCESS_SHADER_READ_BIT
		srcStage = C.VK_PIPELINE_STAGE_TRANSFER_BIT
		dstStage = C.VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT
	} else if oldLayout == C.VK_IMAGE_LAYOUT_UNDEFINED && newLayout == C.VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL {
		barrier.subresourceRange.aspectMask = C.VK_IMAGE_ASPECT_DEPTH_BIT
		if format == C.VK_FORMAT_D32_SFLOAT_S8_UINT || format == C.VK_FORMAT_D24_UNORM_S8_UINT {
			barrier.subresourceRange.aspectMask |= C.VK_IMAGE_ASPECT_STENCIL_BIT
		}
		barrier.srcAccessMask = 0
		barrier.dstAccessMask = C.VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_READ_BIT | C.VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT
		srcStage = C.VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT
		dstStage = C.VK_PIPELINE_STAGE_EARLY_FRAGMENT_TESTS_BIT
	} else {
		barrier.srcAccessMask = 0
		barrier.dstAccessMask = 0
		srcStage = C.VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT
		dstStage = C.VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT
	}
	
	C.vkCmdPipelineBarrier(cmdBuffer, srcStage, dstStage, 0, 0, nil, 0, nil, 1, &barrier)
}

func CopyBufferToImage(cmdBuffer C.VkCommandBuffer, buffer C.VkBuffer, image C.VkImage, width, height uint32) {
	region := C.VkBufferImageCopy{
		bufferOffset:      0,
		bufferRowLength:   0,
		bufferImageHeight: 0,
		imageSubresource: C.VkImageSubresourceLayers{
			aspectMask:     C.VK_IMAGE_ASPECT_COLOR_BIT,
			mipLevel:       0,
			baseArrayLayer: 0,
			layerCount:     1,
		},
		imageOffset: C.VkOffset3D{x: 0, y: 0, z: 0},
		imageExtent: C.VkExtent3D{width: C.uint32_t(width), height: C.uint32_t(height), depth: 1},
	}
	
	C.vkCmdCopyBufferToImage(cmdBuffer, buffer, image, C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &region)
}

func FreeCommandBuffers(device *Device, pool C.VkCommandPool, buffers []CommandBuffer) {
	handles := make([]C.VkCommandBuffer, len(buffers))
	for i, buf := range buffers {
		handles[i] = buf.Handle
	}
	C.vkFreeCommandBuffers(device.Device, pool, C.uint32_t(len(handles)), &handles[0])
}
