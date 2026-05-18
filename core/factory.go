package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// NodeBuilder creates a new TreeNode instance.
type NodeBuilder func(name string, config NodeConfig) TreeNode

// Tree represents a behavior tree instance.
type Tree struct {
	Subtrees  []*TreeSubtree
	Manifests map[string]TreeNodeManifest

	wakeUp      *WakeUpSignal
	uidCounter  uint16
	initialized bool
}

// TreeSubtree represents a subtree within a tree.
type TreeSubtree struct {
	Nodes        []TreeNode
	Blackboard   *Blackboard
	InstanceName string
	TreeID       string
}

// NewTree creates a new empty Tree.
func NewTree() *Tree {
	return &Tree{
		Subtrees:  nil,
		Manifests: make(map[string]TreeNodeManifest),
		wakeUp:    NewWakeUpSignal(),
	}
}

// Initialize prepares the tree for execution.
func (t *Tree) Initialize() {
	if t.initialized {
		return
	}

	for _, subtree := range t.Subtrees {
		for _, node := range subtree.Nodes {
			node.SetWakeUpInstance(t.wakeUp)
		}
	}

	t.initialized = true
}

// RootNode returns the root node of the main tree.
func (t *Tree) RootNode() TreeNode {
	if len(t.Subtrees) == 0 {
		return nil
	}
	if len(t.Subtrees[0].Nodes) == 0 {
		return nil
	}
	return t.Subtrees[0].Nodes[0]
}

// TickExactlyOnce ticks the root node exactly once.
// After the tick, if the root node's status is completed (SUCCESS or FAILURE),
// the root node's status is reset to IDLE, matching C++ behavior.
func (t *Tree) TickExactlyOnce() NodeStatus {
	root := t.RootNode()
	if root == nil {
		return FAILURE
	}
	status := root.ExecuteTick()
	if status.IsCompleted() {
		root.ResetStatus()
	}
	return status
}

// TickOnce ticks the root once, but if a wake-up signal was emitted,
// it will tick again until no wake-up is pending.
func (t *Tree) TickOnce() NodeStatus {
	return t.tickRoot(OnceUnlessWokenUp, 0)
}

// TickWhileRunning ticks the root until it returns a status other than RUNNING.
func (t *Tree) TickWhileRunning(sleepTime time.Duration) NodeStatus {
	return t.tickRoot(WhileRunning, sleepTime)
}

type tickOption int

const (
	ExactlyOnce       tickOption = 0
	OnceUnlessWokenUp tickOption = 1
	WhileRunning      tickOption = 2
)

func (t *Tree) tickRoot(opt tickOption, sleepTime time.Duration) NodeStatus {
	// Ensure initialized
	if !t.initialized {
		t.Initialize()
	}

	root := t.RootNode()
	if root == nil {
		return FAILURE
	}

	switch opt {
	case ExactlyOnce:
		return root.ExecuteTick()

	case OnceUnlessWokenUp:
		status := root.ExecuteTick()
		for t.wakeUp != nil {
			t.wakeUp.mu.Lock()
			fired := t.wakeUp.fired
			t.wakeUp.fired = false
			t.wakeUp.mu.Unlock()
			if !fired {
				break
			}
			status = root.ExecuteTick()
		}
		return status

	case WhileRunning:
		status := root.ExecuteTick()
		for status == RUNNING {
			// Wait for wake-up signal or timeout
			if t.wakeUp != nil {
				t.wakeUp.WaitFor(sleepTime)
			} else {
				time.Sleep(sleepTime)
			}
			status = root.ExecuteTick()
		}
		return status

	default:
		return root.ExecuteTick()
	}
}

// HaltTree halts all nodes in the tree.
func (t *Tree) HaltTree() {
	for _, subtree := range t.Subtrees {
		for _, node := range subtree.Nodes {
			node.Halt()
		}
	}
}

