package action

import (
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/script"
	"log/slog"
)

// ScriptNode executes a piece of script code to set or modify entries in the Blackboard.
// The script is passed via the input port "code".
// The node always returns SUCCESS after executing the script.
type ScriptNode struct {
	core.SyncActionNode
	script   string
	executor core.ScriptFunction
}

func NewScriptNode(name string, config core.NodeConfig) *ScriptNode {
	n := &ScriptNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("ScriptNode")
	n.loadExecutor()
	return n
}

func (n *ScriptNode) loadExecutor() {
	code, err := core.GetInputTyped[string](n, "code")
	if err != nil {
		slog.Error("missing port [code] in Script")
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

func (n *ScriptNode) Tick() core.NodeStatus {
	n.loadExecutor()
	if n.executor != nil {
		env := core.ScriptEnv{
			Blackboard: n.Config().Blackboard,
			Enums:      n.Config().Enums,
		}
		n.executor(env)
	}
	return core.SUCCESS
}
