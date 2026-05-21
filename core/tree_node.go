package core

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"
)

// TreeNode is the interface every behavior tree node must implement.
type TreeNode interface {
	// ExecuteTick is the public entry point, called by parents.
	// It wraps Tick() with pre/post condition checks and status notifications.
	ExecuteTick() NodeStatus

	// Halt interrupts a running node and resets its state.
	Halt()

	// HaltNode halts the node and runs the onHalted post-condition.
	// This is the public halt API that should be called by parents on children.
	HaltNode()

	// Tick contains the node-specific behavior. Called by ExecuteTick.
	Tick() NodeStatus

	// Status returns the current status.
	Status() NodeStatus

	// SetStatus changes status and notifies subscribers.
	// Must NOT set IDLE directly (use ResetStatus).
	SetStatus(status NodeStatus)

	// ResetStatus sets status to IDLE and notifies subscribers.
	ResetStatus()

	// NodeType returns the type classification.
	NodeType() NodeType

	// Name returns the instance name.
	Name() string

	// FullPath returns the hierarchical path including subtrees.
	FullPath() string

	// UID returns the unique numeric identifier.
	UID() uint16

	// RegistrationID returns the type name used at registration.
	RegistrationID() string

	// Config returns the node's configuration.
	Config() *NodeConfig

	// GetInput reads an input port value from the blackboard.
	GetInput(key string, dest interface{}) error

	// SetOutput writes to an output port on the blackboard.
	SetOutput(key string, value interface{}) error

	// GetLockedPortContent returns a locked reference to a port's value.
	GetLockedPortContent(key string) (*AnyPtrLocked, error)

	// EmitWakeUpSignal notifies the tree to tick again.
	EmitWakeUpSignal()

	// RequiresWakeUp returns true if a wake-up signal is registered.
	RequiresWakeUp() bool

	// GetRawPortValue returns the raw string value of a port.
	GetRawPortValue(key string) string

	// SetRegistrationID sets the registration ID.
	SetRegistrationID(id string)

	// SetWakeUpInstance sets the wake-up signal instance.
	SetWakeUpInstance(wakeUp *WakeUpSignal)

	// SetTimerQueueInstance sets the timer queue instance for this node.
	SetTimerQueueInstance(tq *TimerQueue)

	// WaitValidStatus blocks until the node is not halted, returns status.
	WaitValidStatus() NodeStatus

	// SubscribeToStatusChange registers a callback for status transitions.
	SubscribeToStatusChange(callback StatusChangeCallback) StatusChangeSubscriber

	// SetSelf sets the self-reference to the outermost embedding struct.
	SetSelf(self TreeNode)

	// TimerQueue returns the timer queue instance for scheduling delayed callbacks.
	TimerQueue() *TimerQueue
}

// StatusChangeCallback is a function called when a node's status changes.
type StatusChangeCallback func(timestamp time.Time, node TreeNode, prevStatus, status NodeStatus)

// StatusChangeSubscriber is a handle to unsubscribe from status changes.
type StatusChangeSubscriber struct {
	Unsubscribe func()
}

// PreTickCallback can be called before Tick().
// If it returns a completed status (SUCCESS/FAILURE), the actual Tick() is not executed.
type PreTickCallback func(node TreeNode) NodeStatus

// PostTickCallback is called after Tick().
// It receives the node and the status returned by Tick().
type PostTickCallback func(node TreeNode, status NodeStatus) NodeStatus

// TickMonitorCallback is called after Tick() with execution duration info.
type TickMonitorCallback func(node TreeNode, status NodeStatus, duration time.Duration)

// PreScripts holds compiled pre-condition scripts.
type PreScripts [PreCondCount]ScriptFunction

// PostScripts holds compiled post-condition scripts.
type PostScripts [PostCondCount]ScriptFunction

