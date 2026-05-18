package action

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// UnsetBlackboardNode removes an entry from the blackboard and returns SUCCESS.
type UnsetBlackboardNode struct {
	core.SyncActionNode
}

func NewUnsetBlackboardNode(name string, config core.NodeConfig) *UnsetBlackboardNode {
	n := &UnsetBlackboardNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("UnsetBlackboard")
	return n
}

func (n *UnsetBlackboardNode) Tick() core.NodeStatus {
	var key string
	if err := n.GetInput("key", &key); err != nil {
		slog.Error("missing input port [key]")
		return core.FAILURE
	}
	n.Config().Blackboard.Unset(key)
	return core.SUCCESS
}
