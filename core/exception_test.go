package core_test

import (
	"strings"
	"testing"

	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// ThrowingActionNode is a test action that panics in Tick.
type ThrowingActionNode struct {
	core.SyncActionNode
}

func (n *ThrowingActionNode) Tick() core.NodeStatus {
	panic("Test exception from ThrowingAction")
}

// SucceedingActionNode is a test action that always succeeds.
type SucceedingActionNode struct {
	core.SyncActionNode
}

func (n *SucceedingActionNode) Tick() core.NodeStatus {
	return core.SUCCESS
}

func TestExceptionTracking_BasicExceptionCapture(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	_ = factory.RegisterNodeType("ThrowingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &ThrowingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("ThrowingAction")
			return n
		}, core.Action)

	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ThrowingAction name="thrower"/>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := r.(error).Error()
			if !strings.Contains(errMsg, "thrower") {
				t.Errorf("expected error message to contain 'thrower', got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "Test exception from ThrowingAction") {
				t.Errorf("expected error message to contain 'Test exception from ThrowingAction', got: %s", errMsg)
			}
		} else {
			t.Error("expected panic from ThrowingAction")
		}
	}()

	_ = tree.TickExactlyOnce()
}

func TestExceptionTracking_NestedExceptionBacktrace(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	control.RegisterStandardNodes(factory)

	_ = factory.RegisterNodeType("ThrowingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &ThrowingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("ThrowingAction")
			return n
		}, core.Action)

	_ = factory.RegisterNodeType("SucceedingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &SucceedingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("SucceedingAction")
			return n
		}, core.Action)

	// Register RetryUntilSuccessful from decorator package
	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="main_seq">
				<SucceedingAction name="first"/>
				<RetryUntilSuccessful num_attempts="1" name="retry">
					<ThrowingAction name="nested_thrower"/>
				</RetryUntilSuccessful>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := r.(error).Error()
			if !strings.Contains(errMsg, "nested_thrower") {
				t.Errorf("expected error message to contain 'nested_thrower', got: %s", errMsg)
			}
		} else {
			t.Error("expected panic from nested ThrowingAction")
		}
	}()

	_ = tree.TickExactlyOnce()
}

func TestExceptionTracking_NoExceptionNoWrapping(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	control.RegisterStandardNodes(factory)

	_ = factory.RegisterNodeType("SucceedingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &SucceedingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("SucceedingAction")
			return n
		}, core.Action)

	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="main_seq">
				<SucceedingAction name="a"/>
				<SucceedingAction name="b"/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	status := tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

func TestExceptionTracking_BacktraceEntryContents(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	_ = factory.RegisterNodeType("ThrowingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &ThrowingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("ThrowingAction")
			return n
		}, core.Action)

	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ThrowingAction name="my_action"/>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := r.(error).Error()
			if !strings.Contains(errMsg, "my_action") {
				t.Errorf("expected error message to contain node name 'my_action', got: %s", errMsg)
			}
			if !strings.Contains(errMsg, "ThrowingAction") {
				t.Errorf("expected error message to contain registration ID 'ThrowingAction', got: %s", errMsg)
			}
		} else {
			t.Error("expected panic from ThrowingAction")
		}
	}()

	_ = tree.TickExactlyOnce()
}

func TestExceptionTracking_SubtreeExceptionBacktrace(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	control.RegisterStandardNodes(factory)

	_ = factory.RegisterNodeType("ThrowingAction", core.PortsList{},
		func(name string, config core.NodeConfig) core.TreeNode {
			n := &ThrowingActionNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("ThrowingAction")
			return n
		}, core.Action)

	xml := `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
		<BehaviorTree ID="MainTree">
			<Sequence name="outer_seq">
				<SubTree ID="InnerTree" name="subtree_call"/>
			</Sequence>
		</BehaviorTree>
		<BehaviorTree ID="InnerTree">
			<Sequence name="inner_seq">
				<ThrowingAction name="subtree_thrower"/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := r.(error).Error()
			if !strings.Contains(errMsg, "subtree_thrower") {
				t.Errorf("expected error message to contain 'subtree_thrower', got: %s", errMsg)
			}
		} else {
			t.Error("expected panic from ThrowingAction in subtree")
		}
	}()

	_ = tree.TickExactlyOnce()
}
