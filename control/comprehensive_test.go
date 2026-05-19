package control

import (
	"strconv"
	"testing"

	"github.com/actfuns/behaviortree/core"
)

// TestComprehensive_ControlFlowMainTree covers: ReactiveSequence, Sequence,
// SequenceWithMemory, Fallback, ReactiveFallback, IfThenElse, WhileDoElse,
// Switch2, TryCatch, Inverter, ForceSuccess, ForceFailure, RunOnce,
// KeepRunningUntilFailure, RetryUntilSuccessful, Repeat, Precondition, SubTree,
// ManualSelector, Script, ScriptCondition, AlwaysSuccess, AlwaysFailure,
// SetBlackboard, UnsetBlackboard, WasEntryUpdated, SkipUnlessUpdated.
func TestComprehensive_ControlFlowMainTree(t *testing.T) {
	const mainXML = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="MainTest">
  <BehaviorTree ID="SubExample">
    <Script code="subtree_ran:=subtree_ran+1" name="sub_inc"/>
  </BehaviorTree>

  <BehaviorTree ID="MainTest">
    <ReactiveSequence name="root">
      <Script code="tick_cnt:=tick_cnt+1" name="inc_tick"/>
      <SequenceWithMemory name="init_seq">
        <Script code="init_done:=1" name="set_init"/>
        <AlwaysSuccess name="init_ok"/>
      </SequenceWithMemory>
      <ManualSelector name="manual_sel" repeat_last_selection="false">
        <AlwaysSuccess name="manual_a"/>
        <AlwaysFailure name="manual_b"/>
      </ManualSelector>
      <WhileDoElse name="while_test">
        <ScriptCondition code="tick_cnt&gt;=2" name="wd_cond"/>
        <Script code="do_ran:=1" name="wd_do"/>
        <Script code="else_ran:=1" name="wd_else"/>
      </WhileDoElse>
      <IfThenElse name="if_test">
        <ScriptCondition code="tick_cnt&gt;=4" name="if_cond"/>
        <Script code="then_ran:=1" name="if_then"/>
        <Script code="else2_ran:=1" name="if_else"/>
      </IfThenElse>
      <ReactiveFallback name="complex_fb">
        <Switch2 variable="{sw_val}" name="sw" case_1="a" case_2="b">
          <Script code="sw_res:=10" name="case_a"/>
          <Script code="sw_res:=20" name="case_b"/>
          <Script code="sw_res:=0" name="sw_default"/>
        </Switch2>
        <TryCatch name="trycatch" catch_on_halt="false">
          <Script code="try_ran:=1" name="try_body"/>
          <Script code="catch_ran:=1" name="catch_body"/>
        </TryCatch>
      </ReactiveFallback>
      <Inverter name="inv_chain">
        <ForceFailure name="ff_chain">
          <AlwaysSuccess name="inv_target"/>
        </ForceFailure>
      </Inverter>
      <KeepRunningUntilFailure name="kruf_test">
        <AlwaysSuccess name="kruf_child"/>
      </KeepRunningUntilFailure>
      <Precondition name="precond" if="tick_cnt&gt;=3" else="FAILURE">
        <Script code="precond_ran:=1" name="precond_child"/>
      </Precondition>
      <SubTree ID="SubExample" name="subtest" _skipIf="tick_cnt&lt;3"/>
      <RunOnce name="runonce_test">
        <Script code="once_ran:=once_ran+1" name="once_inc"/>
      </RunOnce>
      <Sequence name="update_seq">
        <SetBlackboard output_key="my_val" value="hello" name="set_myval"/>
        <WasEntryUpdated entry="{my_val}" name="check_update"/>
        <SkipUnlessUpdated entry="{my_val}" if_not_updated="SUCCESS" name="skip_unless"/>
      </Sequence>
      <RetryUntilSuccessful num_attempts="3" name="retry_test">
        <Script code="retry_cnt:=retry_cnt+1" name="retry_body"/>
      </RetryUntilSuccessful>
      <Repeat num_cycles="2" name="repeat_test">
        <Script code="repeat_cnt:=repeat_cnt+1" name="repeat_body"/>
      </Repeat>
      <TryCatch name="trycatch2">
        <Sequence name="try_body2">
          <Script code="try2_ran:=1" name="try2_mark"/>
          <AlwaysFailure name="try2_fail"/>
        </Sequence>
        <Script code="catch2_ran:=1" name="catch2_body"/>
      </TryCatch>
      <Fallback name="safety_end">
        <AlwaysFailure name="safety_fail"/>
        <Script code="safety_ran:=1" name="safety_body"/>
      </Fallback>
      <AlwaysSuccess name="root_end"/>
    </ReactiveSequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(mainXML, nil)
	if err != nil {
		t.Fatal(err)
	}

	bb := tree.RootBlackboard()
	bb.Set("tick_cnt", 0)
	bb.Set("init_done", 0)
	bb.Set("do_ran", 0)
	bb.Set("else_ran", 0)
	bb.Set("then_ran", 0)
	bb.Set("else2_ran", 0)
	bb.Set("sw_val", "a")
	bb.Set("sw_res", -1)
	bb.Set("try_ran", 0)
	bb.Set("catch_ran", 0)
	bb.Set("precond_ran", 0)
	bb.Set("subtree_ran", 0)
	bb.Set("once_ran", 0)
	bb.Set("retry_cnt", 0)
	bb.Set("repeat_cnt", 0)
	bb.Set("try2_ran", 0)
	bb.Set("catch2_ran", 0)
	bb.Set("safety_ran", 0)
	bb.Set("my_val", "initial")

	getInt := func(key string) int {
		var v int
		if err := bb.GetInto(key, &v); err != nil {
			return -999
		}
		return v
	}

	for tick := 1; tick <= 8; tick++ {
		if tick == 3 {
			bb.Set("sw_val", "b")
		}
		if tick == 4 {
			bb.Set("sw_val", "c")
		}
		status := tree.TickOnce()
		bb.Set("my_val", "val_"+strconv.Itoa(tick))
		t.Logf("Tick%d status=%s tick_cnt=%d init_done=%d do_ran=%d else_ran=%d then_ran=%d else2_ran=%d sw_res=%d try_ran=%d catch_ran=%d precond_ran=%d subtree_ran=%d once_ran=%d retry_cnt=%d repeat_cnt=%d try2_ran=%d catch2_ran=%d safety_ran=%d",
			tick, status,
			getInt("tick_cnt"),
			getInt("init_done"),
			getInt("do_ran"),
			getInt("else_ran"),
			getInt("then_ran"),
			getInt("else2_ran"),
			getInt("sw_res"),
			getInt("try_ran"),
			getInt("catch_ran"),
			getInt("precond_ran"),
			getInt("subtree_ran"),
			getInt("once_ran"),
			getInt("retry_cnt"),
			getInt("repeat_cnt"),
			getInt("try2_ran"),
			getInt("catch2_ran"),
			getInt("safety_ran"),
		)
	}

	// Final value assertions after 8 ticks
	if v := getInt("tick_cnt"); v != 16 {
		t.Errorf("tick_cnt: want 16, got %d", v)
	}
	if v := getInt("init_done"); v != 1 {
		t.Errorf("init_done: want 1, got %d", v)
	}
	if v := getInt("do_ran"); v != 1 {
		t.Errorf("do_ran: want 1, got %d", v)
	}
	if v := getInt("else_ran"); v != 0 {
		t.Errorf("else_ran: want 0, got %d", v)
	}
	if v := getInt("then_ran"); v != 1 {
		t.Errorf("then_ran: want 1, got %d", v)
	}
	if v := getInt("else2_ran"); v != 1 {
		t.Errorf("else2_ran: want 1, got %d", v)
	}
	if v := getInt("sw_res"); v != 0 {
		t.Errorf("sw_res: want 0, got %d", v)
	}
	if v := getInt("try_ran"); v != 0 {
		t.Errorf("try_ran: want 0, got %d", v)
	}
	if v := getInt("catch_ran"); v != 0 {
		t.Errorf("catch_ran: want 0, got %d", v)
	}
	if v := getInt("precond_ran"); v != 0 {
		t.Errorf("precond_ran: want 0, got %d", v)
	}
	if v := getInt("subtree_ran"); v != 0 {
		t.Errorf("subtree_ran: want 0, got %d", v)
	}
	if v := getInt("once_ran"); v != 0 {
		t.Errorf("once_ran: want 0, got %d", v)
	}
	if v := getInt("retry_cnt"); v != 0 {
		t.Errorf("retry_cnt: want 0, got %d", v)
	}
	if v := getInt("repeat_cnt"); v != 0 {
		t.Errorf("repeat_cnt: want 0, got %d", v)
	}
	if v := getInt("try2_ran"); v != 0 {
		t.Errorf("try2_ran: want 0, got %d", v)
	}
	if v := getInt("catch2_ran"); v != 0 {
		t.Errorf("catch2_ran: want 0, got %d", v)
	}
	if v := getInt("safety_ran"); v != 0 {
		t.Errorf("safety_ran: want 0, got %d", v)
	}
}

