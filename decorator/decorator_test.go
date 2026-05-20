package decorator_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
	"github.com/actfuns/behaviortree/factory"
)

// simpleSequenceNode is a minimal SequenceNode for testing.
type simpleSequenceNode struct {
	core.ControlNode
	childIdx int
}

func (n *simpleSequenceNode) Tick() core.NodeStatus {
	n.SetStatus(core.RUNNING)

	for n.childIdx < len(n.Children()) {
		child := n.Children()[n.childIdx]
		childStatus := child.ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.childIdx++
			child.ResetStatus()
		case core.FAILURE:
			n.HaltChildren()
			n.childIdx = 0
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		case core.SKIPPED:
			n.childIdx++
			child.ResetStatus()
		}
	}

	n.childIdx = 0
	return core.SUCCESS
}

func (n *simpleSequenceNode) Halt() {
	n.HaltChildren()
	n.childIdx = 0
	n.ResetStatus()
}

// --------------------------------------------------------------------
// Direct-construction tests: Deadline, Retry, Repeat
// --------------------------------------------------------------------

// TestDeadlineTriggered verifies that a TimeoutNode with a short deadline will
// eventually return FAILURE when the child keeps running.
func TestDeadlineTriggered(t *testing.T) {
	cfg := core.NewNodeConfig()
	cfg.InputPorts["msec"] = "10"
	timeout := decorator.NewTimeoutNode("deadline", cfg)

	actionCfg := core.NewNodeConfig()
	actionCfg.InputPorts["return_status"] = "SUCCESS"
	actionCfg.InputPorts["async_delay"] = "100"
	action := newTestNode("action", actionCfg)

	timeout.SetChild(action)

	state := timeout.ExecuteTick()
	if state != core.RUNNING {
		t.Fatalf("first tick: want RUNNING, got %v", state)
	}

	// Wait for the timeout to expire naturally.
	time.Sleep(25 * time.Millisecond)

	state = timeout.ExecuteTick()
	if state != core.FAILURE {
		t.Fatalf("after timeout: want FAILURE, got %v", state)
	}
}

// TestDeadlineNotTriggered verifies that when the child finishes before the
// deadline, TimeoutNode returns SUCCESS.
func TestDeadlineNotTriggered(t *testing.T) {
	cfg := core.NewNodeConfig()
	cfg.InputPorts["msec"] = "10"
	timeout := decorator.NewTimeoutNode("deadline", cfg)

	actionCfg := core.NewNodeConfig()
	actionCfg.InputPorts["return_status"] = "SUCCESS"
	actionCfg.InputPorts["async_delay"] = "1"
	action := newTestNode("action", actionCfg)

	timeout.SetChild(action)

	state := timeout.ExecuteTick()
	if state != core.RUNNING {
		t.Fatalf("first tick: want RUNNING, got %v", state)
	}

	// Wait for the async action to complete
	time.Sleep(15 * time.Millisecond)

	state = timeout.ExecuteTick()
	if state != core.SUCCESS {
		t.Fatalf("after child completes: want SUCCESS, got %v", state)
	}
}

// TestRetry verifies that RetryNode ticks the child the expected number of
// times when the child keeps failing, and succeeds when the child finally
// succeeds.
func TestRetry(t *testing.T) {
	// --- First pass: child always fails, retry should fail after 3 attempts ---
	cfg := core.NewNodeConfig()
	cfg.InputPorts["num_attempts"] = "3"
	retry := decorator.NewRetryNode("retry", cfg)
	action := newSyncFailAction("action")
	retry.SetChild(action)

	retry.ExecuteTick()
	if retry.Status() != core.FAILURE {
		t.Fatalf("retry after 3 failures: want FAILURE, got %v", retry.Status())
	}
	if action.tickCount != 3 {
		t.Fatalf("child tick count: want 3, got %d", action.tickCount)
	}

	// --- Second pass: child succeeds, retry should succeed after 1 tick ---
	action2 := newSyncSuccessAction("action2")
	retry2Cfg := core.NewNodeConfig()
	retry2Cfg.InputPorts["num_attempts"] = "3"
	retry2 := decorator.NewRetryNode("retry2", retry2Cfg)
	retry2.SetChild(action2)

	retry2.ExecuteTick()
	if retry2.Status() != core.SUCCESS {
		t.Fatalf("retry with success: want SUCCESS, got %v", retry2.Status())
	}
	if action2.tickCount != 1 {
		t.Fatalf("child tick count: want 1, got %d", action2.tickCount)
	}
}

