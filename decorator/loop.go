package decorator

import (
	"strconv"
	"strings"

	"github.com/actfuns/behaviortree/core"
)

// LoopNode repeatedly executes its child based on a counter type.
type LoopNode struct {
	core.DecoratorNode
	queue    []int
	idx      int
	queueSet bool
	ifEmpty  string
}

func NewLoopNode(name string, config core.NodeConfig) *LoopNode {
	n := &LoopNode{
		ifEmpty: config.InputPorts["if_empty"],
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Loop")
	return n
}

func (n *LoopNode) Tick() core.NodeStatus {
	// Read queue from input port on first execution
	if !n.queueSet {
		if queueStr, err := core.GetInputTyped[string](n, "queue"); err == nil {
			parts := strings.Split(queueStr, ";")
			n.queue = make([]int, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if v, err := strconv.Atoi(p); err == nil {
					n.queue = append(n.queue, v)
				}
			}
		}
		n.queueSet = true
		n.idx = 0
	}

	n.SetStatus(core.RUNNING)

	// If no queue was configured or queue is empty, check if_empty attribute
	if len(n.queue) == 0 {
		return n.handleEmptyQueue()
	}

	for n.idx < len(n.queue) {
		// Set the current value as output
		if err := n.SetOutput("value", n.queue[n.idx]); err != nil {
			_ = err
		}

		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.idx++
			n.ResetChild()
			if n.idx < len(n.queue) && n.RequiresWakeUp() {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}
		case core.FAILURE:
			n.idx = 0
			n.ResetChild()
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		default:
			n.idx++
			n.ResetChild()
		}
	}

	// Queue exhausted
	n.idx = 0
	n.queue = nil
	n.queueSet = false
	return core.SUCCESS
}

// handleEmptyQueue processes the case where the queue is empty.
// It respects the if_empty attribute (FAILURE, SKIPPED, or SUCCESS/default).
func (n *LoopNode) handleEmptyQueue() core.NodeStatus {
	switch strings.TrimSpace(n.ifEmpty) {
	case "FAILURE":
		return core.FAILURE
	case "SKIPPED":
		return core.SKIPPED
	case "SUCCESS":
		return core.SUCCESS
	default:
		// Default behavior: tick child once (backward compat with basic tests)
		childStatus := n.Child().ExecuteTick()
		if childStatus.IsCompleted() {
			n.ResetChild()
		}
		return childStatus
	}
}

func (n *LoopNode) Halt() {
	n.idx = 0
	n.queue = nil
	n.queueSet = false
	n.DecoratorNode.Halt()
}
