// Package factory provides the BehaviorTreeFactory implementation.
// It mirrors C++ BehaviorTree.CPP's BehaviorTreeFactory, registering all
// standard node types in the constructor and managing their lifecycle.
package factory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/actfuns/behaviortree/action"
	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/decorator"
	"github.com/actfuns/behaviortree/xml"
)

// behaviorTreeFactory implements core.BehaviorTreeFactory.
type behaviorTreeFactory struct {
	mu                sync.RWMutex
	builders          map[string]core.NodeBuilder
	manifests         map[string]core.TreeNodeManifest
	builtins          map[string]bool
	enums             core.ScriptingEnumsRegistry
	substitutionRules map[string]core.SubstitutionRule
	registeredTrees   map[string]string
}

// NewBehaviorTreeFactory creates a new factory with all standard nodes registered.
// This mirrors C++ BehaviorTreeFactory constructor which registers all built-in nodes.
func NewBehaviorTreeFactory() core.BehaviorTreeFactory {
	f := &behaviorTreeFactory{
		builders:          make(map[string]core.NodeBuilder),
		manifests:         make(map[string]core.TreeNodeManifest),
		builtins:          make(map[string]bool),
		enums:             make(core.ScriptingEnumsRegistry),
		substitutionRules: make(map[string]core.SubstitutionRule),
		registeredTrees:   make(map[string]string),
	}

	f.registerBuiltinNodes()
	return f
}