// TestRepeat verifies that RepeatNode ticks the child the expected number of
// times when the child succeeds, and stops early when the child fails.
func TestRepeat(t *testing.T) {
	// --- First pass: child fails, repeat should stop immediately ---
	cfg := core.NewNodeConfig()
	cfg.InputPorts["num_cycles"] = "3"
	repeat := decorator.NewRepeatNode("repeat", cfg)
	failAction := newSyncFailAction("action")
	repeat.SetChild(failAction)

	repeat.ExecuteTick()
	if repeat.Status() != core.FAILURE {
		t.Fatalf("repeat with child failure: want FAILURE, got %v", repeat.Status())
	}
	if failAction.tickCount != 1 {
		t.Fatalf("child tick count: want 1, got %d", failAction.tickCount)
	}

	// --- Second pass: child succeeds, repeat should cycle 3 times ---
	successAction := newSyncSuccessAction("action2")
	repeat2Cfg := core.NewNodeConfig()
	repeat2Cfg.InputPorts["num_cycles"] = "3"
	repeat2 := decorator.NewRepeatNode("repeat2", repeat2Cfg)
	repeat2.SetChild(successAction)

	repeat2.ExecuteTick()
	if repeat2.Status() != core.SUCCESS {
		t.Fatalf("repeat with success: want SUCCESS, got %v", repeat2.Status())
	}
	if successAction.tickCount != 3 {
		t.Fatalf("child tick count: want 3, got %d", successAction.tickCount)
	}
}

// TestRepeatAsync verifies that RepeatNode works correctly with an async child.
func TestRepeatAsync(t *testing.T) {
	cfg := core.NewNodeConfig()
	cfg.InputPorts["num_cycles"] = "3"
	repeat := decorator.NewRepeatNode("repeat", cfg)

	actionCfg := core.NewNodeConfig()
	actionCfg.InputPorts["return_status"] = "SUCCESS"
	actionCfg.InputPorts["async_delay"] = "5"
	action := newTestNode("action", actionCfg)
	repeat.SetChild(action)

	res := repeat.ExecuteTick()

	// Loop while RUNNING, waiting for async completions
	for res == core.RUNNING {
		time.Sleep(10 * time.Millisecond)
		res = repeat.ExecuteTick()
	}

	if repeat.Status() != core.SUCCESS {
		t.Fatalf("repeat async: want SUCCESS, got %v", repeat.Status())
	}
}

// TestTimeoutAndRetry_Issue57 verifies that Timeout wrapping Retry does not
// get stuck in an infinite loop (regression test for C++ issue #57).
func TestTimeoutAndRetry_Issue57(t *testing.T) {
	timeoutCfg := core.NewNodeConfig()
	timeoutCfg.InputPorts["msec"] = "5"
	timeout := decorator.NewTimeoutNode("deadline", timeoutCfg)

	retryCfg := core.NewNodeConfig()
	retryCfg.InputPorts["num_attempts"] = "1000"
	retry := decorator.NewRetryNode("retry", retryCfg)

	failAction := newSyncFailAction("action")
	retry.SetChild(failAction)
	timeout.SetChild(retry)

	// This should not loop forever; the timeout should halt the child.
	deadline := time.After(2 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timed out after 2s - possible infinite loop")
		default:
		}
		status := timeout.ExecuteTick()
		if status == core.FAILURE || status == core.SUCCESS {
			break
		}
		time.Sleep(time.Microsecond * 50)
	}

	if timeout.Status() != core.FAILURE {
		t.Fatalf("want FAILURE, got %v", timeout.Status())
	}
}

