package core

// ActionNodeBase is the base class for action nodes.
type ActionNodeBase struct {
	LeafNode
}

// Init initializes an ActionNodeBase.
func (n *ActionNodeBase) Init(name string, config NodeConfig) {
	n.LeafNode.Init(name, config)
}

// NodeType returns ACTION.
func (n *ActionNodeBase) NodeType() NodeType {
	return Action
}

// SyncActionNode is an ActionNode that prevents RUNNING status
// and doesn't require halt() implementation.
type SyncActionNode struct {
	ActionNodeBase
}

// Init initializes a SyncActionNode.
func (n *SyncActionNode) Init(name string, config NodeConfig) {
	n.ActionNodeBase.Init(name, config)
}

// ExecuteTick overrides to prevent RUNNING status.
func (n *SyncActionNode) ExecuteTick() NodeStatus {
	self := n.getSelf()
	if self == nil {
		panic("SyncActionNode: SetSelf() was not called for " + n.name)
	}
	status := n.ExecuteTickImpl(self)
	if status == RUNNING {
		panic(NewLogicError("SyncActionNode [%s] must not return RUNNING", n.Name()))
	}
	return status
}

// Halt resets the status.
func (n *SyncActionNode) Halt() {
	n.ResetStatus()
}

// Tick implements the TreeNode interface.
// Concrete types that embed SyncActionNode should override this method.
func (n *SyncActionNode) Tick() NodeStatus {
	return SUCCESS
}

// StatefulActionNode provides an easier way to implement asynchronous actions
// using onStart()/onRunning()/onHalted() pattern.
type StatefulActionNode struct {
	ActionNodeBase
	haltRequested bool
}

// Init initializes a StatefulActionNode.
func (n *StatefulActionNode) Init(name string, config NodeConfig) {
	n.ActionNodeBase.Init(name, config)
	n.haltRequested = false
}

// OnStart is called when transitioning from IDLE state.
// If it returns RUNNING, this becomes an asynchronous node.
func (n *StatefulActionNode) OnStart() NodeStatus {
	return SUCCESS
}

// OnRunning is invoked when the action is already in the RUNNING state.
func (n *StatefulActionNode) OnRunning() NodeStatus {
	return SUCCESS
}

// OnHalted is called when halt() is called and the action is RUNNING.
func (n *StatefulActionNode) OnHalted() {
}

// HaltRequested returns true if halt was requested.
func (n *StatefulActionNode) HaltRequested() bool {
	return n.haltRequested
}

// Tick implements the stateful pattern.
func (n *StatefulActionNode) Tick() NodeStatus {
	prevStatus := n.Status()
	if prevStatus == IDLE {
		n.haltRequested = false
		newStatus := n.OnStart()
		if newStatus == IDLE {
			panic(NewLogicError("StatefulActionNode::onStart() must not return IDLE"))
		}
		return newStatus
	}
	if prevStatus == RUNNING {
		newStatus := n.OnRunning()
		if newStatus == IDLE {
			panic(NewLogicError("StatefulActionNode::onRunning() must not return IDLE"))
		}
		return newStatus
	}
	return prevStatus
}

// Halt interrupts the action.
func (n *StatefulActionNode) Halt() {
	n.haltRequested = true
	if n.Status() == RUNNING {
		n.OnHalted()
	}
	n.ResetStatus()
}