// treeNodeBase provides shared state and behavior for all tree nodes.
// It is embedded in every concrete node struct.
// ExecuteTick is provided here as a default implementation that delegates
// to ExecuteTickImpl using the self reference (set by SetSelf or by concrete nodes).
type treeNodeBase struct {
	name           string
	uid            uint16
	fullPath       string
	registrationID string
	status         NodeStatus
	config         NodeConfig

	self TreeNode // Set by concrete nodes to point to the outermost embedding struct

	mu              sync.Mutex
	statusChangedAt time.Time
	statusCond      *sync.Cond

	subscribers []StatusChangeCallback
	subMu       sync.RWMutex

	preTickCallback     PreTickCallback
	postTickCallback    PostTickCallback
	tickMonitorCallback TickMonitorCallback
	callbackInjectionMu sync.Mutex

	wakeUp      *WakeUpSignal
	timerQueue  *TimerQueue
	preScripts  PreScripts
	postScripts PostScripts
}

func newTreeNodeBase(name string, config NodeConfig) *treeNodeBase {
	b := &treeNodeBase{
		name:     name,
		uid:      config.UID,
		fullPath: config.Path,
		status:   IDLE,
		config:   config,
	}
	b.statusCond = sync.NewCond(&b.mu)
	return b
}

// ExecuteTickImpl implements the core executeTick logic.
// It is called by the embedding node's ExecuteTick.
func (b *treeNodeBase) ExecuteTickImpl(self TreeNode) NodeStatus {
	// Must capture current status first (matching C++: auto new_status = _p->status;)
	newStatus := b.Status()

	// Capture callbacks under lock
	b.callbackInjectionMu.Lock()
	preTick := b.preTickCallback
	postTick := b.postTickCallback
	monitorTick := b.tickMonitorCallback
	b.callbackInjectionMu.Unlock()

	// Check pre-conditions
	if status, ok := b.checkPreConditions(); ok {
		newStatus = status
	} else {
		// Pre-tick callback injection
		substituted := false
		if preTick != nil && !b.status.IsCompleted() {
			overrideStatus := preTick(self)
			if overrideStatus.IsCompleted() {
				substituted = true
				newStatus = overrideStatus
			}
		}

		if !substituted {
			// Actual tick
			t1 := time.Now()
			func() {
				defer func() {
					if r := recover(); r != nil {
						// If already wrapped by a child node, re-throw as-is
						if _, ok := r.(*NodeExecutionError); ok {
							panic(r)
						}
						panic(NewNodeExecutionError(
							TickBacktraceEntry{
								NodeName:         b.name,
								NodePath:         b.fullPath,
								RegistrationName: b.registrationID,
							},
							fmt.Sprintf("%v", r),
						))
					}
				}()
				newStatus = self.Tick()
			}()
			t2 := time.Now()
			if monitorTick != nil {
				monitorTick(self, newStatus, t2.Sub(t1))
			}
		}
	}

	// Post-conditions
	if newStatus.IsCompleted() {
		b.checkPostConditions(newStatus)
	}

	// Post-tick callback
	if postTick != nil {
		overrideStatus := postTick(self, newStatus)
		if overrideStatus.IsCompleted() {
			newStatus = overrideStatus
		}
	}

	// Preserve IDLE if SKIPPED, but set status otherwise
	if newStatus != SKIPPED {
		self.SetStatus(newStatus)
	}

	return newStatus
}

// checkPreConditions checks the pre-conditions attached to this node.
// Returns (status, true) if a pre-condition triggered, (IDLE, false) otherwise.
func (b *treeNodeBase) checkPreConditions() (NodeStatus, bool) {
	env := ScriptEnv{
		Blackboard: b.config.Blackboard,
		Enums:      b.config.Enums,
	}

	for idx := 0; idx < int(PreCondCount); idx++ {
		fn := b.preScripts[idx]
		if fn == nil {
			continue
		}
		preID := PreCond(idx)

		if b.status == IDLE || b.status == SKIPPED {
			condVal := fn(env)
			condBool := false
			if v, err := Cast[bool](condVal); err == nil {
				condBool = v
			}

			if condBool {
				switch preID {
				case FailureIf:
					return FAILURE, true
				case SuccessIf:
					return SUCCESS, true
				case SkipIf:
					return SKIPPED, true
				case WhileTrue:
					// When _while is true and node is IDLE/SKIPPED,
					// fall through — let the node tick normally
				}
			} else if preID == WhileTrue {
				// While condition is false and node is IDLE/SKIPPED:
				// return SKIPPED (matching C++ behavior)
				return SKIPPED, true
			}
		} else if b.status == RUNNING && preID == WhileTrue {
			condVal := fn(env)
			condBool := false
			if v, err := Cast[bool](condVal); err == nil {
				condBool = v
			}
			if !condBool {
				b.HaltNode()
				return SKIPPED, true
			}
		}
	}

	return IDLE, false
}

