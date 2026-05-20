package control

import (
	"github.com/actfuns/behaviortree/core"
)

// SequenceWithMemory ticks children in order, but remembers the index on failure.
// Unlike SequenceNode, it does NOT reset current_child_idx on failure.
type SequenceWithMemory struct {
	core.ControlNode
	currentChildIdx int
	skippedCount    int
}

func NewSequenceWithMemory(name string, config core.NodeConfig) *SequenceWithMemory {
	n := &SequenceWithMemory{
		currentChildIdx: 0,
		skippedCount:    0,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("SequenceWithMemory")
	return n
}

func (n *SequenceWithMemory) Tick() core.NodeStatus {
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
			return childStatus

		case core.FAILURE:
			// DO NOT reset current_child_idx on failure
			for i := n.currentChildIdx; i < n.ChildrenCount(); i++ {
				n.HaltChild(i)
			}
			return childStatus

		case core.SUCCESS:
			n.currentChildIdx++
			if n.RequiresWakeUp() && prevStatus == core.IDLE &&
				n.currentChildIdx < childrenCount {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}

		case core.SKIPPED:
			n.currentChildIdx++
			n.skippedCount++

		case core.IDLE:
			panic(core.NewLogicError("child returned IDLE during Tick"))

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

func (n *SequenceWithMemory) Halt() {
	n.ControlNode.Halt()
}
