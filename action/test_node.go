package action

import (
	"log/slog"
	"time"

	"github.com/actfuns/behaviortree/core"
)

// TestNodeConfig configures the behavior of TestNode.
type TestNodeConfig struct {
	// ReturnStatus is the status to return when the action is completed.
	ReturnStatus core.NodeStatus

	// SuccessScript is executed when the action returns SUCCESS.
	SuccessScript string

	// FailureScript is executed when the action returns FAILURE.
	FailureScript string

	// PostScript is executed when the action is completed.
	PostScript string

	// AsyncDelayMs if > 0 makes this action asynchronous waiting this amount of time (ms).
	AsyncDelayMs int

	// CompleteFunc is an optional function invoked when the action is completed.
	CompleteFunc func() core.NodeStatus
}

// TestNode is a configurable test node used for testing.
// It can return a specific status, execute post-condition scripts,
// and complete synchronously or asynchronously.
type TestNode struct {
	core.StatefulActionNode
	config          *TestNodeConfig
	successExecutor core.ScriptFunction
	failureExecutor core.ScriptFunction
	postExecutor    core.ScriptFunction
	startTime       time.Time
	msec            int
}

func NewTestNode(name string, cfg core.NodeConfig, testConfig *TestNodeConfig) *TestNode {
	n := &TestNode{
		config: testConfig,
	}
	n.Init(name, cfg)
	n.SetSelf(n)
	n.SetRegistrationID("TestNode")

	if testConfig.ReturnStatus == core.IDLE {
		slog.Error("TestNode can not return IDLE")
		return nil
	}

	if testConfig.SuccessScript != "" {
		n.successExecutor = core.ParseScriptExpr(testConfig.SuccessScript)
	}
	if testConfig.FailureScript != "" {
		n.failureExecutor = core.ParseScriptExpr(testConfig.FailureScript)
	}
	if testConfig.PostScript != "" {
		n.postExecutor = core.ParseScriptExpr(testConfig.PostScript)
	}

	return n
}

// NewTestNodeFromConfig creates a TestNode with the blackboard configuration.
func NewTestNodeFromConfig(name string, config core.NodeConfig) *TestNode {
	return NewTestNode(name, config, &TestNodeConfig{
		ReturnStatus: core.SUCCESS,
	})
}

func (n *TestNode) OnStart() core.NodeStatus {
	if n.config.AsyncDelayMs <= 0 {
		return n.onCompleted()
	}
	n.startTime = time.Now()
	n.msec = n.config.AsyncDelayMs
	return core.RUNNING
}

func (n *TestNode) OnRunning() core.NodeStatus {
	if n.msec > 0 && time.Since(n.startTime) < time.Duration(n.msec)*time.Millisecond {
		return core.RUNNING
	}
	return n.onCompleted()
}

func (n *TestNode) OnHalted() {
}

// Tick dispatches to OnStart or OnRunning based on the current status.
// This is necessary because Go's static dispatch prevents StatefulActionNode.Tick()
// from calling the overridden methods on the embedding struct.
func (n *TestNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		return n.OnStart()
	}
	if prevStatus == core.RUNNING {
		return n.OnRunning()
	}
	return prevStatus
}

func (n *TestNode) onCompleted() core.NodeStatus {
	env := core.ScriptEnv{
		Blackboard: n.Config().Blackboard,
		Enums:      n.Config().Enums,
	}

	var status core.NodeStatus
	if n.config.CompleteFunc != nil {
		status = n.config.CompleteFunc()
	} else {
		status = n.config.ReturnStatus
	}

	if status == core.SUCCESS && n.successExecutor != nil {
		n.successExecutor(env)
	} else if status == core.FAILURE && n.failureExecutor != nil {
		n.failureExecutor(env)
	}
	if n.postExecutor != nil {
		n.postExecutor(env)
	}
	return status
}