// registerBuiltinNodes registers all standard node types, matching C++ behavior.
func (f *behaviorTreeFactory) registerBuiltinNodes() {
	// Control nodes
	f.registerNodeType(core.Control, "Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceNode(name, config)
	})
	f.registerNodeType(core.Control, "SequenceWithMemory", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceWithMemory(name, config)
	})
	f.registerNodeType(core.Control, "ReactiveSequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewReactiveSequence(name, config)
	})
	f.registerNodeType(core.Control, "Fallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewFallbackNode(name, config)
	})
	f.registerNodeType(core.Control, "ReactiveFallback", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewReactiveFallback(name, config)
	})
	f.registerNodeType(core.Control, "IfThenElse", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewIfThenElseNode(name, config)
	})
	f.registerNodeType(core.Control, "WhileDoElse", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewWhileDoElseNode(name, config)
	})
	f.registerNodeType(core.Control, "Parallel", core.PortsList{
		"success_count": core.NewPortInfo(core.INPUT),
		"failure_count": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewParallelNode(name, config)
	})
	f.registerNodeType(core.Control, "ParallelAll", core.PortsList{
		"max_failures": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewParallelAllNode(name, config)
	})
	f.registerNodeType(core.Control, "ManualSelector", core.PortsList{
		"repeat_last_selection": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewManualSelectorNode(name, config)
	})
	f.registerNodeType(core.Control, "TryCatch", core.PortsList{
		"catch_on_halt": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewTryCatchNode(name, config)
	})

	// Switch variants (1..5 cases)
	f.registerNodeType(core.Control, "Switch2", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode { return control.NewSwitchNode(name, config) })
	f.registerNodeType(core.Control, "Switch3", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode { return control.NewSwitchNode(name, config) })
	f.registerNodeType(core.Control, "Switch4", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
		"case_4":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode { return control.NewSwitchNode(name, config) })
	f.registerNodeType(core.Control, "Switch5", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
		"case_4":   core.NewPortInfo(core.INPUT),
		"case_5":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode { return control.NewSwitchNode(name, config) })
	f.registerNodeType(core.Control, "Switch", core.PortsList{
		"variable": core.NewPortInfo(core.INPUT),
		"case_1":   core.NewPortInfo(core.INPUT),
		"case_2":   core.NewPortInfo(core.INPUT),
		"case_3":   core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode { return control.NewSwitchNode(name, config) })

	// Action nodes
	f.registerNodeType(core.Action, "AlwaysSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysSuccessNode(name, config)
	})
	f.registerNodeType(core.Action, "AlwaysFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewAlwaysFailureNode(name, config)
	})
	f.registerNodeType(core.Action, "Sleep", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewSleepNode(name, config)
	})
	f.registerNodeType(core.Action, "Script", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptNode(name, config)
	})
	f.registerNodeType(core.Condition, "ScriptCondition", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptCondition(name, config)
	})
	f.registerNodeType(core.Action, "SetBlackboard", core.PortsList{
		"output_key": core.NewPortInfo(core.INPUT),
		"value":      core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewSetBlackboardNode(name, config)
	})
	f.registerNodeType(core.Action, "UnsetBlackboard", core.PortsList{
		"key": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewUnsetBlackboardNode(name, config)
	})
	f.registerNodeType(core.Action, "UpdatedAction", core.PortsList{
		"entry": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewEntryUpdatedAction(name, config)
	})
	f.registerNodeType(core.Action, "WasEntryUpdated", core.PortsList{
		"entry": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewEntryUpdatedAction(name, config)
	})

	// Decorator nodes
	f.registerNodeType(core.Decorator, "Inverter", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewInverterNode(name, config)
	})
	f.registerNodeType(core.Decorator, "ForceSuccess", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceSuccessNode(name, config)
	})
	f.registerNodeType(core.Decorator, "ForceFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewForceFailureNode(name, config)
	})
	f.registerNodeType(core.Decorator, "RetryUntilSuccessful", core.PortsList{
		"num_attempts": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRetryNode(name, config)
	})
	f.registerNodeType(core.Decorator, "Retry", core.PortsList{
		"num_attempts": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRetryNode(name, config)
	})
	f.registerNodeType(core.Decorator, "Repeat", core.PortsList{
		"num_cycles": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRepeatNode(name, config)
	})
	f.registerNodeType(core.Decorator, "Timeout", core.PortsList{
		"msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewTimeoutNode(name, config)
	})
	f.registerNodeType(core.Decorator, "Delay", core.PortsList{
		"delay_msec": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewDelayNode(name, config)
	})
	f.registerNodeType(core.Decorator, "RunOnce", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewRunOnceNode(name, config)
	})
	f.registerNodeType(core.Decorator, "KeepRunningUntilFailure", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewKeepRunningUntilFailureNode(name, config)
	})
	f.registerNodeType(core.Decorator, "Precondition", core.PortsList(func() map[string]core.PortInfo {
		_, ifPort := core.InputPort[string]("if", "")
		_, elsePort := core.InputPort[core.NodeStatus]("else", "")
		return map[string]core.PortInfo{
			"if":   ifPort,
			"else": elsePort,
		}
	}()), func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewPreconditionNode(name, config)
	})
	f.registerNodeType(core.Decorator, "SkipUnlessUpdated", core.PortsList{
		"entry":          core.NewPortInfo(core.INPUT),
		"if_not_updated": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
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
	})

	f.registerNodeType(core.Decorator, "Loop", core.PortsList{
		"queue":    core.NewPortInfo(core.INPUT),
		"if_empty": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewLoopNode(name, config)
	})

	// SubTree
	f.registerNodeType(core.Decorator, "SubTree", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return decorator.NewSubTreeNode(name, config)
	})
}

// registerNodeType registers a node type and marks it as builtin.
func (f *behaviorTreeFactory) registerNodeType(nodeType core.NodeType, id string, ports core.PortsList, builder func(string, core.NodeConfig) core.TreeNode) {
	manifest := core.TreeNodeManifest{
		Type:           nodeType,
		RegistrationID: id,
		Ports:          ports,
	}
	f.manifests[id] = manifest
	f.builders[id] = builder
	f.builtins[id] = true
}

// RegisterBuilder registers a node builder with a manifest.
func (f *behaviorTreeFactory) RegisterBuilder(manifest core.TreeNodeManifest, builder core.NodeBuilder) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.builders[manifest.RegistrationID]; exists {
		return fmt.Errorf("Node type [%s] is already registered", manifest.RegistrationID)
	}
	f.manifests[manifest.RegistrationID] = manifest
	f.builders[manifest.RegistrationID] = builder
	return nil
}

// RegisterNodeType registers a node type with a builder function.
func (f *behaviorTreeFactory) RegisterNodeType(id string, ports core.PortsList, builder core.NodeBuilder, nodeType core.NodeType) error {
	return f.RegisterBuilder(core.TreeNodeManifest{
		Type:           nodeType,
		RegistrationID: id,
		Ports:          ports,
	}, builder)
}

