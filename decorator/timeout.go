package decorator

import (
	"time"

	"github.com/actfuns/behaviortree/core"
)

// TimeoutNode will halt a running child if it has been RUNNING longer than a given time.
// If timeout is reached, the node returns FAILURE.
type TimeoutNode struct {
	core.DecoratorNode
	msec         int
	startTime    time.Time
	timerStarted bool
	childHalted  bool
}

func NewTimeoutNode(name string, config core.NodeConfig) *TimeoutNode {
	n := &TimeoutNode{}
	n.Init(name, config)
	n.SetSelf(n)
	return n
}

func (n *TimeoutNode) Tick() core.NodeStatus {
	// Read msec from ports
	if v, err := core.GetInputTyped[int](n, "msec"); err == nil {
		n.msec = v
	}

	if !n.timerStarted {
		n.timerStarted = true
		n.childHalted = false
		n.startTime = time.Now()
		n.SetStatus(core.RUNNING)
	}

	// Tick the child first so it can complete
	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.timerStarted = false
		n.ResetChild()
		return childStatus
	}

	// Check timeout after ticking child; if exceeded, halt the child
	if !n.childHalted && n.msec > 0 {
		if time.Since(n.startTime) >= time.Duration(n.msec)*time.Millisecond {
			n.childHalted = true
			n.HaltChild()
		}
	}

	if n.childHalted {
		n.timerStarted = false
		return core.FAILURE
	}

	return childStatus
}

func (n *TimeoutNode) Halt() {
	n.timerStarted = false
	n.DecoratorNode.Halt()
}
