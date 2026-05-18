package control

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// --------------------------------------------------------------------
// ReactiveBackchaining tests
//
// These tests verify the "backchaining" pattern using ReactiveFallback
// and ReactiveSequence. The pattern is:
//   ReactiveFallback {
//     <condition>           // goal condition
//     ReactiveSequence {
//       <sub-condition>     // prerequisite
//       <action>            // action that achieves the goal
//     }
//   }
// --------------------------------------------------------------------

// simpleBBCondition returns a SimpleCondition that reads a bool from the
// blackboard via an input port "value" and returns SUCCESS if true.
func registerBBCondition(factory *core.BehaviorTreeFactory, name, portName string) {
	_ = factory.RegisterNodeType(name, core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}, func(nName string, config core.NodeConfig) core.TreeNode {
		n := &bbConditionNode{
			portName: portName,
		}
		n.Init(nName, config)
		n.SetSelf(n)
		n.SetRegistrationID(name)
		return n
	}, core.Condition)
}

type bbConditionNode struct {
	core.ConditionNode
	portName string
}

func (n *bbConditionNode) Tick() core.NodeStatus {
	var val bool
	if err := n.GetInput("value", &val); err == nil && val {
		return core.SUCCESS
	}
	return core.FAILURE
}

// bbSetAction is a StatefulActionNode that runs for a configurable number
// of ticks (successTicks) and then sets a blackboard entry to true and
// returns SUCCESS.
type bbSetAction struct {
	core.StatefulActionNode
	blackboardKey string
	tickCount     int
	successTicks  int
}

func (n *bbSetAction) OnStart() core.NodeStatus {
	n.tickCount = 0
	return core.RUNNING
}

func (n *bbSetAction) OnRunning() core.NodeStatus {
	n.tickCount++
	if n.tickCount >= n.successTicks {
		if err := n.SetOutput("value", true); err != nil {
			_ = err
		}
		return core.SUCCESS
	}
	return core.RUNNING
}

func (n *bbSetAction) OnHalted() {
}

func (n *bbSetAction) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

func registerBBSetAction(factory *core.BehaviorTreeFactory, name, blackboardKey string, successTicks int) {
	_ = factory.RegisterNodeType(name, core.PortsList{
		"value": core.NewPortInfo(core.OUTPUT),
	}, func(nName string, config core.NodeConfig) core.TreeNode {
		n := &bbSetAction{
			blackboardKey: blackboardKey,
			successTicks:  successTicks,
		}
		n.Init(nName, config)
		n.SetSelf(n)
		n.SetRegistrationID(name)
		// Set the output port to the blackboard key
		config.OutputPorts["value"] = "{" + blackboardKey + "}"
		n.SetSelf(n)
		return n
	}, core.Action)
}

// RegisterStandardBackchainingNodes registers all node types needed for
// reactive backchaining tests.
func RegisterStandardBackchainingNodes(factory *core.BehaviorTreeFactory) {
	// IsWarm condition: reads from "is_warm" blackboard entry
	registerBBCondition(factory, "IsWarm", "is_warm")

	// IsHoldingJacket condition: reads from "holding_jacket" blackboard
	registerBBCondition(factory, "IsHoldingJacket", "holding_jacket")

	// IsNearCloset condition: reads from "near_closet" blackboard
	registerBBCondition(factory, "IsNearCloset", "near_closet")

	// WearJacket action: after 2 ticks sets "is_warm" to true
	registerBBSetAction(factory, "WearJacket", "is_warm", 2)

	// GrabJacket action: after 2 ticks sets "holding_jacket" to true
	registerBBSetAction(factory, "GrabJacket", "holding_jacket", 2)
}

// ====================================================================
// TestReactiveBackchaining_EnsureWarm
//
// Basic PPA (Postcondition-Precondition-Action) test.
// ReactiveFallback checks IsWarm condition first. If not warm, it tries
// a ReactiveSequence that checks IsHoldingJacket and then WearJacket.
// WearJacket runs asynchronously for 2 ticks before succeeding.
// ====================================================================

