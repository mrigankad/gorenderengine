package editor

import (
	"render-engine/core"
	"render-engine/math"
	"render-engine/scene"
)

// Command represents an undoable editor action
type Command interface {
	Execute()
	Undo()
	Description() string
}

// History manages undo/redo stacks
type History struct {
	undoStack []Command
	redoStack []Command
	maxDepth  int
}

// NewHistory creates a new history with the given max undo depth
func NewHistory(maxDepth int) *History {
	return &History{
		undoStack: make([]Command, 0, maxDepth),
		redoStack: make([]Command, 0, maxDepth),
		maxDepth:  maxDepth,
	}
}

// Do executes a command and pushes it to the undo stack
func (h *History) Do(cmd Command) {
	cmd.Execute()
	h.undoStack = append(h.undoStack, cmd)
	if len(h.undoStack) > h.maxDepth {
		h.undoStack = h.undoStack[1:]
	}
	// Clear redo stack on new action
	h.redoStack = h.redoStack[:0]
}

// Undo reverts the last action
func (h *History) Undo() bool {
	if len(h.undoStack) == 0 {
		return false
	}
	cmd := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	cmd.Undo()
	h.redoStack = append(h.redoStack, cmd)
	return true
}

// Redo reapplies the last undone action
func (h *History) Redo() bool {
	if len(h.redoStack) == 0 {
		return false
	}
	cmd := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	cmd.Execute()
	h.undoStack = append(h.undoStack, cmd)
	return true
}

// CanUndo returns whether there are actions to undo
func (h *History) CanUndo() bool { return len(h.undoStack) > 0 }

// CanRedo returns whether there are actions to redo
func (h *History) CanRedo() bool { return len(h.redoStack) > 0 }

// Clear wipes all undo/redo history
func (h *History) Clear() {
	h.undoStack = h.undoStack[:0]
	h.redoStack = h.redoStack[:0]
}

// --- Concrete Commands ---

// TransformCommand records a transform change on a node
type TransformCommand struct {
	Node         *scene.Node
	OldTransform core.Transform
	NewTransform core.Transform
	desc         string
}

func NewTransformCommand(node *scene.Node, newTransform core.Transform, desc string) *TransformCommand {
	return &TransformCommand{
		Node:         node,
		OldTransform: node.Transform,
		NewTransform: newTransform,
		desc:         desc,
	}
}

func (c *TransformCommand) Execute() {
	c.Node.Transform = c.NewTransform
	c.Node.MarkWorldMatrixDirty()
}
func (c *TransformCommand) Undo()               { c.Node.Transform = c.OldTransform; c.Node.MarkWorldMatrixDirty() }
func (c *TransformCommand) Description() string { return c.desc }

// MoveCommand is a shortcut for position-only changes
type MoveCommand struct {
	Node   *scene.Node
	OldPos math.Vec3
	NewPos math.Vec3
}

func NewMoveCommand(node *scene.Node, newPos math.Vec3) *MoveCommand {
	return &MoveCommand{Node: node, OldPos: node.Transform.Position, NewPos: newPos}
}

func (c *MoveCommand) Execute()            { c.Node.SetPosition(c.NewPos) }
func (c *MoveCommand) Undo()               { c.Node.SetPosition(c.OldPos) }
func (c *MoveCommand) Description() string { return "Move " + c.Node.Name }

// RotateCommand records a rotation change
type RotateCommand struct {
	Node   *scene.Node
	OldRot math.Quaternion
	NewRot math.Quaternion
}

func NewRotateCommand(node *scene.Node, newRot math.Quaternion) *RotateCommand {
	return &RotateCommand{Node: node, OldRot: node.Transform.Rotation, NewRot: newRot}
}

func (c *RotateCommand) Execute()            { c.Node.SetRotation(c.NewRot) }
func (c *RotateCommand) Undo()               { c.Node.SetRotation(c.OldRot) }
func (c *RotateCommand) Description() string { return "Rotate " + c.Node.Name }

// ScaleCommand records a scale change
type ScaleCommand struct {
	Node     *scene.Node
	OldScale math.Vec3
	NewScale math.Vec3
}

func NewScaleCommand(node *scene.Node, newScale math.Vec3) *ScaleCommand {
	return &ScaleCommand{Node: node, OldScale: node.Transform.Scale, NewScale: newScale}
}

func (c *ScaleCommand) Execute()            { c.Node.SetScale(c.NewScale) }
func (c *ScaleCommand) Undo()               { c.Node.SetScale(c.OldScale) }
func (c *ScaleCommand) Description() string { return "Scale " + c.Node.Name }

// AddNodeCommand records adding an object to the scene
type AddNodeCommand struct {
	Scene *scene.Scene
	Node  *scene.Node
}

func NewAddNodeCommand(s *scene.Scene, node *scene.Node) *AddNodeCommand {
	return &AddNodeCommand{Scene: s, Node: node}
}

func (c *AddNodeCommand) Execute()            { c.Scene.AddNode(c.Node) }
func (c *AddNodeCommand) Undo()               { c.Scene.RemoveNode(c.Node) }
func (c *AddNodeCommand) Description() string { return "Add " + c.Node.Name }

// DeleteNodeCommand records deleting an object from the scene
type DeleteNodeCommand struct {
	Scene  *scene.Scene
	Node   *scene.Node
	Parent *scene.Node
}

func NewDeleteNodeCommand(s *scene.Scene, node *scene.Node) *DeleteNodeCommand {
	return &DeleteNodeCommand{Scene: s, Node: node, Parent: node.Parent}
}

func (c *DeleteNodeCommand) Execute() { c.Scene.RemoveNode(c.Node) }
func (c *DeleteNodeCommand) Undo() {
	if c.Parent != nil {
		c.Parent.AddChild(c.Node)
	} else {
		c.Scene.AddNode(c.Node)
	}
}
func (c *DeleteNodeCommand) Description() string { return "Delete " + c.Node.Name }

// DuplicateNodeCommand records duplicating an object
type DuplicateNodeCommand struct {
	Scene     *scene.Scene
	Original  *scene.Node
	Duplicate *scene.Node
}

func NewDuplicateNodeCommand(s *scene.Scene, original *scene.Node) *DuplicateNodeCommand {
	dup := scene.NewNode(original.Name + ".copy")
	dup.Transform = original.Transform
	dup.Mesh = original.Mesh // share mesh data
	dup.Visible = original.Visible
	// Offset slightly so it's visible
	dup.Transform.Position = dup.Transform.Position.Add(math.Vec3{X: 0.5, Y: 0, Z: 0})
	return &DuplicateNodeCommand{Scene: s, Original: original, Duplicate: dup}
}

func (c *DuplicateNodeCommand) Execute()            { c.Scene.AddNode(c.Duplicate) }
func (c *DuplicateNodeCommand) Undo()               { c.Scene.RemoveNode(c.Duplicate) }
func (c *DuplicateNodeCommand) Description() string { return "Duplicate " + c.Original.Name }
