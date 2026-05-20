// Package xml provides XML parsing for behavior tree definitions.
// It translates C++ BehaviorTree.CPP src/xml_parsing.cpp to Go.
package xml

import (
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/actfuns/behaviortree/core"
)

// ---------------------------------------------------------------------------
// Internal node representation from XML
// ---------------------------------------------------------------------------

type xmlNode struct {
	name       string            // element name
	attributes map[string]string // all attributes
	children   []*xmlNode
	lineNum    int // approximate line number
}

// ---------------------------------------------------------------------------
// Parser state
// ---------------------------------------------------------------------------

type xmlParser struct {
	factory        core.BehaviorTreeFactory
	treeRoots      map[string]*xmlNode // tree ID -> BehaviorTree element children
	subtreeModels  map[string]subtreeModel
	currentDir     string
	suffixCount    int
	rootAttributes map[string]string // attributes from the <root> element
}

type subtreeModel struct {
	ports map[string]core.PortInfo
}

// NewXMLParser creates a new XML parser bound to a factory.
func NewXMLParser(factory core.BehaviorTreeFactory) *xmlParser {
	return &xmlParser{
		factory:       factory,
		treeRoots:     make(map[string]*xmlNode),
		subtreeModels: make(map[string]subtreeModel),
		currentDir:    ".",
	}
}

// ---------------------------------------------------------------------------
// XML token-based parsing (builds our simple xmlNode tree)
// ---------------------------------------------------------------------------
func (p *xmlParser) LoadFromText(xmlText string, addIncludes bool) error {
	return p.parseXML(xmlText, addIncludes)
}

func (p *xmlParser) loadFromFile(path string, addIncludes bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	oldDir := p.currentDir
	abs, err := filepath.Abs(filepath.Dir(path))
	if err == nil {
		p.currentDir = abs
	}
	defer func() { p.currentDir = oldDir }()
	return p.parseXML(string(data), addIncludes)
}

func (p *xmlParser) parseXML(xmlText string, addIncludes bool) error {
	decoder := xml.NewDecoder(strings.NewReader(xmlText))

	// Parse into raw token tree
	var stack []*xmlNode
	var root *xmlNode

	for {
		tok, err := decoder.Token()
		if err != nil {
			break // EOF or error
		}

		switch t := tok.(type) {
		case xml.StartElement:
			node := &xmlNode{
				name:       t.Name.Local,
				attributes: make(map[string]string),
			}
			for _, attr := range t.Attr {
				node.attributes[attr.Name.Local] = attr.Value
			}

			if len(stack) == 0 {
				root = node
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, node)
			}
			stack = append(stack, node)

		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}

		case xml.CharData:
			// ignore text nodes
		}
	}

	if root == nil {
		return fmt.Errorf("XML parsing error: no root element")
	}
	if root.name != "root" {
		return fmt.Errorf("The XML must have a root node called <root>, got <%s>", root.name)
	}

	return p.processRoot(root, addIncludes)
}

func (p *xmlParser) processRoot(root *xmlNode, addIncludes bool) error {
	format := root.attributes["BTCPP_format"]
	if format == "" {
		log.Print("Warnings: The first tag of the XML (<root>) should contain the attribute [BTCPP_format=\"4\"]\n" +
			"Please check if your XML is compatible with version 4.x of BT.CPP")
	}

	// Store root attributes
	p.rootAttributes = make(map[string]string)
	for k, v := range root.attributes {
		p.rootAttributes[k] = v
	}

	// Process includes recursively
	if addIncludes {
		for _, child := range root.children {
			if child.name == "include" {
				pathAttr := child.attributes["path"]
				if pathAttr == "" {
					return fmt.Errorf("Invalid <include> tag: missing 'path' attribute")
				}
				filePath := pathAttr
				if !filepath.IsAbs(filePath) {
					filePath = filepath.Join(p.currentDir, filePath)
				}
				if err := p.loadFromFile(filePath, addIncludes); err != nil {
					return err
				}
			}
		}
	}

	// Collect registered node types for verification
	manifests := p.factory.Manifests()
	registeredNodes := make(map[string]core.NodeType)
	for id, m := range manifests {
		registeredNodes[id] = m.Type
	}

	// Simplified verification: just check for basic errors
	// Full VerifyXML is skipped in this Go version; node-level validation happens during instantiation

	// Load subtree models from TreeNodesModel
	if err := p.loadSubtreeModels(root); err != nil {
		return err
	}

	// Register each BehaviorTree
	for _, child := range root.children {
		if child.name == "BehaviorTree" {
			treeName := child.attributes["ID"]
			if treeName == "" {
				treeName = fmt.Sprintf("BehaviorTree_%d", p.suffixCount)
				p.suffixCount++
			}
			p.treeRoots[treeName] = child
		}
	}

	return nil
}

