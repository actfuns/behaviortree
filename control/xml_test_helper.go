package control

import (
	"github.com/actfuns/behaviortree/action"
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
)

// RegisterStandardNodes registers all standard node types needed for XML-based tests.
// This includes control flow nodes, leaf nodes, and decorators used across tests.
func RegisterStandardNodes(factory *core.BehaviorTreeFactory) {
	registerControlNodes(factory)
	registerActionNodes(factory)
	registerDecoratorNodes(factory)
}

func registerControlNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSequenceNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("SequenceWithMemory", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSequenceWithMemory(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ReactiveSequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewReactiveSequence(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ReactiveFallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewReactiveFallback(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Fallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewFallbackNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Parallel", core.PortsList{
		"success_count": core.NewPortInfo(core.INPUT),
		"failure_count": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewParallelNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ParallelAll", core.PortsList{
		"max_failures": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewParallelAllNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("IfThenElse", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewIfThenElseNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("WhileDoElse", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewWhileDoElseNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("TryCatch", core.PortsList{
		"catch_on_halt": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewTryCatchNode(name, config)
	}, core.Control)

	// Switch with 1..N cases
	_ = factory.RegisterNodeType("Switch2", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSwitchNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Switch3", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSwitchNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Switch4", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
		"case_4":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSwitchNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Switch5", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
		"case_4":   core.NewPortInfo(core.INPUT),
		"case_5":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSwitchNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Switch", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return NewSwitchNode(name, config)
	}, core.Control)
}

func registerActionNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysSuccessNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("AlwaysFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysFailureNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("Sleep", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewSleepNode(name, config)
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

	_ = factory.RegisterNodeType("TestNode", core.PortsList{
		"return_status": core.NewPortInfo(core.INPUT),
		"async_delay":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewTestNodeFromConfig(name, config)
	}, core.Action)
}

func registerDecoratorNodes(factory *core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("RetryUntilSuccessful", core.PortsList{
		"num_attempts": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRetryNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Retry", core.PortsList{
		"num_attempts": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRetryNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Repeat", core.PortsList{
		"num_cycles": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRepeatNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Timeout", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewTimeoutNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Inverter", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewInverterNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("ForceFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceFailureNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("ForceSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceSuccessNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Precondition", core.PortsList(func() map[string]core.PortInfo {
		_, ifPort := core.InputPort[string]("if", "")
		_, elsePort := core.InputPort[core.NodeStatus]("else", "")
		return map[string]core.PortInfo{
			"if":   ifPort,
			"else": elsePort,
		}
	}()), func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewPreconditionNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("Delay", core.PortsList{
		"delay_msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewDelayNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("KeepRunningUntilFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewKeepRunningUntilFailureNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("RunOnce", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRunOnceNode(name, config)
	}, core.Decorator)

	_ = factory.RegisterNodeType("UpdatedAction", core.PortsList{
		"entry": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewEntryUpdatedAction(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("WasEntryUpdated", core.PortsList{
		"entry": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewEntryUpdatedAction(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("SkipUnlessUpdated", core.PortsList{
		"entry":          core.NewPortInfo(core.INPUT),
		"if_not_updated": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		// Default if_not_updated to FAILURE if not set
		ifNotUpdated := core.FAILURE
		if val, ok := config.InputPorts["if_not_updated"]; ok && val != "" {
			switch val {
			case "SUCCESS":
				ifNotUpdated = core.SUCCESS
			case "SKIPPED":
				ifNotUpdated = core.SKIPPED
			}
		}
		return decorator.NewUpdatedDecorator(name, config, ifNotUpdated)
	}, core.Decorator)
}
