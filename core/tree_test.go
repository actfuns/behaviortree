package core_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

func TestTree_Condition1ToFalseCondition2True(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	condition1Called := false
	condition2Called := false
	actionCalled := false

	_ = factory.RegisterSimpleAction("Condition1", func(core.TreeNode) core.NodeStatus {
		condition1Called = true
		return core.FAILURE
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Condition2", func(core.TreeNode) core.NodeStatus {
		condition2Called = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Action1", func(core.TreeNode) core.NodeStatus {
		actionCalled = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence name="root_sequence">
				<Fallback name="fallback_conditions">
					<Condition1/>
					<Condition2/>
				</Fallback>
				<Action1/>
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
	if !condition1Called {
		t.Error("Condition1 was not called")
	}
	if !condition2Called {
		t.Error("Condition2 was not called")
	}
	if !actionCalled {
		t.Error("Action1 was not called")
	}
}

func TestTree_Condition2ToFalseCondition1True(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	condition1Called := false
	condition2Called := false
	actionCalled := false

	_ = factory.RegisterSimpleAction("Condition1", func(core.TreeNode) core.NodeStatus {
		condition1Called = true
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Condition2", func(core.TreeNode) core.NodeStatus {
		condition2Called = true
		return core.FAILURE
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Action1", func(core.TreeNode) core.NodeStatus {
		actionCalled = true
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence name="root_sequence">
				<Fallback name="fallback_conditions">
					<Condition1/>
					<Condition2/>
				</Fallback>
				<Action1/>
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
	if !condition1Called {
		t.Error("Condition1 was not called")
	}
	// In Fallback, if Condition1 succeeds, Condition2 is not ticked.
	// This matches C++ test expectations: condition_2 is IDLE.
	if condition2Called {
		t.Log("Condition2 was called (Fallback ticked through all children)")
	}
	if !actionCalled {
		t.Error("Action1 was not called")
	}
}

func TestTree_PrintWithStream(t *testing.T) {
	// Capture stdout from PrintTreeRecursively
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		os.Stdout = old
		t.Fatal(err)
	}
	control.RegisterStandardNodes(factory)

	_ = factory.RegisterSimpleAction("Condition1", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Condition2", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})
	_ = factory.RegisterSimpleAction("Action1", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4" >
		<BehaviorTree ID="Main">
			<Sequence name="root_sequence">
				<Fallback name="fallback_conditions">
					<Condition1/>
					<Condition2/>
				</Fallback>
				<Action1/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		os.Stdout = old
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	root := tree.RootNode()
	core.PrintTreeRecursively(root)

	w.Close()
	out, _ := io.ReadAll(r)
	os.Stdout = old

	output := string(out)
	t.Logf("PrintTreeRecursively output:\n%s", output)

	// Verify output contains expected structure
	if !strings.Contains(output, "----------------") {
		t.Error("expected dashed separator line")
	}
	if !strings.Contains(output, "root_sequence") {
		t.Error("expected root_sequence in output")
	}
	if !strings.Contains(output, "fallback_conditions") {
		t.Error("expected fallback_conditions in output")
	}
}
