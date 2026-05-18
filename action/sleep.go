package action

import (
	"log/slog"
	"sync"
	"time"

	"github.com/actfuns/behaviortree/core"
)

// SleepNode sleeps for a specified amount of time (msec port).
type SleepNode struct {
	core.StatefulActionNode
	msec         int
	timerID      uint64
	timer        *core.TimerQueue
	timerWaiting bool
	delayMutex   sync.Mutex
}

func NewSleepNode(name string, config core.NodeConfig) *SleepNode {
	n := &SleepNode{
		timer: core.NewTimerQueue(),
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Sleep")
	return n
}

func (n *SleepNode) OnStart() core.NodeStatus {
	msec := 0
	if err := n.GetInput("msec", &msec); err != nil {
		slog.Error("missing parameter [msec] in SleepNode")
		return core.FAILURE
	}

	if msec <= 0 {
		return core.SUCCESS
	}

	n.SetStatus(core.RUNNING)
	n.timerWaiting = true

	n.timerID = n.timer.Add(time.Duration(msec)*time.Millisecond, func(aborted bool) {
		n.delayMutex.Lock()
		if !aborted {
			n.EmitWakeUpSignal()
		}
		n.timerWaiting = false
		n.delayMutex.Unlock()
	})

	return core.RUNNING
}

func (n *SleepNode) OnRunning() core.NodeStatus {
	// Process expired timers to update timerWaiting state
	n.timer.ProcessExpired()

	if n.timerWaiting {
		return core.RUNNING
	}
	return core.SUCCESS
}

func (n *SleepNode) OnHalted() {
	n.timerWaiting = false
	n.timer.Cancel(n.timerID)
}

// Tick dispatches to OnStart or OnRunning based on the current status.
// This is necessary because Go's static dispatch prevents StatefulActionNode.Tick()
// from calling the overridden OnStart/OnRunning on the embedding struct.
func (n *SleepNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}
