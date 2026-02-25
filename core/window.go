package core

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func init() {
	runtime.LockOSThread()
}

type Window struct {
	Handle *glfw.Window
	Width  int
	Height int
	Title  string
}

type WindowConfig struct {
	Width      int
	Height     int
	Title      string
	Resizable  bool
	VSync      bool
	Fullscreen bool
}

func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		Width:      1280,
		Height:     720,
		Title:      "Render Engine",
		Resizable:  true,
		VSync:      true,
		Fullscreen: false,
	}
}

func NewWindow(config WindowConfig) (*Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize GLFW: %w", err)
	}

	glfw.WindowHint(glfw.ClientAPI, glfw.NoAPI)
	glfw.WindowHint(glfw.Resizable, boolToInt(config.Resizable))

	monitor := (*glfw.Monitor)(nil)
	if config.Fullscreen {
		monitor = glfw.GetPrimaryMonitor()
	}

	handle, err := glfw.CreateWindow(config.Width, config.Height, config.Title, monitor, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create window: %w", err)
	}

	window := &Window{
		Handle: handle,
		Width:  config.Width,
		Height: config.Height,
		Title:  config.Title,
	}

	handle.SetSizeCallback(func(w *glfw.Window, width, height int) {
		window.Width = width
		window.Height = height
	})

	return window, nil
}

func (w *Window) ShouldClose() bool {
	return w.Handle.ShouldClose()
}

func (w *Window) PollEvents() {
	glfw.PollEvents()
}

func (w *Window) SwapBuffers() {
	// Vulkan handles swapchain internally
}

func (w *Window) GetFramebufferSize() (int, int) {
	return w.Handle.GetFramebufferSize()
}

func (w *Window) GetRequiredInstanceExtensions() []string {
	return w.Handle.GetRequiredInstanceExtensions()
}

func (w *Window) CreateWindowSurface(instance uintptr) (uintptr, error) {
	surface, err := w.Handle.CreateWindowSurface(instance, nil)
	return surface, err
}

func (w *Window) Destroy() {
	w.Handle.Destroy()
	glfw.Terminate()
}

func (w *Window) IsKeyPressed(key int) bool {
	return w.Handle.GetKey(glfw.Key(key)) == glfw.Press
}

func (w *Window) SetTitle(title string) {
	w.Handle.SetTitle(title)
	w.Title = title
}

func (w *Window) IsMouseButtonPressed(button int) bool {
	return w.Handle.GetMouseButton(glfw.MouseButton(button)) == glfw.Press
}

func (w *Window) GetCursorPos() (float64, float64) {
	return w.Handle.GetCursorPos()
}

// ScrollCallback is the type for scroll event handlers
type ScrollCallback func(xoff, yoff float64)

