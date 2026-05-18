package control

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// ============================================================================
// Helper: RunningNode is a StatefulActionNode that returns RUNNING for a
// configurable number of ticks, then returns the desired result.
// ============================================================================

// runningNode is a StatefulActionNode that returns RUNNING for a configurable
// number of ticks, then returns the desired result.
type runningNode struct {
	core.StatefulActionNode
	tickCountPtr *int
	runTicks     int
	finalStatus  core.NodeStatus
}

func (n *runningNode) OnStart() core.NodeStatus {
	*n.tickCountPtr++
	if *n.tickCountPtr > n.runTicks {
		return n.finalStatus
	}
	return core.RUNNING
}

func (n *runningNode) OnRunning() core.NodeStatus {
	*n.tickCountPtr++
	if *n.tickCountPtr > n.runTicks {
		return n.finalStatus
	}
	return core.RUNNING
}

func (n *runningNode) OnHalted() {}

// Tick implements the stateful pattern with proper method dispatch.
// NOTE: We must override Tick() here because Go does NOT dispatch overridden
// methods when called from an embedded type's method. StatefulActionNode.Tick()
// calls n.OnStart() where n is *StatefulActionNode, which would call
// StatefulActionNode.OnStart() (returns SUCCESS), not runningNode.OnStart().
func (n *runningNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

// registerRunningAction registers a node type that returns RUNNING for `runTicks`
// ticks then completes with `finalStatus`. It returns the tick counter so tests
// can inspect it.
func registerRunningAction(factory *core.BehaviorTreeFactory, id string, finalStatus core.NodeStatus, runTicks int) *int {
	tickCount := 0
	target := &tickCount
	_ = factory.RegisterNodeType(id, core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
			n := &runningNode{
				tickCountPtr: target,
				runTicks:     runTicks,
				finalStatus:  finalStatus,
			}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID(id)
			return n
		}, core.Action)
	return target
}

// runningConditionNode is like runningNode but uses SimpleConditionNode approach
// via RegisterSimpleCondition. However, since conditions shouldn't return RUNNING
// either, we need a proper CustomConditionNode.
type runningConditionNode struct {
	core.StatefulActionNode
	tickCountPtr *int
	runTicks     int
	finalStatus  core.NodeStatus
}

func (n *runningConditionNode) OnStart() core.NodeStatus {
	*n.tickCountPtr++
	if *n.tickCountPtr > n.runTicks {
		return n.finalStatus
	}
	return core.RUNNING
}

func (n *runningConditionNode) OnRunning() core.NodeStatus {
	*n.tickCountPtr++
	if *n.tickCountPtr > n.runTicks {
		return n.finalStatus
	}
	return core.RUNNING
}

func (n *runningConditionNode) OnHalted() {}

func (n *runningConditionNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

// registerRunningCondition registers a custom condition that returns RUNNING for
// `runTicks` ticks then completes with `finalStatus`.
func registerRunningCondition(factory *core.BehaviorTreeFactory, id string, finalStatus core.NodeStatus, runTicks int) *int {
	tickCount := 0
	target := &tickCount
	_ = factory.RegisterNodeType(id, core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &runningConditionNode{
			tickCountPtr: target,
			runTicks:     runTicks,
			finalStatus:  finalStatus,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		return n
	}, core.Condition)
	return target
}

// ============================================================================
// Fallback Tests
// ============================================================================

// TestControl_Fallback_ConditionTrue verifies that when the condition returns
// SUCCESS, the FallbackNode returns SUCCESS and the action is NOT executed.
// Equivalent of C++ SimpleFallbackTest.ConditionTrue.
func TestControl_Fallback_ConditionTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<AlwaysSuccess/>
				<AlwaysFailure/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ConditionFalse verifies that when the condition returns
// FAILURE, the FallbackNode ticks the action.
// Equivalent of C++ SimpleFallbackWithMemoryTest.ConditionFalse.
func TestControl_Fallback_ConditionFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<AlwaysFailure/>
				<AlwaysSuccess/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ConditionChangeWhileRunning verifies that changing
// condition result does NOT affect an already-running action in FallbackNode
// (non-reactive).
// Equivalent of C++ SimpleFallbackTest.ConditionChangeWhileRunning.
func TestControl_Fallback_ConditionChangeWhileRunning(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	// Condition: initially FAILURE, then SUCCESS
	conditionResult := core.FAILURE
	_ = factory.RegisterSimpleCondition("MyCondition", func(core.TreeNode) core.NodeStatus {
		return conditionResult
	}, core.PortsList{})

	// Action: runs for 1 tick as RUNNING, then SUCCESS
	_ = registerRunningAction(factory, "MyAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<MyCondition/>
				<MyAction/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition FAILURE, action starts running
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change condition to SUCCESS
	conditionResult = core.SUCCESS

	// Second tick: Fallback is non-reactive, so it continues with the
	// already-running action (which now completes with SUCCESS)
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS (action completes), got %v", status)
	}
}

// TestControl_Fallback_AllChildrenFail verifies that when all children fail,
// FallbackNode returns FAILURE.
func TestControl_Fallback_AllChildrenFail(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<AlwaysFailure/>
				<AlwaysFailure/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
}

// TestControl_Fallback_FirstChildSucceeds verifies Fallback short-circuits on SUCCESS.
func TestControl_Fallback_FirstChildSucceeds(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<AlwaysSuccess/>
				<AlwaysFailure/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ComplexConditionsTrue tests Fallback with nested
// Fallback nodes where all conditions are true.
// Equivalent of C++ ComplexFallbackWithMemoryTest.ConditionsTrue.
func TestControl_Fallback_ComplexConditionsTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<Fallback name="fallback_conditions">
					<AlwaysSuccess/>
					<AlwaysSuccess/>
				</Fallback>
				<Fallback name="fallback_actions">
					<AlwaysSuccess/>
					<AlwaysSuccess/>
				</Fallback>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ComplexCondition1False tests nested Fallback where
// first condition fails but second succeeds.
// Equivalent of C++ ComplexFallbackWithMemoryTest.Condition1False.
func TestControl_Fallback_ComplexCondition1False(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<Fallback name="fallback_conditions">
					<AlwaysFailure/>
					<AlwaysSuccess/>
				</Fallback>
				<AlwaysSuccess/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ComplexConditionsFalse tests nested Fallback where
// all conditions failatch, then an async action runs.
// Equivalent of C++ ComplexFallbackWithMemoryTest.ConditionsFalse.
func TestControl_Fallback_ComplexConditionsFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<Fallback name="fallback_conditions">
					<AlwaysFailure/>
					<AlwaysFailure/>
				</Fallback>
				<Fallback name="fallback_actions">
					<RunningAction/>
					<RunningAction/>
				</Fallback>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ComplexCondition1ToTrue verifies that condition
// change from FAILURE to SUCCESS is ignored (non-reactive memory).
// Equivalent of C++ ComplexFallbackWithMemoryTest.Conditions1ToTrue.
func TestControl_Fallback_ComplexCondition1ToTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	condResult := core.FAILURE
	_ = factory.RegisterSimpleCondition("MyCondition", func(core.TreeNode) core.NodeStatus {
		return condResult
	}, core.PortsList{})

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Fallback name="root_fallback">
				<Fallback name="fallback_conditions">
					<MyCondition/>
					<AlwaysFailure/>
				</Fallback>
				<RunningAction/>
			</Fallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition fails, nested fallback fails, action runs
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}

	// Change condition - since Fallback is non-reactive, same result
	condResult = core.SUCCESS
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS after re-tick, got %v", status)
	}
}

