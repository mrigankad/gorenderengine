package editor

import (
	"render-engine/math"
	"render-engine/scene"
)

// SelectionMode defines what can be selected
type SelectionMode int

const (
	SelectObject SelectionMode = iota
	SelectVertex
	SelectEdge
	SelectFace
)

// EditorMode defines the editor's operating mode
type EditorMode int

const (
	ModeObject EditorMode = iota
	ModeEdit
)

// TransformTool defines the active transform gizmo
type TransformTool int

const (
	ToolSelect TransformTool = iota
	ToolTranslate
	ToolRotate
	ToolScale
)

// Selection tracks all selected objects and components
type Selection struct {
	Objects []*scene.Node
	Mode    SelectionMode

	// Edit mode selections (vertex/edge/face indices)
	Vertices []uint32
	Edges    [][2]uint32 // pairs of vertex indices
	Faces    []uint32    // face indices (every 3 indices in the index buffer)

	// Active object (last selected, shown in properties)
	ActiveObject *scene.Node
}

// NewSelection creates an empty selection
func NewSelection() *Selection {
	return &Selection{
		Objects:  make([]*scene.Node, 0),
		Vertices: make([]uint32, 0),
		Edges:    make([][2]uint32, 0),
		Faces:    make([]uint32, 0),
		Mode:     SelectObject,
	}
}

// Clear removes all selections
func (s *Selection) Clear() {
	s.Objects = s.Objects[:0]
	s.Vertices = s.Vertices[:0]
	s.Edges = s.Edges[:0]
	s.Faces = s.Faces[:0]
	s.ActiveObject = nil
}

// SelectSingle selects a single object, clearing the previous selection
func (s *Selection) SelectSingle(node *scene.Node) {
	s.Objects = []*scene.Node{node}
	s.ActiveObject = node
}

// ToggleObject adds/removes an object from the selection (Shift+Click)
func (s *Selection) ToggleObject(node *scene.Node) {
	for i, n := range s.Objects {
		if n == node {
			// Remove from selection
			s.Objects = append(s.Objects[:i], s.Objects[i+1:]...)
			if s.ActiveObject == node {
				if len(s.Objects) > 0 {
					s.ActiveObject = s.Objects[len(s.Objects)-1]
				} else {
					s.ActiveObject = nil
				}
			}
			return
		}
	}
	// Add to selection
	s.Objects = append(s.Objects, node)
	s.ActiveObject = node
}

// IsSelected checks if a node is selected
func (s *Selection) IsSelected(node *scene.Node) bool {
	for _, n := range s.Objects {
		if n == node {
			return true
		}
	}
	return false
}

// GetSelectionCenter returns the center position of all selected objects
func (s *Selection) GetSelectionCenter() math.Vec3 {
	if len(s.Objects) == 0 {
		return math.Vec3Zero
	}

	center := math.Vec3Zero
	for _, obj := range s.Objects {
		center = center.Add(obj.Transform.Position)
	}
	return center.Div(float32(len(s.Objects)))
}

// HasSelection returns true if anything is selected
func (s *Selection) HasSelection() bool {
	return len(s.Objects) > 0 || len(s.Vertices) > 0 || len(s.Edges) > 0 || len(s.Faces) > 0
}
