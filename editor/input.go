package editor

import (
	"render-engine/core"
)

// InputManager tracks mouse and keyboard state for the editor
type InputManager struct {
	// Mouse state
	MouseX, MouseY           float64
	MouseDeltaX, MouseDeltaY float64
	lastMouseX, lastMouseY   float64
	ScrollDelta              float64

	// Button states
	mouseButtons     [8]bool
	mouseButtonsPrev [8]bool

	// Key states
	keys     [512]bool
	keysPrev [512]bool

	// Modifiers
	ShiftDown bool
	CtrlDown  bool
	AltDown   bool

	// Window for key polling
	window     *core.Window
	firstFrame bool
}

// Mouse button constants
const (
	MouseLeft   = 0
	MouseRight  = 1
	MouseMiddle = 2
)

// NewInputManager creates a new input manager with scroll callback
func NewInputManager(window *core.Window) *InputManager {
	im := &InputManager{
		window:     window,
		firstFrame: true,
	}

	window.SetScrollCallback(func(xoff, yoff float64) {
		im.ScrollDelta += yoff
	})

	return im
}

// Update should be called once per frame to compute deltas and poll state
func (im *InputManager) Update() {
	// Poll mouse position
	x, y := im.window.GetCursorPos()
	if im.firstFrame {
		im.lastMouseX = x
		im.lastMouseY = y
		im.firstFrame = false
	}
	im.MouseDeltaX = x - im.lastMouseX
	im.MouseDeltaY = y - im.lastMouseY
	im.lastMouseX = x
	im.lastMouseY = y
	im.MouseX = x
	im.MouseY = y

	// Save previous states
	copy(im.mouseButtonsPrev[:], im.mouseButtons[:])
	copy(im.keysPrev[:], im.keys[:])

	// Poll mouse buttons
	im.mouseButtons[MouseLeft] = im.window.IsMouseButtonPressed(MouseLeft)
	im.mouseButtons[MouseRight] = im.window.IsMouseButtonPressed(MouseRight)
	im.mouseButtons[MouseMiddle] = im.window.IsMouseButtonPressed(MouseMiddle)

	// Poll modifier keys
	im.ShiftDown = im.window.IsKeyPressed(core.KeyLeftShift) || im.window.IsKeyPressed(core.KeyRightShift)
	im.CtrlDown = im.window.IsKeyPressed(core.KeyLeftControl) || im.window.IsKeyPressed(core.KeyRightControl)
	im.AltDown = im.window.IsKeyPressed(core.KeyLeftAlt) || im.window.IsKeyPressed(core.KeyRightAlt)

	// Poll commonly used keys
	commonKeys := []int{
		core.KeyEscape, core.KeyTab, core.KeyDelete, core.KeyBackspace, core.KeyEnter,
		core.KeyA, core.KeyB, core.KeyC, core.KeyD, core.KeyE, core.KeyF, core.KeyG,
		core.KeyH, core.KeyI, core.KeyJ, core.KeyK, core.KeyL, core.KeyM, core.KeyN,
		core.KeyO, core.KeyP, core.KeyQ, core.KeyR, core.KeyS, core.KeyT, core.KeyU,
		core.KeyV, core.KeyW, core.KeyX, core.KeyY, core.KeyZ,
		core.Key0, core.Key1, core.Key2, core.Key3, core.Key4,
		core.Key5, core.Key6, core.Key7, core.Key8, core.Key9,
	}
	for _, k := range commonKeys {
		if k >= 0 && k < len(im.keys) {
			im.keys[k] = im.window.IsKeyPressed(k)
		}
	}
}

// EndFrame clears per-frame state
func (im *InputManager) EndFrame() {
	im.ScrollDelta = 0
}

// --- Mouse Queries ---

func (im *InputManager) IsMouseDown(button int) bool {
	if button < 0 || button >= len(im.mouseButtons) {
		return false
	}
	return im.mouseButtons[button]
}

func (im *InputManager) IsMousePressed(button int) bool {
	if button < 0 || button >= len(im.mouseButtons) {
		return false
	}
	return im.mouseButtons[button] && !im.mouseButtonsPrev[button]
}

func (im *InputManager) IsMouseReleased(button int) bool {
	if button < 0 || button >= len(im.mouseButtons) {
		return false
	}
	return !im.mouseButtons[button] && im.mouseButtonsPrev[button]
}

// --- Key Queries ---

func (im *InputManager) IsKeyDown(key int) bool {
	if key < 0 || key >= len(im.keys) {
		return false
	}
	return im.keys[key]
}

func (im *InputManager) IsKeyPressed(key int) bool {
	if key < 0 || key >= len(im.keys) {
		return false
	}
	return im.keys[key] && !im.keysPrev[key]
}

// IsShortcut checks for a Ctrl+key press
func (im *InputManager) IsShortcut(key int) bool {
	return im.CtrlDown && im.IsKeyPressed(key)
}

// IsShiftShortcut checks for Ctrl+Shift+key press
func (im *InputManager) IsShiftShortcut(key int) bool {
	return im.CtrlDown && im.ShiftDown && im.IsKeyPressed(key)
}
