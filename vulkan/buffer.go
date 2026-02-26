package vulkan

/*
#include <vulkan/vulkan.h>
#include <string.h>
#include <stdbool.h>

bool hasStencilComponent(VkFormat format) {
    return format == VK_FORMAT_D32_SFLOAT_S8_UINT || format == VK_FORMAT_D24_UNORM_S8_UINT;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Buffer struct {
	Handle     C.VkBuffer
	Memory     C.VkDeviceMemory
	Size       uint64
	MappedData unsafe.Pointer
}

type Image struct {
	Handle    C.VkImage
	Memory    C.VkDeviceMemory
	View      C.VkImageView
	Format    C.VkFormat
	Width     uint32
	Height    uint32
	MipLevels uint32
	Layout    C.VkImageLayout
}

func CreateBuffer(device *Device, size uint64, usage C.VkBufferUsageFlags, properties C.VkMemoryPropertyFlags) (*Buffer, error) {
	bufferInfo := C.VkBufferCreateInfo{
		sType:       C.VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		size:        C.VkDeviceSize(size),
		usage:       usage,
		sharingMode: C.VK_SHARING_MODE_EXCLUSIVE,
	}

	buffer := &Buffer{Size: size}

	result := C.vkCreateBuffer(device.Device, &bufferInfo, nil, &buffer.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create buffer: %d", result)
	}

	var memRequirements C.VkMemoryRequirements
	C.vkGetBufferMemoryRequirements(device.Device, buffer.Handle, &memRequirements)

	memType, err := device.FindMemoryType(uint32(memRequirements.memoryTypeBits), properties)
	if err != nil {
		return nil, err
	}

	allocInfo := C.VkMemoryAllocateInfo{
		sType:           C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		allocationSize:  memRequirements.size,
		memoryTypeIndex: C.uint32_t(memType),
	}

	result = C.vkAllocateMemory(device.Device, &allocInfo, nil, &buffer.Memory)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to allocate buffer memory: %d", result)
	}

	result = C.vkBindBufferMemory(device.Device, buffer.Handle, buffer.Memory, 0)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to bind buffer memory: %d", result)
	}

	return buffer, nil
}

func (b *Buffer) Map(device *Device) error {
	result := C.vkMapMemory(device.Device, b.Memory, 0, C.VkDeviceSize(b.Size), 0, &b.MappedData)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to map buffer memory: %d", result)
	}
	return nil
}

func (b *Buffer) Unmap(device *Device) {
	if b.MappedData != nil {
		C.vkUnmapMemory(device.Device, b.Memory)
		b.MappedData = nil
	}
}

func (b *Buffer) CopyData(data unsafe.Pointer, size uint64) {
	if b.MappedData != nil {
		C.memcpy(b.MappedData, data, C.size_t(size))
	}
}

func (b *Buffer) Destroy(device *Device) {
	b.Unmap(device)
	if b.Handle != nil {
		C.vkDestroyBuffer(device.Device, b.Handle, nil)
	}
	if b.Memory != nil {
		C.vkFreeMemory(device.Device, b.Memory, nil)
	}
}

func CopyBuffer(device *Device, srcBuffer, dstBuffer C.VkBuffer, size uint64, commandPool C.VkCommandPool, queue C.VkQueue) error {
	allocInfo := C.VkCommandBufferAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		level:              C.VK_COMMAND_BUFFER_LEVEL_PRIMARY,
		commandPool:        commandPool,
		commandBufferCount: 1,
	}

	var commandBuffer C.VkCommandBuffer
	C.vkAllocateCommandBuffers(device.Device, &allocInfo, &commandBuffer)

	beginInfo := C.VkCommandBufferBeginInfo{
		sType: C.VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		flags: C.VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
	}

	C.vkBeginCommandBuffer(commandBuffer, &beginInfo)

	copyRegion := C.VkBufferCopy{
		size: C.VkDeviceSize(size),
	}

	C.vkCmdCopyBuffer(commandBuffer, srcBuffer, dstBuffer, 1, &copyRegion)

	C.vkEndCommandBuffer(commandBuffer)

	submitInfo := C.VkSubmitInfo{
		sType:              C.VK_STRUCTURE_TYPE_SUBMIT_INFO,
		commandBufferCount: 1,
		pCommandBuffers:    &commandBuffer,
	}

	C.vkQueueSubmit(queue, 1, &submitInfo, nil)
	C.vkQueueWaitIdle(queue)

	C.vkFreeCommandBuffers(device.Device, commandPool, 1, &commandBuffer)

	return nil
}

func CreateImage(device *Device, width, height uint32, format C.VkFormat, tiling C.VkImageTiling, usage C.VkImageUsageFlags, properties C.VkMemoryPropertyFlags, mipLevels uint32) (*Image, error) {
	imageInfo := C.VkImageCreateInfo{
		sType:     C.VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
		imageType: C.VK_IMAGE_TYPE_2D,
		extent: C.VkExtent3D{
			width:  C.uint32_t(width),
			height: C.uint32_t(height),
			depth:  1,
		},
		mipLevels:     C.uint32_t(mipLevels),
		arrayLayers:   1,
		format:        format,
		tiling:        tiling,
		initialLayout: C.VK_IMAGE_LAYOUT_UNDEFINED,
		usage:         usage,
		samples:       C.VK_SAMPLE_COUNT_1_BIT,
		sharingMode:   C.VK_SHARING_MODE_EXCLUSIVE,
	}

	img := &Image{
		Width:     width,
		Height:    height,
		Format:    format,
		MipLevels: mipLevels,
		Layout:    C.VK_IMAGE_LAYOUT_UNDEFINED,
	}

	result := C.vkCreateImage(device.Device, &imageInfo, nil, &img.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create image: %d", result)
	}

	var memRequirements C.VkMemoryRequirements
	C.vkGetImageMemoryRequirements(device.Device, img.Handle, &memRequirements)

	memType, err := device.FindMemoryType(uint32(memRequirements.memoryTypeBits), properties)
	if err != nil {
		return nil, err
	}

	allocInfo := C.VkMemoryAllocateInfo{
		sType:           C.VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		allocationSize:  memRequirements.size,
		memoryTypeIndex: C.uint32_t(memType),
	}

	result = C.vkAllocateMemory(device.Device, &allocInfo, nil, &img.Memory)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to allocate image memory: %d", result)
	}

	result = C.vkBindImageMemory(device.Device, img.Handle, img.Memory, 0)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to bind image memory: %d", result)
	}

	return img, nil
}

func CreateImageView(device *Device, image C.VkImage, format C.VkFormat, aspectFlags C.VkImageAspectFlags, mipLevels uint32) (C.VkImageView, error) {
	viewInfo := C.VkImageViewCreateInfo{
		sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
		image:    image,
		viewType: C.VK_IMAGE_VIEW_TYPE_2D,
		format:   format,
		subresourceRange: C.VkImageSubresourceRange{
			aspectMask:     aspectFlags,
			baseMipLevel:   0,
			levelCount:     C.uint32_t(mipLevels),
			baseArrayLayer: 0,
			layerCount:     1,
		},
	}

	var imageView C.VkImageView
	result := C.vkCreateImageView(device.Device, &viewInfo, nil, &imageView)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create image view: %d", result)
	}

	return imageView, nil
}

func (img *Image) CreateView(device *Device, aspectFlags C.VkImageAspectFlags) error {
	view, err := CreateImageView(device, img.Handle, img.Format, aspectFlags, img.MipLevels)
	if err != nil {
		return err
	}
	img.View = view
	return nil
}

func (img *Image) Destroy(device *Device) {
	if img.View != nil {
		C.vkDestroyImageView(device.Device, img.View, nil)
	}
	if img.Handle != nil {
		C.vkDestroyImage(device.Device, img.Handle, nil)
	}
	if img.Memory != nil {
		C.vkFreeMemory(device.Device, img.Memory, nil)
	}
}

func FindDepthFormat(device *Device) C.VkFormat {
	candidates := []C.VkFormat{
		C.VK_FORMAT_D32_SFLOAT,
		C.VK_FORMAT_D32_SFLOAT_S8_UINT,
		C.VK_FORMAT_D24_UNORM_S8_UINT,
	}

	for _, format := range candidates {
		var props C.VkFormatProperties
		C.vkGetPhysicalDeviceFormatProperties(device.PhysicalDevice, format, &props)

		if props.optimalTilingFeatures&C.VK_FORMAT_FEATURE_DEPTH_STENCIL_ATTACHMENT_BIT != 0 {
			return format
		}
	}

	return C.VK_FORMAT_UNDEFINED
}

func CreateDepthBuffer(device *Device, width, height uint32) (*Image, error) {
	format := FindDepthFormat(device)
	if format == C.VK_FORMAT_UNDEFINED {
		return nil, fmt.Errorf("failed to find supported depth format")
	}

	image, err := CreateImage(device, width, height, format,
		C.VK_IMAGE_TILING_OPTIMAL,
		C.VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT,
		C.VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT, 1)
	if err != nil {
		return nil, err
	}

	aspectFlags := C.VK_IMAGE_ASPECT_DEPTH_BIT
	if C.hasStencilComponent(format) {
		aspectFlags |= C.VK_IMAGE_ASPECT_STENCIL_BIT
	}

	err = image.CreateView(device, C.VkImageAspectFlags(aspectFlags))
	if err != nil {
		image.Destroy(device)
		return nil, err
	}

	return image, nil
}