// Sleep sleeps for a duration, interruptible by wake-up signal.
func (t *Tree) Sleep(timeout time.Duration) bool {
	if t.wakeUp != nil {
		return t.wakeUp.WaitFor(timeout)
	}
	time.Sleep(timeout)
	return false
}

// EmitWakeUpSignal wakes up the tree.
func (t *Tree) EmitWakeUpSignal() {
	if t.wakeUp != nil {
		t.wakeUp.Emit()
	}
}

// WakeUpSignal returns the tree's wake-up signal.
func (t *Tree) WakeUpSignal() *WakeUpSignal {
	return t.wakeUp
}

// RootBlackboard returns the root blackboard.
func (t *Tree) RootBlackboard() *Blackboard {
	if len(t.Subtrees) == 0 {
		return nil
	}
	return t.Subtrees[0].Blackboard
}

// GetUID returns a new unique ID.
func (t *Tree) GetUID() uint16 {
	t.uidCounter++
	return t.uidCounter
}

// ApplyVisitor calls the visitor for each node in the tree recursively.
func (t *Tree) ApplyVisitor(visitor func(node TreeNode)) error {
	root := t.RootNode()
	if root != nil {
		return ApplyRecursiveVisitor(root, visitor)
	}
	return nil
}

// BehaviorTreeFactory is used to create instances of TreeNode at run-time.
type BehaviorTreeFactory struct {
	mu        sync.RWMutex
	builders  map[string]NodeBuilder
	manifests map[string]TreeNodeManifest
	builtins  map[string]bool
	enums     ScriptingEnumsRegistry

	substitutionRules map[string]SubstitutionRule
	registeredTrees   map[string]string      // tree ID -> raw XML text
	registeredBBs     map[string]*Blackboard // tree ID -> pre-associated blackboard (optional)
}

// SubstitutionRule replaces a node with another when the tree is created.
type SubstitutionRule struct {
	ReplaceWith string
}

// NewBehaviorTreeFactory creates a new factory.
func NewBehaviorTreeFactory() (*BehaviorTreeFactory, error) {
	f := &BehaviorTreeFactory{
		builders:          make(map[string]NodeBuilder),
		manifests:         make(map[string]TreeNodeManifest),
		builtins:          make(map[string]bool),
		enums:             make(ScriptingEnumsRegistry),
		substitutionRules: make(map[string]SubstitutionRule),
		registeredTrees:   make(map[string]string),
		registeredBBs:     make(map[string]*Blackboard),
	}
	if err := f.registerBuiltins(); err != nil {
		return nil, err
	}
	return f, nil
}

// subTreeNode is a simple SubTree placeholder node used by the factory.
// It embeds DecoratorNode so it can accept SetChild from the XML parser.
type subTreeNode struct {
	DecoratorNode
	subtreeID string
}

func (n *subTreeNode) Tick() NodeStatus {
	n.SetStatus(RUNNING)
	if n.Child() == nil {
		return FAILURE
	}
	return n.Child().ExecuteTick()
}

func (n *subTreeNode) SetSubtreeID(id string) {
	n.subtreeID = id
}

func (n *subTreeNode) SubtreeID() string {
	return n.subtreeID
}

func (f *BehaviorTreeFactory) registerBuiltins() error {
	// Register SubTree builder.
	// The SubTree node acts as a placeholder in the tree; the actual subtree
	// content is wired by the XML parser's recursivelyCreateSubtree.
	if err := f.RegisterNodeType("SubTree", PortsList{
		"name": NewPortInfo(INPUT),
	}, func(name string, config NodeConfig) TreeNode {
		// Use a simple struct that embeds DecoratorNode for SetChild support
		n := &subTreeNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("SubTree")
		return n
	}, Subtree); err != nil {
		return err
	}
	f.builtins["SubTree"] = true
	return nil
}