func (p *xmlParser) loadSubtreeModels(root *xmlNode) error {
	for _, modelsNode := range root.children {
		if modelsNode.name == "TreeNodesModel" {
			for _, subNode := range modelsNode.children {
				if subNode.name == "SubTree" {
					subtreeID := subNode.attributes["ID"]
					if subtreeID == "" {
						return fmt.Errorf("Missing attribute 'ID' in SubTree element within TreeNodesModel")
					}
					model := subtreeModel{ports: make(map[string]core.PortInfo)}
					for _, portNode := range subNode.children {
						var dir core.PortDirection
						switch portNode.name {
						case "input_port":
							dir = core.INPUT
						case "output_port":
							dir = core.OUTPUT
						case "inout_port":
							dir = core.INOUT
						default:
							continue
						}
						portName := portNode.attributes["name"]
						if portName == "" {
							return fmt.Errorf("Missing attribute [name] in port (SubTree model)")
						}
						if !core.IsAllowedPortName(portName) || core.IsReservedAttribute(portName) {
							return fmt.Errorf("Invalid port name '%s' in SubTree model", portName)
						}
						pi := core.NewPortInfo(dir)
						if def, ok := portNode.attributes["default"]; ok {
							pi.SetDefaultValue(core.AnyOf(def))
						}
						if desc, ok := portNode.attributes["description"]; ok {
							pi.SetDescription(desc)
						}
						model.ports[portName] = pi
					}
					p.subtreeModels[subtreeID] = model
				}
			}
		}
	}
	return nil
}

