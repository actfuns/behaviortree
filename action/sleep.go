package action

import (
	"log/slog"
	"time"

	"github.com/actfuns/behaviortree/core"
)

// SleepNode sleeps for a specified amount of time (msec port).
type SleepNode struct {
	core.StatefulActionNode
	msec         int
	startTime    time.Time
	timerWaiting bool
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

	n.startTime = time.Now()
	n.msec = msec
	n.timerWaiting = true
	n.SetStatus(core.RUNNING)
	return core.RUNNING
}

func (n *SleepNode) OnRunning() core.NodeStatus {
	if time.Since(n.startTime) < time.Duration(n.msec)*time.Millisecond {
		return core.RUNNING
	}
	n.timerWaiting = false
	n.EmitWakeUpSignal()
	return core.SUCCESS
}

func (n *SleepNode) OnHalted() {
	n.timerWaiting = false
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