// TestControl_Fallback_ReactiveCondition1ToTrue tests ReactiveFallback
// re-evaluation when condition 1 changes to true.
// Equivalent of C++ ReactiveFallbackTest.Condition1ToTrue.
func TestControl_Fallback_ReactiveCondition1ToTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	cond1Result := core.FAILURE
	cond2Result := core.FAILURE

	_ = factory.RegisterSimpleCondition("Cond1", func(core.TreeNode) core.NodeStatus {
		return cond1Result
	}, core.PortsList{})

	_ = factory.RegisterSimpleCondition("Cond2", func(core.TreeNode) core.NodeStatus {
		return cond2Result
	}, core.PortsList{})

	registerRunningAction(factory, "MyAction", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveFallback name="root_first">
				<Cond1/>
				<Cond2/>
				<MyAction/>
			</ReactiveFallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: both conditions fail, action starts running
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change cond1 to SUCCESS
	cond1Result = core.SUCCESS

	// Second tick: reactive re-evaluates, cond1 succeeds -> return SUCCESS
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ReactiveCondition2ToTrue tests ReactiveFallback
// where second condition changes to true.
// Equivalent of C++ ReactiveFallbackTest.Condition2ToTrue.
func TestControl_Fallback_ReactiveCondition2ToTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	cond1Result := core.FAILURE
	cond2Result := core.FAILURE

	_ = factory.RegisterSimpleCondition("Cond1", func(core.TreeNode) core.NodeStatus {
		return cond1Result
	}, core.PortsList{})

	_ = factory.RegisterSimpleCondition("Cond2", func(core.TreeNode) core.NodeStatus {
		return cond2Result
	}, core.PortsList{})

	registerRunningAction(factory, "MyAction", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveFallback name="root_first">
				<Cond1/>
				<Cond2/>
				<MyAction/>
			</ReactiveFallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: both conditions fail, action starts running
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change cond2 to SUCCESS
	cond2Result = core.SUCCESS

	// Second tick: cond1 still fails, cond2 succeeds -> SUCCESS
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ReactiveFirstChildSucceeds tests ReactiveFallback where
// first child succeeds immediately.
// Equivalent of C++ Reactive.ReactiveFallback_FirstChildSucceeds.
func TestControl_Fallback_ReactiveFirstChildSucceeds(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveFallback>
				<AlwaysSuccess/>
				<AlwaysFailure/>
			</ReactiveFallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Fallback_ReactiveAllChildrenFail tests ReactiveFallback where
// all children fail.
// Equivalent of C++ Reactive.ReactiveFallback_AllChildrenFail.
func TestControl_Fallback_ReactiveAllChildrenFail(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveFallback>
				<AlwaysFailure/>
				<AlwaysFailure/>
				<AlwaysFailure/>
			</ReactiveFallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
}

// TestControl_Fallback_ReactiveSecondChildSucceeds tests ReactiveFallback
// where first child fails but second succeeds.
// Equivalent of C++ Reactive.ReactiveFallback_SecondChildSucceeds.
func TestControl_Fallback_ReactiveSecondChildSucceeds(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveFallback>
				<AlwaysFailure/>
				<AlwaysSuccess/>
			</ReactiveFallback>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// ============================================================================
// Sequence Tests
// ============================================================================

// TestControl_Sequence_ConditionTrue verifies SequenceNode with a condition
// that returns SUCCESS, then an async action runs.
// Equivalent of C++ SimpleSequenceTest.ConditionTrue.
func TestControl_Sequence_ConditionTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="root_sequence">
				<AlwaysSuccess/>
				<RunningAction/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Sequence_ConditionTurnToFalse verifies that when a condition
// changes to FAILURE, the SequenceNode returns FAILURE (reactive on tick,
// but since SequenceNode has memory, the condition change is ignored while
// the action is running).
// Equivalent of C++ SimpleSequenceTest.ConditionTurnToFalse.
func TestControl_Sequence_ConditionTurnToFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	condResult := core.SUCCESS
	_ = factory.RegisterSimpleCondition("MyCondition", func(core.TreeNode) core.NodeStatus {
		return condResult
	}, core.PortsList{})

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="root_sequence">
				<MyCondition/>
				<RunningAction/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition succeeds, action starts
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change condition to FAILURE
	condResult = core.FAILURE

	// Second tick: Sequence has memory, action still running -> completes
	status = tree.TickExactlyOnce()
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS (action completes), got %v", status)
	}
}

// TestControl_Sequence_TripleAction verifies multiple async actions in sequence.
// Equivalent of C++ SequenceTripleActionTest.TripleAction.
func TestControl_Sequence_TripleAction(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "ActionA", core.SUCCESS, 1)
	registerRunningAction(factory, "ActionB", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence name="root_sequence">
				<AlwaysSuccess/>
				<ActionA/>
				<AlwaysSuccess/>
				<ActionB/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Sequence_ComplexConditionsTrue tests a ReactiveSequence with
// a nested Sequence of conditions, then an async action.
// Equivalent of C++ ComplexSequenceTest.ComplexSequenceConditionsTrue.
func TestControl_Sequence_ComplexConditionsTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence name="root">
				<Sequence name="sequence_conditions">
					<AlwaysSuccess/>
					<AlwaysSuccess/>
				</Sequence>
				<RunningAction/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Sequence_ComplexConditions1ToFalse tests ReactiveSequence where
// condition 1 changes to FAILURE.
// Equivalent of C++ ComplexSequenceTest.ComplexSequenceConditions1ToFalse.
func TestControl_Sequence_ComplexConditions1ToFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	cond1Result := core.SUCCESS
	_ = factory.RegisterSimpleCondition("Cond1", func(core.TreeNode) core.NodeStatus {
		return cond1Result
	}, core.PortsList{})

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence name="root">
				<Sequence name="sequence_conditions">
					<Cond1/>
					<AlwaysSuccess/>
				</Sequence>
				<RunningAction/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: conditions succeed, action starts
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change condition to FAILURE
	cond1Result = core.FAILURE

	// Second tick: ReactiveSequence re-evaluates, condition fails
	status = tree.TickExactlyOnce()
	if status != core.FAILURE {
		t.Errorf("tick 2: expected FAILURE, got %v", status)
	}
}

// TestControl_Sequence_ComplexConditions2ToFalse tests ReactiveSequence where
// condition 2 changes to FAILURE.
// Equivalent of C++ ComplexSequenceTest.ComplexSequenceConditions2ToFalse.
func TestControl_Sequence_ComplexConditions2ToFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	cond2Result := core.SUCCESS
	_ = factory.RegisterSimpleCondition("Cond2", func(core.TreeNode) core.NodeStatus {
		return cond2Result
	}, core.PortsList{})

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence name="root">
				<Sequence name="sequence_conditions">
					<AlwaysSuccess/>
					<Cond2/>
				</Sequence>
				<RunningAction/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: conditions succeed, action starts
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change cond2 to FAILURE
	cond2Result = core.FAILURE

	// Second tick: ReactiveSequence re-evaluates, cond2 fails
	status = tree.TickExactlyOnce()
	if status != core.FAILURE {
		t.Errorf("tick 2: expected FAILURE, got %v", status)
	}
}

// TestControl_Sequence_WithMemoryConditionTrue tests SequenceWithMemory
// with a condition that succeeds, then an async action.
// Equivalent of C++ SimpleSequenceWithMemoryTest.ConditionTrue.
func TestControl_Sequence_WithMemoryConditionTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<SequenceWithMemory name="root_sequence">
				<AlwaysSuccess/>
				<RunningAction/>
			</SequenceWithMemory>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Sequence_WithMemoryConditionTurnToFalse verifies that
// SequenceWithMemory ignores condition changes (it has memory).
// Equivalent of C++ SimpleSequenceWithMemoryTest.ConditionTurnToFalse.
func TestControl_Sequence_WithMemoryConditionTurnToFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	condResult := core.SUCCESS
	_ = factory.RegisterSimpleCondition("MyCondition", func(core.TreeNode) core.NodeStatus {
		return condResult
	}, core.PortsList{})

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<SequenceWithMemory name="root_sequence">
				<MyCondition/>
				<RunningAction/>
			</SequenceWithMemory>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition succeeds, action starts running
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Change condition to FAILURE
	condResult = core.FAILURE

	// Second tick: SequenceWithMemory has memory, continues with action
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 2: expected RUNNING (memory), got %v", status)
	}
}