// --------------------------------------------------------------------
// XML-based decorator tests
// --------------------------------------------------------------------

// TestRunOnce verifies that RunOnce ticks its child only once.
func TestRunOnce(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	counters := []int{0, 0}
	core.RegisterTestTick(factory, "Test", counters)

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Sequence>
	        <RunOnce> <TestA/> </RunOnce>
	        <TestB/>
	      </Sequence>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		status := tree.TickWhileRunning(0)
		if status != core.SUCCESS {
			t.Fatalf("iteration %d: want SUCCESS, got %v", i, status)
		}
	}

	if counters[0] != 1 {
		t.Errorf("TestA count: want 1, got %d", counters[0])
	}
	if counters[1] != 5 {
		t.Errorf("TestB count: want 5, got %d", counters[1])
	}
}

// TestDelayWithXML verifies that DelayNode waits before ticking its child.
func TestDelayWithXML(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Delay delay_msec="100">
	        <AlwaysSuccess/>
	      </Delay>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()

	// First tick should return RUNNING
	status := tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("first tick: want RUNNING, got %v", status)
	}

	// TickOnce after 50ms should still be RUNNING
	time.Sleep(50 * time.Millisecond)
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("after 50ms: want RUNNING, got %v", status)
	}

	// Use TickWhileRunning to poll until the delay completes.
	// This internally processes wake-up signals and re-ticks.
	status = tree.TickWhileRunning(0)
	elapsed := time.Since(start)

	if status != core.SUCCESS {
		t.Fatalf("after delay: want SUCCESS, got %v", status)
	}
	if elapsed.Milliseconds() < 80 {
		t.Errorf("elapsed time: want >= 80ms, got %dms", elapsed.Milliseconds())
	}
	if elapsed.Milliseconds() > 200 {
		t.Errorf("elapsed time: want <= 200ms, got %dms", elapsed.Milliseconds())
	}
}

// TestForceFailure_ChildSuccess verifies that ForceFailure returns FAILURE
// when the child succeeds.
func TestForceFailure_ChildSuccess(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <ForceFailure>
	        <AlwaysSuccess/>
	      </ForceFailure>
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
}

// TestForceFailure_ChildFailure verifies that ForceFailure returns FAILURE
// when the child also fails.
func TestForceFailure_ChildFailure(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <ForceFailure>
	        <AlwaysFailure/>
	      </ForceFailure>
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
}

// TestForceSuccess_ChildFailure verifies that ForceSuccess returns SUCCESS
// even when the child fails.
func TestForceSuccess_ChildFailure(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <ForceSuccess>
	        <AlwaysFailure/>
	      </ForceSuccess>
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
}

// TestForceSuccess_ChildSuccess verifies that ForceSuccess returns SUCCESS
// when the child succeeds.
func TestForceSuccess_ChildSuccess(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <ForceSuccess>
	        <AlwaysSuccess/>
	      </ForceSuccess>
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
}

// TestInverter_ChildSuccess verifies that Inverter returns FAILURE when
// the child succeeds.
func TestInverter_ChildSuccess(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Inverter>
	        <AlwaysSuccess/>
	      </Inverter>
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
}

// TestInverter_ChildFailure verifies that Inverter returns SUCCESS when
// the child fails.
func TestInverter_ChildFailure(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Inverter>
	        <AlwaysFailure/>
	      </Inverter>
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
}

// TestInverterInSequence verifies that Inverter within a Sequence works
// correctly (inverts child FAILURE to SUCCESS so the sequence continues).
func TestInverterInSequence(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Sequence>
	        <Inverter>
	          <AlwaysFailure/>
	        </Inverter>
	        <AlwaysSuccess/>
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
}

