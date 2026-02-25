package vulkan

/*
#include <vulkan/vulkan.h>
#include <stdbool.h>
#include <string.h>

typedef struct {
    VkPhysicalDevice device;
    VkPhysicalDeviceProperties properties;
    VkPhysicalDeviceFeatures features;
    uint32_t graphicsFamily;
    uint32_t presentFamily;
    bool hasGraphicsFamily;
    bool hasPresentFamily;
    uint32_t score;
} DeviceInfo;

void findQueueFamilies(VkPhysicalDevice device, VkSurfaceKHR surface, DeviceInfo* info) {
    uint32_t queueFamilyCount = 0;
    vkGetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, NULL);
    
    VkQueueFamilyProperties* queueFamilies = (VkQueueFamilyProperties*)malloc(queueFamilyCount * sizeof(VkQueueFamilyProperties));
    vkGetPhysicalDeviceQueueFamilyProperties(device, &queueFamilyCount, queueFamilies);
    
    for (uint32_t i = 0; i < queueFamilyCount; i++) {
        if (queueFamilies[i].queueFlags & VK_QUEUE_GRAPHICS_BIT) {
            info->graphicsFamily = i;
            info->hasGraphicsFamily = true;
        }
        
        VkBool32 presentSupport = false;
        vkGetPhysicalDeviceSurfaceSupportKHR(device, i, surface, &presentSupport);
        if (presentSupport) {
            info->presentFamily = i;
            info->hasPresentFamily = true;
        }
        
        if (info->hasGraphicsFamily && info->hasPresentFamily) {
            break;
        }
    }
    
    free(queueFamilies);
}

uint32_t rateDevice(VkPhysicalDevice device, VkSurfaceKHR surface) {
    DeviceInfo info = {0};
    info.device = device;
    vkGetPhysicalDeviceProperties(device, &info.properties);
    vkGetPhysicalDeviceFeatures(device, &info.features);
    findQueueFamilies(device, surface, &info);
    
    if (!info.hasGraphicsFamily || !info.hasPresentFamily) {
        return 0;
    }
    
    if (!info.features.samplerAnisotropy) {
        return 0;
    }
    
    uint32_t score = 0;
    
    // Discrete GPUs have a significant advantage
    if (info.properties.deviceType == VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU) {
        score += 1000;
    }
    
    // Maximum possible size of textures affects graphics quality
    score += info.properties.limits.maxImageDimension2D;
    
    return score;
}

bool checkDeviceExtensionSupport(VkPhysicalDevice device) {
    const char* deviceExtensions[] = {VK_KHR_SWAPCHAIN_EXTENSION_NAME};
    uint32_t extensionCount;
    vkEnumerateDeviceExtensionProperties(device, NULL, &extensionCount, NULL);
    
    VkExtensionProperties* availableExtensions = (VkExtensionProperties*)malloc(extensionCount * sizeof(VkExtensionProperties));
    vkEnumerateDeviceExtensionProperties(device, NULL, &extensionCount, availableExtensions);
    
    for (size_t i = 0; i < sizeof(deviceExtensions) / sizeof(deviceExtensions[0]); i++) {
        bool found = false;
        for (uint32_t j = 0; j < extensionCount; j++) {
            if (strcmp(deviceExtensions[i], availableExtensions[j].extensionName) == 0) {
                found = true;
                break;
            }
        }
        if (!found) {
            free(availableExtensions);
            return false;
        }
    }
    
    free(availableExtensions);
    return true;
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Device struct {
	PhysicalDevice C.VkPhysicalDevice
	Device         C.VkDevice
	GraphicsQueue  C.VkQueue
	PresentQueue   C.VkQueue
	CommandPool    C.VkCommandPool
	
	GraphicsFamily uint32
	PresentFamily  uint32
	Properties     C.VkPhysicalDeviceProperties
	Features       C.VkPhysicalDeviceFeatures
	Limits         C.VkPhysicalDeviceLimits
	MemoryProps    C.VkPhysicalDeviceMemoryProperties
}

func PickPhysicalDevice(instance *Instance, surface C.VkSurfaceKHR) (*Device, error) {
	var deviceCount C.uint32_t
	result := C.vkEnumeratePhysicalDevices(instance.Handle, &deviceCount, nil)
	if result != C.VK_SUCCESS || deviceCount == 0 {
		return nil, fmt.Errorf("failed to find GPUs with Vulkan support")
	}
	
	devices := make([]C.VkPhysicalDevice, deviceCount)
	C.vkEnumeratePhysicalDevices(instance.Handle, &deviceCount, &devices[0])
	
	var bestDevice C.VkPhysicalDevice
	var bestScore C.uint32_t
	
	for _, device := range devices {
		if !C.checkDeviceExtensionSupport(device) {
			continue
		}
		
		score := C.rateDevice(device, surface)
		if score > bestScore {
			bestScore = score
			bestDevice = device
		}
	}
	
	if bestDevice == nil {
		return nil, fmt.Errorf("failed to find a suitable GPU")
	}
	
	d := &Device{
		PhysicalDevice: bestDevice,
	}
	
	C.vkGetPhysicalDeviceProperties(bestDevice, &d.Properties)
	C.vkGetPhysicalDeviceFeatures(bestDevice, &d.Features)
	C.vkGetPhysicalDeviceMemoryProperties(bestDevice, &d.MemoryProps)
	d.Limits = d.Properties.limits
	
	return d, nil
}

func (d *Device) CreateLogicalDevice(surface C.VkSurfaceKHR) error {
	// Find queue families
	var deviceInfo C.DeviceInfo
	C.findQueueFamilies(d.PhysicalDevice, surface, &deviceInfo)
	d.GraphicsFamily = uint32(deviceInfo.graphicsFamily)
	d.PresentFamily = uint32(deviceInfo.presentFamily)
	
	// Create queues
	queueFamilies := []uint32{d.GraphicsFamily}
	if d.GraphicsFamily != d.PresentFamily {
		queueFamilies = append(queueFamilies, d.PresentFamily)
	}
	
	queueCreateInfos := make([]C.VkDeviceQueueCreateInfo, len(queueFamilies))
	queuePriority := C.float(1.0)
	
	for i, family := range queueFamilies {
		queueCreateInfos[i] = C.VkDeviceQueueCreateInfo{
			sType:            C.VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
			queueFamilyIndex: C.uint32_t(family),
			queueCount:       1,
			pQueuePriorities: &queuePriority,
		}
	}
	
	// Device features
	features := C.VkPhysicalDeviceFeatures{
		samplerAnisotropy: C.VK_TRUE,
		fillModeNonSolid:  C.VK_TRUE,
	}
	
	// Device extensions
	extensionName := C.CString(C.VK_KHR_SWAPCHAIN_EXTENSION_NAME)
	defer C.free(unsafe.Pointer(extensionName))
	
	// Create device
	createInfo := C.VkDeviceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
		queueCreateInfoCount:    C.uint32_t(len(queueCreateInfos)),
		pQueueCreateInfos:       &queueCreateInfos[0],
		pEnabledFeatures:        &features,
		enabledExtensionCount:   1,
		ppEnabledExtensionNames: &extensionName,
	}
	
	result := C.vkCreateDevice(d.PhysicalDevice, &createInfo, nil, &d.Device)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to create logical device: %d", result)
	}
	
	// Get queues
	C.vkGetDeviceQueue(d.Device, C.uint32_t(d.GraphicsFamily), 0, &d.GraphicsQueue)
	C.vkGetDeviceQueue(d.Device, C.uint32_t(d.PresentFamily), 0, &d.PresentQueue)
	
	// Create command pool
	poolInfo := C.VkCommandPoolCreateInfo{
		sType:            C.VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		queueFamilyIndex: C.uint32_t(d.GraphicsFamily),
		flags:            C.VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT,
	}
	
	result = C.vkCreateCommandPool(d.Device, &poolInfo, nil, &d.CommandPool)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to create command pool: %d", result)
	}
	
	return nil
}

func (d *Device) Destroy() {
	if d.CommandPool != nil {
		C.vkDestroyCommandPool(d.Device, d.CommandPool, nil)
	}
	if d.Device != nil {
		C.vkDestroyDevice(d.Device, nil)
	}
}

func (d *Device) WaitIdle() {
	C.vkDeviceWaitIdle(d.Device)
}

func (d *Device) GetGPUName() string {
	name := make([]byte, C.VK_MAX_PHYSICAL_DEVICE_NAME_SIZE)
	for i := 0; i < C.VK_MAX_PHYSICAL_DEVICE_NAME_SIZE; i++ {
		name[i] = byte(d.Properties.deviceName[i])
	}
	
	// Find null terminator
	for i, b := range name {
		if b == 0 {
			return string(name[:i])
		}
	}
	return string(name)
}

func (d *Device) GetDeviceType() string {
	switch d.Properties.deviceType {
	case C.VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU:
		return "Integrated GPU"
	case C.VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU:
		return "Discrete GPU"
	case C.VK_PHYSICAL_DEVICE_TYPE_VIRTUAL_GPU:
		return "Virtual GPU"
	case C.VK_PHYSICAL_DEVICE_TYPE_CPU:
		return "CPU"
	default:
		return "Unknown"
	}
}

func (d *Device) FindMemoryType(typeFilter uint32, properties C.VkMemoryPropertyFlags) (uint32, error) {
	for i := uint32(0); i < uint32(d.MemoryProps.memoryTypeCount); i++ {
		if (typeFilter & (1 << i)) != 0 && (d.MemoryProps.memoryTypes[i].propertyFlags & properties) == properties {
			return i, nil
		}
	}
	return 0, fmt.Errorf("failed to find suitable memory type")
}
