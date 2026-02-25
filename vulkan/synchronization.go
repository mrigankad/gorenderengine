package vulkan

/*
#include <vulkan/vulkan.h>
*/
import "C"
import (
	"fmt"
)

type Semaphore struct {
	Handle C.VkSemaphore
}

type Fence struct {
	Handle C.VkFence
}

func CreateSemaphore(device *Device) (*Semaphore, error) {
	semaphoreInfo := C.VkSemaphoreCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
	}
	
	var semaphore C.VkSemaphore
	result := C.vkCreateSemaphore(device.Device, &semaphoreInfo, nil, &semaphore)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create semaphore: %d", result)
	}
	
	return &Semaphore{Handle: semaphore}, nil
}

func (s *Semaphore) Destroy(device *Device) {
	C.vkDestroySemaphore(device.Device, s.Handle, nil)
}

func CreateFence(device *Device, signaled bool) (*Fence, error) {
	flags := C.VkFenceCreateFlags(0)
	if signaled {
		flags = C.VK_FENCE_CREATE_SIGNALED_BIT
	}
	
	fenceInfo := C.VkFenceCreateInfo{
		sType: C.VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
		flags: flags,
	}
	
	var fence C.VkFence
	result := C.vkCreateFence(device.Device, &fenceInfo, nil, &fence)
	if result != C.VK_SUCCESS {
		return nil, fmt.Errorf("failed to create fence: %d", result)
	}
	
	return &Fence{Handle: fence}, nil
}

func (f *Fence) Destroy(device *Device) {
	C.vkDestroyFence(device.Device, f.Handle, nil)
}

func (f *Fence) Wait(device *Device, timeout uint64) error {
	result := C.vkWaitForFences(device.Device, 1, &f.Handle, C.VK_TRUE, C.uint64_t(timeout))
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to wait for fence: %d", result)
	}
	return nil
}

func (f *Fence) Reset(device *Device) error {
	result := C.vkResetFences(device.Device, 1, &f.Handle)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to reset fence: %d", result)
	}
	return nil
}

func SubmitQueue(queue C.VkQueue, commandBuffers []CommandBuffer, waitSemaphores []C.VkSemaphore, signalSemaphores []C.VkSemaphore, fence *Fence) error {
	cmdBufferHandles := make([]C.VkCommandBuffer, len(commandBuffers))
	for i, cb := range commandBuffers {
		cmdBufferHandles[i] = cb.Handle
	}
	
	waitStages := make([]C.VkPipelineStageFlags, len(waitSemaphores))
	for i := range waitStages {
		waitStages[i] = C.VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT
	}
	
	var fenceHandle C.VkFence
	if fence != nil {
		fenceHandle = fence.Handle
	}
	
	submitInfo := C.VkSubmitInfo{
		sType:                C.VK_STRUCTURE_TYPE_SUBMIT_INFO,
		waitSemaphoreCount:   C.uint32_t(len(waitSemaphores)),
		pWaitSemaphores:      &waitSemaphores[0],
		pWaitDstStageMask:    &waitStages[0],
		commandBufferCount:   C.uint32_t(len(cmdBufferHandles)),
		pCommandBuffers:      &cmdBufferHandles[0],
		signalSemaphoreCount: C.uint32_t(len(signalSemaphores)),
		pSignalSemaphores:    &signalSemaphores[0],
	}
	
	result := C.vkQueueSubmit(queue, 1, &submitInfo, fenceHandle)
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to submit draw command buffer: %d", result)
	}
	
	return nil
}

func PresentQueue(queue C.VkQueue, swapchains []C.VkSwapchainKHR, imageIndices []uint32, waitSemaphores []C.VkSemaphore) error {
	result := C.vkQueuePresentKHR(queue, &C.VkPresentInfoKHR{
		sType:              C.VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
		waitSemaphoreCount: C.uint32_t(len(waitSemaphores)),
		pWaitSemaphores:    &waitSemaphores[0],
		swapchainCount:     C.uint32_t(len(swapchains)),
		pSwapchains:        &swapchains[0],
		pImageIndices:      (*C.uint32_t)(&imageIndices[0]),
	})
	
	if result != C.VK_SUCCESS {
		return fmt.Errorf("failed to present swap chain image: %d", result)
	}
	
	return nil
}
