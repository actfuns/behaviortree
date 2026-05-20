package control

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// ReactiveFallback ticks all children every time.
// If a child returns RUNNING, halt other children and return RUNNING.
// If a child returns FAILURE, continue to next sibling.
// If a child returns SUCCESS, stop and return SUCCESS.
//
// IMPORTANT: should not have more than a single asynchronous child.
type ReactiveFallback struct {
	core.ControlNode
	runningChild           int
	throwIfMultipleRunning bool
}

var reactiveFbThrowIfMultipleRunning = false

func NewReactiveFallback(name string, config core.NodeConfig) *ReactiveFallback {
	n := &ReactiveFallback{
		runningChild: -1,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ReactiveFallback")
	return n
}

func ReactiveFallbackEnableException(enable bool) {
	reactiveFbThrowIfMultipleRunning = enable
}

func (n *ReactiveFallback) Tick() core.NodeStatus {
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
			} else if reactiveFbThrowIfMultipleRunning && n.runningChild != i {
				slog.Error("ReactiveFallback: only a single child can return RUNNING")
				return core.FAILURE
			}
			return core.RUNNING

		case core.FAILURE:
			// Continue to next child

		case core.SUCCESS:
			n.ResetChildren()
			return core.SUCCESS

		case core.SKIPPED:
			n.HaltChild(i)

		case core.IDLE:
			panic(core.NewLogicError("child returned IDLE during Tick"))

		}
	}

	n.ResetChildren()
	if allSkipped {
		return core.SKIPPED
	}
	return core.FAILURE
}

func (n *ReactiveFallback) Halt() {
	n.runningChild = -1
	n.ControlNode.Halt()
}
