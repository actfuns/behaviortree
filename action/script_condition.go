package action

import (
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/script"
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
	code, err := core.GetInputTyped[string](n, "code")
	if err != nil {
		slog.Error("missing port [code] in ScriptCondition")
		return
	}
	if code == n.script {
		return
	}
	fn, err := script.ParseScript(code)
	if err != nil {
		slog.Error("script parse error", "err", err)
		return
	}
	n.executor = fn
	n.script = code
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
