package vulkan

/*
#cgo windows LDFLAGS: -lvulkan-1
#include <vulkan/vulkan.h>
#include <stdlib.h>
#include <string.h>
#include <stdbool.h>
#include <stdio.h>

void* vulkan_malloc(size_t size) {
    return malloc(size);
}

const char** getRequiredExtensions(uint32_t* count, const char** glfwExtensions, uint32_t glfwCount, bool enableValidation) {
    uint32_t additionalCount = enableValidation ? 1 : 0;
    *count = glfwCount + additionalCount;
    const char** extensions = (const char**)malloc(*count * sizeof(const char*));

    for (uint32_t i = 0; i < glfwCount; i++) {
        extensions[i] = glfwExtensions[i];
    }

    if (enableValidation) {
        extensions[glfwCount] = VK_EXT_DEBUG_UTILS_EXTENSION_NAME;
    }

    return extensions;
}

VKAPI_ATTR VkBool32 VKAPI_CALL debugCallback(
    VkDebugUtilsMessageSeverityFlagBitsEXT messageSeverity,
    VkDebugUtilsMessageTypeFlagsEXT messageType,
    const VkDebugUtilsMessengerCallbackDataEXT* pCallbackData,
    void* pUserData) {

    const char* severity = "INFO";
    if (messageSeverity >= VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT) {
        severity = "ERROR";
    } else if (messageSeverity >= VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT) {
        severity = "WARNING";
    }

    fprintf(stderr, "[VULKAN %s] %s\n", severity, pCallbackData->pMessage);
    return VK_FALSE;
}

VkResult CreateDebugUtilsMessengerEXT(VkInstance instance, const VkDebugUtilsMessengerCreateInfoEXT* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkDebugUtilsMessengerEXT* pDebugMessenger) {
    PFN_vkCreateDebugUtilsMessengerEXT func = (PFN_vkCreateDebugUtilsMessengerEXT)vkGetInstanceProcAddr(instance, "vkCreateDebugUtilsMessengerEXT");
    if (func != NULL) {
        return func(instance, pCreateInfo, pAllocator, pDebugMessenger);
    } else {
        return VK_ERROR_EXTENSION_NOT_PRESENT;
    }
}

void DestroyDebugUtilsMessengerEXT(VkInstance instance, VkDebugUtilsMessengerEXT debugMessenger, const VkAllocationCallbacks* pAllocator) {
    PFN_vkDestroyDebugUtilsMessengerEXT func = (PFN_vkDestroyDebugUtilsMessengerEXT)vkGetInstanceProcAddr(instance, "vkDestroyDebugUtilsMessengerEXT");
    if (func != NULL) {
        func(instance, debugMessenger, pAllocator);
    }
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Instance struct {
	Handle           C.VkInstance
	DebugMessenger   C.VkDebugUtilsMessengerEXT
	EnableValidation bool
}

type InstanceConfig struct {
	AppName            string
	EngineName         string
	AppVersion         uint32
	EngineVersion      uint32
	EnableValidation   bool
	RequiredExtensions []string
}

func DefaultInstanceConfig() InstanceConfig {
	return InstanceConfig{
		AppName:          "Render Engine App",
		EngineName:       "Render Engine",
		AppVersion:       VK_MAKE_VERSION(1, 0, 0),
		EngineVersion:    VK_MAKE_VERSION(1, 0, 0),
		EnableValidation: true,
	}
}

func NewInstance(config InstanceConfig) (*Instance, error) {
	// Application info
	appName := C.CString(config.AppName)
	defer C.free(unsafe.Pointer(appName))

	engineName := C.CString(config.EngineName)
	defer C.free(unsafe.Pointer(engineName))

	appInfo := (*C.VkApplicationInfo)(C.vulkan_malloc(C.size_t(unsafe.Sizeof(C.VkApplicationInfo{}))))
	defer C.free(unsafe.Pointer(appInfo))
	*appInfo = C.VkApplicationInfo{
		sType:              C.VK_STRUCTURE_TYPE_APPLICATION_INFO,
		pApplicationName:   appName,
		applicationVersion: C.uint32_t(config.AppVersion),
		pEngineName:        engineName,
		engineVersion:      C.uint32_t(config.EngineVersion),
		apiVersion:         C.VK_API_VERSION_1_2,
	}

	// Extensions
	var extensionCount C.uint32_t
	extensions := make([]*C.char, len(config.RequiredExtensions))
	for i, ext := range config.RequiredExtensions {
		extensions[i] = C.CString(ext)
		defer C.free(unsafe.Pointer(extensions[i]))
	}

	extensionsPtr := C.getRequiredExtensions(&extensionCount, &extensions[0], C.uint32_t(len(config.RequiredExtensions)), C.bool(config.EnableValidation))
	defer C.free(unsafe.Pointer(extensionsPtr))

	// Validation layers
	var validationLayersPtr **C.char
	var validationLayerCount C.uint32_t

	if config.EnableValidation {
		validationLayer := C.CString("VK_LAYER_KHRONOS_validation")
		defer C.free(unsafe.Pointer(validationLayer))

		layerArray := (**C.char)(C.vulkan_malloc(C.size_t(unsafe.Sizeof(validationLayer))))
		defer C.free(unsafe.Pointer(layerArray))

		// Copy the pointer to the C array
		ptrSlice := (*[1]*C.char)(unsafe.Pointer(layerArray))[:]
		ptrSlice[0] = validationLayer

		validationLayersPtr = layerArray
		validationLayerCount = 1

		if !checkValidationLayerSupport() {
			return nil, fmt.Errorf("validation layers requested but not available")
		}
	}

	// Create instance
	createInfo := C.VkInstanceCreateInfo{
		sType:                   C.VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
		pApplicationInfo:        appInfo,
		enabledExtensionCount:   extensionCount,
		ppEnabledExtensionNames: extensionsPtr,
		enabledLayerCount:       validationLayerCount,
		ppEnabledLayerNames:     validationLayersPtr,
	}

	// Debug messenger create info
	var debugCreateInfo *C.VkDebugUtilsMessengerCreateInfoEXT
	if config.EnableValidation {
		debugCreateInfo = (*C.VkDebugUtilsMessengerCreateInfoEXT)(C.vulkan_malloc(C.size_t(unsafe.Sizeof(C.VkDebugUtilsMessengerCreateInfoEXT{}))))
		defer C.free(unsafe.Pointer(debugCreateInfo))
		*debugCreateInfo = C.VkDebugUtilsMessengerCreateInfoEXT{
			sType:           C.VK_STRUCTURE_TYPE_DEBUG_UTILS_MESSENGER_CREATE_INFO_EXT,
			messageSeverity: C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_VERBOSE_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_WARNING_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_SEVERITY_ERROR_BIT_EXT,
			messageType:     C.VK_DEBUG_UTILS_MESSAGE_TYPE_GENERAL_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_VALIDATION_BIT_EXT | C.VK_DEBUG_UTILS_MESSAGE_TYPE_PERFORMANCE_BIT_EXT,
			pfnUserCallback: (C.PFN_vkDebugUtilsMessengerCallbackEXT)(C.debugCallback),
		}
		createInfo.pNext = unsafe.Pointer(debugCreateInfo)
	}

	var instance C.VkInstance
	result := C.vkCreateInstance(&createInfo, nil, &instance)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create Vulkan instance: %d", result)
	}

	inst := &Instance{
		Handle:           instance,
		EnableValidation: config.EnableValidation,
	}

	// Setup debug messenger
	if config.EnableValidation {
		result = C.CreateDebugUtilsMessengerEXT(instance, debugCreateInfo, nil, &inst.DebugMessenger)
		if result != C.VK_SUCCESS {
			fmt.Printf("Warning: failed to set up debug messenger: %d\n", result)
		}
	}

	return inst, nil
}

func (i *Instance) Destroy() {
	if i.EnableValidation && i.DebugMessenger != nil {
		C.DestroyDebugUtilsMessengerEXT(i.Handle, i.DebugMessenger, nil)
	}
	C.vkDestroyInstance(i.Handle, nil)
}

func checkValidationLayerSupport() bool {
	var layerCount C.uint32_t
	C.vkEnumerateInstanceLayerProperties(&layerCount, nil)

	availableLayers := make([]C.VkLayerProperties, layerCount)
	C.vkEnumerateInstanceLayerProperties(&layerCount, &availableLayers[0])

	layerName := C.CString("VK_LAYER_KHRONOS_validation")
	defer C.free(unsafe.Pointer(layerName))

	for _, layer := range availableLayers {
		if C.strcmp(&layer.layerName[0], layerName) == 0 {
			return true
		}
	}

	return false
}

func VK_MAKE_VERSION(major, minor, patch uint32) uint32 {
	return (major << 22) | (minor << 12) | patch
}