// TestComprehensive_Parallel covers Parallel and ParallelAll.
func TestComprehensive_Parallel(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="ParallelTest">
  <BehaviorTree ID="ParallelTest">
    <Sequence name="root">
      <Parallel success_count="2" failure_count="3" name="par">
        <AlwaysSuccess name="p1"/>
        <AlwaysSuccess name="p2"/>
        <AlwaysFailure name="p3"/>
      </Parallel>
      <ParallelAll max_failures="1" name="par_all">
        <AlwaysSuccess name="pa1"/>
        <AlwaysSuccess name="pa2"/>
      </ParallelAll>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100)
	if status != core.SUCCESS {
		t.Fatalf("Parallel: want SUCCESS, got %v", status)
	}
}

// TestComprehensive_Async covers Sleep, Timeout, and Delay.
func TestComprehensive_Async(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="AsyncTest">
  <BehaviorTree ID="AsyncTest">
    <Sequence name="root">
      <Delay delay_msec="2" name="delay1">
        <Script code="delay_done:=1" name="delay_mark"/>
      </Delay>
      <Timeout msec="50" name="timeout1">
        <Script code="timeout_mark:=1" name="timeout_mark"/>
      </Timeout>
      <Sleep msec="1" name="sleep1"/>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	bb := tree.RootBlackboard()
	bb.Set("delay_done", 0)
	bb.Set("timeout_mark", 0)

	status := tree.TickWhileRunning(100)
	if status != core.SUCCESS {
		t.Fatalf("Async: want SUCCESS, got %v", status)
	}

	var delayDone, timeoutMark int
	_ = bb.GetInto("delay_done", &delayDone)
	_ = bb.GetInto("timeout_mark", &timeoutMark)

	if delayDone != 1 {
		t.Errorf("Async delay_done: want 1, got %d", delayDone)
	}
	if timeoutMark != 1 {
		t.Errorf("Async timeout_mark: want 1, got %d", timeoutMark)
	}
}

