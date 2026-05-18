package control

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// ReactiveSequence ticks all children every time.
// If a child returns RUNNING, halt remaining siblings and return RUNNING.
// If a child returns SUCCESS, tick the next sibling.
// If a child returns FAILURE, stop and return FAILURE.
//
// IMPORTANT: should not have more than a single asynchronous child.
type ReactiveSequence struct {
	core.ControlNode
	runningChild           int
	throwIfMultipleRunning bool
}

var reactiveSeqThrowIfMultipleRunning = false

func NewReactiveSequence(name string, config core.NodeConfig) *ReactiveSequence {
	n := &ReactiveSequence{
		runningChild: -1,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ReactiveSequence")
	return n
}

func ReactiveSequenceEnableException(enable bool) {
	reactiveSeqThrowIfMultipleRunning = enable
}

func (n *ReactiveSequence) Tick() core.NodeStatus {
	allSkipped := true
	if n.Status() == core.IDLE {
		n.runningChild = -1
	}
	n.SetStatus(core.RUNNING)

	for i := 0; i < n.ChildrenCount(); i++ {
		currentChild := n.Child(i)
		childStatus := currentChild.ExecuteTick()
		allSkipped = allSkipped && (childStatus == core.SKIPPED)

		switch childStatus {
		case core.RUNNING:
			for j := 0; j < n.ChildrenCount(); j++ {
				if j != i {
					n.HaltChild(j)
				}
			}
			if n.runningChild == -1 {
				n.runningChild = i
			} else if reactiveSeqThrowIfMultipleRunning && n.runningChild != i {
				slog.Error("ReactiveSequence: only a single child can return RUNNING")
				return core.FAILURE
			}
			return core.RUNNING

		case core.FAILURE:
			n.ResetChildren()
			return core.FAILURE

		case core.SUCCESS:
			// Continue to next child

		case core.SKIPPED:
			n.HaltChild(i)

		case core.IDLE:
			slog.Error("child returned IDLE during Tick; children should not return IDLE")
			return core.FAILURE
		}
	}

	n.ResetChildren()
	if allSkipped {
		return core.SKIPPED
	}
	return core.SUCCESS
}

func (n *ReactiveSequence) Halt() {
	n.runningChild = -1
	n.ControlNode.Halt()
}