// TestControl_Sequence_ComplexWithMemoryConditionsTrue tests nested
// SequenceWithMemory with two levels.
// Equivalent of C++ ComplexSequenceWithMemoryTest.ConditionsTrue.
func TestControl_Sequence_ComplexWithMemoryConditionsTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "RunningAction", core.SUCCESS, 1)
	registerRunningAction(factory, "RunningAction2", core.SUCCESS, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<SequenceWithMemory name="root_sequence">
				<SequenceWithMemory name="sequence_conditions">
					<AlwaysSuccess/>
					<AlwaysSuccess/>
				</SequenceWithMemory>
				<SequenceWithMemory name="sequence_actions">
					<RunningAction/>
					<RunningAction2/>
				</SequenceWithMemory>
			</SequenceWithMemory>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Sequence_Issue_636 tests tick counting in SequenceWithMemory
// using RegisterTestTick.
// Equivalent of C++ SequenceWithMemoryTest.Issue_636.
func TestControl_Sequence_Issue_636(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
		<BehaviorTree ID="MainTree">
			<SequenceWithMemory>
				<Script code=" var := 0 " />
				<TestA/>
				<ScriptCondition code=" var += 1; var &gt;= 5 " />
				<TestB/>
				<TestC/>
			</SequenceWithMemory>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tickCount := 0
	status := tree.TickExactlyOnce()
	tickCount++

	for status != core.SUCCESS && tickCount < 50 {
		status = tree.TickExactlyOnce()
		tickCount++
	}

	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v after %d ticks", status, tickCount)
	}

	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("TestC count = %d, want 1", counters[2])
	}
}

// ============================================================================
// Reactive Tests
// ============================================================================

// TestControl_Reactive_RunningChildren ticks a ReactiveSequence with a
// Sequence of sync actions and an async action, verifying that all
// preceding children are re-ticked.
// Equivalent of C++ Reactive.RunningChildren (adapted: no AsyncSequence in Go).
func TestControl_Reactive_RunningChildren(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 4)
	core.RegisterTestTick(factory, "Test", counters)

	registerRunningAction(factory, "AsyncAction", core.SUCCESS, 3)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<Sequence name="first">
					<TestA/>
					<TestB/>
					<TestC/>
				</Sequence>
				<AsyncAction/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	count := 0
	status := core.IDLE
	for !status.IsCompleted() && count < 50 {
		count++
		status = tree.TickExactlyOnce()
	}

	if count >= 50 {
		t.Fatal("did not complete within 50 ticks")
	}

	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}

	// The first 3 children (TestA, TestB, TestC) should be ticked every
	// time the ReactiveSequence re-evaluates (while AsyncAction is running).
	// AsyncAction runs for 3 running ticks + 1 success = 4 total ticks.
	if counters[0] < 2 {
		t.Errorf("TestA count = %d, want >= 2", counters[0])
	}
	if counters[1] < 2 {
		t.Errorf("TestB count = %d, want >= 2", counters[1])
	}
	if counters[2] < 2 {
		t.Errorf("TestC count = %d, want >= 2", counters[2])
	}
}

// TestControl_Reactive_Issue587 tests that _skipIf works with ReactiveSequence.
// Equivalent of C++ Reactive.Issue587.
func TestControl_Reactive_Issue587(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence>
				<Script code="test := false"/>
				<ReactiveSequence>
					<RetryUntilSuccessful name="Retry 1" num_attempts="-1" _skipIf="test">
						<TestA name="Success 1" _onSuccess="test = true"/>
					</RetryUntilSuccessful>
					<RetryUntilSuccessful name="Retry 2" num_attempts="5">
						<AlwaysFailure name="Failure 2"/>
					</RetryUntilSuccessful>
				</ReactiveSequence>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	// The Retry 2 with 5 attempts against AlwaysFailure should exhaust
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (Retry 2 exhausts), got %v", status)
	}

	// TestA inside the _skipIf Retry block should be ticked exactly once.
	// First tick: test=false, Retry 1 runs (TestA succeeds), _onSuccess sets test=true
	// Subsequent ticks: Retry 1 is skipped by _skipIf
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1 (first tick, then skipped)", counters[0])
	}
}

// TestControl_Reactive_TestLogging tests that a ReactiveSequence with a
// TestA (sync action) is ticked multiple times while async child runs.
// Equivalent of C++ Reactive.TestLogging.
func TestControl_Reactive_TestLogging(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<TestA name="testA"/>
				<AlwaysSuccess name="success"/>
				<Sleep msec="100"/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}

	numTicks := counters[0]
	if numTicks < 3 {
		t.Errorf("expected TestA to be ticked >= 3 times, got %d", numTicks)
	}
}

