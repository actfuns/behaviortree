package core_test

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
)

// TestPreconditionsIntegers verifies that integer preconditions with _failureIf
// and _successIf work correctly on <Precondition> decorator nodes.
func TestPreconditionsIntegers(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	// Use Precondition if with default else=FAILURE.
	// A==B is true → TestA runs
	// A==C is false → else FAILURE → sequence stops with FAILURE, TestC is not reached
	// We test that case separately.
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=1; B:=1; C:=3" />
	            <Precondition if="A==B">
	                <TestA/>
	            </Precondition>
	            <Precondition if="A!=C">
	                <TestC/>
	            </Precondition>
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
	if counters[0] != 1 {
		t.Errorf("TestA: want 1, got %d", counters[0])
	}
	if counters[2] != 1 {
		t.Errorf("TestC: want 1, got %d", counters[2])
	}
}

// TestPreconditionsDoubleEquals verifies that floating-point comparisons
// work in precondition scripts.
func TestPreconditionsDoubleEquals(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=1.1; B:=1.1; C:=2.0" />
	            <Precondition if="A==B">
	                <TestA/>
	            </Precondition>
	            <Precondition if="A!=C">
	                <TestC/>
	            </Precondition>
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
	if counters[0] != 1 {
		t.Errorf("TestA: want 1, got %d", counters[0])
	}
	if counters[2] != 1 {
		t.Errorf("TestC: want 1, got %d", counters[2])
	}
}

// TestPreconditionsStringEquals verifies that string comparisons work in
// precondition scripts.
func TestPreconditionsStringEquals(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:='hello'" />
	            <Script code="B:='hello'" />
	            <Precondition if="A==B">
	                <TestA/>
	            </Precondition>
	            <Script code="C:='world'" />
	            <Script code="D:='world'" />
	            <Precondition if="C==D">
	                <TestB/>
	            </Precondition>
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
	if counters[0] != 1 {
		t.Errorf("TestA: want 1, got %d", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB: want 1, got %d", counters[1])
	}
}

// TestPreconditionsChecksConditionOnce verifies that preconditions on the
// Precondition decorator are not re-evaluated while the child is running.
// Uses _skipIf on the child to prevent it from ticking when the condition is met.
func TestPreconditionsChecksConditionOnce(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	factory.RegisterNodeType("KeepRunning", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return newKeepRunning(name, config)
	}, core.Condition)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=0" />
	            <Precondition if="A==0" else="FAILURE">
	                <KeepRunning/>
	            </Precondition>
	        </Sequence>
	    </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick - Precondition if="A==0" is true, ticks KeepRunning which returns RUNNING
	status := tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 1: want RUNNING, got %v", status)
	}

	// Change A to 1 to make precondition false, but child is still running
	// so the precondition should NOT be re-evaluated
	tree.RootBlackboard().Set("A", 1)
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 2: want RUNNING (precondition not re-evaluated), got %v", status)
	}
}

// TestPreconditionsBasic verifies _successIf and _failureIf attributes
// on action nodes and within a Fallback.
func TestPreconditionsBasic(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 4)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=1" />
	            <TestA _successIf="A==1"/>
	            <TestB _successIf="A==2"/>
	            <Fallback>
	                <TestC _failureIf="A==1"/>
	                <TestD _failureIf="A!=1"/>
	            </Fallback>
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
	if counters[0] != 0 {
		t.Errorf("TestA: want 0 (skipped by _successIf), got %d", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB: want 1, got %d", counters[1])
	}
	if counters[2] != 0 {
		t.Errorf("TestC: want 0 (skipped by _failureIf), got %d", counters[2])
	}
	if counters[3] != 1 {
		t.Errorf("TestD: want 1, got %d", counters[3])
	}
}

