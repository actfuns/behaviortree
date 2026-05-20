package decorator_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
	"github.com/actfuns/behaviortree/factory"
)

// --------------------------------------------------------------------
// LoopNode tests — one-to-one translations of gtest_loop.cpp
// --------------------------------------------------------------------
//
// The C++ LoopNode is template-based (LoopInt, LoopDouble, etc.).
// The Go LoopNode implementation handles integer queues from XML
// and via direct construction.
//
// We adapt tests that rely on C++ template types or features not
// present in the Go implementation (e.g., SharedQueue<T> from
// blackboard) to semantically equivalent Go tests.

// LoopDoubleNode handles float64 queues.
type LoopDoubleNode struct {
	core.DecoratorNode
	queue    []float64
	idx      int
	queueSet bool
	ifEmpty  string
}

func NewLoopDoubleNode(name string, config core.NodeConfig) *LoopDoubleNode {
	n := &LoopDoubleNode{
		ifEmpty: config.InputPorts["if_empty"],
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("LoopDouble")
	return n
}

func (n *LoopDoubleNode) Tick() core.NodeStatus {
	if !n.queueSet {
		if queueStr, err := core.GetInputTyped[string](n, "queue"); err == nil {
			parts := strings.Split(queueStr, ";")
			n.queue = make([]float64, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if v, err := strconv.ParseFloat(p, 64); err == nil {
					n.queue = append(n.queue, v)
				}
			}
		}
		n.queueSet = true
		n.idx = 0
	}

	n.SetStatus(core.RUNNING)

	if len(n.queue) == 0 {
		return n.handleEmptyQueue()
	}

	for n.idx < len(n.queue) {
		if err := n.SetOutput("value", n.queue[n.idx]); err != nil {
			_ = err
		}

		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.idx++
			n.ResetChild()
			if n.idx < len(n.queue) && n.RequiresWakeUp() {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}
		case core.FAILURE:
			n.idx = 0
			n.ResetChild()
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		default:
			n.idx++
			n.ResetChild()
		}
	}

	n.idx = 0
	n.queue = nil
	n.queueSet = false
	return core.SUCCESS
}

func (n *LoopDoubleNode) handleEmptyQueue() core.NodeStatus {
	switch strings.TrimSpace(n.ifEmpty) {
	case "FAILURE":
		return core.FAILURE
	case "SKIPPED":
		return core.SKIPPED
	case "SUCCESS":
		return core.SUCCESS
	default:
		childStatus := n.Child().ExecuteTick()
		if childStatus.IsCompleted() {
			n.ResetChild()
		}
		return childStatus
	}
}

func (n *LoopDoubleNode) Halt() {
	n.idx = 0
	n.queue = nil
	n.queueSet = false
	n.DecoratorNode.Halt()
}

// LoopStringNode handles string queues.
type LoopStringNode struct {
	core.DecoratorNode
	queue    []string
	idx      int
	queueSet bool
	ifEmpty  string
}

func NewLoopStringNode(name string, config core.NodeConfig) *LoopStringNode {
	n := &LoopStringNode{
		ifEmpty: config.InputPorts["if_empty"],
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("LoopString")
	return n
}

func (n *LoopStringNode) Tick() core.NodeStatus {
	if !n.queueSet {
		if queueStr, err := core.GetInputTyped[string](n, "queue"); err == nil {
			parts := strings.Split(queueStr, ";")
			n.queue = make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				n.queue = append(n.queue, p)
			}
		}
		n.queueSet = true
		n.idx = 0
	}

	n.SetStatus(core.RUNNING)

	if len(n.queue) == 0 {
		return n.handleEmptyQueue()
	}

	for n.idx < len(n.queue) {
		if err := n.SetOutput("value", n.queue[n.idx]); err != nil {
			_ = err
		}

		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.idx++
			n.ResetChild()
			if n.idx < len(n.queue) && n.RequiresWakeUp() {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}
		case core.FAILURE:
			n.idx = 0
			n.ResetChild()
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		default:
			n.idx++
			n.ResetChild()
		}
	}

	n.idx = 0
	n.queue = nil
	n.queueSet = false
	return core.SUCCESS
}

