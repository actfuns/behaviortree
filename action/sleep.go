package action

import (
	"log/slog"
	"sync"

	"github.com/actfuns/behaviortree/core"
)

// SleepNode sleeps for a specified amount of time (msec port).
// Uses a background timer that emits a wake-up signal when the time expires,
// matching C++ BehaviorTree.CPP behavior.
type SleepNode struct {
	core.StatefulActionNode
	msec         int
	timerWaiting bool
	timerID      core.TimerID
	mu           sync.Mutex
}

func NewSleepNode(name string, config core.NodeConfig) *SleepNode {
	n := &SleepNode{}
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

	n.msec = msec
	n.mu.Lock()
	n.timerWaiting = true
	n.mu.Unlock()
	n.SetStatus(core.RUNNING)

	n.timerID = n.TimerQueue().Add(
		core.DurationFromMS(msec),
		func(aborted bool) {
			n.mu.Lock()
			n.timerWaiting = false
			n.mu.Unlock()
			if !aborted {
				n.EmitWakeUpSignal()
			}
		},
	)

	return core.RUNNING
}

func (n *SleepNode) OnRunning() core.NodeStatus {
	n.mu.Lock()
	waiting := n.timerWaiting
	n.mu.Unlock()

	if waiting {
		return core.RUNNING
	}
	return core.SUCCESS
}

func (n *SleepNode) OnHalted() {
	n.TimerQueue().Cancel(n.timerID)
	n.mu.Lock()
	n.timerWaiting = false
	n.mu.Unlock()
}

// Tick dispatches to OnStart or OnRunning based on the current status.
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