func (w *Window) SetScrollCallback(cb ScrollCallback) {
	w.Handle.SetScrollCallback(func(win *glfw.Window, xoff, yoff float64) {
		cb(xoff, yoff)
	})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

const (
	KeySpace        = int(glfw.KeySpace)
	KeyApostrophe   = int(glfw.KeyApostrophe)
	KeyComma        = int(glfw.KeyComma)
	KeyMinus        = int(glfw.KeyMinus)
	KeyPeriod       = int(glfw.KeyPeriod)
	KeySlash        = int(glfw.KeySlash)
	Key0            = int(glfw.Key0)
	Key1            = int(glfw.Key1)
	Key2            = int(glfw.Key2)
	Key3            = int(glfw.Key3)
	Key4            = int(glfw.Key4)
	Key5            = int(glfw.Key5)
	Key6            = int(glfw.Key6)
	Key7            = int(glfw.Key7)
	Key8            = int(glfw.Key8)
	Key9            = int(glfw.Key9)
	KeySemicolon    = int(glfw.KeySemicolon)
	KeyEqual        = int(glfw.KeyEqual)
	KeyA            = int(glfw.KeyA)
	KeyB            = int(glfw.KeyB)
	KeyC            = int(glfw.KeyC)
	KeyD            = int(glfw.KeyD)
	KeyE            = int(glfw.KeyE)
	KeyF            = int(glfw.KeyF)
	KeyG            = int(glfw.KeyG)
	KeyH            = int(glfw.KeyH)
	KeyI            = int(glfw.KeyI)
	KeyJ            = int(glfw.KeyJ)
	KeyK            = int(glfw.KeyK)
	KeyL            = int(glfw.KeyL)
	KeyM            = int(glfw.KeyM)
	KeyN            = int(glfw.KeyN)
	KeyO            = int(glfw.KeyO)
	KeyP            = int(glfw.KeyP)
	KeyQ            = int(glfw.KeyQ)
	KeyR            = int(glfw.KeyR)
	KeyS            = int(glfw.KeyS)
	KeyT            = int(glfw.KeyT)
	KeyU            = int(glfw.KeyU)
	KeyV            = int(glfw.KeyV)
	KeyW            = int(glfw.KeyW)
	KeyX            = int(glfw.KeyX)
	KeyY            = int(glfw.KeyY)
	KeyZ            = int(glfw.KeyZ)
	KeyLeftBracket  = int(glfw.KeyLeftBracket)
	KeyBackslash    = int(glfw.KeyBackslash)
	KeyRightBracket = int(glfw.KeyRightBracket)
	KeyGraveAccent  = int(glfw.KeyGraveAccent)
	KeyWorld1       = int(glfw.KeyWorld1)
	KeyWorld2       = int(glfw.KeyWorld2)
	KeyEscape       = int(glfw.KeyEscape)
	KeyEnter        = int(glfw.KeyEnter)
	KeyTab          = int(glfw.KeyTab)
	KeyBackspace    = int(glfw.KeyBackspace)
	KeyInsert       = int(glfw.KeyInsert)
	KeyDelete       = int(glfw.KeyDelete)
	KeyRight        = int(glfw.KeyRight)
	KeyLeft         = int(glfw.KeyLeft)
	KeyDown         = int(glfw.KeyDown)
	KeyUp           = int(glfw.KeyUp)
	KeyPageUp       = int(glfw.KeyPageUp)
	KeyPageDown     = int(glfw.KeyPageDown)
	KeyHome         = int(glfw.KeyHome)
	KeyEnd          = int(glfw.KeyEnd)
	KeyCapsLock     = int(glfw.KeyCapsLock)
	KeyScrollLock   = int(glfw.KeyScrollLock)
	KeyNumLock      = int(glfw.KeyNumLock)
	KeyPrintScreen  = int(glfw.KeyPrintScreen)
	KeyPause        = int(glfw.KeyPause)
	KeyF1           = int(glfw.KeyF1)
	KeyF2           = int(glfw.KeyF2)
	KeyF3           = int(glfw.KeyF3)
	KeyF4           = int(glfw.KeyF4)
	KeyF5           = int(glfw.KeyF5)
	KeyF6           = int(glfw.KeyF6)
	KeyF7           = int(glfw.KeyF7)
	KeyF8           = int(glfw.KeyF8)
	KeyF9           = int(glfw.KeyF9)
	KeyF10          = int(glfw.KeyF10)
	KeyF11          = int(glfw.KeyF11)
	KeyF12          = int(glfw.KeyF12)
	KeyLeftShift    = int(glfw.KeyLeftShift)
	KeyLeftControl  = int(glfw.KeyLeftControl)
	KeyLeftAlt      = int(glfw.KeyLeftAlt)
	KeyLeftSuper    = int(glfw.KeyLeftSuper)
	KeyRightShift   = int(glfw.KeyRightShift)
	KeyRightControl = int(glfw.KeyRightControl)
	KeyRightAlt     = int(glfw.KeyRightAlt)
	KeyRightSuper   = int(glfw.KeyRightSuper)
	KeyMenu         = int(glfw.KeyMenu)
)