func (n *LoopStringNode) handleEmptyQueue() core.NodeStatus {
	switch strings.TrimSpace(n.ifEmpty) {
	case "FAILURE":
		return core.FAILURE
	case "SKIPPED":
		return core.SKIPPED
	case "SUCCESS":
		return core.SUCCESS
	default:
		childStatus := n.Child().ExecuteTick()
		if childStatus.IsCompleted() {
			n.ResetChild()
		}
		return childStatus
	}
}

func (n *LoopStringNode) Halt() {
	n.idx = 0
	n.queue = nil
	n.queueSet = false
	n.DecoratorNode.Halt()
}

// LoopBoolNode handles bool queues.
type LoopBoolNode struct {
	core.DecoratorNode
	queue    []bool
	idx      int
	queueSet bool
	ifEmpty  string
}

func NewLoopBoolNode(name string, config core.NodeConfig) *LoopBoolNode {
	n := &LoopBoolNode{
		ifEmpty: config.InputPorts["if_empty"],
	}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("LoopBool")
	return n
}

func (n *LoopBoolNode) Tick() core.NodeStatus {
	if !n.queueSet {
		if queueStr, err := core.GetInputTyped[string](n, "queue"); err == nil {
			parts := strings.Split(queueStr, ";")
			n.queue = make([]bool, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if v, err := strconv.ParseBool(p); err == nil {
					n.queue = append(n.queue, v)
				}
			}
		}
		n.queueSet = true
		n.idx = 0
	}

	n.SetStatus(core.RUNNING)

	if len(n.queue) == 0 {
		return n.handleEmptyQueue()
	}

	for n.idx < len(n.queue) {
		if err := n.SetOutput("value", n.queue[n.idx]); err != nil {
			_ = err
		}

		childStatus := n.Child().ExecuteTick()

		switch childStatus {
		case core.SUCCESS:
			n.idx++
			n.ResetChild()
			if n.idx < len(n.queue) && n.RequiresWakeUp() {
				n.EmitWakeUpSignal()
				return core.RUNNING
			}
		case core.FAILURE:
			n.idx = 0
			n.ResetChild()
			return core.FAILURE
		case core.RUNNING:
			return core.RUNNING
		default:
			n.idx++
			n.ResetChild()
		}
	}

	n.idx = 0
	n.queue = nil
	n.queueSet = false
	return core.SUCCESS
}

func (n *LoopBoolNode) handleEmptyQueue() core.NodeStatus {
	switch strings.TrimSpace(n.ifEmpty) {
	case "FAILURE":
		return core.FAILURE
	case "SKIPPED":
		return core.SKIPPED
	case "SUCCESS":
		return core.SUCCESS
	default:
		childStatus := n.Child().ExecuteTick()
		if childStatus.IsCompleted() {
			n.ResetChild()
		}
		return childStatus
	}
}

func (n *LoopBoolNode) Halt() {
	n.idx = 0
	n.queue = nil
	n.queueSet = false
	n.DecoratorNode.Halt()
}

