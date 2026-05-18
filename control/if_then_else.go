package control

import (
	"log/slog"
)

import "github.com/actfuns/behaviortree/core"

// IfThenElseNode must have exactly 2 or 3 children. This node is NOT reactive.
// First child is the "if" condition.
// If it returns SUCCESS, the second child is executed.
// If it returns FAILURE, the third child is executed (if present), otherwise FAILURE.
type IfThenElseNode struct {
	core.ControlNode
	childIdx int
}

func NewIfThenElseNode(name string, config core.NodeConfig) *IfThenElseNode {
	n := &IfThenElseNode{childIdx: 0}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("IfThenElse")
	return n
}

func (n *IfThenElseNode) Tick() core.NodeStatus {
	childrenCount := n.ChildrenCount()
	if childrenCount != 2 && childrenCount != 3 {
		slog.Error("IfThenElseNode must have either 2 or 3 children")
		return core.FAILURE
	}

	n.SetStatus(core.RUNNING)

	if n.childIdx == 0 {
		conditionStatus := n.Child(0).ExecuteTick()

		if conditionStatus == core.RUNNING {
			return conditionStatus
		}
		if conditionStatus == core.SUCCESS {
			n.childIdx = 1
		} else if conditionStatus == core.FAILURE {
			if childrenCount == 3 {
				n.childIdx = 2
			} else {
				return conditionStatus
			}
		}
	}

	if n.childIdx > 0 {
		status := n.Child(n.childIdx).ExecuteTick()
		if status == core.RUNNING {
			return core.RUNNING
		}
		n.ResetChildren()
		n.childIdx = 0
		return status
	}

	slog.Error("Something unexpected happened in IfThenElseNode")
	return core.FAILURE
}

func (n *IfThenElseNode) Halt() {
	n.childIdx = 0
	n.ControlNode.Halt()
}
