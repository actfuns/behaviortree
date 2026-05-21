package decorator

import (
	"sync"

	"github.com/actfuns/behaviortree/core"
)

// DelayNode will wait for a specified time before ticking the child.
// While waiting, it returns RUNNING.
// Uses a background timer that emits a wake-up signal when the delay expires,
// matching C++ BehaviorTree.CPP behavior.
type DelayNode struct {
	core.DecoratorNode
	delayMs       int
	delayStarted  bool
	delayComplete bool
	timerID       core.TimerID
	mu            sync.Mutex
}

func NewDelayNode(name string, config core.NodeConfig) *DelayNode {
	n := &DelayNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Delay")
	return n
}

func (n *DelayNode) Tick() core.NodeStatus {
	// Read delay_msec from ports
	if v, err := core.GetInputTyped[int](n, "delay_msec"); err == nil {
		n.mu.Lock()
		n.delayMs = v
		n.mu.Unlock()
	}

	n.mu.Lock()
	delayComplete := n.delayComplete
	n.mu.Unlock()

	if delayComplete {
		// Delay finished, tick the child
		n.mu.Lock()
		n.delayStarted = false
		n.delayComplete = false
		n.mu.Unlock()

		childStatus := n.Child().ExecuteTick()
		if childStatus.IsCompleted() {
			n.ResetChild()
		}
		return childStatus
	}

	if !n.delayStarted {
		n.mu.Lock()
		n.delayStarted = true
		n.delayComplete = false
		n.mu.Unlock()
		n.SetStatus(core.RUNNING)

		if n.delayMs > 0 {
			n.timerID = n.TimerQueue().Add(
				core.DurationFromMS(n.delayMs),
				func(aborted bool) {
					n.mu.Lock()
					n.delayComplete = !aborted
					n.mu.Unlock()
					if !aborted {
						n.EmitWakeUpSignal()
					}
				},
			)
		} else {
			// Zero delay: complete immediately
			n.mu.Lock()
			n.delayComplete = true
			n.mu.Unlock()
		}
	}

	return core.RUNNING
}

func (n *DelayNode) Halt() {
	n.TimerQueue().Cancel(n.timerID)
	n.mu.Lock()
	n.delayStarted = false
	n.delayComplete = false
	n.mu.Unlock()
	n.DecoratorNode.Halt()
}
