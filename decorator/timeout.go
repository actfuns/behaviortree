package decorator

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/actfuns/behaviortree/core"
)

// TimeoutNode will halt a running child if it has been RUNNING longer than a given time.
// If timeout is reached, the node returns FAILURE.
type TimeoutNode struct {
	core.DecoratorNode
	msec           int
	childHalted    atomic.Bool
	timerID        uint64
	timer          *core.TimerQueue
	timeoutStarted atomic.Bool
	mu             sync.Mutex
}

func NewTimeoutNode(name string, config core.NodeConfig) *TimeoutNode {
	n := &TimeoutNode{
		msec:  0,
		timer: core.NewTimerQueue(),
	}
	n.Init(name, config)
	n.SetSelf(n)
	return n
}

func (n *TimeoutNode) Tick() core.NodeStatus {
	// Read msec from ports
	if v, err := core.GetInputTyped[int](n, "msec"); err == nil {
		n.msec = v
	}

	n.mu.Lock()

	if !n.timeoutStarted.Load() {
		n.timeoutStarted.Store(true)
		n.SetStatus(core.RUNNING)
		n.childHalted.Store(false)

		if n.msec > 0 {
			n.timerID = n.timer.Add(time.Duration(n.msec)*time.Millisecond, func(aborted bool) {
				if aborted {
					return
				}
				n.mu.Lock()
				if n.Child() != nil && n.Child().Status() == core.RUNNING {
					n.childHalted.Store(true)
					n.HaltChild()
					n.EmitWakeUpSignal()
				}
				n.mu.Unlock()
			})
		}
	}

	halted := n.childHalted.Load()
	n.mu.Unlock()

	if halted {
		n.timeoutStarted.Store(false)
		return core.FAILURE
	}

	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.timeoutStarted.Store(false)
		n.timer.Cancel(n.timerID)
		n.ResetChild()
	}
	return childStatus
}

func (n *TimeoutNode) Halt() {
	n.timeoutStarted.Store(false)
	n.timer.CancelAll()
	n.DecoratorNode.Halt()
}
