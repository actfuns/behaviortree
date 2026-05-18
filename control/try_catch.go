package control

import (
	"log/slog"

	"github.com/actfuns/behaviortree/core"
)

// TryCatchNode executes children 1..N-1 as a "try" block (like a Sequence).
//
//   - If all try-children return SUCCESS, this node returns SUCCESS.
//   - If any try-child returns FAILURE, the last child N is executed as a "catch" action.
//     This node returns FAILURE regardless of the catch child's result.
//   - If the catch returns RUNNING, this node returns RUNNING.
//   - SKIPPED try-children are skipped over (not treated as failure).
//
// Port "catch_on_halt" (default false): if true, the catch child is also
// executed when the TryCatch node is halted while the try-block is RUNNING.
//
// Requires at least 2 children.
type TryCatchNode struct {
	core.ControlNode
	currentChildIdx int
	skippedCount    int
	inCatch         bool
}

func NewTryCatchNode(name string, config core.NodeConfig) *TryCatchNode {
	n := &TryCatchNode{
		currentChildIdx: 0,
		skippedCount:    0,
		inCatch:         false,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("TryCatch")
	return n
}

func (n *TryCatchNode) Tick() core.NodeStatus {
	childrenCount := n.ChildrenCount()
	if childrenCount < 2 {
		slog.Error("TryCatch requires at least 2 children", "node", n.Name())
		return core.FAILURE
	}

	if !n.Status().IsActive() {
		n.skippedCount = 0
		n.inCatch = false
	}

	n.SetStatus(core.RUNNING)

	tryCount := childrenCount - 1

	// If we are in catch mode, tick the last child (cleanup)
	if n.inCatch {
		catchChild := n.Child(childrenCount - 1)
		catchStatus := catchChild.ExecuteTick()

		if catchStatus == core.RUNNING {
			return core.RUNNING
		}

		// Catch child finished: return FAILURE regardless of catch result
		n.ResetChildren()
		n.currentChildIdx = 0
		n.inCatch = false
		return core.FAILURE
	}

	// Try-block: execute children 0..N-2 as a Sequence
	for n.currentChildIdx < tryCount {
		currentChild := n.Child(n.currentChildIdx)
		childStatus := currentChild.ExecuteTick()

		switch childStatus {
		case core.RUNNING:
			return core.RUNNING

		case core.FAILURE:
			// Enter catch mode: reset try-block children, then tick catch child
			n.ResetChildren()
			n.currentChildIdx = 0
			n.inCatch = true
			return n.Tick()

		case core.SUCCESS:
			n.currentChildIdx++

		case core.SKIPPED:
			n.currentChildIdx++
			n.skippedCount++

		case core.IDLE:
			slog.Error("child returned IDLE during Tick; child should not return IDLE")
			return core.FAILURE
		}
	}

	// All try-children completed successfully (or were skipped)
	allSkipped := n.skippedCount == tryCount
	n.ResetChildren()
	n.currentChildIdx = 0
	n.skippedCount = 0

	if allSkipped {
		return core.SKIPPED
	}
	return core.SUCCESS
}

func (n *TryCatchNode) Halt() {
	// Check catch_on_halt port
	catchOnHalt := false
	if v, err := core.GetInputTyped[bool](n, "catch_on_halt"); err == nil {
		catchOnHalt = v
	}

	// If catch_on_halt is enabled and we were in try-block (not already in catch),
	// execute the catch child synchronously before halting.
	if catchOnHalt && !n.inCatch && n.Status().IsActive() && n.ChildrenCount() >= 2 {
		// Halt all try-block children first
		for i := 0; i < n.ChildrenCount()-1; i++ {
			n.HaltChild(i)
		}

		// Tick the catch child. If it returns RUNNING, halt it too
		catchChild := n.Child(n.ChildrenCount() - 1)
		catchStatus := catchChild.ExecuteTick()
		if catchStatus == core.RUNNING {
			n.HaltChild(n.ChildrenCount() - 1)
		}
	}

	n.currentChildIdx = 0
	n.skippedCount = 0
	n.inCatch = false
	n.ControlNode.Halt()
}