// checkPostConditions runs post-condition scripts.
func (b *treeNodeBase) checkPostConditions(status NodeStatus) {
	env := ScriptEnv{
		Blackboard: b.config.Blackboard,
		Enums:      b.config.Enums,
	}

	execScript := func(cond PostCond) {
		fn := b.postScripts[cond]
		if fn != nil {
			fn(env)
		}
	}

	switch status {
	case SUCCESS:
		execScript(OnSuccess)
	case FAILURE:
		execScript(OnFailure)
	}
	execScript(Always)
}

// getSelf returns the TreeNode self reference set by SetSelf or concrete node constructors.
func (b *treeNodeBase) getSelf() TreeNode {
	return b.self
}

// SetSelf sets the self-reference to the outermost embedding struct.
// Must be called by all concrete nodes after construction.
func (b *treeNodeBase) SetSelf(self TreeNode) {
	b.self = self
}

// ExecuteTick is the default implementation for all nodes.
// It delegates to ExecuteTickImpl using the self reference.
// Nodes that need special wrapping (DecoratorNode, SyncActionNode) override this.
func (b *treeNodeBase) ExecuteTick() NodeStatus {
	self := b.getSelf()
	if self == nil {
		panic(NewRuntimeError("Node [%s]: getSelf() returned nil. Did you forget to call SetSelf()?", b.name))
	}
	return b.ExecuteTickImpl(self)
}

// HaltNode halts the node by calling halt() and then the onHalted post-condition.
func (b *treeNodeBase) HaltNode() {
	self := b.getSelf()
	if self == nil {
		panic(NewRuntimeError("Node: getSelf() returned nil in HaltNode(). Did you forget to call SetSelf()?"))
	}
	self.Halt()

	fn := b.postScripts[OnHalted]
	if fn != nil {
		env := ScriptEnv{
			Blackboard: b.config.Blackboard,
			Enums:      b.config.Enums,
		}
		fn(env)
	}
}

// SetStatus changes the node's status and notifies subscribers.
// It will panic if you try to set IDLE directly (use ResetStatus).
func (b *treeNodeBase) SetStatus(newStatus NodeStatus) {
	if newStatus == IDLE {
		panic(NewRuntimeError("Node [%s]: you are not allowed to set manually the status to IDLE. Use ResetStatus() instead.", b.name))
	}

	b.mu.Lock()
	prevStatus := b.status
	b.status = newStatus
	b.statusChangedAt = time.Now()
	b.mu.Unlock()

	if prevStatus != newStatus {
		b.statusCond.Broadcast()

		b.subMu.RLock()
		for _, cb := range b.subscribers {
			cb(b.statusChangedAt, b.getSelf(), prevStatus, newStatus)
		}
		b.subMu.RUnlock()
	}
}

// ResetStatus sets the status to IDLE and notifies subscribers.
func (b *treeNodeBase) ResetStatus() {
	b.mu.Lock()
	prevStatus := b.status
	if prevStatus == IDLE {
		b.mu.Unlock()
		return
	}
	b.status = IDLE
	b.statusChangedAt = time.Now()
	b.mu.Unlock()

	b.statusCond.Broadcast()

	b.subMu.RLock()
	for _, cb := range b.subscribers {
		cb(b.statusChangedAt, b.getSelf(), prevStatus, IDLE)
	}
	b.subMu.RUnlock()
}

// Status returns the current node status.
func (b *treeNodeBase) Status() NodeStatus {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status
}

// IsHalted returns true if the node is in IDLE state.
func (b *treeNodeBase) IsHalted() bool {
	return b.Status() == IDLE
}

// WaitValidStatus blocks until the node is not halted, returns its status.
func (b *treeNodeBase) WaitValidStatus() NodeStatus {
	b.mu.Lock()
	for b.status == IDLE {
		b.statusCond.Wait()
	}
	status := b.status
	b.mu.Unlock()
	return status
}

