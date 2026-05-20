package decorator

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// RetryNode executes a child several times if it fails (RetryUntilSuccessful).
// If child returns SUCCESS, the loop stops and this node returns SUCCESS.
// If child returns FAILURE, it tries again up to N times.
type RetryNode struct {
	core.DecoratorNode
	maxAttempts int
	tryCount    int
}

const numAttempts = "num_attempts"

func NewRetryNode(name string, config core.NodeConfig) *RetryNode {
	n := &RetryNode{
		maxAttempts: 3,
		tryCount:    0,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("RetryUntilSuccessful")
	return n
}

func (n *RetryNode) Tick() core.NodeStatus {
	// Read num_attempts from ports, fall back to internal value if not set via ports
	if v, err := core.GetInputTyped[int](n, numAttempts); err == nil {
		n.maxAttempts = v
	}

	doLoop := n.tryCount < n.maxAttempts || n.maxAttempts == -1
	n.SetStatus(core.RUNNING)

	for doLoop {
		prevStatus := n.Child().Status()
		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.tryCount = 0
			n.ResetChild()
			return core.SUCCESS

		case core.FAILURE:
			n.tryCount++
			if v, err := core.GetInputTyped[int](n, numAttempts); err == nil {
				n.maxAttempts = v
			}
			doLoop = n.tryCount < n.maxAttempts || n.maxAttempts == -1
			n.ResetChild()

			if n.RequiresWakeUp() && prevStatus == core.IDLE && doLoop {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}

		case core.RUNNING:
			return core.RUNNING

		case core.SKIPPED:
			n.ResetChild()
			return core.SKIPPED

		case core.IDLE:
			slog.Error("child returned IDLE during Tick; children should not return IDLE")
			return core.FAILURE
		}
	}

	n.tryCount = 0
	return core.FAILURE
}

func (n *RetryNode) Halt() {
	n.tryCount = 0
	n.DecoratorNode.Halt()
}

// RetryNodeTypo is provided for backward compatibility with the deprecated spelling.
type RetryNodeTypo struct {
	*RetryNode
}