// RegisterBuilder registers a node builder with a manifest.
func (f *BehaviorTreeFactory) RegisterBuilder(manifest TreeNodeManifest, builder NodeBuilder) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := manifest.RegistrationID
	if _, exists := f.builders[id]; exists {
		return fmt.Errorf("Builder already registered for ID: %s", id)
	}
	f.builders[id] = builder
	f.manifests[id] = manifest
	return nil
}

// RegisterNodeType registers a node type with a builder function.
func (f *BehaviorTreeFactory) RegisterNodeType(id string, ports PortsList, builder NodeBuilder, nodeType NodeType) error {
	manifest := TreeNodeManifest{
		Type:           nodeType,
		RegistrationID: id,
		Ports:          ports,
	}
	return f.RegisterBuilder(manifest, builder)
}

// RegisterSimpleAction registers a simple action node.
func (f *BehaviorTreeFactory) RegisterSimpleAction(id string, tickFn func(TreeNode) NodeStatus, ports PortsList) error {
	builder := func(name string, config NodeConfig) TreeNode {
		n := &SimpleActionNode{
			tickFn: tickFn,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		return n
	}
	return f.RegisterNodeType(id, ports, builder, Action)
}

// RegisterSimpleCondition registers a simple condition node.
func (f *BehaviorTreeFactory) RegisterSimpleCondition(id string, tickFn func(TreeNode) NodeStatus, ports PortsList) error {
	builder := func(name string, config NodeConfig) TreeNode {
		n := &SimpleConditionNode{
			tickFn: tickFn,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		return n
	}
	return f.RegisterNodeType(id, ports, builder, Condition)
}

// RegisterSimpleDecorator registers a simple decorator node.
func (f *BehaviorTreeFactory) RegisterSimpleDecorator(id string, tickFn func(NodeStatus, TreeNode) NodeStatus, ports PortsList) error {
	builder := func(name string, config NodeConfig) TreeNode {
		n := &SimpleDecoratorNode{
			tickFn: tickFn,
		}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		return n
	}
	return f.RegisterNodeType(id, ports, builder, Decorator)
}

// RegisterScriptingEnum adds an enum value to the scripting language.
func (f *BehaviorTreeFactory) RegisterScriptingEnum(name string, value int) {
	f.enums[name] = value
}

// ScriptingEnums returns the registry of scripting enums.
func (f *BehaviorTreeFactory) ScriptingEnums() *ScriptingEnumsRegistry {
	return &f.enums
}

// InstantiateTreeNode creates an instance of a previously registered TreeNode.
func (f *BehaviorTreeFactory) InstantiateTreeNode(name, id string, config NodeConfig) (TreeNode, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	manifest, hasManifest := f.manifests[id]

	// Apply default port remapping only when NOT from XML.
	// The XML parser already sets InputPorts/OutputPorts with the correct remappings.
	// This must be done BEFORE calling the builder so the node receives the defaults.
	if !config.FromXML && hasManifest {
		if len(config.InputPorts) == 0 {
			config.InputPorts = make(PortsRemapping)
		}
		if len(config.OutputPorts) == 0 {
			config.OutputPorts = make(PortsRemapping)
		}
		AssignDefaultRemapping(&config, manifest.Ports)
	}

	var node TreeNode

	// Check substitution rules (matching C++ behavior)
	substituted := false
	for filter, rule := range f.substitutionRules {
		if filter == name || filter == id || WildcardMatch(config.Path, filter) {
			if rule.ReplaceWith != "" {
				subBuilder, subOk := f.builders[rule.ReplaceWith]
				if !subOk {
					return nil, fmt.Errorf("Substituted Node ID [%s] not found", rule.ReplaceWith)
				}
				node = subBuilder(name, config)
				substituted = true
			}
			break
		}
	}

	if !substituted {
		builder, ok := f.builders[id]
		if !ok {
			return nil, fmt.Errorf("Node not registered: %s", id)
		}
		node = builder(name, config)
	}

	// Set the manifest
	if hasManifest {
		config.Manifest = &manifest
	}

	// Validate port value types when config is from XML and manifest has ports
	if config.FromXML && hasManifest && len(manifest.Ports) > 0 {
		for portName, remapValue := range config.InputPorts {
			if portInfo, exists := manifest.Ports[portName]; exists {
				isBB, _ := IsBlackboardPointer(remapValue)
				if !isBB && portInfo.IsStronglyTyped() && portInfo.Converter() != nil {
					if _, err := portInfo.ParseString(remapValue); err != nil {
						return nil, fmt.Errorf("port value \"%s\" for port \"%s\" cannot be converted to %s",
							remapValue, portName, portInfo.TypeName())
					}
				}
			}
		}
		for portName, remapValue := range config.OutputPorts {
			if portInfo, exists := manifest.Ports[portName]; exists {
				isBB, _ := IsBlackboardPointer(remapValue)
				if !isBB && portInfo.IsStronglyTyped() && portInfo.Converter() != nil {
					if _, err := portInfo.ParseString(remapValue); err != nil {
						return nil, fmt.Errorf("port value \"%s\" for port \"%s\" cannot be converted to %s",
							remapValue, portName, portInfo.TypeName())
					}
				}
			}
		}
	}

	return node, nil
}

// Builders returns all registered builders.
func (f *BehaviorTreeFactory) Builders() map[string]NodeBuilder {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]NodeBuilder)
	for k, v := range f.builders {
		result[k] = v
	}
	return result
}

