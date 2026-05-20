package core_test

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

// Motor is an interface for testing interface-based dependencies.
type Motor interface {
	TurnOn() string
}

// LinearMotor implements the Motor interface.
type LinearMotor struct{}

func (m *LinearMotor) TurnOn() string { return "linear on" }

// PathFollowNode is a StatefulActionNode that uses a Motor interface.
type PathFollowNode struct {
	core.StatefulActionNode
	motor Motor
}

func (n *PathFollowNode) OnStart() core.NodeStatus {
	n.motor.TurnOn()
	return core.RUNNING
}

func (n *PathFollowNode) OnRunning() core.NodeStatus {
	n.motor.TurnOn()
	return core.SUCCESS
}

func (n *PathFollowNode) OnHalted() {}

// Tick dispatches to OnStart or OnRunning based on the current status.
func (n *PathFollowNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

func TestFactory_VirtualInterface(t *testing.T) {
	motor := &LinearMotor{}
	factory := factory.NewBehaviorTreeFactory()

	motorUsed := false
	_ = factory.RegisterNodeType("PathFollow", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &PathFollowNode{motor: motor}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("PathFollow")
			motorUsed = true
			return n
		}, core.Action)

	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="root_sequence">
				<PathFollow/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if !motorUsed {
		t.Error("PathFollow was not created with motor dependency")
	}
}
