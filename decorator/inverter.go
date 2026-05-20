package decorator

import "github.com/actfuns/behaviortree/core"

// InverterNode returns SUCCESS if child fails, FAILURE if child succeeds.
// RUNNING status is propagated.
type InverterNode struct {
	core.DecoratorNode
}

func NewInverterNode(name string, config core.NodeConfig) *InverterNode {
	n := &InverterNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Inverter")
	return n
}

func (n *InverterNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)
	childStatus := n.Child().ExecuteTick()

	switch childStatus {
	case core.SUCCESS:
		n.ResetChild()
		return core.FAILURE

	case core.FAILURE:
		n.ResetChild()
		return core.SUCCESS

	case core.RUNNING, core.SKIPPED:
		return childStatus

	case core.IDLE:
		panic(core.NewLogicError("child returned IDLE during Tick"))
	}

	return n.Status()
}