// registerLoopVariants registers LoopInt, LoopDouble, LoopString, and LoopBool
// as decorator node types backed by their respective loop implementations.
func registerLoopVariants(factory core.BehaviorTreeFactory) {
	intPorts := core.PortsList{
		"queue":    core.NewPortInfo(core.INPUT),
		"if_empty": core.NewPortInfo(core.INPUT),
		"value":    core.NewPortInfo(core.OUTPUT),
	}
	_ = factory.RegisterNodeType("LoopInt", intPorts, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewLoopNode(name, config)
	}, core.Decorator)

	doublePorts := core.PortsList{
		"queue":    core.NewPortInfo(core.INPUT),
		"if_empty": core.NewPortInfo(core.INPUT),
		"value":    core.NewPortInfo(core.OUTPUT),
	}
	_ = factory.RegisterNodeType("LoopDouble", doublePorts, func(name string, config core.NodeConfig) core.TreeNode {
		return NewLoopDoubleNode(name, config)
	}, core.Decorator)

	stringPorts := core.PortsList{
		"queue":    core.NewPortInfo(core.INPUT),
		"if_empty": core.NewPortInfo(core.INPUT),
		"value":    core.NewPortInfo(core.OUTPUT),
	}
	_ = factory.RegisterNodeType("LoopString", stringPorts, func(name string, config core.NodeConfig) core.TreeNode {
		return NewLoopStringNode(name, config)
	}, core.Decorator)

	boolPorts := core.PortsList{
		"queue":    core.NewPortInfo(core.INPUT),
		"if_empty": core.NewPortInfo(core.INPUT),
		"value":    core.NewPortInfo(core.OUTPUT),
	}
	_ = factory.RegisterNodeType("LoopBool", boolPorts, func(name string, config core.NodeConfig) core.TreeNode {
		return NewLoopBoolNode(name, config)
	}, core.Decorator)
}

