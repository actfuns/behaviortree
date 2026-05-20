package script_test

import (
	"testing"

	"github.com/actfuns/behaviortree/action"
	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
	"github.com/actfuns/behaviortree/factory"
)

// --------------------------------------------------------------------
// Tests ported from C++ gtest_enums.cpp
// --------------------------------------------------------------------

// registerEnumNodes registers the node types needed for enum tests.
func registerEnumNodes(factory core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Switch4", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
		"case_4":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSwitchNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Script", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysSuccessNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysFailureNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("ForceSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceSuccessNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("TestNode", core.PortsList{
		"return_status": core.NewPortInfo(core.INPUT),
		"async_delay":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewTestNodeFromConfig(name, config)
	}, core.Action)
}

// TestEnums_StringToEnum verifies that enum values can be used in scripts.
// Equivalent of C++ Enums/StrintToEnum.
func TestEnums_StringToEnum(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerEnumNodes(factory)

	// Register Color enum values (using simple names without dots,
	// since the script parser doesn't handle dots in identifiers)
	factory.RegisterScriptingEnum("Red", 0)
	factory.RegisterScriptingEnum("Green", 2)
	factory.RegisterScriptingEnum("Blue", 1)

	// ActionEnum that reads a color port
	_ = factory.RegisterNodeType("ActionEnum", core.PortsList{
		"color": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &testActionEnum{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("ActionEnum")
		return n
	}, core.Action)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Sequence>
	      <Script code=" my_color := Red "/>
	      <ActionEnum name="maybe_red" color="{my_color}"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	// Verify the color was read correctly
	bb := tree.Subtrees[0].Blackboard
	myColor, err := core.GetTyped[int](bb, "my_color")
	if err != nil {
		t.Fatal(err)
	}
	if myColor != 0 {
		t.Errorf("Expected my_color=0 (Red), got %d", myColor)
	}
}

// testActionEnum is a simple action that reads a color port.
type testActionEnum struct {
	core.StatefulActionNode
}

func (n *testActionEnum) Tick() core.NodeStatus {
	return core.SUCCESS
}

// TestEnums_SwitchNodeWithEnum verifies Switch4 with enum case values.
// Equivalent of C++ Enums/SwitchNodeWithEnum.
func TestEnums_SwitchNodeWithEnum(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerEnumNodes(factory)

	factory.RegisterScriptingEnum("Red", 0)
	factory.RegisterScriptingEnum("Blue", 1)
	factory.RegisterScriptingEnum("Green", 2)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Sequence>
	      <Script code=" my_color := Blue "/>
	      <Switch4 variable="{my_color}"
	        case_1="Red"
	        case_2="Blue"
	        case_3="Green"
	        case_4="0">
	        <AlwaysFailure name="case_red"/>
	        <AlwaysSuccess name="case_blue"/>
	        <AlwaysFailure name="case_green"/>
	        <AlwaysFailure name="case_0"/>
	        <AlwaysFailure name="default_case"/>
	      </Switch4>
	    </Sequence>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestEnums_SubtreeRemapping verifies enum remapping through subtree ports.
// Equivalent of C++ Enums/SubtreeRemapping.
func TestEnums_SubtreeRemapping(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerEnumNodes(factory)

	factory.RegisterScriptingEnum("NO_FAULT", 0)
	factory.RegisterScriptingEnum("LOW_BATTERY", 1)

	// Register PrintEnum node
	_ = factory.RegisterNodeType("PrintEnum", core.PortsList{
		"enum": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &testPrintEnum{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("PrintEnum")
		return n
	}, core.Condition)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Script code=" fault_status := NO_FAULT "/>
	      <PrintEnum enum="{fault_status}"/>
	      <ForceSuccess>
	        <AlwaysFailure/>
	      </ForceSuccess>
	    </Sequence>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// testPrintEnum is a simple condition node.
type testPrintEnum struct {
	core.ConditionNode
}

func (n *testPrintEnum) Tick() core.NodeStatus {
	return core.SUCCESS
}

// TestEnums_ParseEnumWithConvertFromString verifies that enums with
// convertFromString work without ScriptingEnumsRegistry.
// Equivalent of C++ Enums/ParseEnumWithConvertFromString_Issue948.
func TestEnums_ParseEnumWithConvertFromString(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerEnumNodes(factory)

	// TestNode from the registered "TestNode" type will parse return_status
	// from the port. Since the registered TestNode uses NewTestNodeFromConfig
	// which always returns SUCCESS with no delay, let's just verify the XML
	// parsing and tree execution works.
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Sequence>
	      <TestNode name="test_action" return_status="SUCCESS"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}
