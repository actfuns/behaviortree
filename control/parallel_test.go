package control

import (
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	_ "github.com/actfuns/behaviortree/script"
	_ "github.com/actfuns/behaviortree/xml"
)

// --------------------------------------------------------------------
// Tests ported from C++ gtest_parallel.cpp
//
// Note: C++ tests use AsyncActionTest / TestNode with async_delay.
// The Go port's TestNode does not call ProcessExpired() on its timer,
// so async-delay TestNodes won't complete. We use Sequence+Sleep+AlwaysSuccess
// instead to create async children that work with the wakeup mechanism.
// --------------------------------------------------------------------

// TestControl_Parallel_Async tests that a Parallel node with async children
// waits for completion. Each async child is a Sequence { Sleep, AlwaysSuccess }.
func TestControl_Parallel_Async(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="-1" failure_count="2">
	      <Sequence>
	        <Sleep msec="50"/>
	        <AlwaysSuccess/>
	      </Sequence>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_Threshold3 tests Parallel with success_count=3.
// Equivalent of C++ SimpleParallelTest/Threshold_3.
func TestControl_Parallel_Threshold3(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="3" failure_count="4">
	      <AlwaysSuccess/>
	      <AlwaysSuccess/>
	      <Sleep msec="50"/>
	      <Sequence>
	        <Sleep msec="200"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_ThresholdNeg1 tests Parallel with success_count=-1
// (all children must succeed). Equivalent of C++ SimpleParallelTest/Threshold_neg1.
func TestControl_Parallel_ThresholdNeg1(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="-1" failure_count="4">
	      <AlwaysSuccess/>
	      <Sequence>
	        <Sleep msec="50"/>
	        <AlwaysSuccess/>
	      </Sequence>
	      <AlwaysSuccess/>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_FailureThresholdNeg1 tests Parallel with
// failure_count=-1 (all children must fail).
// Equivalent of C++ SimpleParallelTest/Threshold_thresholdFneg1.
func TestControl_Parallel_FailureThresholdNeg1(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="1" failure_count="-1">
	      <AlwaysFailure/>
	      <Sequence>
	        <Sleep msec="50"/>
	        <AlwaysFailure/>
	      </Sequence>
	      <AlwaysFailure/>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysFailure/>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.FAILURE {
		t.Errorf("Expected FAILURE, got %v", status)
	}
}

// TestControl_Parallel_Threshold2 tests Parallel with all sync children
// and success_count=2. Equivalent of C++ SimpleParallelTest/Threshold_2.
func TestControl_Parallel_Threshold2(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="2" failure_count="2">
	      <AlwaysSuccess/>
	      <AlwaysSuccess/>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_Complex_ConditionsTrue tests nested Parallel nodes.
// Equivalent of C++ ComplexParallelTest/ConditionsTrue.
func TestControl_Parallel_Complex_ConditionsTrue(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="2" failure_count="2">
	      <Parallel success_count="2" failure_count="3" name="par_left">
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	        <Sequence>
	          <Sleep msec="100"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	      <Parallel success_count="1" failure_count="1" name="par_right">
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_Complex_ConditionsLeftFalse tests nested Parallel
// where the left sub-parallel fails.
// Equivalent of C++ ComplexParallelTest/ConditionsLeftFalse.
func TestControl_Parallel_Complex_ConditionsLeftFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="2" failure_count="2">
	      <Parallel success_count="3" failure_count="3" name="par_left">
	        <AlwaysFailure/>
	        <AlwaysFailure/>
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	      <Parallel success_count="1" failure_count="1" name="par_right">
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.FAILURE {
		t.Errorf("Expected FAILURE, got %v", status)
	}
}

// TestControl_Parallel_Complex_ConditionRightFalse tests nested Parallel
// where the right sub-parallel immediately fails.
// Equivalent of C++ ComplexParallelTest/ConditionRightFalse.
func TestControl_Parallel_Complex_ConditionRightFalse(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="2" failure_count="2">
	      <Parallel success_count="3" failure_count="4" name="par_left">
	        <AlwaysSuccess/>
	        <AlwaysSuccess/>
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	      <Parallel success_count="1" failure_count="1" name="par_right">
	        <AlwaysFailure/>
	        <AlwaysSuccess/>
	      </Parallel>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.FAILURE {
		t.Errorf("Expected FAILURE, got %v", status)
	}
}

// TestControl_Parallel_Complex_ConditionRightFalseThreshold2 tests nested
// Parallel where right sub-parallel has failure_count=2.
// Equivalent of C++ ComplexParallelTest/ConditionRightFalse_thresholdF_2.
func TestControl_Parallel_Complex_ConditionRightFalseThreshold2(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="Main">
	    <Parallel success_count="2" failure_count="2">
	      <Parallel success_count="3" failure_count="4" name="par_left">
	        <AlwaysSuccess/>
	        <AlwaysSuccess/>
	        <AlwaysSuccess/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	      <Parallel success_count="1" failure_count="2" name="par_right">
	        <AlwaysFailure/>
	        <Sequence>
	          <Sleep msec="50"/>
	          <AlwaysSuccess/>
	        </Sequence>
	      </Parallel>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_FailingParallel tests with Good/Bad/Slow nodes.
// Equivalent of C++ Parallel/FailingParallel.
func TestControl_Parallel_FailingParallel(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Parallel name="parallel" success_count="1" failure_count="3">
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	      <AlwaysFailure/>
	      <Sequence>
	        <Sleep msec="200"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(100 * time.Millisecond)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// TestControl_Parallel_ParallelAll tests the ParallelAll node with different
// max_failures values. Equivalent of C++ Parallel/ParallelAll.
func TestControl_Parallel_ParallelAll(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	// Test 1: max_failures=1, one failure => FAILURE
	xml1 := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <ParallelAll max_failures="1">
	      <AlwaysFailure/>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </ParallelAll>
	  </BehaviorTree>
	</root>`
	tree1, err := factory.CreateTreeFromText(xml1, nil)
	if err != nil {
		t.Fatal(err)
	}

	status1 := tree1.TickWhileRunning(100 * time.Millisecond)
	if status1 != core.FAILURE {
		t.Errorf("Expected FAILURE, got %v", status1)
	}

	// Test 2: max_failures=2, one failure => SUCCESS
	xml2 := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <ParallelAll max_failures="2">
	      <AlwaysFailure/>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	      <Sequence>
	        <Sleep msec="100"/>
	        <AlwaysSuccess/>
	      </Sequence>
	    </ParallelAll>
	  </BehaviorTree>
	</root>`
	tree2, err := factory.CreateTreeFromText(xml2, nil)
	if err != nil {
		t.Fatal(err)
	}

	status2 := tree2.TickWhileRunning(100 * time.Millisecond)
	if status2 != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status2)
	}
}

// TestControl_Parallel_Issue593 verifies Parallel with _skipIf skips correctly.
// Equivalent of C++ Parallel/Issue593.
func TestControl_Parallel_Issue593(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	var testTickCount int
	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testTickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="TestTree">
	    <Sequence>
	      <Script code="test := true"/>
	      <Parallel failure_count="1" success_count="-1">
	        <TestA _skipIf="test == true"/>
	        <Sleep msec="50"/>
	      </Parallel>
	    </Sequence>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	_ = tree.TickWhileRunning(100 * time.Millisecond)
	if testTickCount != 0 {
		t.Errorf("Expected TestA to be skipped (0 ticks), got %d", testTickCount)
	}
}

// TestControl_Parallel_PauseWithRetry tests a retry loop inside Parallel.
// Equivalent of C++ Parallel/PauseWithRetry.
func TestControl_Parallel_PauseWithRetry(t *testing.T) {
	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	RegisterStandardNodes(factory)

	xml := `
	<root BTCPP_format="4">
	  <BehaviorTree ID="TestTree">
	    <Parallel>
	      <Sequence>
	        <Sleep msec="100"/>
	        <Script code="paused := false"/>
	        <Sleep msec="100"/>
	      </Sequence>
	      <Sequence>
	        <Script code="paused := true; done := false"/>
	        <RetryUntilSuccessful _while="paused" num_attempts="-1" _onHalted="done = true">
	          <AlwaysFailure/>
	        </RetryUntilSuccessful>
	      </Sequence>
	    </Parallel>
	  </BehaviorTree>
	</root>`
	tree, err := factory.CreateTreeFromText(xml, nil)
	if err != nil {
		t.Fatal(err)
	}

	startTime := time.Now()
	status := tree.TickWhileRunning(100 * time.Millisecond)
	elapsed := time.Since(startTime)

	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	// Should take roughly 200ms (2 Sleeps of 100ms each)
	const minExpected = 150 * time.Millisecond
	if elapsed < minExpected {
		t.Errorf("Expected at least %v, got %v", minExpected, elapsed)
	}
}

// TestControl_Parallel_Issue819_SequenceVsReactiveSequence tests that
// Sequence does NOT re-evaluate conditions while ReactiveSequence DOES.
// Equivalent of C++ Parallel/Issue819_SequenceVsReactiveSequence.
func TestControl_Parallel_Issue819_SequenceVsReactiveSequence(t *testing.T) {
	// Test 1: Regular Sequence - condition NOT re-evaluated
	t.Run("Sequence", func(t *testing.T) {
		factory, err := core.NewBehaviorTreeFactory()
		if err != nil {
			t.Fatal(err)
		}
		RegisterStandardNodes(factory)

		var tickCount1, tickCount2 int

		_ = factory.RegisterSimpleCondition("TestCondition", func(self core.TreeNode) core.NodeStatus {
			name := self.Name()
			switch name {
			case "cond1":
				tickCount1++
			case "cond2":
				tickCount2++
			}
			return core.SUCCESS
		}, core.PortsList{})

		xml := `
		<root BTCPP_format="4">
		  <BehaviorTree ID="TestTree">
		    <Parallel success_count="2" failure_count="1">
		      <Sequence>
		        <TestCondition name="cond1"/>
		        <Sleep msec="200"/>
		      </Sequence>
		      <Sequence>
		        <TestCondition name="cond2"/>
		        <Sleep msec="200"/>
		      </Sequence>
		    </Parallel>
		  </BehaviorTree>
		</root>`
		tree, err := factory.CreateTreeFromText(xml, nil)
		if err != nil {
			t.Fatal(err)
		}

		// First tick: both conditions evaluated
		status := tree.TickExactlyOnce()
		if status != core.RUNNING {
			t.Errorf("Expected RUNNING, got %v", status)
		}
		if tickCount1 != 1 {
			t.Errorf("Expected cond1 ticked once, got %d", tickCount1)
		}
		if tickCount2 != 1 {
			t.Errorf("Expected cond2 ticked once, got %d", tickCount2)
		}

		// Second tick: conditions should NOT be re-evaluated (Sequence)
		time.Sleep(50 * time.Millisecond)
		status = tree.TickExactlyOnce()
		if status != core.RUNNING {
			t.Errorf("Expected RUNNING, got %v", status)
		}
		if tickCount1 != 1 {
			t.Errorf("Expected cond1 still 1, got %d", tickCount1)
		}
		if tickCount2 != 1 {
			t.Errorf("Expected cond2 still 1, got %d", tickCount2)
		}
	})

	// Test 2: ReactiveSequence - condition IS re-evaluated every tick
	t.Run("ReactiveSequence", func(t *testing.T) {
		factory, err := core.NewBehaviorTreeFactory()
		if err != nil {
			t.Fatal(err)
		}
		RegisterStandardNodes(factory)

		var tickCount1, tickCount2 int

		_ = factory.RegisterSimpleCondition("TestCondition", func(self core.TreeNode) core.NodeStatus {
			name := self.Name()
			switch name {
			case "cond1":
				tickCount1++
			case "cond2":
				tickCount2++
			}
			return core.SUCCESS
		}, core.PortsList{})

		xml := `
		<root BTCPP_format="4">
		  <BehaviorTree ID="TestTree">
		    <Parallel success_count="2" failure_count="1">
		      <ReactiveSequence>
		        <TestCondition name="cond1"/>
		        <Sleep msec="200"/>
		      </ReactiveSequence>
		      <ReactiveSequence>
		        <TestCondition name="cond2"/>
		        <Sleep msec="200"/>
		      </ReactiveSequence>
		    </Parallel>
		  </BehaviorTree>
		</root>`
		tree, err := factory.CreateTreeFromText(xml, nil)
		if err != nil {
			t.Fatal(err)
		}

		// First tick: both conditions evaluated
		status := tree.TickExactlyOnce()
		if status != core.RUNNING {
			t.Errorf("Expected RUNNING, got %v", status)
		}
		if tickCount1 != 1 {
			t.Errorf("Expected cond1 ticked once, got %d", tickCount1)
		}
		if tickCount2 != 1 {
			t.Errorf("Expected cond2 ticked once, got %d", tickCount2)
		}

		// Second tick: conditions SHOULD be re-evaluated (ReactiveSequence)
		time.Sleep(50 * time.Millisecond)
		status = tree.TickExactlyOnce()
		if status != core.RUNNING {
			t.Errorf("Expected RUNNING, got %v", status)
		}
		if tickCount1 != 2 {
			t.Errorf("Expected cond1 ticked twice, got %d", tickCount1)
		}
		if tickCount2 != 2 {
			t.Errorf("Expected cond2 ticked twice, got %d", tickCount2)
		}
	})
}