// Name returns the instance name.
func (b *treeNodeBase) Name() string {
	return b.name
}

// FullPath returns the hierarchical path.
func (b *treeNodeBase) FullPath() string {
	return b.fullPath
}

// UID returns the unique identifier.
func (b *treeNodeBase) UID() uint16 {
	return b.uid
}

// RegistrationID returns the registration ID.
func (b *treeNodeBase) RegistrationID() string {
	return b.registrationID
}

// Config returns the node's configuration.
func (b *treeNodeBase) Config() *NodeConfig {
	return &b.config
}

// SubscribeToStatusChange registers a callback for status transitions.
// Returns a subscriber handle that can be used to unsubscribe.
func (b *treeNodeBase) SubscribeToStatusChange(callback StatusChangeCallback) StatusChangeSubscriber {
	b.subMu.Lock()
	id := len(b.subscribers)
	b.subscribers = append(b.subscribers, callback)
	b.subMu.Unlock()

	return StatusChangeSubscriber{
		Unsubscribe: func() {
			b.subMu.Lock()
			if id < len(b.subscribers) {
				b.subscribers[id] = nil
			}
			b.subMu.Unlock()
		},
	}
}

// SetPreTickFunction sets a pre-tick callback.
func (b *treeNodeBase) SetPreTickFunction(callback PreTickCallback) {
	b.callbackInjectionMu.Lock()
	b.preTickCallback = callback
	b.callbackInjectionMu.Unlock()
}

// SetPostTickFunction sets a post-tick callback.
func (b *treeNodeBase) SetPostTickFunction(callback PostTickCallback) {
	b.callbackInjectionMu.Lock()
	b.postTickCallback = callback
	b.callbackInjectionMu.Unlock()
}

// SetTickMonitorCallback sets a tick monitor callback.
func (b *treeNodeBase) SetTickMonitorCallback(callback TickMonitorCallback) {
	b.callbackInjectionMu.Lock()
	b.tickMonitorCallback = callback
	b.callbackInjectionMu.Unlock()
}

// EmitWakeUpSignal notifies the tree to tick again.
func (b *treeNodeBase) EmitWakeUpSignal() {
	if b.wakeUp != nil {
		b.wakeUp.Emit()
	}
}

// RequiresWakeUp returns true if a wake-up signal is registered.
func (b *treeNodeBase) RequiresWakeUp() bool {
	return b.wakeUp != nil
}

// SetRegistrationID sets the registration ID.
func (b *treeNodeBase) SetRegistrationID(id string) {
	b.registrationID = id
}

// SetWakeUpInstance sets the wake-up signal instance.
func (b *treeNodeBase) SetWakeUpInstance(wakeUp *WakeUpSignal) {
	b.wakeUp = wakeUp
}

// SetTimerQueueInstance sets the timer queue instance for scheduling delayed callbacks.
func (b *treeNodeBase) SetTimerQueueInstance(tq *TimerQueue) {
	b.timerQueue = tq
}

// TimerQueue returns the TimerQueue instance for scheduling delayed callbacks.
// Falls back to a global default if no tree-specific instance was set.
func (b *treeNodeBase) TimerQueue() *TimerQueue {
	if b.timerQueue != nil {
		return b.timerQueue
	}
	return defaultTimerQueue
}

// SetFullPath sets the full path.
func (b *treeNodeBase) SetFullPath(path string) {
	b.fullPath = path
}

// SetUID sets the unique identifier.
func (b *treeNodeBase) SetUID(uid uint16) {
	b.uid = uid
}

// PreScripts returns the precondition scripts array.
func (b *treeNodeBase) PreScripts() *PreScripts {
	return &b.preScripts
}

// PostScripts returns the postcondition scripts array.
func (b *treeNodeBase) PostScripts() *PostScripts {
	return &b.postScripts
}