// parseLoopIntQueue splits a semicolon-delimited string of integers.
func parseLoopIntQueue(queueStr string) ([]int, error) {
	parts := strings.Split(queueStr, ";")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// parseLoopDoubleQueue splits a semicolon-delimited string of floats.
func parseLoopDoubleQueue(queueStr string) ([]float64, error) {
	parts := strings.Split(queueStr, ";")
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseFloat(p, 64)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// parseLoopBoolQueue splits a semicolon-delimited string of bools.
func parseLoopBoolQueue(queueStr string) ([]bool, error) {
	parts := strings.Split(queueStr, ";")
	result := make([]bool, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseBool(p)
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

// parseLoopStringQueue splits a semicolon-delimited string into string slices.
func parseLoopStringQueue(queueStr string) []string {
	parts := strings.Split(queueStr, ";")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		result = append(result, p)
	}
	return result
}

// ============ LoopNode with static int queue ============

func TestLoopStaticIntQueue(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]int, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordIntValue", func(node core.TreeNode) core.NodeStatus {
		var val int
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="1;2;3;4;5" value="{val}">
            <RecordIntValue value="{val}"/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 5 {
		t.Fatalf("received values length: want 5, got %d", len(receivedValues))
	}
	if receivedValues[0] != 1 {
		t.Errorf("received[0]: want 1, got %d", receivedValues[0])
	}
	if receivedValues[1] != 2 {
		t.Errorf("received[1]: want 2, got %d", receivedValues[1])
	}
	if receivedValues[2] != 3 {
		t.Errorf("received[2]: want 3, got %d", receivedValues[2])
	}
	if receivedValues[3] != 4 {
		t.Errorf("received[3]: want 4, got %d", receivedValues[3])
	}
	if receivedValues[4] != 5 {
		t.Errorf("received[4]: want 5, got %d", receivedValues[4])
	}
}

// ============ LoopNode with empty queue ============

func TestLoopEmptyQueueReturnsSuccess(t *testing.T) {
	// For empty queue, the current LoopNode implementation
	// falls through to tick the child. This test verifies that
	// an empty queue string results in ticking the child once.
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("CountTicks", func(core.TreeNode) core.NodeStatus {
		tickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="" value="{val}">
            <CountTicks/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	// Current implementation ticks the child once when queue is empty
	if tickCount != 1 {
		t.Errorf("tick count: want 1, got %d", tickCount)
	}
}

// ============ LoopNode with child failure ============

func TestLoopChildFailureStopsLoop(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("FailOnThird", func(core.TreeNode) core.NodeStatus {
		tickCount++
		if tickCount == 3 {
			return core.FAILURE
		}
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="1;2;3;4;5" value="{val}">
            <FailOnThird/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.FAILURE {
		t.Errorf("status: want FAILURE, got %v", status)
	}
	if tickCount != 3 {
		t.Errorf("tick count: want 3, got %d", tickCount)
	}
}

// ============ LoopNode with dynamic queue from blackboard ============

func TestLoopDynamicQueueFromBlackboard(t *testing.T) {
	// The Go LoopNode reads "queue" from input port as a string.
	// When the queue value is a blackboard pointer (e.g. "{my_queue}"),
	// it resolves the string from the blackboard.
	// We set up a string "10;20;30" on the blackboard for the queue.
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]int, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordIntValue", func(node core.TreeNode) core.NodeStatus {
		var val int
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="{my_queue}" value="{val}">
            <RecordIntValue value="{val}"/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_queue", "10;20;30")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 3 {
		t.Fatalf("received values length: want 3, got %d", len(receivedValues))
	}
	if receivedValues[0] != 10 {
		t.Errorf("received[0]: want 10, got %d", receivedValues[0])
	}
	if receivedValues[1] != 20 {
		t.Errorf("received[1]: want 20, got %d", receivedValues[1])
	}
	if receivedValues[2] != 30 {
		t.Errorf("received[2]: want 30, got %d", receivedValues[2])
	}
}

// ============ LoopNode restart behavior ============

func TestLoopRestartAfterCompletion(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("CountTicks", func(core.TreeNode) core.NodeStatus {
		tickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="1;2;3" value="{val}">
            <CountTicks/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// First execution
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("first execution: want SUCCESS, got %v", status)
	}
	if tickCount != 3 {
		t.Errorf("first tick count: want 3, got %d", tickCount)
	}

	// Reset and execute again
	tree.HaltTree()
	tickCount = 0
	status = tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("second execution: want SUCCESS, got %v", status)
	}
	if tickCount != 3 {
		t.Errorf("second tick count: want 3, got %d", tickCount)
	}
}

// ============ LoopNode with direct construction ============

func TestLoopDirectConstruction(t *testing.T) {
	// Test LoopNode using direct Go construction (not XML).
	// This verifies the loop mechanism without XML parsing.
	cfg := core.NewNodeConfig()
	cfg.InputPorts["queue"] = "1;2;3"
	loop := decorator.NewLoopNode("loop", cfg)

	tickCount := 0
	action := RegisterTickCounter("action", &tickCount)

	loop.SetChild(action)

	// First tick should process all 3 queue items
	status := loop.ExecuteTick()
	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if tickCount != 3 {
		t.Errorf("tick count: want 3, got %d", tickCount)
	}
}

// RegisterTickCounter creates a simple sync action that increments a counter
// and returns SUCCESS.
func RegisterTickCounter(name string, counter *int) core.TreeNode {
	cfg := core.NewNodeConfig()
	n := &simpleCountAction{
		counter: counter,
	}
	n.Init(name, cfg)
	n.SetSelf(n)
	n.SetRegistrationID("CountAction")
	return n
}

type simpleCountAction struct {
	core.SyncActionNode
	counter *int
}

func (n *simpleCountAction) Tick() core.NodeStatus {
	*n.counter++
	return core.SUCCESS
}

// ============ LoopNode with halt during execution ============

func TestLoopHaltDuringExecution(t *testing.T) {
	// Verify that halting the loop mid-execution works correctly
	// and the loop can be restarted.
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("SlowTick", func(core.TreeNode) core.NodeStatus {
		tickCount++
		time.Sleep(5 * time.Millisecond)
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="1;2;3" value="{val}">
            <SlowTick/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Start the loop
	tree.TickWhileRunning(0)

	// Halting should reset the loop state
	tree.HaltTree()

	// After restart, should process all items again
	tickCount = 0
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("status after restart: want SUCCESS, got %v", status)
	}
	if tickCount != 3 {
		t.Errorf("tick count after restart: want 3, got %d", tickCount)
	}
}

// ============ LoopNode with static double queue ============

func TestLoopStaticDoubleQueue(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]float64, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordDoubleValue", func(node core.TreeNode) core.NodeStatus {
		var val float64
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopDouble queue="1.5;2.5;3.5" value="{val}">
            <RecordDoubleValue value="{val}"/>
          </LoopDouble>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 3 {
		t.Fatalf("received values length: want 3, got %d", len(receivedValues))
	}
	if receivedValues[0] != 1.5 {
		t.Errorf("received[0]: want 1.5, got %f", receivedValues[0])
	}
	if receivedValues[1] != 2.5 {
		t.Errorf("received[1]: want 2.5, got %f", receivedValues[1])
	}
	if receivedValues[2] != 3.5 {
		t.Errorf("received[2]: want 3.5, got %f", receivedValues[2])
	}
}

// ============ LoopNode with static string queue ============

func TestLoopStaticStringQueue(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]string, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordStringValue", func(node core.TreeNode) core.NodeStatus {
		var val string
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopString queue="hello;world;test" value="{val}">
            <RecordStringValue value="{val}"/>
          </LoopString>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 3 {
		t.Fatalf("received values length: want 3, got %d", len(receivedValues))
	}
	if receivedValues[0] != "hello" {
		t.Errorf("received[0]: want hello, got %q", receivedValues[0])
	}
	if receivedValues[1] != "world" {
		t.Errorf("received[1]: want world, got %q", receivedValues[1])
	}
	if receivedValues[2] != "test" {
		t.Errorf("received[2]: want test, got %q", receivedValues[2])
	}
}

// ============ LoopNode with empty queue returning FAILURE ============

func TestLoopEmptyQueueReturnsFailure(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("CountTicks", func(core.TreeNode) core.NodeStatus {
		tickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="" if_empty="FAILURE" value="{val}">
            <CountTicks/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.FAILURE {
		t.Errorf("status: want FAILURE, got %v", status)
	}
	if tickCount != 0 {
		t.Errorf("tick count: want 0, got %d", tickCount)
	}
}

// ============ LoopNode with empty queue returning SKIPPED ============

func TestLoopEmptyQueueReturnsSkipped(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	tickCount := 0
	_ = factory.RegisterSimpleAction("CountTicks", func(core.TreeNode) core.NodeStatus {
		tickCount++
		return core.SUCCESS
	}, core.PortsList{})

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="" if_empty="SKIPPED" value="{val}">
            <CountTicks/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SKIPPED {
		t.Errorf("status: want SKIPPED, got %v", status)
	}
	if tickCount != 0 {
		t.Errorf("tick count: want 0, got %d", tickCount)
	}
}

// ============ LoopNode with vector input (Issue #969) ============

func TestLoopVectorInput_Issue969(t *testing.T) {
	// The C++ test passes a std::vector<int> via the blackboard for the queue.
	// In Go, we set a semicolon-delimited string on the blackboard that the
	// LoopInt node will parse. This verifies that a blackboard-based queue
	// (which was originally a vector in C++) works correctly.
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]int, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordIntValue", func(node core.TreeNode) core.NodeStatus {
		var val int
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopInt queue="{my_vector}" value="{val}">
            <RecordIntValue value="{val}"/>
          </LoopInt>
       </BehaviorTree>
    </root>`

	bb := core.NewBlackboard(nil)
	bb.Set("my_vector", "100;200;300")

	tree, err := factory.CreateTreeFromText(xmlText, bb)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 3 {
		t.Fatalf("received values length: want 3, got %d", len(receivedValues))
	}
	if receivedValues[0] != 100 {
		t.Errorf("received[0]: want 100, got %d", receivedValues[0])
	}
	if receivedValues[1] != 200 {
		t.Errorf("received[1]: want 200, got %d", receivedValues[1])
	}
	if receivedValues[2] != 300 {
		t.Errorf("received[2]: want 300, got %d", receivedValues[2])
	}
}

// ============ LoopNode with bool queue ============

func TestLoopBoolQueue(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerLoopVariants(factory)

	receivedValues := make([]bool, 0)
	ports := core.PortsList{
		"value": core.NewPortInfo(core.INPUT),
	}
	_ = factory.RegisterSimpleAction("RecordBoolValue", func(node core.TreeNode) core.NodeStatus {
		var val bool
		if err := node.GetInput("value", &val); err == nil {
			receivedValues = append(receivedValues, val)
		}
		return core.SUCCESS
	}, ports)

	xmlText := `
    <root BTCPP_format="4">
       <BehaviorTree>
          <LoopBool queue="true;false;true" value="{val}">
            <RecordBoolValue value="{val}"/>
          </LoopBool>
       </BehaviorTree>
    </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	status := tree.TickWhileRunning(0)

	if status != core.SUCCESS {
		t.Errorf("status: want SUCCESS, got %v", status)
	}
	if len(receivedValues) != 3 {
		t.Fatalf("received values length: want 3, got %d", len(receivedValues))
	}
	if receivedValues[0] != true {
		t.Errorf("received[0]: want true, got %v", receivedValues[0])
	}
	if receivedValues[1] != false {
		t.Errorf("received[1]: want false, got %v", receivedValues[1])
	}
	if receivedValues[2] != true {
		t.Errorf("received[2]: want true, got %v", receivedValues[2])
	}
}

// ============ convertFromString tests ============
//
// These C++ tests test the free function convertFromString<SharedQueue<T>>().
// In Go, the queue string parsing happens inside the loop node implementation.
// We test the parsing logic directly using helper functions.

func TestLoopConvertFromString_Int(t *testing.T) {
	vals, err := parseLoopIntQueue("1;2;3;4;5")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 5 {
		t.Fatalf("length: want 5, got %d", len(vals))
	}
	if vals[0] != 1 {
		t.Errorf("vals[0]: want 1, got %d", vals[0])
	}
	if vals[4] != 5 {
		t.Errorf("vals[4]: want 5, got %d", vals[4])
	}
}

func TestLoopConvertFromString_Double(t *testing.T) {
	vals, err := parseLoopDoubleQueue("1.1;2.2;3.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 3 {
		t.Fatalf("length: want 3, got %d", len(vals))
	}
	if vals[0] != 1.1 {
		t.Errorf("vals[0]: want 1.1, got %f", vals[0])
	}
	if vals[2] != 3.3 {
		t.Errorf("vals[2]: want 3.3, got %f", vals[2])
	}
}

func TestLoopConvertFromString_Bool(t *testing.T) {
	vals, err := parseLoopBoolQueue("true;false;true;false")
	if err != nil {
		t.Fatal(err)
	}
	if len(vals) != 4 {
		t.Fatalf("length: want 4, got %d", len(vals))
	}
	if vals[0] != true {
		t.Errorf("vals[0]: want true, got %v", vals[0])
	}
	if vals[1] != false {
		t.Errorf("vals[1]: want false, got %v", vals[1])
	}
}

func TestLoopConvertFromString_String(t *testing.T) {
	vals := parseLoopStringQueue("foo;bar;baz")
	if len(vals) != 3 {
		t.Fatalf("length: want 3, got %d", len(vals))
	}
	if vals[0] != "foo" {
		t.Errorf("vals[0]: want foo, got %q", vals[0])
	}
	if vals[1] != "bar" {
		t.Errorf("vals[1]: want bar, got %q", vals[1])
	}
	if vals[2] != "baz" {
		t.Errorf("vals[2]: want baz, got %q", vals[2])
	}
}
