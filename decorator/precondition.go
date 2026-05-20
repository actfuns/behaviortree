package decorator

import (
	"log/slog"

	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/script"
)

// PreconditionNode evaluates a script condition before ticking its child.
// If the script in the "if" port returns true, the child is ticked.
// If the script returns false, the node returns the status specified in the
// "else" port (FAILURE by default). Once the child starts (returns RUNNING),
// subsequent ticks continue executing the child without re-evaluating the
// precondition until completion.
type PreconditionNode struct {
	core.DecoratorNode
	script          string
	executor        core.ScriptFunction
	childrenRunning bool
}

func NewPreconditionNode(name string, config core.NodeConfig) *PreconditionNode {
	n := &PreconditionNode{}
	n.Init(name, config)
	n.SetSelf(n)
	n.SetRegistrationID("Precondition")
	n.loadExecutor()
	return n
}

func (n *PreconditionNode) loadExecutor() {
	code, err := core.GetInputTyped[string](n, "if")
	if err != nil {
		slog.Error("Missing parameter [if] in Precondition")
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

func (n *PreconditionNode) Tick() core.NodeStatus {
	n.loadExecutor()

	elseReturn := core.FAILURE
	if v, err := core.GetInputTyped[core.NodeStatus](n, "else"); err == nil {
		elseReturn = v
	}

	var tickChildren bool
	if n.childrenRunning {
		tickChildren = true
	} else {
		env := core.ScriptEnv{
			Blackboard: n.Config().Blackboard,
			Enums:      n.Config().Enums,
		}
		if n.executor != nil {
			result := n.executor(env)
			if v, err := core.Cast[bool](result); err == nil && v {
				tickChildren = true
				n.childrenRunning = true
			}
		}
	}

	if !tickChildren {
		return elseReturn
	}

	childStatus := n.Child().ExecuteTick()

	if childStatus == core.RUNNING {
		n.SetStatus(core.RUNNING)
		return core.RUNNING
	}

	n.childrenRunning = false
	n.Child().ResetStatus()
	return childStatus
}