// RegisterSimpleAction registers a simple action node.
func (f *behaviorTreeFactory) RegisterSimpleAction(id string, tickFn func(core.TreeNode) core.NodeStatus, ports core.PortsList) error {
	builder := func(name string, config core.NodeConfig) core.TreeNode {
		n := &SimpleActionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		n.tickFn = tickFn
		return n
	}
	return f.RegisterNodeType(id, ports, builder, core.Action)
}

// RegisterSimpleCondition registers a simple condition node.
func (f *behaviorTreeFactory) RegisterSimpleCondition(id string, tickFn func(core.TreeNode) core.NodeStatus, ports core.PortsList) error {
	builder := func(name string, config core.NodeConfig) core.TreeNode {
		n := &SimpleConditionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		n.tickFn = tickFn
		return n
	}
	return f.RegisterNodeType(id, ports, builder, core.Condition)
}

// RegisterSimpleDecorator registers a simple decorator node.
func (f *behaviorTreeFactory) RegisterSimpleDecorator(id string, tickFn func(core.NodeStatus, core.TreeNode) core.NodeStatus, ports core.PortsList) error {
	builder := func(name string, config core.NodeConfig) core.TreeNode {
		n := &SimpleDecoratorNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID(id)
		n.tickFn = tickFn
		return n
	}
	return f.RegisterNodeType(id, ports, builder, core.Decorator)
}

// RegisterScriptingEnum adds an enum value to the scripting language.
func (f *behaviorTreeFactory) RegisterScriptingEnum(name string, value int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if existing, ok := f.enums[name]; ok && existing != value {
		panic(fmt.Sprintf("RegisterScriptingEnum: [%s] was already registered with a different value (%d vs %d)", name, existing, value))
	}
	f.enums[name] = value
}

// ScriptingEnums returns the registry of scripting enums.
func (f *behaviorTreeFactory) ScriptingEnums() *core.ScriptingEnumsRegistry {
	return &f.enums
}

// InstantiateTreeNode creates an instance of a previously registered TreeNode.
func (f *behaviorTreeFactory) InstantiateTreeNode(name, id string, config core.NodeConfig) (core.TreeNode, error) {
	f.mu.RLock()
	builder, hasBuilder := f.builders[id]
	manifest, hasManifest := f.manifests[id]
	f.mu.RUnlock()

	if !hasBuilder {
		return nil, fmt.Errorf("Node type [%s] not registered", id)
	}

	// Apply substitution rules (match by registration ID or instance name, matching C++ behavior)
	if replacementID, ok := f.substitutionRules[id]; ok {
		id = replacementID.ReplaceWith
	} else if replacementID, ok = f.substitutionRules[name]; ok {
		id = replacementID.ReplaceWith
	} else {
		// Check wildcard substitution rules
		for filter, rule := range f.substitutionRules {
			if wildcardMatch(id, filter) || wildcardMatch(name, filter) {
				id = rule.ReplaceWith
				break
			}
		}
	}

	// Re-lookup after possible substitution
	f.mu.RLock()
	builder, hasBuilder = f.builders[id]
	manifest, hasManifest = f.manifests[id]
	f.mu.RUnlock()

	if !hasBuilder {
		return nil, fmt.Errorf("Node type [%s] not registered (after substitution)", id)
	}

	if hasManifest {
		config.Manifest = &manifest
	}

	// Apply default port remapping if not from XML
	if !config.FromXML {
		core.AssignDefaultRemapping(&config, manifest.Ports)
	}

	node := builder(name, config)
	if node == nil {
		return nil, fmt.Errorf("Builder for [%s] returned nil", id)
	}

	// Validate port values from XML
	if config.FromXML && hasManifest && len(manifest.Ports) > 0 {
		for portName, remapValue := range config.InputPorts {
			if portInfo, exists := manifest.Ports[portName]; exists {
				isBB, _ := core.IsBlackboardPointer(remapValue)
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
				isBB, _ := core.IsBlackboardPointer(remapValue)
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
func (f *behaviorTreeFactory) Builders() map[string]core.NodeBuilder {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]core.NodeBuilder)
	for k, v := range f.builders {
		result[k] = v
	}
	return result
}

// Manifests returns all registered manifests.
func (f *behaviorTreeFactory) Manifests() map[string]core.TreeNodeManifest {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]core.TreeNodeManifest)
	for k, v := range f.manifests {
		result[k] = v
	}
	return result
}

