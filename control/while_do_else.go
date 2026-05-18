package control

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// WhileDoElseNode must have exactly 2 or 3 children.
// It is a REACTIVE node of IfThenElseNode.
//
// The first child is the "while" condition.
// - If it returns SUCCESS, the second child is executed (the "do" part).
// - If it returns FAILURE, the third child is executed (if present), otherwise FAILURE.
// - While the condition returns RUNNING, the entire node returns RUNNING.
//
// If the 2nd or 3rd child is RUNNING and the condition changes,
// the RUNNING child will be stopped before starting the sibling.
type WhileDoElseNode struct {
	core.ControlNode
}

func NewWhileDoElseNode(name string, config core.NodeConfig) *WhileDoElseNode {
	n := &WhileDoElseNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("WhileDoElse")
	return n
}

func (n *WhileDoElseNode) Tick() core.NodeStatus {
	childrenCount := n.ChildrenCount()
	if childrenCount != 2 && childrenCount != 3 {
		slog.Error("WhileDoElseNode must have either 2 or 3 children")
		return core.FAILURE
	}

	n.SetStatus(core.RUNNING)

	conditionStatus := n.Child(0).ExecuteTick()

	if conditionStatus == core.RUNNING {
		return conditionStatus
	}

	status := core.IDLE

	if conditionStatus == core.SUCCESS {
		if childrenCount == 3 {
			n.HaltChild(2)
		}
		status = n.Child(1).ExecuteTick()
	} else if conditionStatus == core.FAILURE {
		if childrenCount == 3 {
			n.HaltChild(1)
			status = n.Child(2).ExecuteTick()
		} else if childrenCount == 2 {
			status = core.FAILURE
		}
	}

	if status == core.RUNNING {
		return core.RUNNING
	}
	n.ResetChildren()
	return status
}

func (n *WhileDoElseNode) Halt() {
	n.ControlNode.Halt()
}