// TestKeepRunningUntilFailure verifies that KeepRunningUntilFailure ticks
// the child repeatedly until the child returns FAILURE.
func TestKeepRunningUntilFailure(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()

	tickCount := 0
	factory.RegisterSimpleAction("SuccessThenFail", func(core.TreeNode) core.NodeStatus {
		tickCount++
		if tickCount < 3 {
			return core.SUCCESS
		}
		return core.FAILURE
	}, core.PortsList{})

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <KeepRunningUntilFailure>
	        <SuccessThenFail/>
	      </KeepRunningUntilFailure>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First tick - child succeeds, should return RUNNING
	status := tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 1: want RUNNING, got %v", status)
	}
	if tickCount != 1 {
		t.Fatalf("tick 1 count: want 1, got %d", tickCount)
	}

	// Second tick - child succeeds again, should return RUNNING
	status = tree.TickOnce()
	if status != core.RUNNING {
		t.Fatalf("tick 2: want RUNNING, got %v", status)
	}
	if tickCount != 2 {
		t.Fatalf("tick 2 count: want 2, got %d", tickCount)
	}

	// Third tick - child fails, should return FAILURE
	status = tree.TickOnce()
	if status != core.FAILURE {
		t.Fatalf("tick 3: want FAILURE, got %v", status)
	}
	if tickCount != 3 {
		t.Fatalf("tick 3 count: want 3, got %d", tickCount)
	}
}

// ====================================================================
// Helper types for direct-construction tests
// ====================================================================

// syncSuccessAction is a SyncActionNode that counts ticks and returns SUCCESS.
type syncSuccessAction struct {
	core.SyncActionNode
	tickCount int
}

func newSyncSuccessAction(name string) *syncSuccessAction {
	n := &syncSuccessAction{}
	cfg := core.NewNodeConfig()
	n.Init(name, cfg)
	n.SetSelf(n)
	n.SetRegistrationID("SyncSuccess")
	return n
}

func (n *syncSuccessAction) Tick() core.NodeStatus {
	n.tickCount++
	return core.SUCCESS
}

// syncFailAction is a SyncActionNode that counts ticks and returns FAILURE.
type syncFailAction struct {
	core.SyncActionNode
	tickCount int
}

func newSyncFailAction(name string) *syncFailAction {
	n := &syncFailAction{}
	cfg := core.NewNodeConfig()
	n.Init(name, cfg)
	n.SetSelf(n)
	n.SetRegistrationID("SyncFail")
	return n
}

func (n *syncFailAction) Tick() core.NodeStatus {
	n.tickCount++
	return core.FAILURE
}

// testNode is a simple stateful action node with configurable return status
// and async delay, used for direct-construction testing.
type testNode struct {
	core.StatefulActionNode
	returnStatus core.NodeStatus
	asyncDelayMs int
	startTime    time.Time
}

func newTestNode(name string, cfg core.NodeConfig) *testNode {
	n := &testNode{
		returnStatus: core.SUCCESS,
	}
	// Read config from InputPorts strings before Init (GetInputTyped needs a fully initialized node)
	if v, ok := cfg.InputPorts["return_status"]; ok {
		switch v {
		case "SUCCESS":
			n.returnStatus = core.SUCCESS
		case "FAILURE":
			n.returnStatus = core.FAILURE
		case "RUNNING":
			n.returnStatus = core.RUNNING
		}
	}
	if v, ok := cfg.InputPorts["async_delay"]; ok {
		if delay, err := strconv.Atoi(v); err == nil && delay > 0 {
			n.asyncDelayMs = delay
		}
	}
	n.Init(name, cfg)
	n.SetSelf(n)
	n.SetRegistrationID("TestNodeHelper")
	return n
}

func (n *testNode) OnStart() core.NodeStatus {
	if n.asyncDelayMs <= 0 {
		return n.returnStatus
	}
	n.startTime = time.Now()
	return core.RUNNING
}

func (n *testNode) OnRunning() core.NodeStatus {
	if n.asyncDelayMs <= 0 || time.Since(n.startTime) >= time.Duration(n.asyncDelayMs)*time.Millisecond {
		return n.returnStatus
	}
	return core.RUNNING
}

func (n *testNode) OnHalted() {
}

func (n *testNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}
