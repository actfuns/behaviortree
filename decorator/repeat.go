package decorator

import "github.com/actfuns/behaviortree/core"

// RepeatNode executes a child several times as long as it succeeds.
// If child returns FAILURE, the loop stops and this node returns FAILURE.
// If child returns SUCCESS N times, this node returns SUCCESS.
type RepeatNode struct {
	core.DecoratorNode
	numCycles   int
	repeatCount int
}

const numCycles = "num_cycles"

func NewRepeatNode(name string, config core.NodeConfig) *RepeatNode {
	n := &RepeatNode{
		numCycles:   1,
		repeatCount: 0,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Repeat")
	return n
}

func (n *RepeatNode) Tick() core.NodeStatus {
	// Read num_cycles from ports, fall back to internal if not set via ports
	if v, err := core.GetInputTyped[int](n, numCycles); err == nil {
		n.numCycles = v
	}

	doLoop := n.repeatCount < n.numCycles || n.numCycles == -1
	n.SetStatus(core.RUNNING)

	for doLoop {
		prevStatus := n.Child().Status()
		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.repeatCount++
			doLoop = n.repeatCount < n.numCycles || n.numCycles == -1
			n.ResetChild()

			if n.RequiresWakeUp() && prevStatus == core.IDLE && doLoop {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}

		case core.FAILURE:
			n.repeatCount = 0
			n.ResetChild()
			return core.FAILURE

		case core.RUNNING:
			return core.RUNNING

		case core.SKIPPED:
			n.ResetChild()
			return core.SKIPPED

		case core.IDLE:
			panic(core.NewLogicError("child returned IDLE during Tick"))

		}
	}

	n.repeatCount = 0
	return core.SUCCESS
}

func (n *RepeatNode) Halt() {
	n.repeatCount = 0
	n.DecoratorNode.Halt()
}
