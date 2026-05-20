package core_test

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

// TestPostConditionsBasic verifies that _onSuccess, _onFailure, and _post
// post-condition scripts are executed at the appropriate times.
func TestPostConditionsBasic(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=1; B:=1; C:=1; D:=1" />
	            <AlwaysSuccess _onSuccess="B=42"/>
	            <ForceSuccess>
	                <AlwaysSuccess _failureIf="A!=0" _onFailure="C=42"/>
	            </ForceSuccess>
	            <ForceSuccess>
	                <AlwaysFailure _onFailure="D=42"/>
	            </ForceSuccess>
	        </Sequence>
	    </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Fatalf("want SUCCESS, got %v", status)
	}

	var bVal int
	if err := tree.RootBlackboard().GetInto("B", &bVal); err == nil && bVal != 42 {
		t.Errorf("B: want 42, got %d", bVal)
	}

	var cVal int
	if err := tree.RootBlackboard().GetInto("C", &cVal); err == nil && cVal != 42 {
		t.Errorf("C: want 42, got %d", cVal)
	}

	var dVal int
	if err := tree.RootBlackboard().GetInto("D", &dVal); err == nil && dVal != 42 {
		t.Errorf("D: want 42, got %d", dVal)
	}
}

// TestPostConditionsIssue539 verifies that _onFailure and _post fire on
// every retry attempt in RetryUntilSuccessful.
func TestPostConditionsIssue539(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Script code="x:=0; y:=0" />
	      <RetryUntilSuccessful num_attempts="5">
	        <AlwaysFailure _onFailure="x += 1" _post="y += 1" />
	      </RetryUntilSuccessful>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Fatalf("want FAILURE, got %v", status)
	}

	var xVal int
	if err := tree.RootBlackboard().GetInto("x", &xVal); err == nil && xVal != 5 {
		t.Errorf("x: want 5, got %d", xVal)
	}

	var yVal int
	if err := tree.RootBlackboard().GetInto("y", &yVal); err == nil && yVal != 5 {
		t.Errorf("y: want 5, got %d", yVal)
	}
}

// TestPostConditionsIssue601 verifies that _onHalted fires when a sibling
// causes a node to be halted in a Parallel node.
func TestPostConditionsIssue601(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="test_tree">
	    <Sequence>
	      <Script code="test := 'start'"/>
	        <Parallel failure_count="1"
	                  success_count="-1">
	          <Sleep msec="1000"
	                 _onHalted="test = 'halted'"
	                 _post="test = 'post'"/>
	          <AlwaysFailure/>
	        </Parallel>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	// The tree returns FAILURE because Parallel fails (AlwaysFailure triggers failure_count=1).
	// We still check that _onHalted fires on the Sleep node.
	_ = tree.TickWhileRunning(0)

	var testVal string
	if err := tree.RootBlackboard().GetInto("test", &testVal); err == nil {
		// The _onHalted script should have fired when the Parallel halts the Sleep node.
		// Expected value is "halted" - _onHalted runs during halt, _post runs on normal
		// completion which is never reached because the node is halted.
		if testVal != "halted" {
			t.Errorf("test: want 'halted', got %q", testVal)
		}
	} else {
		t.Fatalf("could not read 'test' from blackboard: %v", err)
	}
}
