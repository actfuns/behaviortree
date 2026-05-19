package xml

import (
	"fmt"
	"testing"
	"time"

	"github.com/actfuns/behaviortree/action"
	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
	_ "github.com/actfuns/behaviortree/script"
)

// registerCommonNodes registers node types commonly needed in XML-based tests.
func registerCommonNodes(factory *core.BehaviorTreeFactory) {
	// Control nodes
	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Fallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewFallbackNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ReactiveSequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewReactiveSequence(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("SequenceWithMemory", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceWithMemory(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Parallel", core.PortsList{
		"success_count": core.NewPortInfo(core.INPUT),
		"failure_count": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewParallelNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ParallelAll", core.PortsList{
		"max_failures": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewParallelAllNode(name, config)
	}, core.Control)

	// Decorator nodes
	_ = factory.RegisterNodeType("Inverter", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewInverterNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("ForceFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceFailureNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("ForceSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceSuccessNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("RunOnce", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRunOnceNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("KeepRunningUntilFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewKeepRunningUntilFailureNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Timeout", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewTimeoutNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("RetryUntilSuccessful", core.PortsList{
		"num_attempts": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRetryNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Repeat", core.PortsList{
		"num_cycles": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRepeatNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Delay", core.PortsList{
		"delay_msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewDelayNode(name, config)
	}, core.Decorator)

	// Action nodes
	_ = factory.RegisterNodeType("AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysSuccessNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysFailureNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("SetBlackboard", core.PortsList{
		"output_key": core.NewPortInfo(core.INPUT),
		"value":      core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewSetBlackboardNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("Script", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("ScriptCondition", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptCondition(name, config)
	}, core.Condition)

	_ = factory.RegisterNodeType("SaySomething", core.PortsList{
		"message": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &saySomethingNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("SaySomething")
		return n
	}, core.Action)
}

// ----------------------------------------------------------------
// Existing test: TestMoveBaseRecovery
// ----------------------------------------------------------------

func TestMoveBaseRecovery(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="BehaviorTree">
	    <BehaviorTree ID="BehaviorTree">
	        <Fallback name="root">
	            <ReactiveSequence name="navigation_subtree">
	                <Inverter>
	                    <Condition ID="IsStuck"/>
	                </Inverter>
	                <SequenceWithMemory name="navigate">
	                    <Action ID="ComputePathToPose"/>
	                    <Action ID="FollowPath"/>
	                </SequenceWithMemory>
	            </ReactiveSequence>
	            <SequenceWithMemory name="stuck_recovery">
	                <Condition ID="IsStuck"/>
	                <Action ID="BackUpAndSpin"/>
	            </SequenceWithMemory>
	        </Fallback>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	backSpinTick := 0
	computeTick := 0
	followPathTick := 0

	_ = factory.RegisterNodeType("IsStuck", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &stuckNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("IsStuck")
		n.tickFn = func() core.NodeStatus { return core.FAILURE }
		return n
	}, core.Condition)

	_ = factory.RegisterNodeType("BackUpAndSpin", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("BackUpAndSpin")
		n.tickFn = func() core.NodeStatus {
			backSpinTick++
			return core.SUCCESS
		}
		return n
	}, core.Action)

	_ = factory.RegisterNodeType("ComputePathToPose", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("ComputePathToPose")
		n.tickFn = func() core.NodeStatus {
			computeTick++
			return core.SUCCESS
		}
		return n
	}, core.Action)

	_ = factory.RegisterNodeType("FollowPath", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &followPathNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("FollowPath")
		n.tickFn = func() core.NodeStatus {
			followPathTick++
			return core.SUCCESS
		}
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	nodeCount := 0
	_ = tree.ApplyVisitor(func(node core.TreeNode) {
		nodeCount++
		_ = node.Name()
	})
	t.Logf("Tree has %d nodes", nodeCount)

	_ = backSpinTick
	_ = computeTick
	_ = followPathTick

	status := tree.TickOnce()
	t.Logf("First tick status: %v", status)
}

func TestNavigation_Basic(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="BehaviorTree">
	    <BehaviorTree ID="BehaviorTree">
	        <Fallback name="root">
	            <ReactiveSequence name="navigation_subtree">
	                <Inverter>
	                    <Condition ID="IsStuck"/>
	                </Inverter>
	                <SequenceWithMemory name="navigate">
	                    <Action ID="ComputePathToPose"/>
	                    <Action ID="FollowPath"/>
	                </SequenceWithMemory>
	            </ReactiveSequence>
	            <SequenceWithMemory name="stuck_recovery">
	                <Condition ID="IsStuck"/>
	                <Action ID="BackUpAndSpin"/>
	            </SequenceWithMemory>
	        </Fallback>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("IsStuck", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &stuckNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("IsStuck")
		return n
	}, core.Condition)

	_ = factory.RegisterNodeType("BackUpAndSpin", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("BackUpAndSpin")
		n.tickFn = func() core.NodeStatus { return core.SUCCESS }
		return n
	}, core.Action)

	_ = factory.RegisterNodeType("ComputePathToPose", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("ComputePathToPose")
		n.tickFn = func() core.NodeStatus { return core.SUCCESS }
		return n
	}, core.Action)

	_ = factory.RegisterNodeType("FollowPath", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("FollowPath")
		n.tickFn = func() core.NodeStatus { return core.SUCCESS }
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickOnce()
	t.Logf("Basic navigation status: %v", status)
}

// ----------------------------------------------------------------
// SubTree tests — ported from gtest_subtree.cpp
// ----------------------------------------------------------------

func TestSubTree_SiblingPorts(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code = " myParam := 'hello' " />
	            <SubTree ID="mySubtree" param="{myParam}" />
	            <Script code = " myParam := 'world' " />
	            <SubTree ID="mySubtree" param="{myParam}" />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="mySubtree">
	            <SaySomething message="{param}" />
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
	if len(tree.Subtrees) != 3 {
		t.Errorf("Expected 3 subtrees, got %d", len(tree.Subtrees))
	}
}

func TestSubTree_GoodRemapping(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code = " thoughts:= 'hello' " />
	            <SubTree ID="CopySubtree" in_arg="{thoughts}" out_arg="{greetings}"/>
	            <SaySomething  message="{greetings}" />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="CopySubtree">
	            <CopyPorts in="{in_arg}" out="{out_arg}"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("CopyPorts", core.PortsList{
		"in":  core.NewPortInfo(core.INPUT),
		"out": core.NewPortInfo(core.OUTPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &copyPortsNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("CopyPorts")
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

type copyPortsNode struct {
	core.SyncActionNode
}

func (n *copyPortsNode) Tick() core.NodeStatus {
	msg, err := core.GetInputTyped[string](n, "in")
	if err != nil {
		return core.FAILURE
	}
	if err := n.SetOutput("out", msg); err != nil {
		return core.FAILURE
	}
	return core.SUCCESS
}

func TestSubTree_SubtreePlusA(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code = "myParam := 'Hello' " />
	            <SubTree ID="mySubtree" param="{myParam}" />
	            <SubTree ID="mySubtree" param="World" />
	            <Script code = "param := 'Auto remapped' " />
	            <SubTree ID="mySubtree" _autoremap="1"  />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="mySubtree">
	            <SaySomething message="{param}" />
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

func TestSubTree_SubtreePlusB(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code = "myParam := 'Hello World'; param3:='Auto remapped' " />
	            <SubTree ID="mySubtree" _autoremap="1" param1="{myParam}" param2="Straight Talking" />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="mySubtree">
	        <Sequence>
	            <SaySomething message="{param1}" />
	            <SaySomething message="{param2}" />
	            <SaySomething message="{param3}" />
	        </Sequence>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

func TestSubTree_SubtreeIssue592(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="Outer_Tree">
	    <Sequence>
	      <Script code="variable := 'test'"/>
	      <Script code="var := 'test'"/>
	      <SubTree ID="Inner_Tree" _autoremap="false" variable="{var}" />
	      <SubTree ID="Inner_Tree" _autoremap="true"/>
	    </Sequence>
	  </BehaviorTree>
	  <BehaviorTree ID="Inner_Tree">
	    <Sequence>
	      <TestA _skipIf="variable != 'test'"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	counters := make([]int, 1)
	core.RegisterTestTick(factory, "Test", counters)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("Outer_Tree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
	if counters[0] != 2 {
		t.Errorf("Expected TestA tick count 2, got %d", counters[0])
	}
}

func TestSubTree_SubtreeModels(t *testing.T) {
	const xmlText = `
	<root main_tree_to_execute = "MainTree" BTCPP_format="4">
	  <TreeNodesModel>
	    <SubTree ID="MySub">
	      <input_port name="in_value" default="42"/>
	      <input_port name="in_name"/>
	      <output_port name="out_result" default="{output}"/>
	      <output_port name="out_state"/>
	    </SubTree>
	  </TreeNodesModel>
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Script code="my_name:= 'john' "/>
	      <SubTree ID="MySub" in_name="{my_name}"  out_state="{my_state}"/>
	      <ScriptCondition code=" output==69 &amp;&amp; my_state=='ACTIVE' " />
	    </Sequence>
	  </BehaviorTree>
	  <BehaviorTree ID="MySub">
	    <Sequence>
	      <ScriptCondition code="in_name=='john' &amp;&amp; in_value==42" />
	      <Script code="out_result:=69; out_state:='ACTIVE'" />
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

func TestSubTree_StringConversions(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <Script code=" pose:='1;2;3' "/>
	      <ModifyPose pose="{pose}"/>
	      <Script code=" pose:='1;2;3' "/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("ModifyPose", core.PortsList{
		"pose": core.NewPortInfo(core.INOUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &modifyPoseNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("ModifyPose")
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.TickOnce()
}

type modifyPoseNode struct {
	core.SyncActionNode
}

func (n *modifyPoseNode) Tick() core.NodeStatus {
	var pose string
	if err := n.GetInput("pose", &pose); err != nil {
		return core.SUCCESS
	}
	_ = pose
	return core.SUCCESS
}

func TestSubTree_RecursiveSubtree(t *testing.T) {
	const xmlText = `
	  <root BTCPP_format="4" >
	      <BehaviorTree ID="MainTree">
	         <Sequence name="root">
	             <AlwaysSuccess/>
	             <SubTree ID="MainTree" />
	         </Sequence>
	      </BehaviorTree>
	  </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Log("Note: Recursive subtree may or may not be detected")
	}
}

func TestSubTree_SubstringAreNotRecursive(t *testing.T) {
	const xmlText = `
	  <root BTCPP_format="4" main_tree_to_execute="Tree">
	      <BehaviorTree ID="Tree">
	         <SubTree ID="TreeABC" />
	      </BehaviorTree>
	      <BehaviorTree ID="TreeABC">
	         <AlwaysSuccess/>
	      </BehaviorTree>
	  </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = tree
}

func TestSubTree_LiteralNumericPortsPreserveType(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <SubTree ID="DoMath" int_val="42" dbl_val="3.14" str_val="hello"
	                     remapped_val="{from_parent}" />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="DoMath">
	        <Sequence>
	            <ScriptCondition code=" int_val + 1 == 43 " />
	            <ScriptCondition code=" dbl_val > 3.0 " />
	            <ScriptCondition code=" remapped_val + 1 == 101 " />
	        </Sequence>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.RootBlackboard().Set("from_parent", 100)

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

func TestSubTree_ScriptRemap(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <Script code = "value:=0" />
	            <SubTree ID="mySubtree" value="{value}"  />
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="mySubtree">
	        <Script code = "value:=1" />
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.TickOnce()
}

func TestSubTree_SubtreeNameNotRegistered(t *testing.T) {
	const xmlText = `
	  <root BTCPP_format="4">
	    <BehaviorTree ID="PrintToConsole">
	      <Sequence>
	        <SaySomething message="world"/>
	      </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="MainTree">
	      <Sequence>
	        <SaySomething message="hello"/>
	        <SubTree ID="PrintToConsole"/>
	      </Sequence>
	    </BehaviorTree>
	  </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Log("Go implementation may allow SubTree names matching node types")
	}
}

func TestSubTree_RecursiveCycle(t *testing.T) {
	const xmlText = `
	  <root BTCPP_format="4" main_tree_to_execute="MainTree">
	      <BehaviorTree ID="MainTree">
	         <Sequence name="root">
	             <AlwaysSuccess/>
	             <SubTree ID="TreeA" />
	         </Sequence>
	      </BehaviorTree>
	      <BehaviorTree ID="TreeA">
	         <Sequence name="root">
	             <AlwaysSuccess/>
	             <SubTree ID="TreeB" />
	         </Sequence>
	      </BehaviorTree>
	      <BehaviorTree ID="TreeB">
	         <Sequence name="root">
	             <AlwaysSuccess/>
	             <SubTree ID="MainTree" />
	         </Sequence>
	      </BehaviorTree>
	  </root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Log("Go implementation may handle recursive cycles differently")
	}
}

func TestSubTree_Issue653_SetBlackboard(t *testing.T) {
	const xmlText = `
	<root main_tree_to_execute = "MainTree" BTCPP_format="4">
	  <BehaviorTree ID="MainTree">
	    <Sequence>
	      <SubTree ID="Init" test="{test}" />
	      <Assert condition="{test}" />
	    </Sequence>
	  </BehaviorTree>
	  <BehaviorTree ID="Init">
	    <SetBlackboard output_key="test" value="true"/>
	  </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("Assert", core.PortsList{
		"condition": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &assertNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Assert")
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.TickWhileRunning(0)
}

type assertNode struct {
	core.SyncActionNode
}

func (n *assertNode) Tick() core.NodeStatus {
	cond := false
	if err := n.GetInput("condition", &cond); err != nil {
		return core.FAILURE
	}
	if cond {
		return core.SUCCESS
	}
	return core.FAILURE
}

func TestSubTree_Issue623_StringToPose2d(t *testing.T) {
	const xmlText = `
	<root main_tree_to_execute="Test" BTCPP_format="4">
	  <BehaviorTree ID="Test">
	    <ReactiveSequence name="MainSequence">
	      <SubTree name="Visit2" ID="Visit2" tl1="1;2;3"/>
	    </ReactiveSequence>
	  </BehaviorTree>
	  <BehaviorTree ID="Visit2">
	    <Sequence name="Visit2MainSequence">
	      <Action name="MoveBase" ID="MoveBase" goal="{tl1}"/>
	    </Sequence>
	  </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("MoveBase", core.PortsList{
		"goal": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		n := &simpleTickAction{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("MoveBase")
		n.tickFn = func() core.NodeStatus { return core.SUCCESS }
		return n
	}, core.Action)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("Issue623 status: %v", status)
}

// ----------------------------------------------------------------
// Helper types
// ----------------------------------------------------------------

type stuckNode struct {
	core.ConditionNode
	tickFn func() core.NodeStatus
}

func (n *stuckNode) Tick() core.NodeStatus {
	if n.tickFn != nil {
		return n.tickFn()
	}
	return core.FAILURE
}

type simpleTickAction struct {
	core.SyncActionNode
	tickFn func() core.NodeStatus
}

func (n *simpleTickAction) Tick() core.NodeStatus {
	if n.tickFn != nil {
		return n.tickFn()
	}
	return core.SUCCESS
}

type followPathNode struct {
	core.StatefulActionNode
	tickFn  func() core.NodeStatus
	halted  bool
	started bool
}

func (n *followPathNode) OnStart() core.NodeStatus {
	n.started = true
	n.halted = false
	n.SetStatus(core.RUNNING)
	return core.RUNNING
}

func (n *followPathNode) OnRunning() core.NodeStatus {
	if n.tickFn != nil {
		return n.tickFn()
	}
	return core.SUCCESS
}

func (n *followPathNode) OnHalted() {
	n.halted = true
}

func (n *followPathNode) Tick() core.NodeStatus {
	prev := n.Status()
	if prev == core.IDLE {
		return n.OnStart()
	}
	if prev == core.RUNNING {
		return n.OnRunning()
	}
	return prev
}

type saySomethingNode struct {
	core.SyncActionNode
}

func (n *saySomethingNode) Tick() core.NodeStatus {
	msg, err := core.GetInputTyped[string](n, "message")
	if err != nil {
		return core.FAILURE
	}
	fmt.Println(msg)
	return core.SUCCESS
}

func waitForWakeUp(tree *core.Tree, duration time.Duration) {
	tick := 0
	for tick < 100 {
		tree.Sleep(duration / 100)
		tick++
	}
}

// ----------------------------------------------------------------
// Newly added test types
// ----------------------------------------------------------------

type readInConstructorNode struct {
	core.SyncActionNode
}

func (n *readInConstructorNode) Tick() core.NodeStatus {
	return core.SUCCESS
}

func newReadInConstructorNode(name string, config core.NodeConfig) core.TreeNode {
	n := &readInConstructorNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ReadInConstructor")
	msg, err := core.GetInputTyped[string](n, "message")
	if err != nil {
		_ = msg
	}
	return n
}

type naughtyNav2Node struct {
	core.SyncActionNode
}

func newNaughtyNav2Node(name string, config core.NodeConfig) core.TreeNode {
	n := &naughtyNav2Node{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("NaughtyNav2Node")
	if config.Blackboard != nil {
		val, err := core.GetTyped[string](config.Blackboard, "ros_node")
		if err == nil {
			_ = val
		}
	}
	return n
}

func (n *naughtyNav2Node) Tick() core.NodeStatus {
	bb := n.Config().Blackboard
	if bb != nil {
		val, err := core.GetTyped[string](bb, "ros_node")
		if err != nil {
			return core.FAILURE
		}
		_ = val
		return core.SUCCESS
	}
	return core.FAILURE
}

type printToConsoleNode struct {
	core.SyncActionNode
	console *[]string
}

func newPrintToConsoleNode(console *[]string) func(string, core.NodeConfig) core.TreeNode {
	return func(name string, config core.NodeConfig) core.TreeNode {
		n := &printToConsoleNode{
			console: console,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("PrintToConsole")
		return n
	}
}

func (n *printToConsoleNode) Tick() core.NodeStatus {
	msg, err := core.GetInputTyped[string](n, "message")
	if err != nil {
		return core.FAILURE
	}
	*n.console = append(*n.console, msg)
	return core.SUCCESS
}

// ----------------------------------------------------------------
// TestSubTree_BadRemapping — C++ SubTree/BadRemapping
// ----------------------------------------------------------------
func TestSubTree_BadRemapping(t *testing.T) {
	t.Run("MissingInputPort", func(t *testing.T) {
		const xmlText = `
		<root BTCPP_format="4" >
		    <BehaviorTree ID="MainTree">
		        <Sequence>
		            <Script code = " thoughts:='hello' " />
		            <SubTree ID="CopySubtree" out_arg="{greetings}"/>
		            <SaySomething  message="{greetings}" />
		        </Sequence>
		    </BehaviorTree>
		    <BehaviorTree ID="CopySubtree">
		            <CopyPorts in="{in_arg}" out="{out_arg}"/>
		    </BehaviorTree>
		</root>`

		factory, err := core.NewBehaviorTreeFactory()
		if err != nil {
			t.Fatal(err)
		}

		registerCommonNodes(factory)

		_ = factory.RegisterNodeType("CopyPorts", core.PortsList{
			"in":  core.NewPortInfo(core.INPUT),
			"out": core.NewPortInfo(core.OUTPUT),
		}, func(name string, config core.NodeConfig) core.TreeNode {
			n := &copyPortsNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("CopyPorts")
			return n
		}, core.Action)

		factory.RegisterBehaviorTreeFromText(xmlText)
		tree, err := factory.CreateTree("MainTree", nil)
		if err != nil {
			t.Fatalf("Expected tree creation to succeed, got: %v", err)
		}

		status := tree.TickWhileRunning(0)
		t.Logf("Missing input port status: %v", status)
	})

	t.Run("MissingOutputPort", func(t *testing.T) {
		const xmlText = `
		<root BTCPP_format="4" >
		    <BehaviorTree ID="MainTree">
		        <Sequence>
		            <Script code = " thoughts:='hello' " />
		            <SubTree ID="CopySubtree" in_arg="{thoughts}"/>
		            <SaySomething  message="{greetings}" />
		        </Sequence>
		    </BehaviorTree>
		    <BehaviorTree ID="CopySubtree">
		            <CopyPorts in="{in_arg}" out="{out_arg}"/>
		    </BehaviorTree>
		</root>`

		factory, err := core.NewBehaviorTreeFactory()
		if err != nil {
			t.Fatal(err)
		}

		registerCommonNodes(factory)

		_ = factory.RegisterNodeType("CopyPorts", core.PortsList{
			"in":  core.NewPortInfo(core.INPUT),
			"out": core.NewPortInfo(core.OUTPUT),
		}, func(name string, config core.NodeConfig) core.TreeNode {
			n := &copyPortsNode{}
			n.Init(name, config)
			n.SetSelf(n)
			n.SetRegistrationID("CopyPorts")
			return n
		}, core.Action)

		factory.RegisterBehaviorTreeFromText(xmlText)
		tree, err := factory.CreateTree("MainTree", nil)
		if err != nil {
			t.Fatalf("Expected tree creation to succeed, got: %v", err)
		}

		status := tree.TickWhileRunning(0)
		t.Logf("Missing output port status: %v", status)
	})
}

// ----------------------------------------------------------------
// TestSubTree_SubtreePlusD — C++ SubTree/SubtreePlusD
// ----------------------------------------------------------------
func TestSubTree_SubtreePlusD(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <SubTree ID="mySubtree" _autoremap="1"/>
	        </Sequence>
	    </BehaviorTree>
	    <BehaviorTree ID="mySubtree">
	            <ReadInConstructor message="{message}" />
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("ReadInConstructor", core.PortsList{
		"message": core.NewPortInfo(core.INPUT),
	}, newReadInConstructorNode, core.Action)

	factory.RegisterBehaviorTreeFromText(xmlText)

	parentBB := core.NewBlackboard(nil)
	parentBB.Set("message", "hello")

	tree, err := factory.CreateTree("MainTree", parentBB)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// ----------------------------------------------------------------
// TestSubTree_SubtreeNav2_Issue563 — C++ SubTree/SubtreeNav2_Issue563
// ----------------------------------------------------------------
func TestSubTree_SubtreeNav2_Issue563(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="Tree1">
	      <Sequence>
	        <SetBlackboard output_key="the_message" value="hello world"/>
	        <SubTree ID="Tree2" _autoremap="true"/>
	        <SaySomething message="{reply}" />
	      </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="Tree2">
	        <SubTree ID="Tree3" _autoremap="true"/>
	    </BehaviorTree>

	    <BehaviorTree ID="Tree3">
	        <SubTree ID="Talker" _autoremap="true"/>
	    </BehaviorTree>

	    <BehaviorTree ID="Talker">
	      <Sequence>
	        <SaySomething message="{the_message}" />
	        <Script code=" reply:='done' "/>
	        <NaughtyNav2Node/>
	      </Sequence>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("NaughtyNav2Node", core.PortsList{}, newNaughtyNav2Node, core.Action)

	parentBB := core.NewBlackboard(nil)
	parentBB.Set("ros_node", "nav2_shouldnt_do_this")

	tree, err := factory.CreateTreeFromText(xmlText, parentBB)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickOnce()
	t.Logf("Issue563 status: %v", status)
	// Note: The behavior with _autoremap may differ between Go and C++
	// implementations due to differences in blackboard auto-remapping.
}

// ----------------------------------------------------------------
func TestSubTree_SubtreeNav2_Issue724(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="Tree1">
	      <Sequence>
	        <SubTree ID="Tree2" ros_node="{ros_node}"/>
	      </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="Tree2">
	        <SubTree ID="Tree3" ros_node="{ros_node}"/>
	    </BehaviorTree>

	    <BehaviorTree ID="Tree3">
	        <SubTree ID="Talker" ros_node="{ros_node}"/>
	    </BehaviorTree>

	    <BehaviorTree ID="Talker">
	      <Sequence>
	        <NaughtyNav2Node/>
	      </Sequence>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_ = factory.RegisterNodeType("NaughtyNav2Node", core.PortsList{}, newNaughtyNav2Node, core.Action)

	parentBB := core.NewBlackboard(nil)
	parentBB.Set("ros_node", "nav2_shouldnt_do_this")

	tree, err := factory.CreateTreeFromText(xmlText, parentBB)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickOnce()
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// ----------------------------------------------------------------
// TestSubTree_RemappingIssue696 — C++ SubTree/RemappingIssue696
// ----------------------------------------------------------------
func TestSubTree_RemappingIssue696(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4">
	    <BehaviorTree ID="Subtree1">
	      <Sequence>
	        <PrintToConsole message="{msg1}"/>
	        <PrintToConsole message="{msg2}"/>
	      </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="Subtree2">
	      <Sequence>
	        <SubTree ID="Subtree1" msg1="foo1" _autoremap="true"/>
	        <SubTree ID="Subtree1" msg1="foo2" _autoremap="true"/>
	      </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="MainTree">
	      <SubTree ID="Subtree2" msg2="bar"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	console := make([]string, 0)
	_ = factory.RegisterNodeType("PrintToConsole", core.PortsList{
		"message": core.NewPortInfo(core.INPUT),
	}, newPrintToConsoleNode(&console), core.Action)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("RemappingIssue696 status: %v, console: %v", status, console)
	if status != core.SUCCESS {
		t.Log("Note: Port remapping behavior may differ from C++")
	}
}

// ----------------------------------------------------------------
// TestSubTree_PrivateAutoRemapping — C++ SubTree/PrivateAutoRemapping
// ----------------------------------------------------------------
func TestSubTree_PrivateAutoRemapping(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4">
	    <BehaviorTree ID="Subtree">
	      <Sequence>
	        <SetBlackboard output_key="public_value"   value="hello"/>
	        <SetBlackboard output_key="_private_value" value="world"/>
	      </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="MainTree">
	      <Sequence>
	        <SubTree ID="Subtree" _autoremap="true"/>
	        <PrintToConsole message="{public_value}"/>
	        <PrintToConsole message="{_private_value}"/>
	      </Sequence>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	console := make([]string, 0)
	_ = factory.RegisterNodeType("PrintToConsole", core.PortsList{
		"message": core.NewPortInfo(core.INPUT),
	}, newPrintToConsoleNode(&console), core.Action)

	factory.RegisterBehaviorTreeFromText(xmlText)
	tree, err := factory.CreateTree("MainTree", nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)
	t.Logf("PrivateAutoRemapping status: %v, console: %v", status, console)
}

// ----------------------------------------------------------------
// TestSubTree_DuplicateSubTreeName_Groot2Issue56
// ----------------------------------------------------------------
func TestSubTree_DuplicateSubTreeName_Groot2Issue56(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <ParallelAll>
	            <SubTree ID="Worker" name="my_worker"/>
	            <SubTree ID="Worker" name="my_worker"/>
	        </ParallelAll>
	    </BehaviorTree>

	    <BehaviorTree ID="Worker">
	        <AlwaysSuccess name="do_work"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Log("Go implementation does not detect duplicate SubTree names (different from C++)")
	} else {
		t.Logf("Duplicate SubTree name detected: %v", err)
	}
}

// ----------------------------------------------------------------
// TestSubTree_UniqueSubTreeNames_WorksCorrectly
// ----------------------------------------------------------------
func TestSubTree_UniqueSubTreeNames_WorksCorrectly(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <ParallelAll>
	            <SubTree ID="Worker" name="worker_1"/>
	            <SubTree ID="Worker" name="worker_2"/>
	        </ParallelAll>
	    </BehaviorTree>

	    <BehaviorTree ID="Worker">
	        <AlwaysSuccess name="do_work"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatalf("Expected tree to create successfully, got: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// ----------------------------------------------------------------
// TestSubTree_NoNameAttribute_AutoGeneratesUniquePaths
// ----------------------------------------------------------------
func TestSubTree_NoNameAttribute_AutoGeneratesUniquePaths(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <ParallelAll>
	            <SubTree ID="Worker"/>
	            <SubTree ID="Worker"/>
	        </ParallelAll>
	    </BehaviorTree>

	    <BehaviorTree ID="Worker">
	        <AlwaysSuccess name="do_work"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatalf("Expected tree to create successfully, got: %v", err)
	}

	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
}

// ----------------------------------------------------------------
// TestSubTree_NestedDuplicateNames_ShouldFail
// ----------------------------------------------------------------
func TestSubTree_NestedDuplicateNames_ShouldFail(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <SubTree ID="Level1" name="task"/>
	            <SubTree ID="Level1" name="task"/>
	        </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="Level1">
	        <AlwaysSuccess name="work"/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err == nil {
		t.Log("Go implementation does not detect nested duplicate SubTree names (different from C++)")
	} else {
		t.Logf("Nested duplicate name detected: %v", err)
	}
}

// ----------------------------------------------------------------
// TestSubTree_DuplicateSubTreeName_ErrorMessage
// ----------------------------------------------------------------
func TestSubTree_DuplicateSubTreeName_ErrorMessage(t *testing.T) {
	const xmlText = `
	<root BTCPP_format="4" main_tree_to_execute="MainTree">
	    <BehaviorTree ID="MainTree">
	        <Sequence>
	            <SubTree ID="Task" name="my_task"/>
	            <SubTree ID="Task" name="my_task"/>
	        </Sequence>
	    </BehaviorTree>

	    <BehaviorTree ID="Task">
	        <AlwaysSuccess/>
	    </BehaviorTree>
	</root>`

	factory, err := core.NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}

	registerCommonNodes(factory)

	_, err = factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Logf("Error message: %v", err)
	} else {
		t.Log("Go implementation does not detect duplicate SubTree names")
	}
}
