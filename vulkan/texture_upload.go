package vulkan

/*
#include <vulkan/vulkan.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// TextureUploadResult contains the results of uploading texture data to the GPU
type TextureUploadResult struct {
	Image   *Image
	Sampler C.VkSampler
}

// UploadTextureData uploads raw RGBA pixel data to the GPU as a textured image with sampler
func UploadTextureData(device *Device, width, height uint32, pixels []byte) (*TextureUploadResult, error) {
	imageSize := uint64(width) * uint64(height) * 4

	// Create staging buffer
	stagingBuffer, err := CreateBuffer(device, imageSize,
		C.VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
		C.VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT|C.VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)
	if err != nil {
		return nil, fmt.Errorf("failed to create staging buffer: %w", err)
	}
	defer stagingBuffer.Destroy(device)

	// Copy pixel data to staging buffer
	if err := stagingBuffer.Map(device); err != nil {
		return nil, err
	}
	stagingBuffer.CopyData(unsafe.Pointer(&pixels[0]), imageSize)
	stagingBuffer.Unmap(device)

	// Create Vulkan image
	vkImage, err := CreateImage(device, width, height,
		C.VK_FORMAT_R8G8B8A8_SRGB,
		C.VK_IMAGE_TILING_OPTIMAL,
		C.VK_IMAGE_USAGE_TRANSFER_DST_BIT|C.VK_IMAGE_USAGE_SAMPLED_BIT,
		C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	// Execute single-time commands for layout transitions + copy
	err = ExecuteSingleTimeCommands(device, func(cmdBuffer C.VkCommandBuffer) {
		TransitionImageLayout(cmdBuffer, vkImage.Handle, vkImage.Format,
			C.VK_IMAGE_LAYOUT_UNDEFINED, C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1)

		CopyBufferToImage(cmdBuffer, stagingBuffer.Handle, vkImage.Handle, width, height)

		TransitionImageLayout(cmdBuffer, vkImage.Handle, vkImage.Format,
			C.VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL, 1)
	})
	if err != nil {
		vkImage.Destroy(device)
		return nil, fmt.Errorf("failed to upload texture data: %w", err)
	}

	// Create image view
	if err := vkImage.CreateView(device, C.VK_IMAGE_ASPECT_COLOR_BIT); err != nil {
		vkImage.Destroy(device)
		return nil, fmt.Errorf("failed to create image view: %w", err)
	}

	// Create sampler
	sampler, err := CreateSampler(device,
		C.VK_FILTER_LINEAR, C.VK_FILTER_LINEAR,
		C.VK_SAMPLER_ADDRESS_MODE_REPEAT, 16.0)
	if err != nil {
		vkImage.Destroy(device)
		return nil, fmt.Errorf("failed to create sampler: %w", err)
	}

	return &TextureUploadResult{
		Image:   vkImage,
		Sampler: sampler,
	}, nil
}

// ExecuteSingleTimeCommands allocates a one-time command buffer, runs fn, then submits and waits
func ExecuteSingleTimeCommands(device *Device, fn func(cmdBuffer C.VkCommandBuffer)) error {
	allocInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandPool:        device.CommandPool,
		commandBufferCount: 1,
	}

	var cmdBuffer C.VkCommandBuffer
	result := C.vkAllocateCommandBuffers(device.Device, &allocInfo, &cmdBuffer)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to allocate command buffer: %d", result)
	}

	beginInfo := C.VkCommandBufferBeginInfo{
		sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		flags: C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	C.vkBeginCommandBuffer(cmdBuffer, &beginInfo)

	fn(cmdBuffer)

	C.vkEndCommandBuffer(cmdBuffer)

	submitInfo := C.VkSubmitInfo{
		sType:              C.VK_STRUCTURE_TYPE_SUBMIT_INFO,
		commandBufferCount: 1,
		pCommandBuffers:    &cmdBuffer,
	}

	C.vkQueueSubmit(device.GraphicsQueue, 1, &submitInfo, nil)
	C.vkQueueWaitIdle(device.GraphicsQueue)

	C.vkFreeCommandBuffers(device.Device, device.CommandPool, 1, &cmdBuffer)

	return nil
}

// DestroyTextureUpload cleans up a texture upload result
func (r *TextureUploadResult) Destroy(device *Device) {
	if r.Sampler != nil {
		DestroySampler(device, r.Sampler)
	}
	if r.Image != nil {
		r.Image.Destroy(device)
	}
}