func TestReactiveBackchaining_EnsureWarm(t *testing.T) {
	xmlText := `
    <root BTCPP_format="4">
      <BehaviorTree ID="EnsureWarm">
        <ReactiveFallback>
          <IsWarm name="warm" value="{is_warm}"/>
          <ReactiveSequence>
            <IsHoldingJacket name="jacket" value="{holding_jacket}"/>
            <WearJacket name="wear" value="{is_warm}"/>
          </ReactiveSequence>
        </ReactiveFallback>
      </BehaviorTree>
    </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardControlNodes(factory)
	RegisterStandardBackchainingNodes(factory)

	bb := core.NewBlackboard(nil)
	bb.Set("is_warm", false)
	bb.Set("holding_jacket", true)

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	// first tick: not warm, have a jacket: start wearing it
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: want RUNNING, got %v", status)
	}
	var isWarm bool
	bb.Get("is_warm", &isWarm)
	if isWarm {
		t.Error("tick 1: is_warm should be false")
	}

	// second tick: not warm (still wearing)
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 2: want RUNNING, got %v", status)
	}
	bb.Get("is_warm", &isWarm)
	if isWarm {
		t.Error("tick 2: is_warm should be false")
	}

	// third tick: warm (wearing succeeded)
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 3: want SUCCESS, got %v", status)
	}
	bb.Get("is_warm", &isWarm)
	if !isWarm {
		t.Error("tick 3: is_warm should be true")
	}

	// fourth tick: still warm (just the condition ticked)
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 4: want SUCCESS, got %v", status)
	}
}

// ====================================================================
// TestReactiveBackchaining_EnsureWarmWithEnsureHoldingJacket
//
// More complex test with nested backchaining.
// EnsureWarm backchains on holding jacket:
//   EnsureWarm -> EnsureHoldingJacket (subtree) -> WearJacket
// EnsureHoldingJacket:
//   IsHoldingJacket -> GrabJacket (if near closet)
// ====================================================================

func TestReactiveBackchaining_EnsureWarmWithEnsureHoldingJacket(t *testing.T) {
	xmlText := `
    <root BTCPP_format="4">
      <BehaviorTree ID="EnsureWarm">
        <ReactiveFallback>
          <IsWarm value="{is_warm}"/>
          <ReactiveSequence>
            <SubTree ID="EnsureHoldingJacket" _autoremap="true" />
            <WearJacket value="{is_warm}"/>
          </ReactiveSequence>
        </ReactiveFallback>
      </BehaviorTree>

      <BehaviorTree ID="EnsureHoldingJacket">
        <ReactiveFallback>
          <IsHoldingJacket value="{holding_jacket}"/>
          <ReactiveSequence>
            <IsNearCloset value="{near_closet}"/>
            <GrabJacket value="{holding_jacket}"/>
          </ReactiveSequence>
        </ReactiveFallback>
      </BehaviorTree>
    </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardControlNodes(factory)
	RegisterStandardBackchainingNodes(factory)

	factory.RegisterBehaviorTreeFromText(xmlText)

	bb := core.NewBlackboard(nil)
	bb.Set("is_warm", false)
	bb.Set("holding_jacket", false)
	bb.Set("near_closet", true)

	tree, err := factory.CreateTree("EnsureWarm", bb)
	if err != nil {
		t.Fatal(err)
	}

	// first tick: not warm, no jacket, near closet: start GrabJacket
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: want RUNNING, got %v", status)
	}
	var isWarm, holdingJacket, nearCloset bool
	bb.Get("is_warm", &isWarm)
	bb.Get("holding_jacket", &holdingJacket)
	bb.Get("near_closet", &nearCloset)
	if isWarm {
		t.Error("tick 1: is_warm should be false")
	}
	if holdingJacket {
		t.Error("tick 1: holding_jacket should be false")
	}
	if !nearCloset {
		t.Error("tick 1: near_closet should be true")
	}

	// second tick: still GrabJacket
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 2: want RUNNING, got %v", status)
	}

	// third tick: GrabJacket succeeded, start wearing
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 3: want RUNNING, got %v", status)
	}
	bb.Get("is_warm", &isWarm)
	bb.Get("holding_jacket", &holdingJacket)
	if isWarm {
		t.Error("tick 3: is_warm should be false")
	}
	if !holdingJacket {
		t.Error("tick 3: holding_jacket should be true")
	}

	// fourth tick: still WearingJacket
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 4: want RUNNING, got %v", status)
	}

	// fifth tick: warm (WearingJacket succeeded)
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 5: want SUCCESS, got %v", status)
	}
	bb.Get("is_warm", &isWarm)
	if !isWarm {
		t.Error("tick 5: is_warm should be true")
	}

	// sixth tick: still warm (just the condition ticked)
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 6: want SUCCESS, got %v", status)
	}
}

// RegisterStandardControlNodes registers control flow nodes needed for
// reactive backchaining tests. This is separate from RegisterStandardNodes
// to avoid circular dependency with decorator package test nodes.
func RegisterStandardControlNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("ReactiveSequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewReactiveSequence(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ReactiveFallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewReactiveFallback(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSequenceNode(name, config)
	}, core.Control)
}
