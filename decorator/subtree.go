package decorator

import "github.com/actfuns/behaviortree/core"

// SubTreeNode wraps a subtree, delegating tick to the subtree root.
type SubTreeNode struct {
	core.DecoratorNode
	subtreeID string
}

func NewSubTreeNode(name string, config core.NodeConfig) *SubTreeNode {
	n := &SubTreeNode{subtreeID: ""}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("SubTree")
	return n
}

func (n *SubTreeNode) Tick() core.NodeStatus {
	if n.Status() == core.IDLE {
		n.SetStatus(core.RUNNING)
	}

	if n.Child() == nil {
		return core.FAILURE
	}

	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.ResetChild()
	}
	return childStatus
}

// SetSubtreeID sets the subtree ID for this node.
func (n *SubTreeNode) SetSubtreeID(id string) {
	n.subtreeID = id
}

// SubtreeID returns the subtree ID.
func (n *SubTreeNode) SubtreeID() string {
	return n.subtreeID
}
