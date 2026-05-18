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
	completedList    map[int]struct{}
	failureCount     int
	failureThreshold int
}

func NewParallelAllNode(name string, config core.NodeConfig) *ParallelAllNode {
	n := &ParallelAllNode{
		completedList:    make(map[int]struct{}),
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

	for index := 0; index < childrenCount; index++ {
		childNode := n.Child(index)

		// Already completed
		if _, ok := n.completedList[index]; ok {
			continue
		}

		childStatus := childNode.ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.completedList[index] = struct{}{}

		case core.FAILURE:
			n.completedList[index] = struct{}{}
			n.failureCount++

		case core.RUNNING:
			// Still working. Check the next

		case core.SKIPPED:
			skippedCount++

		case core.IDLE:
			slog.Error("child returned IDLE during Tick; children should not return IDLE")
			return core.FAILURE
		}
	}

	if skippedCount == childrenCount {
		return core.SKIPPED
	}
	if skippedCount+len(n.completedList) >= childrenCount {
		// DONE
		n.HaltChildren()
		n.completedList = make(map[int]struct{})
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
	n.completedList = make(map[int]struct{})
	n.failureCount = 0
	n.ControlNode.Halt()
}
