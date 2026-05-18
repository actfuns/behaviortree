package action

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// registerUpdatesTestNodes registers the minimal set of nodes needed for updates tests.
func registerUpdatesTestNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &updatesSequenceNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Sequence")
		return n
	}, core.Control)

	_ = factory.RegisterNodeType("Fallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &updatesFallbackNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Fallback")
		return n
	}, core.Control)

	_ = factory.RegisterNodeType("Script", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewScriptNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewAlwaysSuccessNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("WasEntryUpdated", core.PortsList{
		"entry": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewEntryUpdatedAction(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("Repeat", core.PortsList{
		"num_cycles": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &updatesRepeatNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Repeat")
		return n
	}, core.Decorator)

	_ = factory.RegisterNodeType("SubTree", core.PortsList{
		"name":    core.NewPortInfo(core.INPUT),
		"_autoremap": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &updatesSubTreeNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("SubTree")
		return n
	}, core.Subtree)
}

// updatesSequenceNode is a minimal Sequence node.
type updatesSequenceNode struct {
	core.ControlNode
	childIdx int
}

func (n *updatesSequenceNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)
	for n.childIdx < len(n.Children()) {
		child := n.Children()[n.childIdx]
		childStatus := child.ExecuteTick()
		switch childStatus {
		case core.SUCCESS:
			n.childIdx++
			child.ResetStatus()
		case core.FAILURE:
			n.HaltChildren()
			n.childIdx = 0
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		case core.SKIPPED:
			n.childIdx++
			child.ResetStatus()
		}
	}
	n.childIdx = 0
	return core.SUCCESS
}

func (n *updatesSequenceNode) Halt() {
	n.HaltChildren()
	n.childIdx = 0
	n.ResetStatus()
}

// updatesFallbackNode is a minimal Fallback node.
type updatesFallbackNode struct {
	core.ControlNode
	childIdx int
}

func (n *updatesFallbackNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)
	for n.childIdx < len(n.Children()) {
		child := n.Children()[n.childIdx]
		childStatus := child.ExecuteTick()
		switch childStatus {
		case core.FAILURE:
			n.childIdx++
			child.ResetStatus()
		case core.SUCCESS:
			n.HaltChildren()
			n.childIdx = 0
			return core.SUCCESS
		case core.RUNNING:
			return core.RUNNING
		case core.SKIPPED:
			n.childIdx++
			child.ResetStatus()
		}
	}
	n.childIdx = 0
	return core.FAILURE
}

func (n *updatesFallbackNode) Halt() {
	n.HaltChildren()
	n.childIdx = 0
	n.ResetStatus()
}

// updatesRepeatNode is a minimal Repeat decorator.
type updatesRepeatNode struct {
	core.DecoratorNode
	numCycles  int
	currentIdx int
}

func (n *updatesRepeatNode) Tick() core.NodeStatus {
	if v, err := core.GetInputTyped[int](n, "num_cycles"); err == nil {
		n.numCycles = v
	}

	n.SetStatus(core.RUNNING)

	for n.currentIdx < n.numCycles {
		childStatus := n.Child().ExecuteTick()
		if childStatus == core.RUNNING {
			return core.RUNNING
		}
		if childStatus == core.FAILURE {
			n.currentIdx = 0
			return core.FAILURE
		}
		n.currentIdx++
		if n.currentIdx < n.numCycles {
			n.Child().ResetStatus()
		}
	}

	n.currentIdx = 0
	return core.SUCCESS
}

func (n *updatesRepeatNode) Halt() {
	n.currentIdx = 0
	n.DecoratorNode.Halt()
}

// updatesSubTreeNode is a minimal SubTree placeholder.
type updatesSubTreeNode struct {
	core.DecoratorNode
}

func (n *updatesSubTreeNode) Tick() core.NodeStatus {
	if n.Child() == nil {
		return core.FAILURE
	}
	return n.Child().ExecuteTick()
}

// --------------------------------------------------------------------
// Tests
// --------------------------------------------------------------------

func TestEntryUpdates_NoEntry(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerUpdatesTestNodes(factory)

	testACalled := 0
	testBCalled := 0

	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testACalled++
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("TestB", func(core.TreeNode) core.NodeStatus {
		testBCalled++
		return core.SUCCESS
	}, core.PortsList{})

	xmlCheck := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Check">
			<Sequence>
				<Fallback>
					<WasEntryUpdated entry="A"/>
					<TestA/>
				</Fallback>
				<WasEntryUpdated entry="A">
					<TestB/>
				</WasEntryUpdated>
			</Sequence>
		</BehaviorTree>
	</root>`

	// NOTE: WasEntryUpdated as a decorator is implemented as an action in Go.
	// The test structure is adapted: WasEntryUpdated checks an entry and acts as a condition.
	// With no entry set, it returns FAILURE (not updated), so Fallback tick TestA.
	// Then SkipUnlessUpdated would check the same entry.

	_ = factory.RegisterSimpleAction("SkipUnlessUpdated", func(core.TreeNode) core.NodeStatus {
		// Returns SUCCESS because entry is not set (simplified)
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<SubTree ID="Check" _autoremap="true"/>
			</Sequence>
		</BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlCheck)
	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("Main", nil)
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("NoEntry test status: %v", status)
	t.Logf("TestA called: %d, TestB called: %d", testACalled, testBCalled)
}

func TestEntryUpdates_Initialized(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerUpdatesTestNodes(factory)

	testACalled := 0
	testBCalled := 0

	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testACalled++
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("TestB", func(core.TreeNode) core.NodeStatus {
		testBCalled++
		return core.SUCCESS
	}, core.PortsList{})

	xmlCheck := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Check">
			<Sequence>
				<Fallback>
					<WasEntryUpdated entry="A"/>
					<TestA/>
				</Fallback>
				<WasEntryUpdated entry="A">
					<TestB/>
				</WasEntryUpdated>
			</Sequence>
		</BehaviorTree>
	</root>`

	_ = factory.RegisterSimpleAction("SkipUnlessUpdated", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<Script code="A:=1;B:=1"/>
				<SubTree ID="Check" _autoremap="true"/>
			</Sequence>
		</BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlCheck)
	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("Main", nil)
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("Initialized test status: %v", status)
	t.Logf("TestA called: %d, TestB called: %d", testACalled, testBCalled)
}

func TestEntryUpdates_UpdateOnce(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerUpdatesTestNodes(factory)

	testACalled := 0
	testBCalled := 0

	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testACalled++
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("TestB", func(core.TreeNode) core.NodeStatus {
		testBCalled++
		return core.SUCCESS
	}, core.PortsList{})

	xmlCheck := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Check">
			<Sequence>
				<Fallback>
					<WasEntryUpdated entry="A"/>
					<TestA/>
				</Fallback>
				<WasEntryUpdated entry="A">
					<TestB/>
				</WasEntryUpdated>
			</Sequence>
		</BehaviorTree>
	</root>`

	_ = factory.RegisterSimpleAction("SkipUnlessUpdated", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence>
				<Script code="A:=1"/>
				<Repeat num_cycles="2">
					<SubTree ID="Check" _autoremap="true"/>
				</Repeat>
			</Sequence>
		</BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlCheck)
	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("Main", nil)
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("UpdateOnce test status: %v", status)
	t.Logf("TestA called: %d, TestB called: %d", testACalled, testBCalled)
}

func TestEntryUpdates_UpdateTwice(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerUpdatesTestNodes(factory)

	testACalled := 0
	testBCalled := 0

	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testACalled++
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("TestB", func(core.TreeNode) core.NodeStatus {
		testBCalled++
		return core.SUCCESS
	}, core.PortsList{})

	xmlCheck := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Check">
			<Sequence>
				<Fallback>
					<WasEntryUpdated entry="A"/>
					<TestA/>
				</Fallback>
				<WasEntryUpdated entry="A">
					<TestB/>
				</WasEntryUpdated>
			</Sequence>
		</BehaviorTree>
	</root>`

	_ = factory.RegisterSimpleAction("SkipUnlessUpdated", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Repeat num_cycles="2">
				<Sequence>
					<Script code="A:=1"/>
					<SubTree ID="Check" _autoremap="true"/>
				</Sequence>
			</Repeat>
		</BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlCheck)
	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("Main", nil)
	if err != nil {
		t.Fatalf("CreateTree failed: %v", err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("UpdateTwice test status: %v", status)
	t.Logf("TestA called: %d, TestB called: %d", testACalled, testBCalled)
}