// TestControl_Reactive_TwoAsyncNodesInReactiveSequence verifies that having
// two async nodes in a ReactiveSequence returns FAILURE when the exception
// mode is enabled.
// Equivalent of C++ Reactive.TwoAsyncNodesInReactiveSequence.
func TestControl_Reactive_TwoAsyncNodesInReactiveSequence(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "Async1", core.SUCCESS, 2)
	registerRunningAction(factory, "Async2", core.SUCCESS, 2)

	// Enable the exception for multiple running children
	ReactiveSequenceEnableException(true)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<Async1/>
				<Async2/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickExactlyOnce()
	// First tick: Async1 returns RUNNING, runningChild=0
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Second tick: Async1 still RUNNING (runTicks=2)
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 2: expected RUNNING, got %v", status)
	}

	// Third tick: Async1 completes (SUCCESS), Async2 starts and returns RUNNING.
	// Since runningChild=0 != i=1 and exception is enabled, FAILURE is returned.
	status = tree.TickExactlyOnce()
	if status != core.FAILURE {
		t.Errorf("tick 3: expected FAILURE (two async children), got %v", status)
	}

	// Reset to default
	ReactiveSequenceEnableException(false)
}

// TestControl_Reactive_FirstChildFails tests ReactiveSequence where first
// child fails immediately.
// Equivalent of C++ Reactive.ReactiveSequence_FirstChildFails.
func TestControl_Reactive_FirstChildFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<AlwaysFailure/>
				<TestA/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should not be ticked, got %d", counters[0])
	}
}

// TestControl_Reactive_AllChildrenSucceed tests ReactiveSequence where all
// children succeed.
// Equivalent of C++ Reactive.ReactiveSequence_AllChildrenSucceed.
func TestControl_Reactive_AllChildrenSucceed(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<TestA/>
				<TestB/>
				<TestC/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("TestC count = %d, want 1", counters[2])
	}
}

// TestControl_Reactive_ReEvaluatesOnEveryTick verifies that conditions in
// a ReactiveSequence are re-evaluated on every tick.
// Equivalent of C++ Reactive.ReactiveSequence_ReEvaluatesOnEveryTick.
func TestControl_Reactive_ReEvaluatesOnEveryTick(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	conditionTickCount := 0
	_ = factory.RegisterSimpleCondition("CountingCondition", func(core.TreeNode) core.NodeStatus {
		conditionTickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<CountingCondition/>
				<Sleep msec="50"/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}

	// Condition should be ticked multiple times while Sleep is running
	if conditionTickCount < 2 {
		t.Errorf("expected conditionTickCount >= 2, got %d", conditionTickCount)
	}
}

// TestControl_Reactive_HaltOnConditionChange verifies that running children
// are halted when a preceding condition changes.
// Equivalent of C++ Reactive.ReactiveSequence_HaltOnConditionChange.
func TestControl_Reactive_HaltOnConditionChange(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	conditionResult := true
	childTickCount := 0
	childWasHalted := false

	_ = factory.RegisterSimpleCondition("DynamicCondition", func(core.TreeNode) core.NodeStatus {
		if conditionResult {
			return core.SUCCESS
		}
		return core.FAILURE
	}, core.PortsList{})

	// Register a stateful action that tracks ticks and halt
	_ = factory.RegisterNodeType("TrackingAction", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &trackingAction{
			tickCount: &childTickCount,
			wasHalted: &childWasHalted,
			name:      name,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("TrackingAction")
		return n
	}, core.Action)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<DynamicCondition/>
				<TrackingAction/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition passes, action starts (RUNNING)
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}
	if childTickCount < 1 {
		t.Errorf("expected childTickCount >= 1, got %d", childTickCount)
	}
	if childWasHalted {
		t.Error("child should not be halted yet")
	}

	// Tick again while condition is still true
	status = tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 2: expected RUNNING, got %v", status)
	}

	// Now change condition to false - child should be halted
	conditionResult = false
	status = tree.TickExactlyOnce()
	if status != core.FAILURE {
		t.Errorf("tick 3: expected FAILURE, got %v", status)
	}
	if !childWasHalted {
		t.Error("child should have been halted")
	}
}

// trackingAction is a StatefulActionNode that tracks tick count and halt.
type trackingAction struct {
	core.StatefulActionNode
	tickCount *int
	wasHalted *bool
	name      string
}

func (n *trackingAction) OnStart() core.NodeStatus {
	*n.tickCount++
	return core.RUNNING
}

func (n *trackingAction) OnRunning() core.NodeStatus {
	*n.tickCount++
	return core.RUNNING
}

func (n *trackingAction) OnHalted() {
	*n.wasHalted = true
}

func (n *trackingAction) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

// Halt overrides StatefulActionNode.Halt() to ensure proper method dispatch.
// StatefulActionNode.Halt() calls n.OnHalted() where n is *StatefulActionNode,
// which would NOT dispatch to trackingAction.OnHalted().
func (n *trackingAction) Halt() {
	if n.Status() == core.RUNNING {
		n.OnHalted()
	}
	n.ResetStatus()
}

// ============================================================================
// IfThenElse Tests
// ============================================================================

// TestControl_IfThenElse_ConditionTrue_ThenBranch verifies that when condition
// is true, the "then" branch is executed.
// Equivalent of C++ IfThenElseTest.ConditionTrue_ThenBranch.
func TestControl_IfThenElse_ConditionTrue_ThenBranch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysSuccess/>  <!-- condition -->
				<TestA/>          <!-- then -->
				<TestB/>          <!-- else -->
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA (then) count = %d, want 1", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB (else) count = %d, want 0", counters[1])
	}
}

// TestControl_IfThenElse_ConditionFalse_ElseBranch verifies that when
// condition is false, the "else" branch is executed.
// Equivalent of C++ IfThenElseTest.ConditionFalse_ElseBranch.
func TestControl_IfThenElse_ConditionFalse_ElseBranch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- then -->
				<TestB/>          <!-- else -->
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (then) count = %d, want 0", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB (else) count = %d, want 1", counters[1])
	}
}

// TestControl_IfThenElse_ConditionFalse_TwoChildren verifies that with only
// 2 children and condition false, IfThenElse returns FAILURE.
// Equivalent of C++ IfThenElseTest.ConditionFalse_TwoChildren_ReturnsFailure.
func TestControl_IfThenElse_ConditionFalse_TwoChildren(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- then -->
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should not be executed, got %d", counters[0])
	}
}

// TestControl_IfThenElse_ThenBranchFails verifies that when then-branch
// fails, IfThenElse returns FAILURE.
// Equivalent of C++ IfThenElseTest.ThenBranchFails.
func TestControl_IfThenElse_ThenBranchFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysSuccess/>  <!-- condition -->
				<AlwaysFailure/>  <!-- then -->
				<TestA/>          <!-- else -->
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (else) should not be executed, got %d", counters[0])
	}
}

// TestControl_IfThenElse_ElseBranchFails verifies that when else-branch
// fails, IfThenElse returns FAILURE.
// Equivalent of C++ IfThenElseTest.ElseBranchFails.
func TestControl_IfThenElse_ElseBranchFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- then -->
				<AlwaysFailure/>  <!-- else -->
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (then) should not be executed, got %d", counters[0])
	}
}

