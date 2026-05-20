package control

import "github.com/actfuns/behaviortree/core"

// FallbackNode tries different strategies until one succeeds.
// If any child returns RUNNING, previous children will NOT be ticked again.
//
// - If all the children return FAILURE, this node returns FAILURE.
// - If a child returns RUNNING, this node returns RUNNING.
// - If a child returns SUCCESS, stop the loop and return SUCCESS.
type FallbackNode struct {
	core.ControlNode
	currentChildIdx int
	skippedCount    int
	async           bool
}

func NewFallbackNode(name string, config core.NodeConfig) *FallbackNode {
	n := &FallbackNode{
		currentChildIdx: 0,
		skippedCount:    0,
		async:           false,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Fallback")
	return n
}

func (n *FallbackNode) Tick() core.NodeStatus {
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

		case core.SUCCESS:
			n.ResetChildren()
			n.currentChildIdx = 0
			return childStatus

		case core.FAILURE:
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
	return core.FAILURE
}

func (n *FallbackNode) Halt() {
	n.currentChildIdx = 0
	n.skippedCount = 0
	n.ControlNode.Halt()
}
