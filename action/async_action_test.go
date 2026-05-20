package action_test

import (
	"sync"
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

// TestAsyncAction_NoHalt verifies that halting an IDLE node is a no-op.
func TestAsyncAction_NoHalt(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

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
	factory := factory.NewBehaviorTreeFactory()

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
	factory := factory.NewBehaviorTreeFactory()

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
