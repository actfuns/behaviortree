package core

// ConditionNode is a leaf node used to check a condition.
// It should NOT return RUNNING and should NOT have side effects.
type ConditionNode struct {
	LeafNode
}

// Init initializes a ConditionNode.
func (n *ConditionNode) Init(name string, config NodeConfig) {
	n.LeafNode.Init(name, config)
}

// NodeType returns CONDITION.
func (n *ConditionNode) NodeType() NodeType {
	return Condition
}

// Halt resets the status.
func (n *ConditionNode) Halt() {
	n.ResetStatus()
}

// Tick implements the TreeNode interface.
// Concrete types that embed ConditionNode should override this method.
func (n *ConditionNode) Tick() NodeStatus {
	return SUCCESS
}
