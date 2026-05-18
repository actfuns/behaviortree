package decorator

import "github.com/actfuns/behaviortree/core"

// ForceFailureNode always returns FAILURE regardless of child's result.
type ForceFailureNode struct {
	core.DecoratorNode
}

func NewForceFailureNode(name string, config core.NodeConfig) *ForceFailureNode {
	n := &ForceFailureNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ForceFailure")
	return n
}

func (n *ForceFailureNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)
	childStatus := n.Child().ExecuteTick()

	if childStatus == core.RUNNING {
		return core.RUNNING
	}

	n.ResetChild()
	return core.FAILURE
}