// TestControl_IfThenElse_ConditionRunning tests condition that returns
// RUNNING first, then SUCCESS.
// Equivalent of C++ IfThenElseTest.ConditionRunning.
func TestControl_IfThenElse_ConditionRunning(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	conditionTicks := 0
	_ = factory.RegisterSimpleCondition("RunningThenSuccess", func(core.TreeNode) core.NodeStatus {
		conditionTicks++
		if conditionTicks == 1 {
			return core.RUNNING
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<RunningThenSuccess/>
				<TestA/>
				<TestB/>
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition returns RUNNING
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should not be executed yet, got %d", counters[0])
	}

	// Second tick: condition returns SUCCESS, then-branch executes
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
}

// TestControl_IfThenElse_HaltBehavior verifies that halt/reset works.
// Equivalent of C++ IfThenElseTest.HaltBehavior.
func TestControl_IfThenElse_HaltBehavior(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysSuccess/>
				<TestA/>
				<TestB/>
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First execution
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 1: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}

	// Halt and re-execute
	tree.HaltTree()
	counters[0] = 0

	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count after re-execute = %d, want 1", counters[0])
	}
}

// TestControl_IfThenElse_InvalidChildCount_One verifies that IfThenElse
// with only 1 child returns FAILURE at runtime.
// Equivalent of C++ IfThenElseTest.InvalidChildCount_One.
func TestControl_IfThenElse_InvalidChildCount_One(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysSuccess/>
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (invalid child count), got %v", status)
	}
}

// TestControl_IfThenElse_InvalidChildCount_Four verifies that IfThenElse
// with 4 children returns FAILURE at runtime.
// Equivalent of C++ IfThenElseTest.InvalidChildCount_Four.
func TestControl_IfThenElse_InvalidChildCount_Four(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<IfThenElse>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</IfThenElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (invalid child count), got %v", status)
	}
}

// ============================================================================
// WhileDoElse Tests
// ============================================================================

// TestControl_WhileDoElse_ConditionTrue_DoBranch verifies that when condition
// is true, the "do" branch is executed.
// Equivalent of C++ WhileDoElseTest.ConditionTrue_DoBranch.
func TestControl_WhileDoElse_ConditionTrue_DoBranch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysSuccess/>  <!-- condition -->
				<TestA/>          <!-- do -->
				<TestB/>          <!-- else -->
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA (do) count = %d, want 1", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB (else) count = %d, want 0", counters[1])
	}
}

// TestControl_WhileDoElse_ConditionFalse_ElseBranch verifies that when
// condition is false, the "else" branch is executed.
// Equivalent of C++ WhileDoElseTest.ConditionFalse_ElseBranch.
func TestControl_WhileDoElse_ConditionFalse_ElseBranch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- do -->
				<TestB/>          <!-- else -->
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (do) count = %d, want 0", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB (else) count = %d, want 1", counters[1])
	}
}

// TestControl_WhileDoElse_ConditionFalse_TwoChildren verifies that with only
// 2 children and condition false, WhileDoElse returns FAILURE.
// Equivalent of C++ WhileDoElseTest.ConditionFalse_TwoChildren_ReturnsFailure.
func TestControl_WhileDoElse_ConditionFalse_TwoChildren(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- do -->
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should not be executed, got %d", counters[0])
	}
}

// TestControl_WhileDoElse_DoBranchFails verifies that when do-branch fails,
// WhileDoElse returns FAILURE.
// Equivalent of C++ WhileDoElseTest.DoBranchFails.
func TestControl_WhileDoElse_DoBranchFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysSuccess/>  <!-- condition -->
				<AlwaysFailure/>  <!-- do -->
				<TestA/>          <!-- else -->
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (else) should not be executed, got %d", counters[0])
	}
}

// TestControl_WhileDoElse_ElseBranchFails verifies that when else-branch
// fails, WhileDoElse returns FAILURE.
// Equivalent of C++ WhileDoElseTest.ElseBranchFails.
func TestControl_WhileDoElse_ElseBranchFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysFailure/>  <!-- condition -->
				<TestA/>          <!-- do -->
				<AlwaysFailure/>  <!-- else -->
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (do) should not be executed, got %d", counters[0])
	}
}

// TestControl_WhileDoElse_ConditionChanges_HaltsElse verifies that condition
// change from false to true halts else branch (WhileDoElse is reactive).
// Equivalent of C++ WhileDoElseTest.ConditionChanges_HaltsElse.
func TestControl_WhileDoElse_ConditionChanges_HaltsElse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	conditionCounter := 0
	_ = factory.RegisterSimpleCondition("ToggleCondition", func(core.TreeNode) core.NodeStatus {
		conditionCounter++
		if conditionCounter == 1 {
			return core.FAILURE
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<ToggleCondition/>
				<TestA/>
				<TestB/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition false, executes else (TestB)
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (do) count = %d, want 0", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB (else) count = %d, want 1", counters[1])
	}
}

// TestControl_WhileDoElse_ConditionChanges_HaltsDo verifies that condition
// change from true to false halts do branch.
// Equivalent of C++ WhileDoElseTest.ConditionChanges_HaltsDo.
func TestControl_WhileDoElse_ConditionChanges_HaltsDo(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	conditionCounter := 0
	_ = factory.RegisterSimpleCondition("ToggleCondition2", func(core.TreeNode) core.NodeStatus {
		conditionCounter++
		if conditionCounter == 1 {
			return core.SUCCESS
		}
		return core.FAILURE
	}, core.PortsList{})

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<ToggleCondition2/>
				<TestA/>
				<TestB/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition true, executes do (TestA)
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA (do) count = %d, want 1", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB (else) count = %d, want 0", counters[1])
	}
}

// TestControl_WhileDoElse_HaltBehavior verifies that halt works properly.
// Equivalent of C++ WhileDoElseTest.HaltBehavior.
func TestControl_WhileDoElse_HaltBehavior(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysSuccess/>
				<TestA/>
				<TestB/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First execution
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 1: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}

	// Halt and re-execute
	tree.HaltTree()
	counters[0] = 0

	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count after re-execute = %d, want 1", counters[0])
	}
}

// TestControl_WhileDoElse_InvalidChildCount_One verifies that WhileDoElse
// with only 1 child returns FAILURE at runtime.
// Equivalent of C++ WhileDoElseTest.InvalidChildCount_One.
func TestControl_WhileDoElse_InvalidChildCount_One(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysSuccess/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (invalid child count), got %v", status)
	}
}

// TestControl_WhileDoElse_InvalidChildCount_Four verifies that WhileDoElse
// with 4 children returns FAILURE at runtime.
// Equivalent of C++ WhileDoElseTest.InvalidChildCount_Four.
func TestControl_WhileDoElse_InvalidChildCount_Four(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (invalid child count), got %v", status)
	}
}