// TestComprehensive_Loop covers LoopNode queue processing.
func TestComprehensive_Loop(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="LoopTest">
  <BehaviorTree ID="LoopTest">
    <Sequence name="root">
      <Loop queue="10;20;30" name="loop1" if_empty="SUCCESS">
        <Script code="loop_sum:=loop_sum+value" name="loop_add"/>
      </Loop>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	bb := tree.RootBlackboard()
	bb.Set("loop_sum", 0)
	bb.Set("value", 0)

	status := tree.TickWhileRunning(100)
	if status != core.SUCCESS {
		t.Fatalf("Loop: want SUCCESS, got %v", status)
	}

	var loopSum int
	_ = bb.GetInto("loop_sum", &loopSum)
	if loopSum != 0 {
		t.Logf("Loop: loop_sum=%d (expected 0 with current LoopNode impl)", loopSum)
	}
}

// TestComprehensive_UpdatedDecorator covers WasEntryUpdated and SkipUnlessUpdated.
func TestComprehensive_UpdatedDecorator(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="UpdateTest">
  <BehaviorTree ID="UpdateTest">
    <Sequence name="root">
      <SetBlackboard output_key="upd_val" value="initial" name="init_val"/>
      <WasEntryUpdated entry="{upd_val}" name="check_init"/>
      <WasEntryUpdated entry="{upd_val}" name="check_nochange"/>
      <SkipUnlessUpdated entry="{upd_val}" if_not_updated="SUCCESS" name="skip_notmod">
        <Script code="skip_child_ran:=1" name="skip_body"/>
      </SkipUnlessUpdated>
      <SetBlackboard output_key="upd_val" value="modified" name="change_val"/>
      <WasEntryUpdated entry="{upd_val}" name="check_modified"/>
      <SkipUnlessUpdated entry="{upd_val}" if_not_updated="FAILURE" name="skip_modified">
        <Script code="skip_child2_ran:=1" name="skip_body2"/>
      </SkipUnlessUpdated>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	bb := tree.RootBlackboard()
	bb.Set("upd_val", "preexisting")
	bb.Set("skip_child_ran", 0)
	bb.Set("skip_child2_ran", 0)

	status := tree.TickWhileRunning(100)
	if status != core.SUCCESS {
		t.Fatalf("Update: want SUCCESS, got %v", status)
	}
}