// Manifests returns all registered manifests.
func (f *BehaviorTreeFactory) Manifests() map[string]TreeNodeManifest {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]TreeNodeManifest)
	for k, v := range f.manifests {
		result[k] = v
	}
	return result
}

// RegisteredBehaviorTrees returns the IDs of registered behavior trees
// in sorted order for deterministic iteration.
func (f *BehaviorTreeFactory) RegisteredBehaviorTrees() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	ids := make([]string, 0, len(f.registeredTrees))
	for id := range f.registeredTrees {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ClearRegisteredBehaviorTrees clears previously-registered behavior trees.
func (f *BehaviorTreeFactory) ClearRegisteredBehaviorTrees() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredTrees = make(map[string]string)
	f.registeredBBs = make(map[string]*Blackboard)
}

// ParseXMLFunc is a function type that parses an XML text and registers trees in the factory.
type ParseXMLFunc func(factory *BehaviorTreeFactory, xmlText string)

// parseXML is a package-level function pointer that can be set by the xml package.
var parseXML ParseXMLFunc

// RegisterXMLParser sets the XML parsing function used by the factory.
// This is called by the bt/xml package's init() function to avoid circular imports.
func RegisterXMLParser(fn ParseXMLFunc) {
	parseXML = fn
}

// RegisterBehaviorTreeFromText registers a behavior tree definition from XML text.
func (f *BehaviorTreeFactory) RegisterBehaviorTreeFromText(xmlText string) {
	if parseXML != nil {
		parseXML(f, xmlText)
	} else {
		// XML parser not registered; store raw text to be parsed later
		f.mu.Lock()
		defer f.mu.Unlock()
		// Store the XML text keyed by a generated name
		// The actual tree names will be extracted when the XML parser is available
		f.registeredTrees["_pending_"] = xmlText
	}
}

// StoreRegisteredTreeXML stores the XML text for a specific tree ID.
// Called by the XML parser during Registration.
func (f *BehaviorTreeFactory) StoreRegisteredTreeXML(treeID string, xmlText string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredTrees[treeID] = xmlText
}

// GetRegisteredTreeXML returns the XML text for a specific tree ID.
func (f *BehaviorTreeFactory) GetRegisteredTreeXML(treeID string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.registeredTrees[treeID]
}

// CreateTree creates a tree from a registered behavior tree.
func (f *BehaviorTreeFactory) CreateTree(treeName string, blackboard *Blackboard) (*Tree, error) {
	if ParseXMLAndCreateTree != nil {
		return ParseXMLAndCreateTree(f, treeName, blackboard)
	}
	return nil, fmt.Errorf("XML parser not registered. Import bt/xml package.")
}

