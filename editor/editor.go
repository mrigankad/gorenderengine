package editor

import (
	"fmt"

	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
)

// Editor is the top-level editor state machine
type Editor struct {
	// Core state
	Mode       EditorMode
	ActiveTool TransformTool
	Selection  *Selection
	History    *History
	Input      *InputManager
	Scene      *scene.Scene
	Window     *core.Window

	// Camera
	OrbitCamera *scene.OrbitCamera

	// Status info
	StatusText string
}

// NewEditor initializes a new editor instance
func NewEditor(window *core.Window, s *scene.Scene) *Editor {
	camera := scene.NewOrbitCamera(math.Vec3Zero, 5.0, 1.0472, float32(window.Width)/float32(window.Height))
	s.SetCamera(&camera.Camera)

	return &Editor{
		Mode:        ModeObject,
		ActiveTool:  ToolTranslate,
		Selection:   NewSelection(),
		History:     NewHistory(100),
		Input:       NewInputManager(window),
		Scene:       s,
		Window:      window,
		OrbitCamera: camera,
		StatusText:  "Ready",
	}
}

// Update processes one frame of editor logic
func (e *Editor) Update(deltaTime float32) {
	e.Input.Update()

	e.handleShortcuts()
	e.handleCameraControls()
	e.handleMouseSelection()

	e.Input.EndFrame()
}

func (e *Editor) handleShortcuts() {
	// Undo: Ctrl+Z
	if e.Input.IsShortcut(core.KeyZ) && !e.Input.ShiftDown {
		if e.History.Undo() {
			e.StatusText = "Undo"
		}
	}

	// Redo: Ctrl+Shift+Z
	if e.Input.IsShiftShortcut(core.KeyZ) {
		if e.History.Redo() {
			e.StatusText = "Redo"
		}
	}

	// Delete: X or Delete
	if e.Input.IsKeyPressed(core.KeyX) || e.Input.IsKeyPressed(core.KeyDelete) {
		if !e.Input.CtrlDown {
			e.deleteSelected()
		}
	}

	// Duplicate: Shift+D
	if e.Input.ShiftDown && e.Input.IsKeyPressed(core.KeyD) {
		e.duplicateSelected()
	}

	// Transform tool shortcuts
	if e.Input.IsKeyPressed(core.KeyG) {
		e.ActiveTool = ToolTranslate
		e.StatusText = "Tool: Move"
	}
	if e.Input.IsKeyPressed(core.KeyR) && !e.Input.CtrlDown {
		e.ActiveTool = ToolRotate
		e.StatusText = "Tool: Rotate"
	}
	if e.Input.IsKeyPressed(core.KeyS) && !e.Input.CtrlDown {
		e.ActiveTool = ToolScale
		e.StatusText = "Tool: Scale"
	}

	// Toggle edit mode: Tab
	if e.Input.IsKeyPressed(core.KeyTab) {
		if e.Mode == ModeObject {
			if e.Selection.ActiveObject != nil {
				e.Mode = ModeEdit
				e.Selection.Mode = SelectVertex
				e.StatusText = "Edit Mode (Vertex)"
			}
		} else {
			e.Mode = ModeObject
			e.Selection.Mode = SelectObject
			e.Selection.Vertices = e.Selection.Vertices[:0]
			e.Selection.Edges = e.Selection.Edges[:0]
			e.Selection.Faces = e.Selection.Faces[:0]
			e.StatusText = "Object Mode"
		}
	}

	// Selection modes in edit mode
	if e.Mode == ModeEdit {
		if e.Input.IsKeyPressed(core.Key1) {
			e.Selection.Mode = SelectVertex
			e.StatusText = "Select: Vertex"
		}
		if e.Input.IsKeyPressed(core.Key2) {
			e.Selection.Mode = SelectEdge
			e.StatusText = "Select: Edge"
		}
		if e.Input.IsKeyPressed(core.Key3) {
			e.Selection.Mode = SelectFace
			e.StatusText = "Select: Face"
		}
	}
}

func (e *Editor) handleCameraControls() {
	// Scroll zoom
	if e.Input.ScrollDelta != 0 {
		e.OrbitCamera.Zoom(-float32(e.Input.ScrollDelta) * 0.5)
	}

	// MMB orbit / Shift+MMB pan
	if e.Input.IsMouseDown(MouseMiddle) {
		dx := float32(e.Input.MouseDeltaX) * 0.01
		dy := float32(e.Input.MouseDeltaY) * 0.01

		if e.Input.ShiftDown {
			// Pan
			right := e.OrbitCamera.GetRight()
			up := e.OrbitCamera.GetUp()
			panSpeed := e.OrbitCamera.Distance * 0.002
			offset := right.Mul(-dx * panSpeed).Add(up.Mul(dy * panSpeed))
			e.OrbitCamera.Target = e.OrbitCamera.Target.Add(offset)
			e.OrbitCamera.UpdatePosition()
		} else {
			// Orbit
			e.OrbitCamera.Orbit(-dx, -dy)
		}
	}
}

func (e *Editor) handleMouseSelection() {
	if e.Mode != ModeObject {
		return
	}

	// Left click select
	if e.Input.IsMousePressed(MouseLeft) {
		ray := ScreenToRay(
			float32(e.Input.MouseX), float32(e.Input.MouseY),
			float32(e.Window.Width), float32(e.Window.Height),
			&e.OrbitCamera.Camera,
		)

		hit := RaycastScene(ray, e.Scene)
		if hit.Hit && hit.Node != nil {
			if e.Input.ShiftDown {
				e.Selection.ToggleObject(hit.Node)
			} else {
				e.Selection.SelectSingle(hit.Node)
			}
			e.StatusText = fmt.Sprintf("Selected: %s", hit.Node.Name)
		} else if !e.Input.ShiftDown {
			e.Selection.Clear()
			e.StatusText = "Selection cleared"
		}
	}
}

func (e *Editor) deleteSelected() {
	for _, node := range e.Selection.Objects {
		cmd := NewDeleteNodeCommand(e.Scene, node)
		e.History.Do(cmd)
	}
	e.Selection.Clear()
	e.StatusText = "Deleted"
}

func (e *Editor) duplicateSelected() {
	for _, node := range e.Selection.Objects {
		cmd := NewDuplicateNodeCommand(e.Scene, node)
		e.History.Do(cmd)
	}
	e.StatusText = "Duplicated"
}

// GetStats returns scene statistics for the status bar
func (e *Editor) GetStats() (objectCount, vertexCount, faceCount int) {
	e.Scene.Root.Traverse(func(n *scene.Node) {
		if n.Mesh != nil {
			objectCount++
			vertexCount += len(n.Mesh.Vertices)
			faceCount += len(n.Mesh.Indices) / 3
		}
	})
	return
}