// RegisteredBehaviorTrees returns the IDs of registered behavior trees.
func (f *behaviorTreeFactory) RegisteredBehaviorTrees() []string {
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
func (f *behaviorTreeFactory) ClearRegisteredBehaviorTrees() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredTrees = make(map[string]string)
}

// RegisterBehaviorTreeFromText registers a behavior tree definition from XML text.
func (f *behaviorTreeFactory) RegisterBehaviorTreeFromText(xmlText string) {
	parser := xml.NewXMLParser(f)
	if err := parser.LoadFromText(xmlText, true); err != nil {
		panic(fmt.Sprintf("XML parse error: %v", err))
	}
	for _, name := range parser.RegisteredTreeNames() {
		f.StoreRegisteredTreeXML(name, xmlText)
	}
}

// RegisterBehaviorTreeFromFile registers a behavior tree definition from an XML file.
func (f *behaviorTreeFactory) RegisterBehaviorTreeFromFile(path string) error {
	parser := xml.NewXMLParser(f)
	if err := parser.LoadFromFile(path, true); err != nil {
		return fmt.Errorf("RegisterBehaviorTreeFromFile: %w", err)
	}
	for _, name := range parser.RegisteredTreeNames() {
		f.StoreRegisteredTreeXML(name, "")
	}
	return nil
}

// StoreRegisteredTreeXML stores the XML text for a specific tree ID.
func (f *behaviorTreeFactory) StoreRegisteredTreeXML(treeID string, xmlText string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.registeredTrees[treeID] = xmlText
}

// GetRegisteredTreeXML returns the XML text for a specific tree ID.
func (f *behaviorTreeFactory) GetRegisteredTreeXML(treeID string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.registeredTrees[treeID]
}

// CreateTree creates a tree from a registered behavior tree.
func (f *behaviorTreeFactory) CreateTree(treeName string, blackboard *core.Blackboard) (*core.Tree, error) {
	return xml.ParseXMLAndCreateTree(f, treeName, blackboard)
}

// CreateTreeFromText creates a tree directly from XML text.
func (f *behaviorTreeFactory) CreateTreeFromText(xmlText string, blackboard *core.Blackboard) (*core.Tree, error) {
	return xml.ParseXMLAndCreateTreeFromText(f, xmlText, blackboard)
}

// AddSubstitutionRule adds a rule to substitute nodes during tree creation.
func (f *behaviorTreeFactory) AddSubstitutionRule(filter string, rule core.SubstitutionRule) {
	f.substitutionRules[filter] = rule
}

// ClearSubstitutionRules removes all substitution rules.
func (f *behaviorTreeFactory) ClearSubstitutionRules() {
	f.substitutionRules = make(map[string]core.SubstitutionRule)
}

// SubstitutionRules returns the current substitution rules.
func (f *behaviorTreeFactory) SubstitutionRules() map[string]core.SubstitutionRule {
	f.mu.RLock()
	defer f.mu.RUnlock()
	result := make(map[string]core.SubstitutionRule)
	for k, v := range f.substitutionRules {
		result[k] = v
	}
	return result
}

// LoadSubstitutionRuleFromJSON parses a JSON string containing substitution rules.
func (f *behaviorTreeFactory) LoadSubstitutionRuleFromJSON(jsonText string) error {
	var cfg jsonSubstitutionRule
	if err := json.Unmarshal([]byte(jsonText), &cfg); err != nil {
		return fmt.Errorf("LoadSubstitutionRuleFromJSON: %w", err)
	}

	registeredConfigs := make(map[string]bool)
	for name, testCfg := range cfg.TestNodeConfigs {
		_, err := convertNodeStatusString(testCfg.ReturnStatus)
		if err != nil {
			return fmt.Errorf("LoadSubstitutionRuleFromJSON: invalid return_status '%s' for config '%s': %w",
				testCfg.ReturnStatus, name, err)
		}

		configName := name
		_ = f.RegisterNodeType(configName, core.PortsList{}, func(n string, config core.NodeConfig) core.TreeNode {
			n2 := &SimpleActionNode{}
			n2.Init(n, config)
			n2.SetSelf(n2)
			n2.SetRegistrationID(configName)
			return n2
		}, core.Action)

		registeredConfigs[name] = true
	}

	for nodeName, target := range cfg.Substitution {
		f.AddSubstitutionRule(nodeName, core.SubstitutionRule{ReplaceWith: target})
	}

	return nil
}