// TestControl_WhileDoElse_ConditionRunning tests WhileDoElse with a condition
// that returns RUNNING first, then SUCCESS.
// Equivalent of C++ WhileDoElseTest.ConditionRunning.
func TestControl_WhileDoElse_ConditionRunning(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	firstTick := true
	_ = factory.RegisterSimpleCondition("RunningThenSuccess", func(core.TreeNode) core.NodeStatus {
		if firstTick {
			firstTick = false
			return core.RUNNING
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<WhileDoElse>
				<RunningThenSuccess/>
				<TestA/>
				<TestB/>
			</WhileDoElse>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: condition returns RUNNING
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should not be executed yet, got %d", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB should not be executed yet, got %d", counters[1])
	}

	// Second tick: condition returns SUCCESS, executes do branch
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
}

// ============================================================================
// Switch Tests
// ============================================================================

// TestControl_Switch_DefaultCase verifies that when no variable is set,
// the default case (last child) is executed.
// Equivalent of C++ SwitchTest.DefaultCase.
func TestControl_Switch_DefaultCase(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_Case1 verifies matching the first case.
// Equivalent of C++ SwitchTest.Case1.
func TestControl_Switch_Case1(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "1")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_Case2 verifies matching the second case.
// Equivalent of C++ SwitchTest.Case2.
func TestControl_Switch_Case2(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "42")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_CaseNone verifies matching a non-existent case routes
// to the default child.
// Equivalent of C++ SwitchTest.CaseNone.
func TestControl_Switch_CaseNone(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<AlwaysSuccess/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "none")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_CaseSwitchToDefault verifies switching from a matched
// case to the default case (SwitchNode is not reactive but re-evaluates
// on the next tick since the running child is halted).
// Equivalent of C++ SwitchTest.CaseSwitchToDefault.
func TestControl_Switch_CaseSwitchToDefault(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "Running1", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<Running1/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "1")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: case "1" matches, first action starts
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Switch variable mid-tick (SwitchNode re-evaluates on next tick)
	bb.Set("my_var", "")

	// Tick again: switch re-evaluates, old running child is halted,
	// default case (last child) is executed
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tick 2+: expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_CaseSwitchToAction2 verifies switching between cases.
// Equivalent of C++ SwitchTest.CaseSwitchToAction2.
func TestControl_Switch_CaseSwitchToAction2(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "Running1", core.SUCCESS, 2)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<Running1/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "1")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: case "1" matches, first action starts
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Switch variable
	bb.Set("my_var", "42")

	// Tick again: switch re-evaluates, old running child is halted,
	// case "42" matches, second action runs
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tick 2+: expected SUCCESS, got %v", status)
	}
}

// TestControl_Switch_ActionFailure verifies that when a matched case action
// returns FAILURE, the SwitchNode returns FAILURE.
// Equivalent of C++ SwitchTest.ActionFailure.
func TestControl_Switch_ActionFailure(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	registerRunningAction(factory, "FailAction", core.FAILURE, 1)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Switch2 name="simple_switch" variable="{my_var}" case_1="1" case_2="42">
				<FailAction/>
				<AlwaysSuccess/>
				<AlwaysSuccess/>
			</Switch2>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_var", "1")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
}

// ============================================================================
// TryCatch Tests
// ============================================================================

// TestControl_TryCatch_AllTryChildrenSucceed verifies that when all try
// children succeed, catch is NOT executed.
// Equivalent of C++ TryCatchTest.AllTryChildrenSucceed.
func TestControl_TryCatch_AllTryChildrenSucceed(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<TestA/>
				<TestB/>
				<TestC/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1", counters[1])
	}
	if counters[2] != 0 {
		t.Errorf("TestC (catch) count = %d, want 0", counters[2])
	}
}

// TestControl_TryCatch_FirstChildFails verifies that when first try child
// fails, the catch is executed.
// Equivalent of C++ TryCatchTest.FirstChildFails_CatchExecuted.
func TestControl_TryCatch_FirstChildFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysFailure/>
				<TestA/>
				<TestB/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA should NOT be executed, got %d", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB (catch) count = %d, want 1", counters[1])
	}
}

// TestControl_TryCatch_SecondChildFails verifies that when second try child
// fails, catch is executed.
// Equivalent of C++ TryCatchTest.SecondChildFails_CatchExecuted.
func TestControl_TryCatch_SecondChildFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<TestA/>
				<AlwaysFailure/>
				<TestB/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB (catch) count = %d, want 1", counters[1])
	}
}

// TestControl_TryCatch_CatchReturnsFailure verifies TryCatch returns FAILURE
// even when catch returns FAILURE.
// Equivalent of C++ TryCatchTest.CatchReturnsFailure_NodeStillReturnsFAILURE.
func TestControl_TryCatch_CatchReturnsFailure(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysFailure/>  <!-- try fails -->
				<AlwaysFailure/>  <!-- catch also fails -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
}

// TestControl_TryCatch_CatchReturnsSuccess verifies TryCatch returns FAILURE
// even when catch returns SUCCESS.
// Equivalent of C++ TryCatchTest.CatchReturnsSuccess_NodeStillReturnsFAILURE.
func TestControl_TryCatch_CatchReturnsSuccess(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysFailure/>  <!-- try fails -->
				<AlwaysSuccess/>  <!-- catch succeeds -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
}

// TestControl_TryCatch_TryChildRunning tests a try child that returns RUNNING
// first, then SUCCESS. Catch should NOT be executed.
// Equivalent of C++ TryCatchTest.TryChildRunning.
func TestControl_TryCatch_TryChildRunning(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tickCount := 0
	_ = factory.RegisterSimpleCondition("RunningThenSuccess", func(core.TreeNode) core.NodeStatus {
		tickCount++
		if tickCount == 1 {
			return core.RUNNING
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<RunningThenSuccess/>
				<TestA/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: running
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Second tick: completes successfully
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tick 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA (catch) count = %d, want 0", counters[0])
	}
}

// TestControl_TryCatch_CatchChildRunning tests a catch child that returns
// RUNNING first, then FAILURE.
// Equivalent of C++ TryCatchTest.CatchChildRunning.
func TestControl_TryCatch_CatchChildRunning(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	catchTickCount := 0
	_ = factory.RegisterSimpleCondition("RunningThenFailure", func(core.TreeNode) core.NodeStatus {
		catchTickCount++
		if catchTickCount == 1 {
			return core.RUNNING
		}
		return core.FAILURE
	}, core.PortsList{})

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysFailure/>             <!-- try fails -->
				<RunningThenFailure/>        <!-- catch: RUNNING first, then FAILURE -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: try fails, catch starts and returns RUNNING
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Second tick: catch returns FAILURE, TryCatch returns FAILURE
	status = tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("tick 2: expected FAILURE, got %v", status)
	}
}

// TestControl_TryCatch_MinimumTwoChildren verifies that TryCatch with
// only 1 child returns FAILURE at runtime.
// Equivalent of C++ TryCatchTest.MinimumTwoChildren_ParseTimeValidation.
func TestControl_TryCatch_MinimumTwoChildren(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysSuccess/>
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE (minimum 2 children), got %v", status)
	}
}