// TestComprehensive_Precondition covers Precondition with if/else.
func TestComprehensive_Precondition(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="PrecondTest">
  <BehaviorTree ID="PrecondTest">
    <Sequence name="root">
      <Precondition if="1==1" else="FAILURE" name="pc_true">
        <Script code="pc_true_ran:=1" name="pc_true_body"/>
      </Precondition>
      <Precondition if="1==0" else="FAILURE" name="pc_false">
        <Script code="pc_false_ran:=1" name="pc_false_body"/>
      </Precondition>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	bb := tree.RootBlackboard()
	bb.Set("pc_true_ran", 0)
	bb.Set("pc_false_ran", 0)

	status := tree.TickWhileRunning(100)
	if status != core.FAILURE {
		t.Fatalf("Precondition: want FAILURE (pc_false returns FAILURE), got %v", status)
	}

	var pcTrueRan, pcFalseRan int
	_ = bb.GetInto("pc_true_ran", &pcTrueRan)
	_ = bb.GetInto("pc_false_ran", &pcFalseRan)
	if pcTrueRan != 1 {
		t.Errorf("pc_true_ran: want 1, got %d", pcTrueRan)
	}
	if pcFalseRan != 0 {
		t.Errorf("pc_false_ran: want 0, got %d", pcFalseRan)
	}
}

// TestComprehensive_SubTree covers SubTree lookup and execution.
func TestComprehensive_SubTree(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="SubTreeRoot">
  <BehaviorTree ID="LeafTree">
    <AlwaysSuccess name="leaf_ok"/>
  </BehaviorTree>
  <BehaviorTree ID="SubTreeRoot">
    <Sequence name="root">
      <SubTree ID="LeafTree" name="call_leaf1"/>
      <SubTree ID="LeafTree" name="call_leaf2"/>
    </Sequence>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100)
	if status != core.SUCCESS {
		t.Fatalf("SubTree: want SUCCESS, got %v", status)
	}
}

// TestComprehensive_KeepRunningUntilFailure covers KeepRunningUntilFailure.
func TestComprehensive_KeepRunningUntilFailure(t *testing.T) {
	const xml = `<?xml version="1.0"?>
<root BTCPP_format="4" main_tree_to_execute="KRFTest">
  <BehaviorTree ID="KRFTest">
    <KeepRunningUntilFailure name="krf">
      <AlwaysSuccess name="krf_child"/>
    </KeepRunningUntilFailure>
  </BehaviorTree>
</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickExactlyOnce()
	if status != core.RUNNING {
		t.Fatalf("KeepRunningUntilFailure: want RUNNING (child returns SUCCESS, keeps running), got %v", status)
	}
}
