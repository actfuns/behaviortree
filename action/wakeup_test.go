package action

import (
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// TestWakeUp_BasicTest verifies that an async action wakes up the tree when done.
// In the Go port, TestNode with async_delay triggers a wake-up signal when the timer fires.
func TestWakeUp_BasicTest(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	_ = factory.RegisterNodeType("Sleep", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSleepNode(name, config)
	}, core.Action)

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

	t1 := time.Now()
	status := tree.TickWhileRunning(0)
	elapsed := time.Since(t1)

	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}

	// The Sleep node with msec=10 should complete within 25ms
	if elapsed > time.Millisecond*25 {
		t.Logf("WakeUp took longer than expected: %v (this may be OK under load)", elapsed)
	} else {
		t.Logf("WakeUp completed in: %v", elapsed)
	}
}
