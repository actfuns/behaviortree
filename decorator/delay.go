package decorator

import (
	"time"

	"github.com/actfuns/behaviortree/core"
)

// DelayNode will wait for a specified time before ticking the child.
// While waiting, it returns RUNNING.
type DelayNode struct {
	core.DecoratorNode
	delayMs      int
	startTime    time.Time
	delayStarted bool
}

func NewDelayNode(name string, config core.NodeConfig) *DelayNode {
	n := &DelayNode{}
	n.Init(name, config)
	n.SetSelf(n)
	return n
}

func (n *DelayNode) Tick() core.NodeStatus {
	// Read delay_msec from ports
	if v, err := core.GetInputTyped[int](n, "delay_msec"); err == nil {
		n.delayMs = v
	}

	if !n.delayStarted {
		n.startTime = time.Now()
		n.delayStarted = true
		n.SetStatus(core.RUNNING)
	}

	// Check if enough real time has passed
	if n.delayMs > 0 {
		elapsed := time.Since(n.startTime)
		if elapsed < time.Duration(n.delayMs)*time.Millisecond {
			return core.RUNNING
		}
	}

	// Delay complete, tick the child
	n.EmitWakeUpSignal()
	childStatus := n.Child().ExecuteTick()
	if childStatus.IsCompleted() {
		n.delayStarted = false
		n.ResetChild()
	}
	return childStatus
}

func (n *DelayNode) Halt() {
	n.delayStarted = false
	n.DecoratorNode.Halt()
}
