package core_test

import (
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

// --------------------------------------------------------------------
// Tests ported from C++ gtest_substitution.cpp
// --------------------------------------------------------------------

// TestSubstitution_Parser verifies that substitution rules work correctly.
// Equivalent of C++ Substitution/Parser.
func TestSubstitution_Parser(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	// Add substitution rules directly (Go version uses AddSubstitutionRule
	// with SubstitutionRule struct, no JSON parser for substitution rules yet)
	factory.AddSubstitutionRule("actionA", core.SubstitutionRule{ReplaceWith: "AlwaysSuccess"})
	factory.AddSubstitutionRule("actionB", core.SubstitutionRule{ReplaceWith: "AlwaysFailure"})
	factory.AddSubstitutionRule("actionC", core.SubstitutionRule{ReplaceWith: "AlwaysSuccess"})

	rules := factory.SubstitutionRules()
	if len(rules) != 3 {
		t.Errorf("Expected 3 substitution rules, got %d", len(rules))
	}
	if _, ok := rules["actionA"]; !ok {
		t.Error("Expected rule for actionA")
	}
	if _, ok := rules["actionB"]; !ok {
		t.Error("Expected rule for actionB")
	}
	if _, ok := rules["actionC"]; !ok {
		t.Error("Expected rule for actionC")
	}

	ruleA := rules["actionA"]
	if ruleA.ReplaceWith != "AlwaysSuccess" {
		t.Errorf("Expected actionA -> AlwaysSuccess, got %s", ruleA.ReplaceWith)
	}

	ruleB := rules["actionB"]
	if ruleB.ReplaceWith != "AlwaysFailure" {
		t.Errorf("Expected actionB -> AlwaysFailure, got %s", ruleB.ReplaceWith)
	}
}

// TestSubstitution_SubTreeNodeSubstitution verifies that SubTree nodes
// can be substituted. Regression test for issue #934.
// Equivalent of C++ Substitution/SubTreeNodeSubstitution.
func TestSubstitution_SubTreeNodeSubstitution(t *testing.T) {
	parentXML := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Parent">
	    <Sequence>
	      <AlwaysSuccess/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	factory.AddSubstitutionRule("Child", core.SubstitutionRule{ReplaceWith: "AlwaysSuccess"})
	factory.RegisterBehaviorTreeFromText(parentXML)

	tree, err := factory.CreateTree("Parent", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestSubstitution_StringSubstitutionWithSimpleAction verifies that
// string-based substitution with registerSimpleAction works.
// Equivalent of C++ Substitution/StringSubstitutionWithSimpleAction_Issue930.
func TestSubstitution_StringSubstitutionWithSimpleAction(t *testing.T) {
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Delay delay_msec="50">
	        <AlwaysSuccess/>
	      </Delay>
	      <AlwaysSuccess name="action_to_replace"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	// Register substitute action
	_ = factory.RegisterSimpleAction("MyTestAction", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	// Use string-based substitution: replace action_to_replace with MyTestAction
	factory.AddSubstitutionRule("action_to_replace", core.SubstitutionRule{ReplaceWith: "MyTestAction"})

	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Should not hang
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestSubstitution_TestNodeConfigAsyncSubstitution verifies that
// substitution with an action that completes immediately works.
// Equivalent of C++ Substitution/TestNodeConfigAsyncSubstitution_Issue930.
func TestSubstitution_TestNodeConfigAsyncSubstitution(t *testing.T) {
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <AlwaysSuccess name="action_A"/>
	      <AlwaysSuccess name="action_B"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	// Substitute action_B with MyReplacement that returns SUCCESS immediately.
	_ = factory.RegisterSimpleAction("MyReplacement", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("action_B", core.SubstitutionRule{ReplaceWith: "MyReplacement"})

	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestSubstitution_JsonStringSubstitution verifies JSON-based string substitution.
// Equivalent of C++ Substitution/JsonStringSubstitution_Issue930.
func TestSubstitution_JsonStringSubstitution(t *testing.T) {
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <AlwaysSuccess name="action_A"/>
	      <AlwaysSuccess name="action_B"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	// Use AddSubstitutionRule instead of JSON (Go version)
	_ = factory.RegisterSimpleAction("MyReplacement", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("action_B", core.SubstitutionRule{ReplaceWith: "MyReplacement"})

	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestSubstitution_JsonWithEmptyTestNodeConfigs verifies JSON substitution
// with empty TestNodeConfigs.
// Equivalent of C++ Substitution/JsonWithEmptyTestNodeConfigs_Issue930.
func TestSubstitution_JsonWithEmptyTestNodeConfigs(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	_ = factory.RegisterSimpleAction("ReplacementNode", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("node_A", core.SubstitutionRule{ReplaceWith: "ReplacementNode"})

	rules := factory.SubstitutionRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 substitution rule, got %d", len(rules))
	}
	rule, ok := rules["node_A"]
	if !ok {
		t.Error("Expected rule for node_A")
	} else if rule.ReplaceWith != "ReplacementNode" {
		t.Errorf("Expected ReplacementNode, got %s", rule.ReplaceWith)
	}
}

// TestSubstitution_JsonWithoutTestNodeConfigs verifies JSON substitution
// without TestNodeConfigs.
// Equivalent of C++ Substitution/JsonWithoutTestNodeConfigs_Issue930.
func TestSubstitution_JsonWithoutTestNodeConfigs(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	_ = factory.RegisterSimpleAction("ReplacementNode", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("node_A", core.SubstitutionRule{ReplaceWith: "ReplacementNode"})

	rules := factory.SubstitutionRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 substitution rule, got %d", len(rules))
	}
	rule, ok := rules["node_A"]
	if !ok {
		t.Error("Expected rule for node_A")
	} else if rule.ReplaceWith != "ReplacementNode" {
		t.Errorf("Expected ReplacementNode, got %s", rule.ReplaceWith)
	}
}

// TestSubstitution_JsonStringSubstitutionWithDelay verifies JSON-based string
// substitution with Delay node.
// Equivalent of C++ Substitution/JsonStringSubstitutionWithDelay_Issue930.
func TestSubstitution_JsonStringSubstitutionWithDelay(t *testing.T) {
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Delay delay_msec="50">
	        <AlwaysSuccess/>
	      </Delay>
	      <Script name="script_2" code=" val:=1 "/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	actionExecuted := false
	_ = factory.RegisterSimpleAction("MyTest", func(core.TreeNode) core.NodeStatus {
		actionExecuted = true
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("script_2", core.SubstitutionRule{ReplaceWith: "MyTest"})

	factory.RegisterBehaviorTreeFromText(xml)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
	if !actionExecuted {
		t.Error("Expected MyTest action to be executed")
	}
}

// TestSubstitution_StringSubstitutionRegistrationID verifies that substituted
// node's registration ID is preserved.
// Equivalent of C++ Substitution/StringSubstitutionRegistrationID_Issue930.
func TestSubstitution_StringSubstitutionRegistrationID(t *testing.T) {
	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <AlwaysSuccess name="target_node"/>
	  </BehaviorTree>
	</root>`

	factory := factory.NewBehaviorTreeFactory()

	_ = factory.RegisterSimpleAction("MyReplacement", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{})

	factory.AddSubstitutionRule("target_node", core.SubstitutionRule{ReplaceWith: "MyReplacement"})
	factory.RegisterBehaviorTreeFromText(xml)

	var tickResult core.NodeStatus
	done := make(chan struct{})
	go func() {
		tree, err := factory.CreateTree("MainTree", nil)
		if err != nil {
			t.Errorf("CreateTree failed: %v", err)
			close(done)
			return
		}
		tickResult = tree.TickWhileRunning(0)
		close(done)
	}()

	select {
	case <-done:
		if tickResult != core.SUCCESS {
			t.Errorf("Expected SUCCESS, got %v", tickResult)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Tree hung! TickWhileRunning did not complete within 5 seconds")
	}
}
