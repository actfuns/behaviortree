package control

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// ParallelAllNode executes all its children concurrently (but not in separate threads).
// It differs from ParallelNode because the latter may stop and halt other children
// if a certain number of SUCCESS/FAILURES is reached, whilst this one will always
// complete the execution of ALL its children.
type ParallelAllNode struct {
	core.ControlNode
	completedList    []bool
	completedCount   int
	failureCount     int
	failureThreshold int
}

func NewParallelAllNode(name string, config core.NodeConfig) *ParallelAllNode {
	n := &ParallelAllNode{
		completedList:    nil,
		failureThreshold: 1,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ParallelAll")
	return n
}

func (n *ParallelAllNode) FailureThreshold() int {
	return n.failureThreshold
}

func (n *ParallelAllNode) SetFailureThreshold(threshold int) {
	if threshold < 0 {
		n.failureThreshold = n.resolveThreshold(threshold, n.ChildrenCount())
	} else {
		n.failureThreshold = threshold
	}
}

func (n *ParallelAllNode) Tick() core.NodeStatus {
	maxFailures := 1
	if v, err := core.GetInputTyped[int](n, "max_failures"); err == nil {
		maxFailures = v
	}
	childrenCount := n.ChildrenCount()
	n.SetFailureThreshold(maxFailures)

	if childrenCount < n.failureThreshold {
		slog.Error("Number of children is less than threshold. Can never fail.")
		return core.FAILURE
	}

	n.SetStatus(core.RUNNING)

	skippedCount := 0

	if len(n.completedList) != childrenCount {
		n.completedList = make([]bool, childrenCount)
	}

	for index := 0; index < childrenCount; index++ {
		childNode := n.Child(index)

		// Already completed
		if n.completedList[index] {
			continue
		}

		childStatus := childNode.ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.completedList[index] = true
			n.completedCount++

		case core.FAILURE:
			n.completedList[index] = true
			n.completedCount++
			n.failureCount++

		case core.RUNNING:
			// Still working. Check the next

		case core.SKIPPED:
			skippedCount++

		case core.IDLE:
			panic(core.NewLogicError("child returned IDLE during Tick"))

		}
	}

	if skippedCount == childrenCount {
		return core.SKIPPED
	}
	if skippedCount+n.completedCount >= childrenCount {
		// DONE
		n.HaltChildren()
		clear(n.completedList)
		n.completedCount = 0
		status := core.SUCCESS
		if n.failureCount >= n.failureThreshold {
			status = core.FAILURE
		}
		n.failureCount = 0
		return status
	}

	// Some children haven't finished yet
	return core.RUNNING
}

func (n *ParallelAllNode) resolveThreshold(threshold int, childrenCount int) int {
	if threshold < 0 {
		result := childrenCount + threshold + 1
		if result < 0 {
			return 0
		}
		return result
	}
	return threshold
}

func (n *ParallelAllNode) Halt() {
	clear(n.completedList)
	n.completedCount = 0
	n.failureCount = 0
	n.ControlNode.Halt()
}