// AddMetadataToManifest adds metadata to an existing node manifest.
func (f *behaviorTreeFactory) AddMetadataToManifest(registrationID string, metadata core.KeyValueVector) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if m, ok := f.manifests[registrationID]; ok {
		m.Metadata = metadata
		f.manifests[registrationID] = m
	}
}

// UnregisterBuilder removes a previously registered node builder.
func (f *behaviorTreeFactory) UnregisterBuilder(id string) error {
	if f.builtins[id] {
		return fmt.Errorf("You can not remove the builtin registration ID [%s]", id)
	}
	delete(f.builders, id)
	delete(f.manifests, id)
	return nil
}

// BuiltinNodes returns the set of builtin node IDs.
func (f *behaviorTreeFactory) BuiltinNodes() map[string]bool {
	result := make(map[string]bool)
	for id := range f.builtins {
		result[id] = true
	}
	return result
}

// -- JSON substitution rule types (internal) --

type jsonSubstitutionRule struct {
	TestNodeConfigs map[string]jsonTestNodeConfig `json:"TestNodeConfigs"`
	Substitution    map[string]string             `json:"SubstitutionRules"`
}

type jsonTestNodeConfig struct {
	ReturnStatus  string `json:"return_status"`
	AsyncDelayMs  int    `json:"async_delay"`
	PostScript    string `json:"post_script"`
	SuccessScript string `json:"success_script"`
	FailureScript string `json:"failure_script"`
}

// SimpleActionNode is a sync action that uses a tick function.
type SimpleActionNode struct {
	core.SyncActionNode
	tickFn func(core.TreeNode) core.NodeStatus
}

func (n *SimpleActionNode) Tick() core.NodeStatus {
	prevStatus := n.Status()
	if prevStatus == core.IDLE {
		n.SetStatus(core.RUNNING)
		prevStatus = core.RUNNING
	}
	status := n.tickFn(n)
	if status != prevStatus {
		n.SetStatus(status)
	}
	return status
}

// SimpleConditionNode is a condition that uses a tick function.
type SimpleConditionNode struct {
	core.ConditionNode
	tickFn func(core.TreeNode) core.NodeStatus
}

func (n *SimpleConditionNode) Tick() core.NodeStatus {
	return n.tickFn(n)
}

// SimpleDecoratorNode is a decorator that uses a tick function.
type SimpleDecoratorNode struct {
	core.DecoratorNode
	tickFn func(core.NodeStatus, core.TreeNode) core.NodeStatus
}

func (n *SimpleDecoratorNode) Tick() core.NodeStatus {
	return n.tickFn(n.Child().ExecuteTick(), n)
}

// wildcardMatch checks if a string matches a wildcard pattern.
func wildcardMatch(str string, filter string) bool {
	if filter == "" || filter == "*" {
		return true
	}

	parts := splitFilter(filter)

	if len(parts) > 0 && parts[0] != "*" {
		if !strings.HasPrefix(str, parts[0]) {
			return false
		}
		str = str[len(parts[0]):]
		parts = parts[1:]
	}

	if len(parts) > 0 && parts[len(parts)-1] != "*" {
		last := parts[len(parts)-1]
		if !strings.HasSuffix(str, last) {
			return false
		}
		str = str[:len(str)-len(last)]
		parts = parts[:len(parts)-1]
	}

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

// convertNodeStatusString parses a NodeStatus string.
func convertNodeStatusString(s string) (core.NodeStatus, error) {
	switch s {
	case "SUCCESS":
		return core.SUCCESS, nil
	case "FAILURE":
		return core.FAILURE, nil
	case "RUNNING":
		return core.RUNNING, nil
	case "SKIPPED":
		return core.SKIPPED, nil
	case "IDLE":
		return core.IDLE, nil
	default:
		return core.IDLE, fmt.Errorf("unknown NodeStatus: %s", s)
	}
}
