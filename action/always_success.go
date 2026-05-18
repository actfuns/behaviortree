package action

import (
	"github.com/actfuns/behaviortree/core"
)

// AlwaysSuccessNode is a simple action that always returns SUCCESS.
type AlwaysSuccessNode struct {
	core.SyncActionNode
}

func NewAlwaysSuccessNode(name string, config core.NodeConfig) *AlwaysSuccessNode {
	n := &AlwaysSuccessNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("AlwaysSuccess")
	return n
}

func (n *AlwaysSuccessNode) Tick() core.NodeStatus {
	return core.SUCCESS
}
