package vulkan

/*
#include <vulkan/vulkan.h>

typedef struct {
    VkSurfaceCapabilitiesKHR capabilities;
    VkSurfaceFormatKHR* formats;
    uint32_t formatCount;
    VkPresentModeKHR* presentModes;
    uint32_t presentModeCount;
} SwapChainSupportDetails;

void querySwapChainSupport(VkPhysicalDevice device, VkSurfaceKHR surface, SwapChainSupportDetails* details) {
    vkGetPhysicalDeviceSurfaceCapabilitiesKHR(device, surface, &details->capabilities);
    
    vkGetPhysicalDeviceSurfaceFormatsKHR(device, surface, &details->formatCount, NULL);
    if (details->formatCount != 0) {
        details->formats = (VkSurfaceFormatKHR*)malloc(details->formatCount * sizeof(VkSurfaceFormatKHR));
        vkGetPhysicalDeviceSurfaceFormatsKHR(device, surface, &details->formatCount, details->formats);
    }
    
    vkGetPhysicalDeviceSurfacePresentModesKHR(device, surface, &details->presentModeCount, NULL);
    if (details->presentModeCount != 0) {
        details->presentModes = (VkPresentModeKHR*)malloc(details->presentModeCount * sizeof(VkPresentModeKHR));
        vkGetPhysicalDeviceSurfacePresentModesKHR(device, surface, &details->presentModeCount, details->presentModes);
    }
}

void freeSwapChainSupportDetails(SwapChainSupportDetails* details) {
    free(details->formats);
    free(details->presentModes);
}

VkSurfaceFormatKHR chooseSwapSurfaceFormat(const VkSurfaceFormatKHR* availableFormats, uint32_t count) {
    for (uint32_t i = 0; i < count; i++) {
        if (availableFormats[i].format == VK_FORMAT_B8G8R8A8_SRGB && 
            availableFormats[i].colorSpace == VK_COLOR_SPACE_SRGB_NONLINEAR_KHR) {
            return availableFormats[i];
        }
    }
    return availableFormats[0];
}

VkPresentModeKHR chooseSwapPresentMode(const VkPresentModeKHR* availablePresentModes, uint32_t count) {
    for (uint32_t i = 0; i < count; i++) {
        if (availablePresentModes[i] == VK_PRESENT_MODE_MAILBOX_KHR) {
            return availablePresentModes[i];
        }
    }
    return VK_PRESENT_MODE_FIFO_KHR;
}

VkExtent2D chooseSwapExtent(const VkSurfaceCapabilitiesKHR* capabilities, uint32_t width, uint32_t height) {
    if (capabilities->currentExtent.width != UINT32_MAX) {
        return capabilities->currentExtent;
    } else {
        VkExtent2D actualExtent = {width, height};
        
        if (actualExtent.width < capabilities->minImageExtent.width) {
            actualExtent.width = capabilities->minImageExtent.width;
        } else if (actualExtent.width > capabilities->maxImageExtent.width) {
            actualExtent.width = capabilities->maxImageExtent.width;
        }
        
        if (actualExtent.height < capabilities->minImageExtent.height) {
            actualExtent.height = capabilities->minImageExtent.height;
        } else if (actualExtent.height > capabilities->maxImageExtent.height) {
            actualExtent.height = capabilities->maxImageExtent.height;
        }
        
        return actualExtent;
    }
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type SwapChain struct {
	Handle       C.VkSwapchainKHR
	Images       []C.VkImage
	ImageViews   []C.VkImageView
	Framebuffers []C.VkFramebuffer
	Format       C.VkFormat
	ColorSpace   C.VkColorSpaceKHR
	PresentMode  C.VkPresentModeKHR
	Extent       C.VkExtent2D
	ImageCount   uint32
}

type SwapChainConfig struct {
	Width         uint32
	Height        uint32
	VSync         bool
	TripleBuffer  bool
}

func CreateSwapChain(device *Device, surface C.VkSurfaceKHR, config SwapChainConfig) (*SwapChain, error) {
	details := C.SwapChainSupportDetails{}
	C.querySwapChainSupport(device.PhysicalDevice, surface, &details)
	defer C.freeSwapChainSupportDetails(&details)
	
	if details.formatCount == 0 || details.presentModeCount == 0 {
		return nil, fmt.Errorf("swapchain does not have available formats or present modes")
	}
	
	// Choose settings
	surfaceFormat := C.chooseSwapSurfaceFormat(details.formats, details.formatCount)
	presentMode := C.VK_PRESENT_MODE_FIFO_KHR
	if !config.VSync {
		presentMode = C.chooseSwapPresentMode(details.presentModes, details.presentModeCount)
	}
	extent := C.chooseSwapExtent(&details.capabilities, C.uint32_t(config.Width), C.uint32_t(config.Height))
	
	// Determine image count
	imageCount := details.capabilities.minImageCount + 1
	if details.capabilities.maxImageCount > 0 && imageCount > details.capabilities.maxImageCount {
		imageCount = details.capabilities.maxImageCount
	}
	
	// Create swapchain
	createInfo := C.VkSwapchainCreateInfoKHR{
		sType:            C.VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
		surface:          surface,
		minImageCount:    imageCount,
		imageFormat:      surfaceFormat.format,
		imageColorSpace:  surfaceFormat.colorSpace,
		imageExtent:      extent,
		imageArrayLayers: 1,
		imageUsage:       C.VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
		preTransform:     details.capabilities.currentTransform,
		compositeAlpha:   C.VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
		presentMode:      presentMode,
		clipped:          C.VK_TRUE,
		oldSwapchain:     nil,
	}
	
	queueFamilyIndices := []C.uint32_t{C.uint32_t(device.GraphicsFamily), C.uint32_t(device.PresentFamily)}
	if device.GraphicsFamily != device.PresentFamily {
		createInfo.imageSharingMode = C.VK_SHARING_MODE_CONCURRENT
		createInfo.queueFamilyIndexCount = 2
		createInfo.pQueueFamilyIndices = &queueFamilyIndices[0]
	} else {
		createInfo.imageSharingMode = C.VK_SHARING_MODE_EXCLUSIVE
	}
	
	sc := &SwapChain{
		Format:      surfaceFormat.format,
		ColorSpace:  surfaceFormat.colorSpace,
		PresentMode: presentMode,
		Extent:      extent,
		ImageCount:  uint32(imageCount),
	}
	
	result := C.vkCreateSwapchainKHR(device.Device, &createInfo, nil, &sc.Handle)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create swapchain: %d", result)
	}
	
	// Get swapchain images
	var actualImageCount C.uint32_t
	C.vkGetSwapchainImagesKHR(device.Device, sc.Handle, &actualImageCount, nil)
	sc.Images = make([]C.VkImage, actualImageCount)
	C.vkGetSwapchainImagesKHR(device.Device, sc.Handle, &actualImageCount, &sc.Images[0])
	
	// Create image views
	sc.ImageViews = make([]C.VkImageView, len(sc.Images))
	for i, image := range sc.Images {
		viewInfo := C.VkImageViewCreateInfo{
			sType:    C.VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
			image:    image,
			viewType: C.VK_IMAGE_VIEW_TYPE_2D,
			format:   sc.Format,
			subresourceRange: C.VkImageSubresourceRange{
				aspectMask:     C.VK_IMAGE_ASPECT_COLOR_BIT,
				baseMipLevel:   0,
				levelCount:     1,
				baseArrayLayer: 0,
				layerCount:     1,
			},
		}
		
		result = C.vkCreateImageView(device.Device, &viewInfo, nil, &sc.ImageViews[i])
		if result != C.VK_SUCCESS {
			return nil, fmt.Errorf("failed to create image view: %d", result)
		}
	}
	
	return sc, nil
}

func (sc *SwapChain) CreateFramebuffers(device *Device, renderPass C.VkRenderPass, depthImageView C.VkImageView) error {
	sc.Framebuffers = make([]C.VkFramebuffer, len(sc.ImageViews))
	
	for i, imageView := range sc.ImageViews {
		attachments := []C.VkImageView{imageView}
		if depthImageView != nil {
			attachments = append(attachments, depthImageView)
		}
		
		framebufferInfo := C.VkFramebufferCreateInfo{
			sType:           C.VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
			renderPass:      renderPass,
			attachmentCount: C.uint32_t(len(attachments)),
			pAttachments:    &attachments[0],
			width:           sc.Extent.width,
			height:          sc.Extent.height,
			layers:          1,
		}
		
		result := C.vkCreateFramebuffer(device.Device, &framebufferInfo, nil, &sc.Framebuffers[i])
		if result != C.VK_SUCCESS {
			return fmt.Errorf("failed to create framebuffer: %d", result)
		}
	}
	
	return nil
}

func (sc *SwapChain) Destroy(device *Device) {
	for _, framebuffer := range sc.Framebuffers {
		C.vkDestroyFramebuffer(device.Device, framebuffer, nil)
	}
	for _, imageView := range sc.ImageViews {
		C.vkDestroyImageView(device.Device, imageView, nil)
	}
	C.vkDestroySwapchainKHR(device.Device, sc.Handle, nil)
}

func (sc *SwapChain) AcquireNextImage(device *Device, semaphore C.VkSemaphore, timeout uint64) (uint32, error) {
	var imageIndex C.uint32_t
	result := C.vkAcquireNextImageKHR(device.Device, sc.Handle, C.uint64_t(timeout), semaphore, nil, &imageIndex)
	
	if result == C.VK_ERROR_OUT_OF_DATE_KHR {
		return uint32(imageIndex), fmt.Errorf("swapchain out of date")
	} else if result != C.VK_SUCCESS && result != C.VK_SUBOPTIMAL_KHR {
		return uint32(imageIndex), fmt.Errorf("failed to acquire swapchain image: %d", result)
	}
	
	return uint32(imageIndex), nil
}
