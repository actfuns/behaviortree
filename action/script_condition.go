package action

import (
	"github.com/actfuns/behaviortree/core"
	"log/slog"
)

// ScriptCondition executes a script, and if the result is true, returns SUCCESS,
// FAILURE otherwise.
type ScriptCondition struct {
	core.ConditionNode
	script   string
	executor core.ScriptFunction
}

func NewScriptCondition(name string, config core.NodeConfig) *ScriptCondition {
	n := &ScriptCondition{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ScriptCondition")
	n.loadExecutor()
	return n
}

func (n *ScriptCondition) loadExecutor() {
	script, err := core.GetInputTyped[string](n, "code")
	if err != nil {
		slog.Error("missing port [code] in ScriptCondition")
		return
	}
	if script == n.script {
		return
	}
	executor := core.ParseScriptExpr(script)
	if executor == nil {
		slog.Error("script parse error")
		return
	}
	n.executor = executor
	n.script = script
}

func (n *ScriptCondition) Tick() core.NodeStatus {
	n.loadExecutor()

	env := core.ScriptEnv{
		Blackboard: n.Config().Blackboard,
		Enums:      n.Config().Enums,
	}
	result := n.executor(env)
	if v, err := core.Cast[bool](result); err == nil && v {
		return core.SUCCESS
	}
	return core.FAILURE
}