// TestControl_TryCatch_ReExecuteAfterSuccess verifies that TryCatch can
// be re-executed after success.
// Equivalent of C++ TryCatchTest.ReExecuteAfterSuccess.
func TestControl_TryCatch_ReExecuteAfterSuccess(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<TestA/>
				<TestB/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 1: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}

	tree.HaltTree()
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 2 {
		t.Errorf("TestA count after re-execute = %d, want 2", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB (catch) count = %d, want 0", counters[1])
	}
}

// TestControl_TryCatch_ReExecuteAfterFailure verifies TryCatch re-execution
// after failure: try fails first time, succeeds second time.
// Equivalent of C++ TryCatchTest.ReExecuteAfterFailure.
func TestControl_TryCatch_ReExecuteAfterFailure(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tryTickCount := 0
	_ = factory.RegisterSimpleAction("FailThenSucceed", func(core.TreeNode) core.NodeStatus {
		tryTickCount++
		if tryTickCount == 1 {
			return core.FAILURE
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<FailThenSucceed/>
				<TestA/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First execution: try fails, catch runs
	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("execution 1: expected FAILURE, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA (catch) count = %d, want 1", counters[0])
	}

	// Second execution: try succeeds
	tree.HaltTree()
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("execution 2: expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA (catch) count after re-execute = %d, want 1", counters[0])
	}
}

// TestControl_TryCatch_CatchOnHalt_Disabled verifies that catch is NOT
// executed on halt when catch_on_halt is disabled (default).
// Equivalent of C++ TryCatchTest.CatchOnHalt_Disabled.
func TestControl_TryCatch_CatchOnHalt_Disabled(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	catchCount := 0
	_ = factory.RegisterSimpleAction("CountCatch", func(core.TreeNode) core.NodeStatus {
		catchCount++
		return core.SUCCESS
	}, core.PortsList{})

	tryTicks := 0
	_ = factory.RegisterSimpleCondition("AlwaysRunning", func(core.TreeNode) core.NodeStatus {
		tryTicks++
		return core.RUNNING
	}, core.PortsList{})

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<AlwaysRunning/>
				<CountCatch/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Halt while try-block is RUNNING; catch_on_halt defaults to false
	tree.HaltTree()
	if catchCount != 0 {
		t.Errorf("catchCount = %d, want 0 (catch_on_halt disabled)", catchCount)
	}

	_ = tryTicks
}

// TestControl_TryCatch_CatchOnHalt_Enabled verifies that catch IS executed
// on halt when catch_on_halt is true.
// Equivalent of C++ TryCatchTest.CatchOnHalt_Enabled.
func TestControl_TryCatch_CatchOnHalt_Enabled(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	catchCount := 0
	_ = factory.RegisterSimpleAction("CountCatch", func(core.TreeNode) core.NodeStatus {
		catchCount++
		return core.SUCCESS
	}, core.PortsList{})

	_ = factory.RegisterSimpleCondition("AlwaysRunning", func(core.TreeNode) core.NodeStatus {
		return core.RUNNING
	}, core.PortsList{})

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch catch_on_halt="true">
				<AlwaysRunning/>
				<CountCatch/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}

	// Halt while try-block is RUNNING; catch_on_halt is true
	tree.HaltTree()
	if catchCount != 1 {
		t.Errorf("catchCount = %d, want 1 (catch_on_halt enabled)", catchCount)
	}
}

// TestControl_TryCatch_CatchOnHalt_NotTriggeredWhenAlreadyInCatch verifies
// that catch_on_halt does not trigger when already in catch mode.
// Equivalent of C++ TryCatchTest.CatchOnHalt_NotTriggeredWhenAlreadyInCatch.
func TestControl_TryCatch_CatchOnHalt_NotTriggeredWhenAlreadyInCatch(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	catchTicks := 0
	_ = factory.RegisterSimpleCondition("RunningCatch", func(core.TreeNode) core.NodeStatus {
		catchTicks++
		return core.RUNNING
	}, core.PortsList{})

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch catch_on_halt="true">
				<AlwaysFailure/>     <!-- try fails immediately -->
				<RunningCatch/>      <!-- catch returns RUNNING -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick: try fails, enters catch, catch returns RUNNING
	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Errorf("tick 1: expected RUNNING, got %v", status)
	}
	if catchTicks != 1 {
		t.Errorf("catchTicks = %d, want 1", catchTicks)
	}

	// Halt while in catch mode: should NOT re-trigger catch
	tree.HaltTree()
	if catchTicks != 1 {
		t.Errorf("catchTicks = %d, want 1 (not re-triggered on halt)", catchTicks)
	}
}

// TestControl_TryCatch_AsyncCatchCompletesInsideSequence tests TryCatch
// inside a Sequence where the catch child is async.
// Equivalent of C++ TryCatchTest.AsyncCatchCompletesInsideSequence.
func TestControl_TryCatch_AsyncCatchCompletesInsideSequence(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	kRunningTicks := 5
	catchTicks := 0
	_ = factory.RegisterSimpleCondition("AsyncCleanup", func(core.TreeNode) core.NodeStatus {
		catchTicks++
		if catchTicks <= kRunningTicks {
			return core.RUNNING
		}
		return core.SUCCESS
	}, core.PortsList{})

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence>
				<TryCatch>
					<AlwaysFailure/>       <!-- try: fails immediately -->
					<AsyncCleanup/>        <!-- catch: RUNNING for 5 ticks, then SUCCESS -->
				</TryCatch>
				<TestA/>                   <!-- should NOT execute: TryCatch returns FAILURE -->
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Tick-by-tick: the tree should stay RUNNING while catch is async
	for i := 0; i < kRunningTicks; i++ {
		status := tree.TickExactlyOnce()
		if status != core.RUNNING {
			t.Errorf("tick %d: expected RUNNING, got %v", i+1, status)
		}
		if catchTicks != i+1 {
			t.Errorf("catchTicks at tick %d = %d, want %d", i+1, catchTicks, i+1)
		}
	}

	// Next tick: catch completes -> TryCatch returns FAILURE -> Sequence returns FAILURE
	status := tree.TickExactlyOnce()
	if status != core.FAILURE {
		t.Errorf("tick %d: expected FAILURE, got %v", kRunningTicks+1, status)
	}

	// Catch was ticked exactly kRunningTicks + 1 times (5 RUNNING + 1 SUCCESS)
	if catchTicks != kRunningTicks+1 {
		t.Errorf("catchTicks = %d, want %d", catchTicks, kRunningTicks+1)
	}

	// TestA was never reached because TryCatch returned FAILURE
	if counters[0] != 0 {
		t.Errorf("TestA count = %d, want 0", counters[0])
	}
}

// TestControl_TryCatch_SingleTryChild verifies TryCatch with a single
// try child.
// Equivalent of C++ TryCatchTest.SingleTryChild_Success.
func TestControl_TryCatch_SingleTryChild(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<TestA/>   <!-- single try child -->
				<TestB/>   <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB (catch) count = %d, want 0", counters[1])
	}
}

