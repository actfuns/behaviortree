package core

import "log/slog"

// ControlNode is the base for nodes that can have multiple children.
// It embeds treeNodeBase and adds a children list.
type ControlNode struct {
	*treeNodeBase
	children []TreeNode
}

// Init initializes a ControlNode.
func (n *ControlNode) Init(name string, config NodeConfig) {
	n.treeNodeBase = newTreeNodeBase(name, config)
	n.children = nil
}

// AddChild adds a child node.
func (n *ControlNode) AddChild(child TreeNode) {
	n.children = append(n.children, child)
}

// ChildrenCount returns the number of children.
func (n *ControlNode) ChildrenCount() int {
	return len(n.children)
}

// Children returns the list of children.
func (n *ControlNode) Children() []TreeNode {
	return n.children
}

// Child returns the child at the given index.
func (n *ControlNode) Child(idx int) TreeNode {
	if idx >= 0 && idx < len(n.children) {
		return n.children[idx]
	}
	return nil
}

// NodeType returns CONTROL.
func (n *ControlNode) NodeType() NodeType {
	return Control
}

// Tick is a placeholder. Concrete control nodes must override this.
func (n *ControlNode) Tick() NodeStatus {
	slog.Error("ControlNode::Tick() should not be called directly, override in concrete node", "node", n.Name())
	return FAILURE
}

// Halt interrupts execution and resets children.
func (n *ControlNode) Halt() {
	n.ResetChildren()
	n.ResetStatus()
}

// ResetChildren halts all RUNNING children and resets all to IDLE.
func (n *ControlNode) ResetChildren() {
	for _, child := range n.children {
		if child.Status() == RUNNING {
			n.haltNode(child)
		}
		child.ResetStatus()
	}
}

// HaltChildren halts all children from the given index onward.
func (n *ControlNode) HaltChildren() {
	for _, child := range n.children {
		n.haltChild(child)
	}
}

// HaltChild halts a single child at the given index.
func (n *ControlNode) HaltChild(idx int) {
	if idx >= 0 && idx < len(n.children) {
		n.haltChild(n.children[idx])
	}
}

func (n *ControlNode) haltChild(child TreeNode) {
	if child.Status() == RUNNING {
		n.haltNode(child)
	}
	child.ResetStatus()
}

func (n *ControlNode) haltNode(child TreeNode) {
	child.HaltNode()
}

// ResetChildrenFrom halts and resets children from a given index.
func (n *ControlNode) ResetChildrenFrom(first int) {
	for i := first; i < len(n.children); i++ {
		n.haltChild(n.children[i])
	}
}
