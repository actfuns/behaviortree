package control

import (
	"fmt"
	"log/slog"
	"math"
	"strconv"

	"github.com/actfuns/behaviortree/core"
)

// SwitchNode is equivalent to a switch statement, where a certain
// branch (child) is executed according to the value of a blackboard entry.
// It requires N children for cases plus one default child (total = N + 1).
type SwitchNode struct {
	core.ControlNode
	runningChild int
}

// NewSwitchNode creates a SwitchNode.
func NewSwitchNode(name string, config core.NodeConfig) *SwitchNode {
	n := &SwitchNode{
		runningChild: -1,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Switch")
	return n
}

func (n *SwitchNode) Tick() core.NodeStatus {
	numCases := n.ChildrenCount() - 1
	if numCases < 1 {
		slog.Error("SwitchNode requires at least 2 children (1 case + 1 default)")
		return core.FAILURE
	}

	matchIndex := numCases // default index

	// No variable? Jump to default
	if variable, err := core.GetInputTyped[string](n, "variable"); err == nil {
		// Check each case until you find a match
		for index := 0; index < numCases; index++ {
			caseKey := fmt.Sprintf("case_%d", index+1)
			if value, err := core.GetInputTyped[string](n, caseKey); err == nil {
				if checkStringEquality(variable, value, n.Config().Enums) {
					matchIndex = index
					break
				}
			}
		}
	}

	// If another one was running earlier, halt it
	if n.runningChild != -1 && n.runningChild != matchIndex {
		n.HaltChild(n.runningChild)
	}

	selectedChild := n.Child(matchIndex)
	if selectedChild == nil {
		return core.FAILURE
	}
	ret := selectedChild.ExecuteTick()
	switch ret {
	case core.SKIPPED:
		n.runningChild = -1
		return core.SKIPPED
	case core.RUNNING:
		n.runningChild = matchIndex
	default:
		n.ResetChildren()
		n.runningChild = -1
	}
	return ret
}

func (n *SwitchNode) Halt() {
	n.runningChild = -1
	n.ControlNode.Halt()
}

// checkStringEquality compares two strings checking for literal, integer,
// and real-number equality (same as C++ version).
func checkStringEquality(v1, v2 string, enums *core.ScriptingEnumsRegistry) bool {
	// Compare strings first
	if v1 == v2 {
		return true
	}

	// Compare as integers next
	toInt := func(str string) (int, bool) {
		if enums != nil {
			if val, ok := (*enums)[str]; ok {
				return val, true
			}
		}
		val, err := strconv.Atoi(str)
		if err != nil {
			return 0, false
		}
		return val, true
	}

	if v1Int, ok1 := toInt(v1); ok1 {
		if v2Int, ok2 := toInt(v2); ok2 && v1Int == v2Int {
			return true
		}
	}

	// Compare as real numbers next
	toReal := func(str string) (float64, bool) {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return 0, false
		}
		return val, true
	}

	if v1Real, ok1 := toReal(v1); ok1 {
		if v2Real, ok2 := toReal(v2); ok2 {
			const eps = 1.1920928955078125e-07 // float32 epsilon
			return math.Abs(v1Real-v2Real) <= eps
		}
	}

	return false
}