// GetInput reads an input port value from the blackboard.
func (b *treeNodeBase) GetInput(key string, dest interface{}) error {
	portValueStr := ""

	// Check input ports remapping
	if val, ok := b.config.InputPorts[key]; ok {
		portValueStr = val
	} else if b.config.Manifest != nil {
		// Check manifest for default value
		portInfo, ok := b.config.Manifest.Ports[key]
		if !ok {
			return NewRuntimeError("getInput() of node '%s' failed because the manifest doesn't contain the key: [%s]", b.fullPath, key)
		}
		if !portInfo.DefaultValue().IsEmpty() {
			if portInfo.DefaultValue().IsString() {
				s, _ := portInfo.DefaultValue().ToString()
				portValueStr = s
			} else {
				// Direct assignment from default value
				return assignValue(portInfo.DefaultValue(), dest)
			}
		} else {
			return NewRuntimeError("getInput() of node '%s' failed: key [%s] not found", b.fullPath, key)
		}
	} else {
		return NewRuntimeError("getInput() of node '%s' failed: invalid config", b.fullPath)
	}

	// Try to resolve as blackboard pointer
	if resolvedKey, ok := GetRemappedKey(key, portValueStr); ok {
		if b.config.Blackboard == nil {
			return NewRuntimeError("getInput(): trying to access an invalid Blackboard")
		}
		return b.config.Blackboard.GetInto(resolvedKey, dest)
	}

	// Pure string value, parse it using the port's typed converter if available
	if b.config.Manifest != nil {
		if portInfo, ok := b.config.Manifest.Ports[key]; ok {
			if portInfo.IsStronglyTyped() && portInfo.Converter() != nil {
				parsed, err := portInfo.ParseString(portValueStr)
				if err != nil {
					return err
				}
				return assignValue(parsed, dest)
			}
		}
	}
	return parseStringValue(portValueStr, dest)
}

// GetInputTyped reads an input port and returns the typed value.
func GetInputTyped[T any](node TreeNode, key string) (T, error) {
	var dest T
	err := node.GetInput(key, &dest)
	return dest, err
}

// SetOutput writes to an output port on the blackboard.
func (b *treeNodeBase) SetOutput(key string, value interface{}) error {
	if b.config.Blackboard == nil {
		return NewRuntimeError("setOutput() failed: trying to access an invalid Blackboard")
	}

	remappedKey, ok := b.config.OutputPorts[key]
	if !ok {
		return NewRuntimeError("setOutput() failed: NodeConfig::OutputPorts does not contain the key: [%s]", key)
	}

	// If the value is an untyped Any, validate that the port was declared as Any
	if _, isAny := value.(Any); isAny {
		if b.config.Manifest != nil {
			if portInfo, ok := b.config.Manifest.Ports[key]; ok {
				portType := portInfo.Type()
				if portType != nil && portType != reflect.TypeOf(Any{}) && portInfo.IsStronglyTyped() {
					return NewLogicError("setOutput<Any> is not allowed, unless the port " +
						"was declared using OutputPort<Any>")
				}
			}
		}
	}

	if remappedKey == "{=}" || remappedKey == "=" {
		b.config.Blackboard.Set(key, value)
		return nil
	}

	if ok, strippedKey := IsBlackboardPointer(remappedKey); ok {
		b.config.Blackboard.Set(strippedKey, value)
		return nil
	}

	return NewRuntimeError("setOutput requires a blackboard pointer. Use {}")
}

// ModifyPortsRemapping updates existing input and output port remappings
// with new values from the given remapping.
func (b *treeNodeBase) ModifyPortsRemapping(newRemapping PortsRemapping) {
	for key, newVal := range newRemapping {
		if _, ok := b.config.InputPorts[key]; ok {
			b.config.InputPorts[key] = newVal
		}
		if _, ok := b.config.OutputPorts[key]; ok {
			b.config.OutputPorts[key] = newVal
		}
	}
}

// GetRawPortValue returns the raw string value of a port.
func (b *treeNodeBase) GetRawPortValue(key string) string {
	if val, ok := b.config.InputPorts[key]; ok {
		return val
	}
	if val, ok := b.config.OutputPorts[key]; ok {
		return val
	}
	return ""
}

