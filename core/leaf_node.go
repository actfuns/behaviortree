package core

// LeafNode is the base for leaf nodes (actions and conditions).
type LeafNode struct {
	*treeNodeBase
}

// Init initializes a LeafNode.
func (n *LeafNode) Init(name string, config NodeConfig) {
	n.treeNodeBase = newTreeNodeBase(name, config)
}

// Halt resets the status.
func (n *LeafNode) Halt() {
	n.ResetStatus()
}

// Tick implements the TreeNode interface.
// Concrete types that embed LeafNode should override this method.
func (n *LeafNode) Tick() NodeStatus {
	return SUCCESS
}

// NodeType returns Undefined for the base LeafNode.
func (n *LeafNode) NodeType() NodeType {
	return Undefined
}
