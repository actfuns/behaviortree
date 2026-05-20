package decorator

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// KeepRunningUntilFailureNode ticks the child repeatedly until it returns FAILURE.
// If child returns SUCCESS, it is ticked again until FAILURE.
// If child returns RUNNING, this node returns RUNNING.
type KeepRunningUntilFailureNode struct {
	core.DecoratorNode
}

func NewKeepRunningUntilFailureNode(name string, config core.NodeConfig) *KeepRunningUntilFailureNode {
	n := &KeepRunningUntilFailureNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("KeepRunningUntilFailure")
	return n
}

func (n *KeepRunningUntilFailureNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)

	childStatus := n.Child().ExecuteTick()

	switch childStatus {
	case core.SUCCESS:
		n.ResetChild()
		return core.RUNNING

	case core.FAILURE:
		n.ResetChild()
		return core.FAILURE

	case core.RUNNING:
		return core.RUNNING

	case core.SKIPPED:
		n.ResetChild()
		return core.SKIPPED

	case core.IDLE:
		panic(core.NewLogicError("child returned IDLE during Tick"))
	}

	slog.Error("Unexpected status in KeepRunningUntilFailure", "node", n.Name())
	return core.FAILURE
}