// GetLockedPortContent returns a locked reference to a port's value.
func (b *treeNodeBase) GetLockedPortContent(key string) (*AnyPtrLocked, error) {
	raw := b.GetRawPortValue(key)
	if remappedKey, ok := GetRemappedKey(key, raw); ok {
		if b.config.Blackboard != nil {
			entry := b.config.Blackboard.GetEntry(remappedKey)
			if entry != nil {
				return &AnyPtrLocked{
					entry: entry,
				}, nil
			}
			// Try creating the entry from manifest
			if b.config.Manifest != nil {
				if portInfo, ok := b.config.Manifest.Ports[key]; ok {
					b.config.Blackboard.CreateEntry(remappedKey,
						NewPortInfoTyped(portInfo.Direction(), portInfo.TypeInfo))
					entry = b.config.Blackboard.GetEntry(remappedKey)
					if entry != nil {
						return &AnyPtrLocked{entry: entry}, nil
					}
				}
			}
		}
	}
	return nil, NewRuntimeError("getLockedPortContent: key [%s] not found", key)
}

// assignValue assigns an Any value to a destination pointer.
func assignValue(src Any, dest interface{}) error {
	if dest == nil {
		return fmt.Errorf("assignValue: destination is nil")
	}

	switch d := dest.(type) {
	case *int:
		v, err := src.ToInt64()
		if err != nil {
			return err
		}
		*d = int(v)
	case *int8:
		v, err := src.ToInt64()
		if err != nil {
			return err
		}
		if v < math.MinInt8 || v > math.MaxInt8 {
			return fmt.Errorf("value %d out of range for int8", v)
		}
		*d = int8(v)
	case *int16:
		v, err := src.ToInt64()
		if err != nil {
			return err
		}
		if v < math.MinInt16 || v > math.MaxInt16 {
			return fmt.Errorf("value %d out of range for int16", v)
		}
		*d = int16(v)
	case *int32:
		v, err := src.ToInt64()
		if err != nil {
			return err
		}
		if v < math.MinInt32 || v > math.MaxInt32 {
			return fmt.Errorf("value %d out of range for int32", v)
		}
		*d = int32(v)
	case *int64:
		v, err := src.ToInt64()
		if err != nil {
			return err
		}
		*d = v
	case *uint:
		v, err := src.ToUint64()
		if err != nil {
			return err
		}
		*d = uint(v)
	case *uint8:
		v, err := src.ToUint64()
		if err != nil {
			return err
		}
		if v > math.MaxUint8 {
			return fmt.Errorf("value %d out of range for uint8", v)
		}
		*d = uint8(v)
	case *uint16:
		v, err := src.ToUint64()
		if err != nil {
			return err
		}
		if v > math.MaxUint16 {
			return fmt.Errorf("value %d out of range for uint16", v)
		}
		*d = uint16(v)
	case *uint32:
		v, err := src.ToUint64()
		if err != nil {
			return err
		}
		if v > math.MaxUint32 {
			return fmt.Errorf("value %d out of range for uint32", v)
		}
		*d = uint32(v)
	case *uint64:
		v, err := src.ToUint64()
		if err != nil {
			return err
		}
		*d = v
	case *float32:
		v, err := src.ToFloat64()
		if err != nil {
			return err
		}
		*d = float32(v)
	case *float64:
		v, err := src.ToFloat64()
		if err != nil {
			return err
		}
		*d = v
	case *string:
		v, err := src.ToString()
		if err != nil {
			return err
		}
		*d = v
	case *bool:
		v, err := src.ToBool()
		if err != nil {
			return err
		}
		*d = v
	case *Any:
		*d = src
	default:
		// Reflection fallback for unknown pointer types
		dv := reflect.ValueOf(dest)
		if dv.Kind() != reflect.Ptr || dv.IsNil() {
			return fmt.Errorf("assignValue: destination must be a non-nil pointer")
		}
		elemType := dv.Elem().Type()
		if src.originalType != nil && src.originalType.AssignableTo(elemType) {
			dv.Elem().Set(reflect.ValueOf(src.value))
			return nil
		}
		return fmt.Errorf("cannot assign Any value to type %s", elemType)
	}

	return nil
}

// parseStringValue parses a string and assigns it to a destination pointer.
func parseStringValue(str string, dest interface{}) error {
	return assignValue(AnyOf(str), dest)
}