// CreateTreeFromText creates a tree directly from XML text.
func (f *BehaviorTreeFactory) CreateTreeFromText(xmlText string, blackboard *Blackboard) (*Tree, error) {
	if ParseXMLAndCreateTreeFromText != nil {
		return ParseXMLAndCreateTreeFromText(f, xmlText, blackboard)
	}
	return nil, fmt.Errorf("XML parser not registered. Import bt/xml package.")
}

// ParseXMLAndCreateTree creates a tree from a previously registered XML tree.
var ParseXMLAndCreateTree func(factory *BehaviorTreeFactory, treeName string, blackboard *Blackboard) (*Tree, error)

// ParseXMLAndCreateTreeFromText parses XML text and creates a tree immediately.
var ParseXMLAndCreateTreeFromText func(factory *BehaviorTreeFactory, xmlText string, blackboard *Blackboard) (*Tree, error)

// AddSubstitutionRule adds a rule to substitute nodes during tree creation.
func (f *BehaviorTreeFactory) AddSubstitutionRule(filter string, rule SubstitutionRule) {
	f.substitutionRules[filter] = rule
}

// ClearSubstitutionRules removes all substitution rules.
func (f *BehaviorTreeFactory) ClearSubstitutionRules() {
	f.substitutionRules = make(map[string]SubstitutionRule)
}

// SubstitutionRules returns the current substitution rules.
func (f *BehaviorTreeFactory) SubstitutionRules() map[string]SubstitutionRule {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]SubstitutionRule)
	for k, v := range f.substitutionRules {
		result[k] = v
	}
	return result
}

// AddMetadataToManifest adds metadata to an existing node manifest.
// The metadata replaces any previously set metadata for the given registration ID.
func (f *BehaviorTreeFactory) AddMetadataToManifest(registrationID string, metadata KeyValueVector) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if m, ok := f.manifests[registrationID]; ok {
		m.Metadata = metadata
		f.manifests[registrationID] = m
	}
}

// UnregisterBuilder removes a previously registered node builder.
// Builtin nodes (registered in registerBuiltins) cannot be unregistered.
func (f *BehaviorTreeFactory) UnregisterBuilder(id string) error {
	if f.builtins[id] {
		return fmt.Errorf("You can not remove the builtin registration ID [%s]", id)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.builders, id)
	delete(f.manifests, id)
	return nil
}

// PolymorphicCastRegistry is not needed in Go (handled by interface assertions).
type PolymorphicCastRegistry struct{}

// WildcardMatch checks if a string matches a wildcard pattern.
func WildcardMatch(str string, filter string) bool {
	// Simple wildcard matching: * matches any sequence
	if filter == "" || filter == "*" {
		return true
	}

	parts := splitFilter(filter)

	// Check prefix (first part is not *)
	if len(parts) > 0 && parts[0] != "*" {
		if !strings.HasPrefix(str, parts[0]) {
			return false
		}
		str = str[len(parts[0]):]
		parts = parts[1:]
	}

	// Check suffix (last part is not *)
	if len(parts) > 0 && parts[len(parts)-1] != "*" {
		last := parts[len(parts)-1]
		if !strings.HasSuffix(str, last) {
			return false
		}
		str = str[:len(str)-len(last)]
		parts = parts[:len(parts)-1]
	}

	// Match remaining middle parts in order
	for _, p := range parts {
		if p == "*" {
			continue
		}
		pos := indexOf(str, p)
		if pos < 0 {
			return false
		}
		str = str[pos+len(p):]
	}

	return true
}

