package core

// DecoratorNode is the base for nodes that have exactly one child.
// It embeds treeNodeBase and adds a single child reference.
// DecoratorNode overrides ExecuteTick to wrap child status reset logic.
type DecoratorNode struct {
	*treeNodeBase
	child TreeNode
}

// Init initializes a DecoratorNode.
func (n *DecoratorNode) Init(name string, config NodeConfig) {
	n.treeNodeBase = newTreeNodeBase(name, config)
	n.child = nil
}

// SetChild sets the child node. Panics if a child is already assigned.
func (n *DecoratorNode) SetChild(child TreeNode) {
	if n.child != nil {
		panic(NewBehaviorTreeError("Decorator [%s] has already a child assigned", n.name))
	}
	n.child = child
}

// Child returns the child node.
func (n *DecoratorNode) Child() TreeNode {
	return n.child
}

// NodeType returns DECORATOR.
func (n *DecoratorNode) NodeType() NodeType {
	return Decorator
}

// Halt interrupts execution and resets the child.
func (n *DecoratorNode) Halt() {
	n.ResetChild()
	n.ResetStatus()
}

// HaltChild resets the child.
func (n *DecoratorNode) HaltChild() {
	n.ResetChild()
}

// ResetChild halts and resets the child if RUNNING.
func (n *DecoratorNode) ResetChild() {
	if n.child == nil {
		return
	}
	if n.child.Status() == RUNNING {
		n.child.HaltNode()
	}
	n.child.ResetStatus()
}

// Tick is a placeholder. Concrete decorator nodes must override this.
func (n *DecoratorNode) Tick() NodeStatus {
	panic(NewBehaviorTreeError("DecoratorNode::Tick() should never be called directly. Override in concrete node."))
}

// ExecuteTick overrides the default to match C++ DecoratorNode::executeTick.
// It runs the node's tick (which calls the child's ExecuteTick), then
// resets the child status if it completed.
func (n *DecoratorNode) ExecuteTick() NodeStatus {
	self := n.getSelf()
	if self == nil {
		panic(NewRuntimeError("Node [%s]: getSelf() returned nil. Did you forget to call SetSelf()?", n.name))
	}
	status := n.ExecuteTickImpl(self)
	if n.child != nil {
		childStatus := n.child.Status()
		if childStatus == SUCCESS || childStatus == FAILURE {
			n.child.ResetStatus()
		}
	}
	return status
}
