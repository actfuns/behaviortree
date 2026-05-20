package control

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// ParallelNode executes all children concurrently (not in separate threads).
// It completes when the THRESHOLD_SUCCESS or THRESHOLD_FAILURE is reached.
// Negative thresholds are python-style indices (-1 = number of children).
type ParallelNode struct {
	core.ControlNode
	successThreshold       int
	failureThreshold       int
	completedList          []bool
	successCount           int
	failureCount           int
	readParameterFromPorts bool
}

func NewParallelNode(name string, config core.NodeConfig) *ParallelNode {
	n := &ParallelNode{
		successThreshold:       -1,
		failureThreshold:       1,
		completedList:          nil,
		readParameterFromPorts: false,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Parallel")

	// Check if success_count/failure_count ports are set on the config
	if _, hasSuccess := config.InputPorts["success_count"]; hasSuccess {
		n.readParameterFromPorts = true
	}
	if _, hasFailure := config.InputPorts["failure_count"]; hasFailure {
		n.readParameterFromPorts = true
	}

	return n
}

func (n *ParallelNode) SuccessThreshold() int {
	if n.successThreshold < 0 {
		val := n.ChildrenCount() + n.successThreshold + 1
		if val < 0 {
			return 0
		}
		return val
	}
	return n.successThreshold
}

func (n *ParallelNode) FailureThreshold() int {
	if n.failureThreshold < 0 {
		val := n.ChildrenCount() + n.failureThreshold + 1
		if val < 0 {
			return 0
		}
		return val
	}
	return n.failureThreshold
}

func (n *ParallelNode) SetSuccessThreshold(threshold int) {
	n.successThreshold = threshold
}

func (n *ParallelNode) SetFailureThreshold(threshold int) {
	n.failureThreshold = threshold
}

func (n *ParallelNode) Tick() core.NodeStatus {
	if n.readParameterFromPorts {
		if v, err := core.GetInputTyped[int](n, "success_count"); err == nil {
			n.successThreshold = v
		} else {
			slog.Error("Missing parameter [success_count] in ParallelNode")
			return core.FAILURE
		}
		if v, err := core.GetInputTyped[int](n, "failure_count"); err == nil {
			n.failureThreshold = v
		} else {
			slog.Error("Missing parameter [failure_count] in ParallelNode")
			return core.FAILURE
		}
	}

	childrenCount := n.ChildrenCount()

	if childrenCount < n.SuccessThreshold() {
		slog.Error("Number of children is less than threshold. Can never succeed.")
		return core.FAILURE
	}
	if childrenCount < n.FailureThreshold() {
		slog.Error("Number of children is less than threshold. Can never fail.")
		return core.FAILURE
	}

	n.SetStatus(core.RUNNING)

	skippedCount := 0

	if len(n.completedList) != childrenCount {
		n.completedList = make([]bool, childrenCount)
	}

	for i := 0; i < childrenCount; i++ {
		if n.completedList[i] {
			continue
		}

		childNode := n.Child(i)
		childStatus := childNode.ExecuteTick()

		switch childStatus {
		case core.SKIPPED:
			skippedCount++

		case core.SUCCESS:
			n.completedList[i] = true
			n.successCount++

		case core.FAILURE:
			n.completedList[i] = true
			n.failureCount++

		case core.RUNNING:
			// Still working, check next

		case core.IDLE:
			panic(core.NewLogicError("child returned IDLE during Tick"))

		}

		requiredSuccess := n.SuccessThreshold()

		if n.successCount >= requiredSuccess ||
			(n.successThreshold < 0 && (n.successCount+skippedCount) >= requiredSuccess) {
			n.clear()
			n.ResetChildren()
			return core.SUCCESS
		}

		if ((childrenCount - n.failureCount) < requiredSuccess) ||
			(n.failureCount == n.FailureThreshold()) {
			n.clear()
			n.ResetChildren()
			return core.FAILURE
		}
	}

	if skippedCount == childrenCount {
		return core.SKIPPED
	}
	return core.RUNNING
}

func (n *ParallelNode) clear() {
	clear(n.completedList)
	n.successCount = 0
	n.failureCount = 0
}

func (n *ParallelNode) Halt() {
	n.clear()
	n.ControlNode.Halt()
}