func splitFilter(filter string) []string {
	var parts []string
	current := ""
	for i := 0; i < len(filter); i++ {
		if filter[i] == '*' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			parts = append(parts, "*")
		} else {
			current += string(filter[i])
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// SimpleActionNode is a sync action that uses a tick function.
type SimpleActionNode struct {
	SyncActionNode
	tickFn func(TreeNode) NodeStatus
}

func (n *SimpleActionNode) Tick() NodeStatus {
	prevStatus := n.Status()
	if prevStatus == IDLE {
		n.SetStatus(RUNNING)
		prevStatus = RUNNING
	}
	status := n.tickFn(n)
	if status != prevStatus {
		n.SetStatus(status)
	}
	return status
}

// SimpleConditionNode is a condition that uses a tick function.
type SimpleConditionNode struct {
	ConditionNode
	tickFn func(TreeNode) NodeStatus
}

func (n *SimpleConditionNode) Tick() NodeStatus {
	return n.tickFn(n)
}

// SimpleDecoratorNode is a decorator that uses a tick function.
type SimpleDecoratorNode struct {
	DecoratorNode
	tickFn func(NodeStatus, TreeNode) NodeStatus
}

func (n *SimpleDecoratorNode) Tick() NodeStatus {
	return n.tickFn(n.Child().ExecuteTick(), n)
}

// BuiltinNodes returns the set of builtin node IDs.
func (f *BehaviorTreeFactory) BuiltinNodes() map[string]bool {
	result := make(map[string]bool)
	for id := range f.builtins {
		result[id] = true
	}
	return result
}

// String conversion helpers
func NodeStatusToString(s NodeStatus) string {
	return s.String()
}

// PrintTreeRecursively prints the tree hierarchy.
func PrintTreeRecursively(root TreeNode) {
	var recursivePrint func(indent int, node TreeNode)
	recursivePrint = func(indent int, node TreeNode) {
		if node == nil {
			fmt.Printf("%s!nullptr!\n", indentStr(indent))
			return
		}
		fmt.Printf("%s%s\n", indentStr(indent), node.Name())
		indent++

		if ctrl, ok := node.(*ControlNode); ok {
			// Can't easily detect ControlNode via interface
			// Use the children method
			_ = ctrl
		}

		if control, ok := interface{}(node).(interface{ Children() []TreeNode }); ok {
			for _, child := range control.Children() {
				recursivePrint(indent, child)
			}
		} else if deco, ok := interface{}(node).(interface{ Child() TreeNode }); ok {
			recursivePrint(indent, deco.Child())
		}
	}
	fmt.Println("----------------")
	recursivePrint(0, root)
	fmt.Println("----------------")
}

func indentStr(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "   "
	}
	return s
}

// ApplyRecursiveVisitor calls the visitor for each node in the tree recursively.
func ApplyRecursiveVisitor(node TreeNode, visitor func(TreeNode)) error {
	if node == nil {
		return fmt.Errorf("One of the children of a DecoratorNode or ControlNode is nullptr")
	}

	visitor(node)

	if control, ok := interface{}(node).(interface{ Children() []TreeNode }); ok {
		for _, child := range control.Children() {
			if err := ApplyRecursiveVisitor(child, visitor); err != nil {
				return err
			}
		}
	} else if deco, ok := interface{}(node).(interface{ Child() TreeNode }); ok {
		if deco.Child() != nil {
			if err := ApplyRecursiveVisitor(deco.Child(), visitor); err != nil {
				return err
			}
		}
	}
	return nil
}

// SerializedTreeStatus is a vector of UID/status pairs.
type SerializedTreeStatus []struct {
	UID    uint16
	Status uint8
}

// BuildSerializedStatusSnapshot creates a snapshot of all node statuses.
func BuildSerializedStatusSnapshot(root TreeNode) (SerializedTreeStatus, error) {
	var result SerializedTreeStatus
	if err := ApplyRecursiveVisitor(root, func(node TreeNode) {
		result = append(result, struct {
			UID    uint16
			Status uint8
		}{node.UID(), uint8(node.Status())})
	}); err != nil {
		return nil, err
	}
	return result, nil
}
