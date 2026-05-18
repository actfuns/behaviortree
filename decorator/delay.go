package decorator

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/actfuns/behaviortree/core"
)

// DelayNode will wait for a specified time before ticking the child.
// While waiting, it returns RUNNING.
type DelayNode struct {
	core.DecoratorNode
	delayMs        int
	timerID        uint64
	timer          *core.TimerQueue
	delayStarted   bool
	delayCompleted atomic.Bool
	delayAborted   bool
	mu             sync.Mutex
}

func NewDelayNode(name string, config core.NodeConfig) *DelayNode {
	n := &DelayNode{
		delayMs: 0,
		timer:   core.NewTimerQueue(),
	}
	n.Init(name, config)
	n.SetSelf(n)
	return n
}

func (n *DelayNode) Tick() core.NodeStatus {
	// Read delay_msec from ports
	if v, err := core.GetInputTyped[int](n, "delay_msec"); err == nil {
		n.delayMs = v
	}

	n.mu.Lock()

	if !n.delayStarted {
		n.delayCompleted.Store(false)
		n.delayAborted = false
		n.delayStarted = true
		n.SetStatus(core.RUNNING)

		n.timerID = n.timer.Add(time.Duration(n.delayMs)*time.Millisecond, func(aborted bool) {
			n.mu.Lock()
			n.delayCompleted.Store(!aborted)
			if !aborted {
				n.EmitWakeUpSignal()
			}
			n.mu.Unlock()
		})
	}

	if n.delayAborted {
		n.delayAborted = false
		n.delayStarted = false
		n.mu.Unlock()
		return core.FAILURE
	}

	n.mu.Unlock()

	// Process expired timers outside the lock, since the callback acquires n.mu
	n.timer.ProcessExpired()

	if !n.delayCompleted.Load() {
		return core.RUNNING
	}

	// Delay complete, tick the child
	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.delayStarted = false
		n.delayAborted = false
		n.ResetChild()
	}
	return childStatus
}

func (n *DelayNode) Halt() {
	n.mu.Lock()
	n.delayStarted = false
	n.mu.Unlock()
	n.timer.CancelAll()
	n.DecoratorNode.Halt()
}
