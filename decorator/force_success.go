package decorator

import "github.com/actfuns/behaviortree/core"

// ForceSuccessNode always returns SUCCESS regardless of child's result.
type ForceSuccessNode struct {
	core.DecoratorNode
}

func NewForceSuccessNode(name string, config core.NodeConfig) *ForceSuccessNode {
	n := &ForceSuccessNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ForceSuccess")
	return n
}

func (n *ForceSuccessNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)
	childStatus := n.Child().ExecuteTick()

	if childStatus == core.RUNNING {
		return core.RUNNING
	}

	n.ResetChild()
	return core.SUCCESS
}