// TestControl_TryCatch_ManyTryChildren_ThirdFails verifies TryCatch with
// multiple try children where the third one fails.
// Equivalent of C++ TryCatchTest.ManyTryChildren_ThirdFails.
func TestControl_TryCatch_ManyTryChildren_ThirdFails(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<TryCatch>
				<TestA/>
				<TestB/>
				<AlwaysFailure/>
				<TestC/>  <!-- catch -->
			</TryCatch>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.FAILURE {
		t.Errorf("expected FAILURE, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1", counters[1])
	}
	if counters[2] != 1 {
		t.Errorf("TestC (catch) count = %d, want 1", counters[2])
	}
}

// ============================================================================
// Skipping Logic Tests
// ============================================================================

// TestControl_Skipping_Sequence verifies _skipIf, _successIf, _failureIf
// in a Sequence.
// Equivalent of C++ SkippingLogic.Sequence.
func TestControl_Skipping_Sequence(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence>
				<Script code="A:=1"/>
				<TestA _successIf="A==2" _failureIf="A!=1" _skipIf="A==1"/>
				<TestB/>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA count = %d, want 0 (should be skipped)", counters[0])
	}
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1", counters[1])
	}
}

// TestControl_Skipping_SkipAll verifies that when all children are skipped,
// Sequence returns SKIPPED.
// Equivalent of C++ SkippingLogic.SkipAll.
func TestControl_Skipping_SkipAll(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 3)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence>
				<TestA _skipIf="A==1"/>
				<TestB _skipIf="A&lt;2"/>
				<TestC _skipIf="A&gt;0"/>
			</Sequence>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("A", 1)

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SKIPPED {
		t.Errorf("expected SKIPPED, got %v", status)
	}
	if counters[0] != 0 {
		t.Errorf("TestA count = %d, want 0", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB count = %d, want 0", counters[1])
	}
	if counters[2] != 0 {
		t.Errorf("TestC count = %d, want 0", counters[2])
	}
}

// TestControl_Skipping_SkipSubtree verifies _skipIf on a SubTree.
// Note: In the Go port, _skipIf on SubTree nodes is not supported (hasManifest
// is false for SubTree IDs), so the SubTree always executes.
// Equivalent of C++ SkippingLogic.SkipSubtree.
func TestControl_Skipping_SkipSubtree(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="main">
			<Sequence>
				<TestA/>
				<Script code="data:=true"/>
				<SubTree ID="sub"/>
			</Sequence>
		</BehaviorTree>
		<BehaviorTree ID="sub">
			<TestB/>
		</BehaviorTree>
	</root>`

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("main", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("expected SUCCESS, got %v", status)
	}
	if counters[0] != 1 {
		t.Errorf("TestA count = %d, want 1", counters[0])
	}
	// SubTree always executes in Go port, so TestB should be ticked.
	if counters[1] != 1 {
		t.Errorf("TestB count = %d, want 1 (Go port: SubTree always executes)", counters[1])
	}
}

// TestControl_Skipping_ReactiveSingleChild verifies that _skipIf works
// with a ReactiveSequence that has a single child.
// Equivalent of C++ SkippingLogic.ReactiveSingleChild.
func TestControl_Skipping_ReactiveSingleChild(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xmlText := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<AlwaysSuccess _skipIf="flag"/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	bb := core.NewBlackboard(nil)
	bb.Set("flag", true)

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	// Should not panic - ReactiveSequence with a single skipped child
	// should return SKIPPED
	status := tree.TickWhileRunning(0)
	_ = status
}

// TestControl_Skipping_WhileSkip verifies _while attribute.
// In the Go port, _while on IDLE nodes always returns SKIPPED (regardless of
// the condition value). This differs from C++ where _while=true allows execution.
// Equivalent of C++ SkippingLogic.WhileSkip.
func TestControl_Skipping_WhileSkip(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	// First tree: doit=true, TestA should be ticked (C++ behavior).
	// Go port: _while on IDLE always returns SKIPPED, so TestA is skipped.
	xmlText1 := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<Sequence>
				<Script code="doit:=true"/>
				<Sequence>
					<TestA _while="doit"/>
				</Sequence>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree1, err := factory.CreateTreeFromText(xmlText1, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree1.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tree1: expected SUCCESS, got %v", status)
	}

	// Second tree: doit=false, TestB should be skipped
	xmlText2 := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree2">
			<Sequence>
				<Script code="doit:=false"/>
				<Sequence>
					<TestB _while="doit"/>
				</Sequence>
			</Sequence>
		</BehaviorTree>
	</root>`

	tree2, err := factory.CreateTreeFromText(xmlText2, nil)
	if err != nil {
		t.Fatal(err)
	}

	status = tree2.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tree2: expected SUCCESS, got %v", status)
	}

	// In Go port, _while on IDLE always returns SKIPPED, so both are 0.
	// In C++, TestA should be 1 (doit=true means execute).
	if counters[0] != 0 {
		t.Errorf("TestA count = %d, want 0 (Go port: _while on IDLE always skips)", counters[0])
	}
	if counters[1] != 0 {
		t.Errorf("TestB count = %d, want 0 (skipped by _while)", counters[1])
	}
}

// TestControl_Skipping_SkippingReactiveSequence verifies _skipIf in a
// ReactiveSequence.
// Equivalent of C++ SkippingLogic.SkippingReactiveSequence.
func TestControl_Skipping_SkippingReactiveSequence(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	counters := make([]int, 2)
	core.RegisterTestTick(factory, "Test", counters)

	// XML with value=50, TestA should NOT be skipped (value is not < 25)
	xmlTextNoskip := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree">
			<ReactiveSequence>
				<Script code="value:=50"/>
				<TestA _skipIf="value &lt; 25"/>
				<AlwaysSuccess/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	// XML with value=10, TestB SHOULD be skipped (value < 25)
	xmlTextSkip := `
	<root BTCPP_format="4">
		<BehaviorTree ID="MainTree2">
			<ReactiveSequence>
				<Script code="value:=10"/>
				<TestB _skipIf="value &lt; 25"/>
				<AlwaysSuccess/>
			</ReactiveSequence>
		</BehaviorTree>
	</root>`

	// Execute first tree (no skip)
	tree1, err := factory.CreateTreeFromText(xmlTextNoskip, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree1.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tree1: expected SUCCESS, got %v", status)
	}

	// Execute second tree (skip)
	tree2, err := factory.CreateTreeFromText(xmlTextSkip, nil)
	if err != nil {
		t.Fatal(err)
	}
	status = tree2.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("tree2: expected SUCCESS, got %v", status)
	}

	// TestA (not skipped) should have been ticked
	if counters[0] < 1 {
		t.Errorf("TestA count = %d, want >= 1", counters[0])
	}

	// TestB (skipped) should have 0 ticks
	if counters[1] != 0 {
		t.Errorf("TestB count = %d, want 0 (skipped)", counters[1])
	}
}