// RegisteredTreeNames returns the names of all registered trees in sorted order.
func (p *xmlParser) RegisteredTreeNames() []string {
	var names []string
	for name := range p.treeRoots {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// InstantiateTree creates a Tree from a previously-loaded XML definition.
func (p *xmlParser) InstantiateTree(blackboard *core.Blackboard, mainTreeID string) (*core.Tree, error) {
	tree := core.NewTree()

	if mainTreeID == "" {
		// Try to find main_tree_to_execute from root attributes
		if id, ok := p.rootAttributes["main_tree_to_execute"]; ok {
			mainTreeID = id
		} else if len(p.treeRoots) == 1 {
			for name := range p.treeRoots {
				mainTreeID = name
			}
		} else {
			return nil, fmt.Errorf("[main_tree_to_execute] was not specified correctly")
		}
	}

	if blackboard == nil {
		blackboard = core.NewBlackboard(nil)
	}

	if blackboard == nil {
		return nil, fmt.Errorf("XMLParser::instantiateTree needs a non-empty root_blackboard")
	}

	ancestors := make(map[string]bool)
	if err := p.recursivelyCreateSubtree(mainTreeID, "", "", tree, blackboard, nil, ancestors); err != nil {
		return nil, err
	}

	tree.Initialize()
	return tree, nil
}

func (p *xmlParser) recursivelyCreateSubtree(treeID, treePath, prefixPath string, outputTree *core.Tree,
	blackboard *core.Blackboard, rootNode core.TreeNode, ancestors map[string]bool) error {

	if ancestors[treeID] {
		return fmt.Errorf("Recursive behavior tree cycle detected: tree '%s' references itself", treeID)
	}
	ancestors[treeID] = true
	defer delete(ancestors, treeID)

	btNode, ok := p.treeRoots[treeID]
	if !ok {
		return fmt.Errorf("Can't find a tree with name: %s", treeID)
	}

	// The BehaviorTree element's first child is the actual root node
	if len(btNode.children) == 0 {
		return fmt.Errorf("BehaviorTree '%s' has no children", treeID)
	}
	rootElement := btNode.children[0]

	// Create a new subtree entry
	subtree := &core.TreeSubtree{
		Blackboard:   blackboard,
		InstanceName: treePath,
		TreeID:       treeID,
	}

	outputTree.Subtrees = append(outputTree.Subtrees, subtree)

	// Recursively create nodes
	var recStep func(parentNode core.TreeNode, element *xmlNode, depth int, prefix string) error
	recStep = func(parentNode core.TreeNode, element *xmlNode, depth int, prefix string) error {
		if depth > 256 {
			return fmt.Errorf("Maximum XML nesting depth exceeded")
		}

		node, err := p.createNodeFromXML(element, blackboard, parentNode, prefix, outputTree)
		if err != nil {
			return err
		}
		subtree.Nodes = append(subtree.Nodes, node)

		// Check if this is a SubTreeNode
		isSubTree := false
		if element.attributes["ID"] != "" {
			if id := element.attributes["ID"]; id != "" {
				// Check if this element is a SubTree built-in
				if element.name == "SubTree" || p.isSubTreeType(element, id) {
					isSubTree = true
				}
			}
		}
		if element.name == "SubTree" {
			isSubTree = true
		}

		if isSubTree {
			// Create new blackboard for subtree
			newBB := core.NewBlackboard(blackboard)
			doAutoRemap := false
			subtreeRemapping := make(map[string]string)

			for key, val := range element.attributes {
				if key == "_autoremap" {
					doAutoRemap = (val == "true" || val == "1")
					newBB.EnableAutoRemapping(doAutoRemap)
					continue
				}
				if !core.IsAllowedPortName(key) {
					continue
				}
				if val == "{=}" {
					val = "{" + key + "}"
				}
				subtreeRemapping[key] = val
			}

			// Check subtree model for mandatory ports
			subtreeID := element.attributes["ID"]
			if model, ok := p.subtreeModels[subtreeID]; ok {
				for portName, portInfo := range model.ports {
					if _, exists := subtreeRemapping[portName]; !exists && !doAutoRemap {
						if defaultVal := portInfo.DefaultValueString(); defaultVal != "" {
							subtreeRemapping[portName] = defaultVal
						} else {
							return fmt.Errorf("In the <TreeNodesModel> the <Subtree ID=\"%s\"> is defining a mandatory port called [%s], but you are not remapping it", subtreeID, portName)
						}
					}
				}
			}

			for attrName, attrValue := range subtreeRemapping {
				if ok, key := core.IsBlackboardPointer(attrValue); ok {
					newBB.AddSubtreeRemapping(attrName, key)
				} else {
					// Constant value — disable auto-remap, set value, restore
					newBB.EnableAutoRemapping(false)
					// Match C++ std::from_chars: strict full-string parsing.
					// Try integer first (no decimal point, no exponent).
					stored := false
					if attrValue != "" &&
						!strings.ContainsAny(attrValue, ".eE") {
						if v, err := strconv.ParseInt(attrValue, 10, 64); err == nil {
							// Verify entire string was consumed (from_chars semantics)
							if strconv.FormatInt(v, 10) == attrValue {
								const maxInt32 = 1<<31 - 1
								const minInt32 = -1 << 31
								if v >= minInt32 && v <= maxInt32 {
									if err := newBB.Set(attrName, int(v)); err != nil {
										return err
									}
								} else {
									if err := newBB.Set(attrName, v); err != nil {
										return err
									}
								}
								stored = true
							}
						}
					}
					if !stored {
						if v, err := strconv.ParseFloat(attrValue, 64); err == nil {
							// Verify entire string was consumed (from_chars semantics)
							if strconv.FormatFloat(v, 'f', -1, 64) == attrValue ||
								strconv.FormatFloat(v, 'e', -1, 64) == attrValue ||
								strconv.FormatFloat(v, 'g', -1, 64) == attrValue {
								if err := newBB.Set(attrName, v); err != nil {
									return err
								}
								stored = true
							}
						}
					}
					if !stored {
						if err := newBB.Set(attrName, attrValue); err != nil {
							return err
						}
					}
					newBB.EnableAutoRemapping(doAutoRemap)
				}
			}

			subtreePath := treePath
			if subtreePath != "" {
				subtreePath += "/"
			}
			if name, ok := element.attributes["name"]; ok {
				subtreePath += name
			} else {
				subtreePath += subtreeID + "::" + strconv.Itoa(int(node.UID()))
			}

			// Check for duplicate paths
			for _, sub := range outputTree.Subtrees {
				if sub.InstanceName == subtreePath {
					return fmt.Errorf("Duplicate SubTree path detected: '%s'", subtreePath)
				}
			}

			if err := p.recursivelyCreateSubtree(subtreeID, subtreePath, subtreePath+"/", outputTree, newBB, node, ancestors); err != nil {
				return err
			}
		} else {
			// Regular node: process children
			for _, childElement := range element.children {
				if err := recStep(node, childElement, depth+1, prefix); err != nil {
					return err
				}
			}
		}
		return nil
	}

	return recStep(rootNode, rootElement, 0, prefixPath)
}

func (p *xmlParser) isSubTreeType(_ *xmlNode, id string) bool {
	// Check if a SubTree element has been registered via the factory
	// SubTree nodes have NodeType == Subtree (5)
	manifests := p.factory.Manifests()
	if m, ok := manifests[id]; ok && m.Type == core.Subtree {
		return true
	}
	return false
}

func (p *xmlParser) createNodeFromXML(element *xmlNode, blackboard *core.Blackboard,
	parentNode core.TreeNode, prefixPath string, outputTree *core.Tree) (core.TreeNode, error) {

	elementName := element.name
	elementID := element.attributes["ID"]

	// Determine the type ID for factory lookup
	nodeType := parseNodeType(elementName)
	var typeID string

	if elementID == "" {
		// Custom element name: <MyCustomNode>
		if nodeType == core.Undefined {
			typeID = elementName
		} else {
			// Built-in without ID — e.g. <Sequence/> (but element is Sequence which isn't a standard built-in name)
			// Actually, built-in elements like Action/Decorator/Control/Condition/SubTree require ID
			typeID = elementName
		}
	} else {
		if nodeType != core.Undefined {
			typeID = elementID
		} else {
			return nil, fmt.Errorf("Attribute [ID] is not allowed in <%s>", typeID)
		}
	}

	// For built-in type elements (Action, Decorator, Control, Condition, SubTree), use ID
	if nodeType != core.Undefined {
		if elementID == "" {
			return nil, fmt.Errorf("Attribute [ID] is mandatory in <%s>", elementName)
		}
		typeID = elementID
	} else {
		typeID = elementName
		if elementID != "" {
			return nil, fmt.Errorf("Attribute [ID] is not allowed in <%s>", elementName)
		}
	}

	// Instance name: defaults to typeID unless 'name' attribute is present
	instanceName := typeID
	if name, ok := element.attributes["name"]; ok {
		instanceName = name
	}

	// Get manifest
	manifests := p.factory.Manifests()
	manifest, hasManifest := manifests[typeID]

	// Build port remapping
	portRemap := make(map[string]string)
	otherAttributes := make(core.NonPortAttributes)

	for key, val := range element.attributes {
		if key == "name" || key == "ID" {
			continue
		}
		if core.IsAllowedPortName(key) {
			if hasManifest {
				portInfo, portExists := manifest.Ports[key]
				if !portExists {
					return nil, fmt.Errorf("a port with name [%s] is found in the XML (<%s>) but not in the providedPorts() of its registered node type.",
						key, elementName)
				}
				// Validate string→type conversion if the value is not a blackboard pointer.
				isBB, _ := core.IsBlackboardPointer(val)
				if !isBB && portInfo.IsStronglyTyped() && portInfo.Converter() != nil {
					if err := tryConvert(portInfo, val); err != nil {
						return nil, fmt.Errorf("The port with name \"%s\" and value \"%s\" can not be converted to %s",
							key, val, portInfo.TypeName())
					}
				}
			}
			portRemap[key] = val
		} else if !core.IsReservedAttribute(key) {
			otherAttributes[key] = val
		}
	}

	// Build NodeConfig
	config := core.NewNodeConfig()
	config.Blackboard = blackboard
	config.Enums = p.factory.ScriptingEnums()
	config.Path = prefixPath + instanceName
	config.UID = outputTree.GetUID()
	config.FromXML = true

	if typeID == instanceName {
		config.Path += "::" + strconv.Itoa(int(config.UID))
	}

	if hasManifest {
		config.Manifest = &manifest
	}

	// Pre/post conditions from reserved attributes
	readCondition := func(attrName string, condType core.PreCond) {
		if script, ok := element.attributes[attrName]; ok {
			config.PreConditions[condType] = script
			delete(otherAttributes, attrName)
		}
	}
	readCondition("_failureIf", core.FailureIf)
	readCondition("_successIf", core.SuccessIf)
	readCondition("_skipIf", core.SkipIf)
	readCondition("_while", core.WhileTrue)

	readPostCondition := func(attrName string, condType core.PostCond) {
		if script, ok := element.attributes[attrName]; ok {
			config.PostConditions[condType] = script
			delete(otherAttributes, attrName)
		}
	}
	readPostCondition("_onHalted", core.OnHalted)
	readPostCondition("_onFailure", core.OnFailure)
	readPostCondition("_onSuccess", core.OnSuccess)
	readPostCondition("_post", core.Always)

	config.OtherAttributes = otherAttributes

	// Check if this is a SubTree
	isSubTree := (nodeType == core.Subtree)

	if isSubTree {
		config.InputPorts = portRemap

		// Set pre/post conditions from scripts
		for pre, script := range config.PreConditions {
			_ = pre
			_ = script
		}
	} else {
		if !hasManifest {
			return nil, fmt.Errorf("Missing manifest for element: %s", typeID)
		}

		// Validate port remapping against manifest
		for portName := range portRemap {
			if _, exists := manifest.Ports[portName]; !exists {
				return nil, fmt.Errorf("Possible typo? In the XML, you tried to remap port \"%s\" in node [%s (type %s)], but the manifest of this node does not contain a port with this name.",
					portName, config.Path, typeID)
			}
		}

		// Initialize port entries in blackboard from manifest
		for portName, portInfo := range manifest.Ports {
			remapValue, hasRemap := portRemap[portName]
			if !hasRemap {
				continue
			}
			if key, ok := core.GetRemappedKey(portName, remapValue); ok {
				// Check if entry already exists with type info
				existing := blackboard.GetEntry(key)
				if existing == nil {
					if _, err := blackboard.CreateEntry(key, portInfo); err != nil {
						return nil, err
					}
				}
			}
		}

		// Split ports by direction
		config.InputPorts = make(core.PortsRemapping)
		config.OutputPorts = make(core.PortsRemapping)

		for portName, remapValue := range portRemap {
			if portInfo, exists := manifest.Ports[portName]; exists {
				dir := portInfo.Direction()
				if dir != core.OUTPUT {
					config.InputPorts[portName] = remapValue
				}
				if dir != core.INPUT {
					config.OutputPorts[portName] = remapValue
				}
			} else {
				// Port not in manifest, put in input by default
				config.InputPorts[portName] = remapValue
			}
		}

		// Apply default values for unset ports
		for portName, portInfo := range manifest.Ports {
			defaultStr := portInfo.DefaultValueString()
			if defaultStr == "" {
				continue
			}
			dir := portInfo.Direction()
			if dir != core.OUTPUT {
				if _, set := config.InputPorts[portName]; !set {
					config.InputPorts[portName] = defaultStr
				}
			}
			if dir != core.INPUT {
				if _, set := config.OutputPorts[portName]; !set {
					if ok, _ := core.IsBlackboardPointer(defaultStr); ok {
						config.OutputPorts[portName] = defaultStr
					}
				}
			}
		}
	}

	// Create node via factory
	isSubTreeNode := func(_, _ string, config core.NodeConfig) (core.TreeNode, error) {
		if isSubTree {
			return p.factory.InstantiateTreeNode(instanceName, "SubTree", config)
		}
		return p.factory.InstantiateTreeNode(instanceName, typeID, config)
	}

	newNode, err := isSubTreeNode(instanceName, typeID, config)
	if err != nil {
		return nil, err
	}

	// Set pre/post condition scripts on the node
	if hasManifest {
		for pre, script := range config.PreConditions {
			setPreScript(newNode, pre, script)
		}
		for post, script := range config.PostConditions {
			setPostScript(newNode, post, script)
		}
	}

	// Add to parent
	if parentNode != nil {
		if ctrl, ok := parentNode.(interface{ AddChild(core.TreeNode) }); ok {
			ctrl.AddChild(newNode)
		} else if deco, ok := parentNode.(interface{ SetChild(core.TreeNode) }); ok {
			if panicVal := func() (r interface{}) {
				defer func() { r = recover() }()
				deco.SetChild(newNode)
				return nil
			}(); panicVal != nil {
				return nil, fmt.Errorf("Error setting child on decorator: %v", panicVal)
			}
		}
	}

	// If SubTree, set the subtree ID
	if isSubTree {
		if st, ok := newNode.(interface{ SetSubtreeID(string) }); ok {
			st.SetSubtreeID(typeID)
		}
	}

	return newNode, nil
}

func setPreScript(node core.TreeNode, cond core.PreCond, script string) {
	// Access the treeNodeBase's PreScripts via the Config's method
	// Use the treeNodeBase interface check with type assertion
	type preScriptSetter interface {
		PreScripts() *core.PreScripts
	}
	if psNode, ok := node.(preScriptSetter); ok {
		ps := psNode.PreScripts()
		ps[cond] = core.ParseScriptExpr(script)
	}
}

func setPostScript(node core.TreeNode, cond core.PostCond, script string) {
	type postScriptSetter interface {
		PostScripts() *core.PostScripts
	}
	if psNode, ok := node.(postScriptSetter); ok {
		ps := psNode.PostScripts()
		ps[cond] = core.ParseScriptExpr(script)
	}
}

// parseNodeType converts an element name to a NodeType.
func parseNodeType(name string) core.NodeType {
	switch name {
	case "Action":
		return core.Action
	case "Condition":
		return core.Condition
	case "Control":
		return core.Control
	case "Decorator":
		return core.Decorator
	case "SubTree":
		return core.Subtree
	default:
		return core.Undefined
	}
}

// ---------------------------------------------------------------------------
// Public API — callback registration
// ---------------------------------------------------------------------------

// ParseXMLAndCreateTree parses previously registered XML and creates a tree.
func ParseXMLAndCreateTree(factory core.BehaviorTreeFactory, treeName string, blackboard *core.Blackboard) (*core.Tree, error) {
	xmlText := factory.GetRegisteredTreeXML(treeName)
	if xmlText == "" {
		return nil, fmt.Errorf("No registered behavior tree found with ID: %s", treeName)
	}

	parser := NewXMLParser(factory)

	// Load all registered trees so SubTree references can be resolved.
	for _, tn := range factory.RegisteredBehaviorTrees() {
		prevXML := factory.GetRegisteredTreeXML(tn)
		if prevXML != "" && prevXML != xmlText {
			if err := parser.LoadFromText(prevXML, false); err != nil {
				return nil, fmt.Errorf("XML parse error for registered tree '%s': %v", tn, err)
			}
		}
	}

	if err := parser.LoadFromText(xmlText, true); err != nil {
		return nil, fmt.Errorf("XML parse error: %v", err)
	}

	if blackboard == nil {
		blackboard = core.NewBlackboard(nil)
	}

	tree, err := parser.InstantiateTree(blackboard, treeName)
	if err != nil {
		return nil, fmt.Errorf("Tree instantiation error: %v", err)
	}

	tree.Manifests = factory.Manifests()
	return tree, nil
}

// ParseXMLAndCreateTreeFromText parses XML text and creates a tree immediately.
func ParseXMLAndCreateTreeFromText(factory core.BehaviorTreeFactory, xmlText string, blackboard *core.Blackboard) (*core.Tree, error) {
	parser := NewXMLParser(factory)

	// Load previously-registered trees from the factory first.
	// This allows SubTree references to trees registered via
	// RegisterBehaviorTreeFromText to be found.
	for _, treeName := range factory.RegisteredBehaviorTrees() {
		prevXML := factory.GetRegisteredTreeXML(treeName)
		if prevXML != "" && prevXML != xmlText {
			if err := parser.LoadFromText(prevXML, false); err != nil {
				return nil, fmt.Errorf("XML parse error for registered tree '%s': %v", treeName, err)
			}
		}
	}

	if err := parser.LoadFromText(xmlText, true); err != nil {
		return nil, fmt.Errorf("XML parse error: %v", err)
	}

	if blackboard == nil {
		blackboard = core.NewBlackboard(nil)
	}

	// Find the main tree to execute
	mainTreeID := ""
	// First check for main_tree_to_execute attribute
	if id, ok := parser.rootAttributes["main_tree_to_execute"]; ok {
		mainTreeID = id
	} else if len(parser.treeRoots) == 1 {
		for name := range parser.treeRoots {
			mainTreeID = name
		}
	} else {
		// Sort tree names for deterministic selection
		names := make([]string, 0, len(parser.treeRoots))
		for name := range parser.treeRoots {
			names = append(names, name)
		}
		sort.Strings(names)
		mainTreeID = names[0]
	}
	if mainTreeID == "" {
		return nil, fmt.Errorf("No BehaviorTree found in XML")
	}

	tree, err := parser.InstantiateTree(blackboard, mainTreeID)
	if err != nil {
		return nil, fmt.Errorf("Tree instantiation error: %v", err)
	}

	// Set the factory manifests
	tree.Manifests = factory.Manifests()
	return tree, nil
}

// ---------------------------------------------------------------------------
// XML writing — TreeNodesModel serialization
// ---------------------------------------------------------------------------

// tryConvert attempts to parse a string value into the port's type.
func tryConvert(portInfo core.PortInfo, val string) error {
	_, err := portInfo.ParseString(val)
	return err
}
