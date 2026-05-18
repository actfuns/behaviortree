package decorator

import "github.com/actfuns/behaviortree/core"

// RunOnceNode ticks the child only once, then returns SKIPPED on subsequent ticks
// (if then_skip is true, the default) or repeats the same status (if then_skip is false).
type RunOnceNode struct {
	core.DecoratorNode
	alreadyTicked  bool
	returnedStatus core.NodeStatus
}

func NewRunOnceNode(name string, config core.NodeConfig) *RunOnceNode {
	n := &RunOnceNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("RunOnce")
	return n
}

func (n *RunOnceNode) Tick() core.NodeStatus {
	skip := true
	if v, err := core.GetInputTyped[bool](n, "then_skip"); err == nil {
		skip = v
	}

	if n.alreadyTicked {
		if skip {
			return core.SKIPPED
		}
		return n.returnedStatus
	}

	n.SetStatus(core.RUNNING)
	childStatus := n.Child().ExecuteTick()

	if childStatus.IsCompleted() {
		n.alreadyTicked = true
		n.returnedStatus = childStatus
		n.ResetChild()
	}
	return childStatus
}

func (n *RunOnceNode) Halt() {
	n.alreadyTicked = false
	n.returnedStatus = core.IDLE
	n.DecoratorNode.Halt()
}
