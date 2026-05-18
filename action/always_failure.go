package action

import (
	"github.com/actfuns/behaviortree/core"
)

// AlwaysFailureNode is a simple action that always returns FAILURE.
type AlwaysFailureNode struct {
	core.SyncActionNode
}

func NewAlwaysFailureNode(name string, config core.NodeConfig) *AlwaysFailureNode {
	n := &AlwaysFailureNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("AlwaysFailure")
	return n
}

func (n *AlwaysFailureNode) Tick() core.NodeStatus {
	return core.FAILURE
}
