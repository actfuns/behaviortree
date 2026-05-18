package control

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// SequenceNode ticks children in an ordered sequence.
// If a child returns RUNNING, previous children will NOT be ticked again.
// If a child returns FAILURE, reset and return FAILURE (reset_on_failure).
type SequenceNode struct {
	core.ControlNode
	currentChildIdx int
	skippedCount    int
	async           bool
}

func NewSequenceNode(name string, config core.NodeConfig) *SequenceNode {
	n := &SequenceNode{
		currentChildIdx: 0,
		skippedCount:    0,
		async:           false,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Sequence")
	return n
}

func (n *SequenceNode) Tick() core.NodeStatus {
	childrenCount := n.ChildrenCount()

	if !n.Status().IsActive() {
		n.skippedCount = 0
	}

	n.SetStatus(core.RUNNING)

	for n.currentChildIdx < childrenCount {
		currentChild := n.Child(n.currentChildIdx)
		prevStatus := currentChild.Status()
		childStatus := currentChild.ExecuteTick()

		switch childStatus {
		case core.RUNNING:
			return core.RUNNING

		case core.FAILURE:
			n.ResetChildren()
			n.currentChildIdx = 0
			return childStatus

		case core.SUCCESS:
			n.currentChildIdx++
			if n.async && n.RequiresWakeUp() && prevStatus == core.IDLE &&
				n.currentChildIdx < childrenCount {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}

		case core.SKIPPED:
			n.currentChildIdx++
			n.skippedCount++

		case core.IDLE:
			slog.Error("child returned IDLE during Tick; children should not return IDLE")
			return core.FAILURE
		}
	}

	allSkipped := n.skippedCount == childrenCount
	if n.currentChildIdx == childrenCount {
		n.ResetChildren()
		n.currentChildIdx = 0
		n.skippedCount = 0
	}
	if allSkipped {
		return core.SKIPPED
	}
	return core.SUCCESS
}

func (n *SequenceNode) Halt() {
	n.currentChildIdx = 0
	n.skippedCount = 0
	n.ControlNode.Halt()
}

// SetAsync enables async mode. When enabled, the sequence returns RUNNING
// after each successful child, allowing other tasks to tick between children.
func (n *SequenceNode) SetAsync(async bool) {
	n.async = async
}
