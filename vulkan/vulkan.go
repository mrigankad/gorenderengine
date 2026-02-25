// Package vulkan provides Vulkan graphics API bindings for the render engine.
// This package uses CGO to interface with the Vulkan C API.
package vulkan

// #cgo windows LDFLAGS: -lvulkan-1
// #cgo linux LDFLAGS: -lvulkan
// #cgo darwin LDFLAGS: -framework MoltenVK
// #include <vulkan/vulkan.h>
import "C"

// Version constants
const (
	VulkanVersion10 = C.VK_API_VERSION_1_0
	VulkanVersion11 = C.VK_API_VERSION_1_1
	VulkanVersion12 = C.VK_API_VERSION_1_2
)

// Common Vulkan constants used throughout the package
const (
	MaxPhysicalDeviceNameSize = C.VK_MAX_PHYSICAL_DEVICE_NAME_SIZE
	UuidSize                 = C.VK_UUID_SIZE
	LuidSize                 = C.VK_LUID_SIZE
	MaxExtensionNameSize     = C.VK_MAX_EXTENSION_NAME_SIZE
	MaxDescriptionSize       = C.VK_MAX_DESCRIPTION_SIZE
	MaxMemoryTypes           = C.VK_MAX_MEMORY_TYPES
	MaxMemoryHeaps           = C.VK_MAX_MEMORY_HEAPS
)
