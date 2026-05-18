package control

import (
	"github.com/actfuns/behaviortree/core"
)

// ManualSelectorNode allows manual selection of which child to execute.
// The C++ version uses ncurses; in Go we provide a simplified version
// that ticks the first child by default, with support for the
// "repeat_last_selection" port.
type ManualSelectorNode struct {
	core.ControlNode
	runningChildIdx       int
	previouslyExecutedIdx int
}

func NewManualSelectorNode(name string, config core.NodeConfig) *ManualSelectorNode {
	n := &ManualSelectorNode{
		runningChildIdx:       -1,
		previouslyExecutedIdx: -1,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ManualSelector")
	return n
}

func (n *ManualSelectorNode) Tick() core.NodeStatus {
	childrenCount := n.ChildrenCount()

	if childrenCount == 0 {
		return core.FAILURE
	}

	repeatLast := false
	if v, err := core.GetInputTyped[bool](n, "repeat_last_selection"); err == nil {
		repeatLast = v
	}

	idx := 0

	if repeatLast && n.previouslyExecutedIdx >= 0 {
		idx = n.previouslyExecutedIdx
	} else {
		n.SetStatus(core.RUNNING)
		idx = 0 // Default: tick the first child
		n.previouslyExecutedIdx = idx
	}

	ret := n.Child(idx).ExecuteTick()
	if ret == core.RUNNING {
		n.runningChildIdx = idx
	}
	return ret
}

func (n *ManualSelectorNode) Halt() {
	if n.runningChildIdx >= 0 {
		n.HaltChild(n.runningChildIdx)
	}
	n.runningChildIdx = -1
	n.ControlNode.Halt()
}

// ManualNode is an alias for ManualSelectorNode with registration ID "Manual".
type ManualNode struct {
	ManualSelectorNode
}

func NewManualNode(name string, config core.NodeConfig) *ManualNode {
	n := &ManualNode{}
	n.ManualSelectorNode.runningChildIdx = -1
	n.ManualSelectorNode.previouslyExecutedIdx = -1
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Manual")
	return n
}
