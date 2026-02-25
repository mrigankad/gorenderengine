package scene

import (
	"render-engine/core"
	"render-engine/math"
)

// Node represents an object in the scene graph
type Node struct {
	Name       string
	Transform  core.Transform
	Parent     *Node
	Children   []*Node
	Mesh       *Mesh
	Visible    bool
	Id         uint32
	
	// Cached world transform
	worldMatrixDirty bool
	worldMatrix      math.Mat4
}

var nodeIdCounter uint32 = 0

func NewNode(name string) *Node {
	nodeIdCounter++
	return &Node{
		Name:             name,
		Transform:        core.NewTransform(),
		Children:         make([]*Node, 0),
		Visible:          true,
		Id:               nodeIdCounter,
		worldMatrixDirty: true,
	}
}

func (n *Node) AddChild(child *Node) {
	if child.Parent != nil {
		child.Parent.RemoveChild(child)
	}
	child.Parent = n
	n.Children = append(n.Children, child)
}

func (n *Node) RemoveChild(child *Node) {
	for i, c := range n.Children {
		if c == child {
			n.Children = append(n.Children[:i], n.Children[i+1:]...)
			child.Parent = nil
			child.MarkWorldMatrixDirty()
			return
		}
	}
}

func (n *Node) GetWorldMatrix() math.Mat4 {
	if n.worldMatrixDirty {
		localMatrix := n.Transform.GetMatrix()
		if n.Parent != nil {
			n.worldMatrix = n.Parent.GetWorldMatrix().Mul(localMatrix)
		} else {
			n.worldMatrix = localMatrix
		}
		n.worldMatrixDirty = false
	}
	return n.worldMatrix
}

func (n *Node) MarkWorldMatrixDirty() {
	n.worldMatrixDirty = true
	for _, child := range n.Children {
		child.MarkWorldMatrixDirty()
	}
}

func (n *Node) SetPosition(pos math.Vec3) {
	n.Transform.Position = pos
	n.MarkWorldMatrixDirty()
}

func (n *Node) SetRotation(rot math.Quaternion) {
	n.Transform.Rotation = rot
	n.MarkWorldMatrixDirty()
}

func (n *Node) SetScale(scale math.Vec3) {
	n.Transform.Scale = scale
	n.MarkWorldMatrixDirty()
}

func (n *Node) Translate(delta math.Vec3) {
	n.Transform.Position = n.Transform.Position.Add(delta)
	n.MarkWorldMatrixDirty()
}

func (n *Node) Rotate(axis math.Vec3, angle float32) {
	rotation := math.QuaternionFromAxisAngle(axis, angle)
	n.Transform.Rotation = n.Transform.Rotation.Mul(rotation).Normalize()
	n.MarkWorldMatrixDirty()
}

func (n *Node) GetForward() math.Vec3 {
	return n.Transform.GetForward()
}

func (n *Node) GetRight() math.Vec3 {
	return n.Transform.GetRight()
}

func (n *Node) GetUp() math.Vec3 {
	return n.Transform.GetUp()
}

// Update updates the node and its children
func (n *Node) Update(deltaTime float32) {
	// Update mesh if any
	if n.Mesh != nil {
		n.Mesh.Update(deltaTime)
	}
	
	// Update children
	for _, child := range n.Children {
		child.Update(deltaTime)
	}
}

// Traverse visits all nodes in the graph
func (n *Node) Traverse(callback func(*Node)) {
	callback(n)
	for _, child := range n.Children {
		child.Traverse(callback)
	}
}

// Find finds a node by name
func (n *Node) Find(name string) *Node {
	if n.Name == name {
		return n
	}
	for _, child := range n.Children {
		if found := child.Find(name); found != nil {
			return found
		}
	}
	return nil
}
