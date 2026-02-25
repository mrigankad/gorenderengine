package vulkan

/*
#include <vulkan/vulkan.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type DescriptorPool struct {
	Handle C.VkDescriptorPool
}

type DescriptorSet struct {
	Handle C.VkDescriptorSet
}

func CreateDescriptorSetLayout(device *Device, bindings []C.VkDescriptorSetLayoutBinding) (C.VkDescriptorSetLayout, error) {
	layoutInfo := C.VkDescriptorSetLayoutCreateInfo{
		sType:        C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
		bindingCount: C.uint32_t(len(bindings)),
		pBindings:    &bindings[0],
	}
	
	var layout C.VkDescriptorSetLayout
	result := C.vkCreateDescriptorSetLayout(device.Device, &layoutInfo, nil, &layout)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create descriptor set layout: %d", result)
	}
	
	return layout, nil
}

func CreateDescriptorPool(device *Device, poolSizes []C.VkDescriptorPoolSize, maxSets uint32) (*DescriptorPool, error) {
	poolInfo := C.VkDescriptorPoolCreateInfo{
		sType:         C.VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
		poolSizeCount: C.uint32_t(len(poolSizes)),
		pPoolSizes:    &poolSizes[0],
		maxSets:       C.uint32_t(maxSets),
	}
	
	pool := &DescriptorPool{}
	result := C.vkCreateDescriptorPool(device.Device, &poolInfo, nil, &pool.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create descriptor pool: %d", result)
	}
	
	return pool, nil
}

func (p *DescriptorPool) Destroy(device *Device) {
	C.vkDestroyDescriptorPool(device.Device, p.Handle, nil)
}

func (p *DescriptorPool) AllocateDescriptorSets(device *Device, layouts []C.VkDescriptorSetLayout) ([]DescriptorSet, error) {
	allocInfo := C.VkDescriptorSetAllocateInfo{
		sType:              C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
		descriptorPool:     p.Handle,
		descriptorSetCount: C.uint32_t(len(layouts)),
		pSetLayouts:        &layouts[0],
	}
	
	sets := make([]DescriptorSet, len(layouts))
	handles := make([]C.VkDescriptorSet, len(layouts))
	
	result := C.vkAllocateDescriptorSets(device.Device, &allocInfo, &handles[0])
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to allocate descriptor sets: %d", result)
	}
	
	for i := range sets {
		sets[i].Handle = handles[i]
	}
	
	return sets, nil
}

func UpdateDescriptorSetBuffer(device *Device, set C.VkDescriptorSet, binding uint32, buffer C.VkBuffer, offset, range_ uint64) {
	bufferInfo := C.VkDescriptorBufferInfo{
		buffer: buffer,
		offset: C.VkDeviceSize(offset),
		range:  C.VkDeviceSize(range_),
	}
	
	descriptorWrite := C.VkWriteDescriptorSet{
		sType:           C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          set,
		dstBinding:      C.uint32_t(binding),
		dstArrayElement: 0,
		descriptorType:  C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
		descriptorCount: 1,
		pBufferInfo:     &bufferInfo,
	}
	
	C.vkUpdateDescriptorSets(device.Device, 1, &descriptorWrite, 0, nil)
}

func UpdateDescriptorSetImage(device *Device, set C.VkDescriptorSet, binding uint32, imageView C.VkImageView, sampler C.VkSampler) {
	imageInfo := C.VkDescriptorImageInfo{
		sampler:     sampler,
		imageView:   imageView,
		imageLayout: C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
	}
	
	descriptorWrite := C.VkWriteDescriptorSet{
		sType:           C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
		dstSet:          set,
		dstBinding:      C.uint32_t(binding),
		dstArrayElement: 0,
		descriptorType:  C.VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
		descriptorCount: 1,
		pImageInfo:      &imageInfo,
	}
	
	C.vkUpdateDescriptorSets(device.Device, 1, &descriptorWrite, 0, nil)
}

func CreateSampler(device *Device, magFilter, minFilter C.VkFilter, addressMode C.VkSamplerAddressMode, anisotropy float32) (C.VkSampler, error) {
	samplerInfo := C.VkSamplerCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
		magFilter:               magFilter,
		minFilter:               minFilter,
		addressModeU:            addressMode,
		addressModeV:            addressMode,
		addressModeW:            addressMode,
		anisotropyEnable:        C.VK_TRUE,
		maxAnisotropy:           C.float(anisotropy),
		borderColor:             C.VK_BORDER_COLOR_INT_OPAQUE_BLACK,
		unnormalizedCoordinates: C.VK_FALSE,
		compareEnable:           C.VK_FALSE,
		compareOp:               C.VK_COMPARE_OP_ALWAYS,
		mipmapMode:              C.VK_SAMPLER_MIPMAP_MODE_LINEAR,
	}
	
	var sampler C.VkSampler
	result := C.vkCreateSampler(device.Device, &samplerInfo, nil, &sampler)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create texture sampler: %d", result)
	}
	
	return sampler, nil
}

func DestroySampler(device *Device, sampler C.VkSampler) {
	C.vkDestroySampler(device.Device, sampler, nil)
}

// Uniform buffer object layouts
func GetVertexLayoutBinding(binding uint32, stride uint32) C.VkVertexInputBindingDescription {
	return C.VkVertexInputBindingDescription{
		binding:   C.uint32_t(binding),
		stride:    C.uint32_t(stride),
		inputRate: C.VK_VERTEX_INPUT_RATE_VERTEX,
	}
}

func GetVertexAttributeLocation(location, binding uint32, format C.VkFormat, offset uint32) C.VkVertexInputAttributeDescription {
	return C.VkVertexInputAttributeDescription{
		location: C.uint32_t(location),
		binding:  C.uint32_t(binding),
		format:   format,
		offset:   C.uint32_t(offset),
	}
}

// Helper to create uniform buffer binding
func UniformBufferBinding(binding uint32, stageFlags C.VkShaderStageFlags) C.VkDescriptorSetLayoutBinding {
	return C.VkDescriptorSetLayoutBinding{
		binding:            C.uint32_t(binding),
		descriptorType:     C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
		descriptorCount:    1,
		stageFlags:         stageFlags,
		pImmutableSamplers: nil,
	}
}

// Helper to create combined image sampler binding
func CombinedImageSamplerBinding(binding uint32, stageFlags C.VkShaderStageFlags) C.VkDescriptorSetLayoutBinding {
	return C.VkDescriptorSetLayoutBinding{
		binding:            C.uint32_t(binding),
		descriptorType:     C.VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
		descriptorCount:    1,
		stageFlags:         stageFlags,
		pImmutableSamplers: nil,
	}
}
