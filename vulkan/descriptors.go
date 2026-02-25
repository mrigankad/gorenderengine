package vulkan

/*
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
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
	// Allocate bindings in C memory
	bindingsSize := C.size_t(len(bindings)) * C.size_t(unsafe.Sizeof(C.VkDescriptorSetLayoutBinding{}))
	cBindings := C.malloc(bindingsSize)
	defer C.free(cBindings)
	C.memcpy(cBindings, unsafe.Pointer(&bindings[0]), bindingsSize)
	
	// Allocate layout info in C memory
	layoutInfo := (*C.VkDescriptorSetLayoutCreateInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkDescriptorSetLayoutCreateInfo{}))))
	defer C.free(unsafe.Pointer(layoutInfo))
	C.memset(unsafe.Pointer(layoutInfo), 0, C.size_t(unsafe.Sizeof(C.VkDescriptorSetLayoutCreateInfo{})))
	layoutInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO
	layoutInfo.bindingCount = C.uint32_t(len(bindings))
	layoutInfo.pBindings = (*C.VkDescriptorSetLayoutBinding)(cBindings)

	var layout C.VkDescriptorSetLayout
	result := C.vkCreateDescriptorSetLayout(device.Device, layoutInfo, nil, &layout)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create descriptor set layout: %d", result)
	}

	return layout, nil
}

func CreateDescriptorPool(device *Device, poolSizes []C.VkDescriptorPoolSize, maxSets uint32) (*DescriptorPool, error) {
	// Allocate pool sizes in C memory
	poolSizesSize := C.size_t(len(poolSizes)) * C.size_t(unsafe.Sizeof(C.VkDescriptorPoolSize{}))
	cPoolSizes := C.malloc(poolSizesSize)
	defer C.free(cPoolSizes)
	C.memcpy(cPoolSizes, unsafe.Pointer(&poolSizes[0]), poolSizesSize)
	
	// Allocate pool info in C memory
	poolInfo := (*C.VkDescriptorPoolCreateInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkDescriptorPoolCreateInfo{}))))
	defer C.free(unsafe.Pointer(poolInfo))
	C.memset(unsafe.Pointer(poolInfo), 0, C.size_t(unsafe.Sizeof(C.VkDescriptorPoolCreateInfo{})))
	poolInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO
	poolInfo.poolSizeCount = C.uint32_t(len(poolSizes))
	poolInfo.pPoolSizes = (*C.VkDescriptorPoolSize)(cPoolSizes)
	poolInfo.maxSets = C.uint32_t(maxSets)

	pool := &DescriptorPool{}
	result := C.vkCreateDescriptorPool(device.Device, poolInfo, nil, &pool.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create descriptor pool: %d", result)
	}

	return pool, nil
}

func (p *DescriptorPool) Destroy(device *Device) {
	C.vkDestroyDescriptorPool(device.Device, p.Handle, nil)
}

func (p *DescriptorPool) AllocateDescriptorSets(device *Device, layouts []C.VkDescriptorSetLayout) ([]DescriptorSet, error) {
	// Allocate layouts in C memory
	layoutsSize := C.size_t(len(layouts)) * C.size_t(unsafe.Sizeof(C.VkDescriptorSetLayout(nil)))
	cLayouts := C.malloc(layoutsSize)
	defer C.free(cLayouts)
	C.memcpy(cLayouts, unsafe.Pointer(&layouts[0]), layoutsSize)
	
	// Allocate alloc info in C memory
	allocInfo := (*C.VkDescriptorSetAllocateInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkDescriptorSetAllocateInfo{}))))
	defer C.free(unsafe.Pointer(allocInfo))
	C.memset(unsafe.Pointer(allocInfo), 0, C.size_t(unsafe.Sizeof(C.VkDescriptorSetAllocateInfo{})))
	allocInfo.sType = C.VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO
	allocInfo.descriptorPool = p.Handle
	allocInfo.descriptorSetCount = C.uint32_t(len(layouts))
	allocInfo.pSetLayouts = (*C.VkDescriptorSetLayout)(cLayouts)

	sets := make([]DescriptorSet, len(layouts))
	handles := make([]C.VkDescriptorSet, len(layouts))

	result := C.vkAllocateDescriptorSets(device.Device, allocInfo, &handles[0])
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to allocate descriptor sets: %d", result)
	}

	for i := range sets {
		sets[i].Handle = handles[i]
	}

	return sets, nil
}

func UpdateDescriptorSetBuffer(device *Device, set C.VkDescriptorSet, binding uint32, buffer C.VkBuffer, offset, range_ uint64) {
	// Allocate buffer info in C memory
	bufferInfo := (*C.VkDescriptorBufferInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkDescriptorBufferInfo{}))))
	defer C.free(unsafe.Pointer(bufferInfo))
	bufferInfo.buffer = buffer
	bufferInfo.offset = C.VkDeviceSize(offset)
	bufferInfo._range = C.VkDeviceSize(range_)

	// Allocate write info in C memory
	descriptorWrite := (*C.VkWriteDescriptorSet)(C.malloc(C.size_t(unsafe.Sizeof(C.VkWriteDescriptorSet{}))))
	defer C.free(unsafe.Pointer(descriptorWrite))
	C.memset(unsafe.Pointer(descriptorWrite), 0, C.size_t(unsafe.Sizeof(C.VkWriteDescriptorSet{})))
	descriptorWrite.sType = C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET
	descriptorWrite.dstSet = set
	descriptorWrite.dstBinding = C.uint32_t(binding)
	descriptorWrite.dstArrayElement = 0
	descriptorWrite.descriptorType = C.VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
	descriptorWrite.descriptorCount = 1
	descriptorWrite.pBufferInfo = bufferInfo

	C.vkUpdateDescriptorSets(device.Device, 1, descriptorWrite, 0, nil)
}

func UpdateDescriptorSetImage(device *Device, set C.VkDescriptorSet, binding uint32, imageView C.VkImageView, sampler C.VkSampler) {
	// Allocate image info in C memory
	imageInfo := (*C.VkDescriptorImageInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkDescriptorImageInfo{}))))
	defer C.free(unsafe.Pointer(imageInfo))
	imageInfo.sampler = sampler
	imageInfo.imageView = imageView
	imageInfo.imageLayout = C.VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL

	// Allocate write info in C memory
	descriptorWrite := (*C.VkWriteDescriptorSet)(C.malloc(C.size_t(unsafe.Sizeof(C.VkWriteDescriptorSet{}))))
	defer C.free(unsafe.Pointer(descriptorWrite))
	C.memset(unsafe.Pointer(descriptorWrite), 0, C.size_t(unsafe.Sizeof(C.VkWriteDescriptorSet{})))
	descriptorWrite.sType = C.VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET
	descriptorWrite.dstSet = set
	descriptorWrite.dstBinding = C.uint32_t(binding)
	descriptorWrite.dstArrayElement = 0
	descriptorWrite.descriptorType = C.VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER
	descriptorWrite.descriptorCount = 1
	descriptorWrite.pImageInfo = imageInfo

	C.vkUpdateDescriptorSets(device.Device, 1, descriptorWrite, 0, nil)
}

func CreateSampler(device *Device, magFilter, minFilter C.VkFilter, addressMode C.VkSamplerAddressMode, anisotropy float32) (C.VkSampler, error) {
	// Allocate sampler info in C memory
	samplerInfo := (*C.VkSamplerCreateInfo)(C.malloc(C.size_t(unsafe.Sizeof(C.VkSamplerCreateInfo{}))))
	defer C.free(unsafe.Pointer(samplerInfo))
	C.memset(unsafe.Pointer(samplerInfo), 0, C.size_t(unsafe.Sizeof(C.VkSamplerCreateInfo{})))
	samplerInfo.sType = C.VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO
	samplerInfo.magFilter = magFilter
	samplerInfo.minFilter = minFilter
	samplerInfo.addressModeU = addressMode
	samplerInfo.addressModeV = addressMode
	samplerInfo.addressModeW = addressMode
	samplerInfo.anisotropyEnable = C.VK_TRUE
	samplerInfo.maxAnisotropy = C.float(anisotropy)
	samplerInfo.borderColor = C.VK_BORDER_COLOR_INT_OPAQUE_BLACK
	samplerInfo.unnormalizedCoordinates = C.VK_FALSE
	samplerInfo.compareEnable = C.VK_FALSE
	samplerInfo.compareOp = C.VK_COMPARE_OP_ALWAYS
	samplerInfo.mipmapMode = C.VK_SAMPLER_MIPMAP_MODE_LINEAR

	var sampler C.VkSampler
	result := C.vkCreateSampler(device.Device, samplerInfo, nil, &sampler)
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
