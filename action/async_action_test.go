package action

import (
	"sync"
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// registerAsyncTestNodes registers nodes needed for async halt tests.
func registerAsyncTestNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("Sleep", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSleepNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewAlwaysSuccessNode(name, config)
	}, core.Action)
}

// TestAsyncAction_NoHalt verifies that halting an IDLE node is a no-op.
func TestAsyncAction_NoHalt(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerAsyncTestNodes(factory)

	// Create an XML tree with a Sleep node
	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sleep msec="50"/>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	// Halting the tree when it's not even started should be a no-op
	tree.HaltTree()
	t.Log("Halt on IDLE node completed without error")
}

// TestAsyncAction_Halt verifies that halting a RUNNING node works.
func TestAsyncAction_Halt(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerAsyncTestNodes(factory)

	// Use a Sleep node with a long delay so we can halt it while running
	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sleep msec="500"/>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatalf("CreateTreeFromText failed: %v", err)
	}

	// Tick the tree in a goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		status := tree.TickWhileRunning(time.Millisecond * 1)
		t.Logf("Halt test: tree finished with status: %v", status)
	}()

	// Wait a bit for the tree to start running
	time.Sleep(time.Millisecond * 10)

	// Halt the tree
	tree.HaltTree()
	wg.Wait()
	t.Log("Halt on RUNNING node completed successfully")
}

// TestAsyncAction_NormalRoutine verifies that an async action completes successfully.
func TestAsyncAction_NormalRoutine(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	registerAsyncTestNodes(factory)

	// Use a Sleep node with a short delay
	xml := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sleep msec="10"/>
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
}
