package decorator

import (
	"sync"

	"github.com/actfuns/behaviortree/core"
)

// TimeoutNode will halt a running child if it has been RUNNING longer than a given time.
// If timeout is reached, the node returns FAILURE.
//
// Uses a TimerQueue so the timeout fires from a background goroutine and sends a
// wake-up signal. On the next tick, the node halts the child and returns FAILURE,
// matching C++ BehaviorTree.CPP behavior.
type TimeoutNode struct {
	core.DecoratorNode
	msec         int
	timerStarted bool
	childHalted  bool
	timerID      core.TimerID
	mu           sync.Mutex
}

func NewTimeoutNode(name string, config core.NodeConfig) *TimeoutNode {
	n := &TimeoutNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Timeout")
	return n
}

func (n *TimeoutNode) Tick() core.NodeStatus {
	// Read msec from ports
	if v, err := core.GetInputTyped[int](n, "msec"); err == nil {
		n.mu.Lock()
		n.msec = v
		n.mu.Unlock()
	}

	n.mu.Lock()
	halted := n.childHalted
	n.mu.Unlock()

	// If timer already fired and halted the child, return FAILURE immediately
	// without ticking the child. Matches C++: check child_halted_ before executeTick.
	if halted {
		n.mu.Lock()
		n.timerStarted = false
		n.mu.Unlock()
		n.HaltChild()
		return core.FAILURE
	}

	// Start timer on first time through
	n.mu.Lock()
	if !n.timerStarted {
		n.timerStarted = true
		n.childHalted = false
		n.mu.Unlock()
		n.SetStatus(core.RUNNING)

		if n.msec > 0 {
			n.timerID = n.TimerQueue().Add(
				core.DurationFromMS(n.msec),
				func(aborted bool) {
					if !aborted {
						n.mu.Lock()
						n.childHalted = true
						n.mu.Unlock()
						n.EmitWakeUpSignal()
					}
				},
			)
		}
		// Fall through to tick the child on this same tick (matches C++)
	} else {
		n.mu.Unlock()
	}

	// Tick the child
	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.TimerQueue().Cancel(n.timerID)
		n.mu.Lock()
		n.timerStarted = false
		n.mu.Unlock()
		n.ResetChild()
		return childStatus
	}

	return childStatus
}

func (n *TimeoutNode) Halt() {
	n.TimerQueue().Cancel(n.timerID)
	n.mu.Lock()
	n.timerStarted = false
	n.childHalted = false
	n.mu.Unlock()
	n.DecoratorNode.Halt()
}