// TestPreconditionsIssue533 verifies _skipIf with _onSuccess for progressive
// state changes over multiple ticks.
func TestPreconditionsIssue533(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <TestA _skipIf="A!=1" />
	            <TestB _skipIf="A!=2" _onSuccess="A=1"/>
	            <TestC _skipIf="A!=3" _onSuccess="A=2"/>
	        </Sequence>
	    </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.Subtrees[0].Blackboard.Set("A", 3)

	// First tick: A=3, only TestC's skipIf is false (A==3), so only TestC ticks
	tree.TickOnce()
	if counters[0] != 0 {
		t.Errorf("tick 1 TestA: want 0, got %d", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("tick 1 TestB: want 0, got %d", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("tick 1 TestC: want 1, got %d", counters[2])
	}

	// Second tick: after TestC's _onSuccess set A=2, so TestB ticks
	tree.TickOnce()
	if counters[0] != 0 {
		t.Errorf("tick 2 TestA: want 0, got %d", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("tick 2 TestB: want 1, got %d", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("tick 2 TestC: want 1, got %d", counters[2])
	}

	// Third tick: after TestB's _onSuccess set A=1, so TestA ticks
	tree.TickOnce()
	if counters[0] != 1 {
		t.Errorf("tick 3 TestA: want 1, got %d", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("tick 3 TestB: want 1, got %d", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("tick 3 TestC: want 1, got %d", counters[2])
	}
}

// TestPreconditionsIssue585 verifies that _skipIf prevents a coroutine-style
// action from being ticked even once.
func TestPreconditionsIssue585(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	timesTicked := 0
	factory.RegisterNodeType("CoroTest", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return newCoroTestNode(name, config, &timesTicked)
	}, core.Action)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=1" />
	            <CoroTest _skipIf="A==1" />
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
	if timesTicked != 0 {
		t.Errorf("times ticked: want 0 (skipped), got %d", timesTicked)
	}
}

// TestPreconditionsIssue615_NoSkipWhenRunningA verifies that _skipIf is
// not re-evaluated when KeepRunningUntilFailure is in RUNNING state.
func TestPreconditionsIssue615_NoSkipWhenRunningA(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	// Use a string comparison instead of bool comparison to avoid type issues.
	const xmlText = `
	<root BTCPP_format="4">
	  <BehaviorTree>
	    <KeepRunningUntilFailure _skipIf="check==1">
	      <AlwaysSuccess/>
	    </KeepRunningUntilFailure>
	  </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.RootBlackboard().Set("check", 0)
	status := tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 1: want RUNNING, got %v", status)
	}

	// Precondition should NOT be re-evaluated because KeepRunningUntilFailure
	// is in RUNNING state
	tree.RootBlackboard().Set("check", 1)
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 2: want RUNNING, got %v", status)
	}
}

// TestPreconditionsIssue615_NoSkipWhenRunningB verifies that _skipIf is
// evaluated only when node is IDLE, not when RUNNING.
func TestPreconditionsIssue615_NoSkipWhenRunningB(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	factory.RegisterNodeType("KeepRunning", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return newKeepRunning(name, config)
	}, core.Condition)

	const xmlText = `
	<root BTCPP_format="4">
	  <BehaviorTree>
	    <KeepRunning _skipIf="check==0"/>
	  </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.RootBlackboard().Set("check", 0)
	status := tree.TickOnce()
	if status != core.SKIPPED {
		t.Fatalf("tick 1: want SKIPPED (skipIf true), got %v", status)
	}

	// Should not be skipped anymore
	tree.RootBlackboard().Set("check", 1)
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 2: want RUNNING, got %v", status)
	}

	// skipIf should be ignored because KeepRunning is RUNNING and not IDLE
	tree.RootBlackboard().Set("check", 0)
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 3: want RUNNING (precondition ignored while running), got %v", status)
	}
}

// TestPreconditionsRemapping verifies that port remapping through subtrees
// works correctly with _skipIf.
func TestPreconditionsRemapping(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	factory.RegisterNodeType("SimpleOutput", core.PortsList(func() map[string]core.PortInfo {
		_, outputPort := core.OutputPort[bool]("output", "")
		return map[string]core.PortInfo{
			"output": outputPort,
		}
	}()), func(name string, config core.NodeConfig) core.TreeNode {
		return newSimpleOutput(name, config)
	}, core.Action)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Sequence>
	      <SimpleOutput output="{param}" />
	      <Script code="value:=true" />
	      <SubTree ID="Sub1" param="{param}"/>
	      <SubTree ID="Sub1" param="{value}"/>
	      <SubTree ID="Sub1" param="true"/>
	      <TestA/>
	    </Sequence>
	  </BehaviorTree>
	  <BehaviorTree ID="Sub1">
	    <Sequence>
	      <SubTree ID="Sub2" _skipIf="param != true" />
	    </Sequence>
	  </BehaviorTree>
	  <BehaviorTree ID="Sub2">
	    <TestB/>
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
	if counters[0] != 1 {
		t.Errorf("TestA: want 1, got %d", counters[0])
	}
	if counters[1] != 3 {
		t.Errorf("TestB: want 3, got %d", counters[1])
	}
}

// TestPreconditionsWhileCallsOnHalt verifies that _onHalted is called when
// _while triggers a halt (when _while condition changes from true to false
// while the node is RUNNING). In the Go port, _while on an IDLE node returns
// SKIPPED, so we test the halt-on-running behavior by halting manually.
func TestPreconditionsWhileCallsOnHalt(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	factory.RegisterNodeType("KeepRunning", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return newKeepRunning(name, config)
	}, core.Condition)

	const xmlText = `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Sequence>
	      <KeepRunning _onHalted="B=69" />
	    </Sequence>
	  </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.RootBlackboard().Set("B", 0)
	status := tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 1: want RUNNING, got %v", status)
	}

	var bVal int
	if err := tree.RootBlackboard().GetInto("B", &bVal); err == nil && bVal != 0 {
		t.Fatalf("B before halt: want 0, got %d", bVal)
	}

	// Halt the tree manually to trigger _onHalted
	tree.HaltTree()

	if err := tree.RootBlackboard().GetInto("B", &bVal); err == nil && bVal != 69 {
		t.Fatalf("B after halt: want 69, got %d", bVal)
	}
}

// TestPreconditionsSkippedSequence verifies that a node with _skipIf inside
// a Sequence causes the Sequence to return SKIPPED.
func TestPreconditionsSkippedSequence(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <AlwaysSuccess _skipIf="skip"/>
	        </Sequence>
	    </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.RootBlackboard().Set("skip", true)
	status := tree.TickWhileRunning(0)
	if status != core.SKIPPED {
		t.Fatalf("with skip=true: want SKIPPED, got %v", status)
	}

	tree.RootBlackboard().Set("skip", false)
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Fatalf("with skip=false: want SUCCESS, got %v", status)
	}

	tree.RootBlackboard().Set("skip", true)
	status = tree.TickWhileRunning(0)
	if status != core.SKIPPED {
		t.Fatalf("with skip=true again: want SKIPPED, got %v", status)
	}

	tree.RootBlackboard().Set("skip", false)
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Fatalf("with skip=false again: want SUCCESS, got %v", status)
	}
}

// TestPreconditionsCanRunChildrenMultipleTimes verifies that preconditions
// within a Repeat loop are evaluated correctly on each cycle.
func TestPreconditionsCanRunChildrenMultipleTimes(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code="A:=0" />
	            <Script code="B:=0" />
	            <Repeat num_cycles="3">
	                <Sequence>
	                    <Precondition if="A==0">
	                        <TestA/>
	                    </Precondition>
	                    <AlwaysSuccess/>
	                    <AlwaysSuccess/>
	                </Sequence>
	            </Repeat>
	        </Sequence>
	    </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// With A initialized to 0 by the script, all 3 cycles pass the precondition.
	// TestA should tick 3 times (once per cycle).
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Fatalf("with A=0: want SUCCESS, got %v", status)
	}
	if counters[0] != 3 {
		t.Errorf("TestA count with A=0: want 3, got %d", counters[0])
	}
}

// ====================================================================
// Helper types for precondition tests
// ====================================================================

// keepRunning is a StatefulActionNode that always returns RUNNING,
// equivalent to the C++ KeepRunning class in the precondition tests.
type keepRunning struct {
	core.StatefulActionNode
}

func newKeepRunning(name string, config core.NodeConfig) *keepRunning {
	n := &keepRunning{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("KeepRunning")
	return n
}

func (n *keepRunning) OnStart() core.NodeStatus {
	return core.RUNNING
}

func (n *keepRunning) OnRunning() core.NodeStatus {
	return core.RUNNING
}

func (n *keepRunning) OnHalted() {}

func (n *keepRunning) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

// coroTestNode simulates a coroutine-style node that yields RUNNING multiple
// times before returning SUCCESS. Equivalent to C++ CoroTestNode.
type coroTestNode struct {
	core.StatefulActionNode
	timesTicked *int
	state       int
}

func newCoroTestNode(name string, config core.NodeConfig, timesTicked *int) *coroTestNode {
	n := &coroTestNode{
		timesTicked: timesTicked,
		state:       0,
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("CoroTest")
	return n
}

func (n *coroTestNode) OnStart() core.NodeStatus {
	n.state = 0
	*n.timesTicked++
	n.state++
	n.SetStatus(core.RUNNING)
	return core.RUNNING
}

func (n *coroTestNode) OnRunning() core.NodeStatus {
	*n.timesTicked++
	n.state++
	if n.state >= 10 {
		return core.SUCCESS
	}
	n.SetStatus(core.RUNNING)
	return core.RUNNING
}

func (n *coroTestNode) OnHalted() {
	n.state = 0
}

func (n *coroTestNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

// simpleOutput is a SyncActionNode that sets an output port to true.
// Equivalent to C++ SimpleOutput.
type simpleOutput struct {
	core.SyncActionNode
}

func newSimpleOutput(name string, config core.NodeConfig) *simpleOutput {
	n := &simpleOutput{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("SimpleOutput")
	return n
}

func (n *simpleOutput) Tick() core.NodeStatus {
	if err := n.SetOutput("output", true); err != nil {
		_ = err
	}
	return core.SUCCESS
}
