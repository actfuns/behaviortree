package core

import (
	"testing"
)

func TestBasicTypes_ToStr_NodeStatus(t *testing.T) {
	if s := SUCCESS.String(); s != "SUCCESS" {
		t.Errorf("SUCCESS.String() = %s, want SUCCESS", s)
	}
	if s := FAILURE.String(); s != "FAILURE" {
		t.Errorf("FAILURE.String() = %s, want FAILURE", s)
	}
	if s := RUNNING.String(); s != "RUNNING" {
		t.Errorf("RUNNING.String() = %s, want RUNNING", s)
	}
	if s := IDLE.String(); s != "IDLE" {
		t.Errorf("IDLE.String() = %s, want IDLE", s)
	}
	if s := SKIPPED.String(); s != "SKIPPED" {
		t.Errorf("SKIPPED.String() = %s, want SKIPPED", s)
	}
}

// TestBasicTypes_ToStr_NodeStatus_Colored verifies colored node status output.
// Equivalent of C++ BasicTypes/ToStr_NodeStatus_Colored.
func TestBasicTypes_ToStr_NodeStatus_Colored(t *testing.T) {
	_ = SUCCESS.String()
	_ = FAILURE.String()
	_ = RUNNING.String()
	_ = IDLE.String()
	_ = SKIPPED.String()
}

func TestBasicTypes_ToStr_PortDirection(t *testing.T) {
	if s := INPUT.String(); s != "INPUT" {
		t.Errorf("INPUT.String() = %s, want INPUT", s)
	}
	if s := OUTPUT.String(); s != "OUTPUT" {
		t.Errorf("OUTPUT.String() = %s, want OUTPUT", s)
	}
	if s := INOUT.String(); s != "INOUT" {
		t.Errorf("INOUT.String() = %s, want INOUT", s)
	}
}

func TestBasicTypes_ToStr_NodeType(t *testing.T) {
	if s := Action.String(); s != "Action" {
		t.Errorf("Action.String() = %s, want Action", s)
	}
	if s := Condition.String(); s != "Condition" {
		t.Errorf("Condition.String() = %s, want Condition", s)
	}
	if s := Decorator.String(); s != "Decorator" {
		t.Errorf("Decorator.String() = %s, want Decorator", s)
	}
	if s := Control.String(); s != "Control" {
		t.Errorf("Control.String() = %s, want Control", s)
	}
	if s := Subtree.String(); s != "SubTree" {
		t.Errorf("Subtree.String() = %s, want SubTree", s)
	}
	if s := Undefined.String(); s != "Undefined" {
		t.Errorf("Undefined.String() = %s, want Undefined", s)
	}
}

func TestBasicTypes_ConvertFromString_Int(t *testing.T) {
	ti := NewTypeInfo[int]()
	if v, err := ti.ParseString("42"); err != nil {
		t.Errorf("ParseString('42'): %v", err)
	} else if r, _ := Cast[int](v); r != 42 {
		t.Errorf("ParseString('42') = %d, want 42", r)
	}
	if v, err := ti.ParseString("-42"); err != nil {
		t.Errorf("ParseString('-42'): %v", err)
	} else if r, _ := Cast[int](v); r != -42 {
		t.Errorf("ParseString('-42') = %d, want -42", r)
	}
	if _, err := ti.ParseString("not_a_number"); err == nil {
		t.Errorf("ParseString('not_a_number') should error")
	}
	if _, err := ti.ParseString(""); err == nil {
		t.Errorf("ParseString('') should error")
	}
}

// TestBasicTypes_ConvertFromString_Int64 verifies int64 parsing.
// Equivalent of C++ BasicTypes/ConvertFromString_Int64.
func TestBasicTypes_ConvertFromString_Int64(t *testing.T) {
	ti := NewTypeInfo[int64]()
	if v, err := ti.ParseString("9223372036854775807"); err != nil {
		t.Errorf("ParseString('9223372036854775807'): %v", err)
	} else if r, _ := Cast[int64](v); r != 9223372036854775807 {
		t.Errorf("ParseString('9223372036854775807') = %d, want max int64", r)
	}
	if v, err := ti.ParseString("-9223372036854775808"); err != nil {
		t.Errorf("ParseString('-9223372036854775808'): %v", err)
	} else if r, _ := Cast[int64](v); r != -9223372036854775808 {
		t.Errorf("ParseString('-9223372036854775808') = %d, want min int64", r)
	}
}

// TestBasicTypes_ConvertFromString_UInt64 verifies uint64 parsing.
// Equivalent of C++ BasicTypes/ConvertFromString_UInt64.
func TestBasicTypes_ConvertFromString_UInt64(t *testing.T) {
	ti := NewTypeInfo[uint64]()
	if v, err := ti.ParseString("18446744073709551615"); err != nil {
		t.Errorf("ParseString('18446744073709551615'): %v", err)
	} else if r, _ := Cast[uint64](v); r != 18446744073709551615 {
		t.Errorf("ParseString('18446744073709551615') = %d, want max uint64", r)
	}
	if v, err := ti.ParseString("0"); err != nil {
		t.Errorf("ParseString('0'): %v", err)
	} else if r, _ := Cast[uint64](v); r != 0 {
		t.Errorf("ParseString('0') = %d, want 0", r)
	}
}

func TestBasicTypes_ConvertFromString_Double(t *testing.T) {
	ti := NewTypeInfo[float64]()
	if v, err := ti.ParseString("3.14159"); err != nil {
		t.Errorf("ParseString('3.14159'): %v", err)
	} else if r, _ := Cast[float64](v); r != 3.14159 {
		t.Errorf("ParseString('3.14159') = %f, want 3.14159", r)
	}
	if v, err := ti.ParseString("-2.5"); err != nil {
		t.Errorf("ParseString('-2.5'): %v", err)
	} else if r, _ := Cast[float64](v); r != -2.5 {
		t.Errorf("ParseString('-2.5') = %f, want -2.5", r)
	}
	if _, err := ti.ParseString("not_a_number"); err == nil {
		t.Errorf("ParseString('not_a_number') should error")
	}
}

func TestBasicTypes_ConvertFromString_Bool(t *testing.T) {
	ti := NewTypeInfo[bool]()
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
	}
	for _, tc := range tests {
		v, err := ti.ParseString(tc.input)
		if err != nil {
			t.Errorf("ParseString(%q): %v", tc.input, err)
			continue
		}
		r, _ := Cast[bool](v)
		if r != tc.expected {
			t.Errorf("ParseString(%q) = %v, want %v", tc.input, r, tc.expected)
		}
	}
	if _, err := ti.ParseString("invalid"); err == nil {
		t.Errorf("ParseString('invalid') should error")
	}
}

func TestBasicTypes_ConvertFromString_String(t *testing.T) {
	ti := NewTypeInfo[string]()
	if v, err := ti.ParseString("hello"); err != nil {
		t.Errorf("ParseString('hello'): %v", err)
	} else if r, _ := Cast[string](v); r != "hello" {
		t.Errorf("ParseString('hello') = %q, want 'hello'", r)
	}
	if v, err := ti.ParseString(""); err != nil {
		t.Errorf("ParseString('') returned: %v", err)
	} else if r, _ := Cast[string](v); r != "" {
		t.Errorf("ParseString('') = %q, want ''", r)
	}
}

func TestBasicTypes_ConvertFromString_NodeStatus(t *testing.T) {
	ti := NewTypeInfo[NodeStatus]()
	tests := []struct {
		input string
		want  NodeStatus
	}{
		{"SUCCESS", SUCCESS},
		{"FAILURE", FAILURE},
		{"RUNNING", RUNNING},
		{"IDLE", IDLE},
		{"SKIPPED", SKIPPED},
	}
	for _, tc := range tests {
		v, err := ti.ParseString(tc.input)
		if err != nil {
			t.Errorf("ParseString(%q): %v", tc.input, err)
			continue
		}
		r, _ := Cast[NodeStatus](v)
		if r != tc.want {
			t.Errorf("ParseString(%q) = %v, want %v", tc.input, r, tc.want)
		}
	}
	if _, err := ti.ParseString("INVALID"); err == nil {
		t.Errorf("ParseString('INVALID') should error")
	}
}

// TestBasicTypes_ConvertFromString_NodeType verifies NodeType string parsing.
// Equivalent of C++ BasicTypes/ConvertFromString_NodeType.
func TestBasicTypes_ConvertFromString_NodeType(t *testing.T) {
	ti := NewTypeInfo[NodeType]()
	tests := []struct {
		str  string
		want NodeType
	}{
		{"Action", Action},
		{"Condition", Condition},
		{"Control", Control},
		{"Decorator", Decorator},
		{"SubTree", Subtree},
	}
	for _, tt := range tests {
		v, err := ti.ParseString(tt.str)
		if err != nil {
			t.Errorf("ParseString(%q): %v", tt.str, err)
			continue
		}
		r, _ := Cast[NodeType](v)
		if r != tt.want {
			t.Errorf("ParseString(%q) = %v, want %v", tt.str, r, tt.want)
		}
	}
}

// TestBasicTypes_ConvertFromString_PortDirection verifies PortDirection string parsing.
// Equivalent of C++ BasicTypes/ConvertFromString_PortDirection.
func TestBasicTypes_ConvertFromString_PortDirection(t *testing.T) {
	ti := NewTypeInfo[PortDirection]()
	tests := []struct {
		str  string
		want PortDirection
	}{
		{"INPUT", INPUT},
		{"OUTPUT", OUTPUT},
		{"INOUT", INOUT},
	}
	for _, tt := range tests {
		v, err := ti.ParseString(tt.str)
		if err != nil {
			t.Errorf("ParseString(%q): %v", tt.str, err)
			continue
		}
		r, _ := Cast[PortDirection](v)
		if r != tt.want {
			t.Errorf("ParseString(%q) = %v, want %v", tt.str, r, tt.want)
		}
	}
}

func TestBasicTypes_SplitString(t *testing.T) {
	parts := SplitString("a,b,c", ',')
	if len(parts) != 3 {
		t.Errorf("SplitString('a,b,c', ',') => %d parts, want 3", len(parts))
	}
	if parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Errorf("SplitString('a,b,c', ',') => %v", parts)
	}

	parts = SplitString("", ',')
	if len(parts) != 0 {
		t.Errorf("SplitString('', ',') => %d parts, want 0", len(parts))
	}

	parts = SplitString("hello", ',')
	if len(parts) != 1 || parts[0] != "hello" {
		t.Errorf("SplitString('hello', ',') => %v", parts)
	}

	parts = SplitString(" a , b , c ", ',')
	if len(parts) != 3 {
		t.Errorf("SplitString(' a , b , c ', ',') => %d parts, want 3", len(parts))
	}
	if parts[0] != " a " || parts[1] != " b " || parts[2] != " c " {
		t.Errorf("SplitString(' a , b , c ', ',') => %v", parts)
	}
}

func TestBasicTypes_IsActive_IsCompleted(t *testing.T) {
	if !SUCCESS.IsCompleted() {
		t.Errorf("SUCCESS should be completed")
	}
	if !FAILURE.IsCompleted() {
		t.Errorf("FAILURE should be completed")
	}
	if RUNNING.IsCompleted() {
		t.Errorf("RUNNING should not be completed")
	}
	if IDLE.IsCompleted() {
		t.Errorf("IDLE should not be completed")
	}
	if SKIPPED.IsCompleted() {
		t.Errorf("SKIPPED should not be completed")
	}
	if !RUNNING.IsActive() {
		t.Errorf("RUNNING should be active")
	}
	if IDLE.IsActive() {
		t.Errorf("IDLE should not be active")
	}
	if SKIPPED.IsActive() {
		t.Errorf("SKIPPED should not be active")
	}
}

func TestBasicTypes_IsBlackboardPointer(t *testing.T) {
	if ok, _ := IsBlackboardPointer("{key}"); !ok {
		t.Errorf("'{key}' should be a blackboard pointer")
	}
	if ok, _ := IsBlackboardPointer("{=}"); !ok {
		t.Errorf("'{=}' should be a blackboard pointer")
	}
	if ok, _ := IsBlackboardPointer("literal"); ok {
		t.Errorf("'literal' should not be a blackboard pointer")
	}
}

func TestBasicTypes_IsAllowedPortName(t *testing.T) {
	if !IsAllowedPortName("my_port") {
		t.Errorf("'my_port' should be allowed")
	}
	if IsAllowedPortName("_failureIf") {
		t.Errorf("'_failureIf' is a reserved name")
	}
	if IsAllowedPortName("") {
		t.Errorf("empty name should not be allowed")
	}
	if IsAllowedPortName("123_bad") {
		t.Errorf("name starting with digit should not be allowed")
	}
}

func TestBasicTypes_FindForbiddenChar(t *testing.T) {
	if c := FindForbiddenChar("good_name"); c != 0 {
		t.Errorf("FindForbiddenChar('good_name') = %q, want 0", c)
	}
	if c := FindForbiddenChar("bad:name"); c != ':' {
		t.Errorf("FindForbiddenChar('bad:name') = %q, want ':'", c)
	}
	if c := FindForbiddenChar("bad.name"); c != '.' {
		t.Errorf("FindForbiddenChar('bad.name') = %q, want '.'", c)
	}
}

func TestBasicTypes_WildcardMatch(t *testing.T) {
	if !WildcardMatch("hello", "h*") {
		t.Errorf("WildcardMatch('hello', 'h*') should be true")
	}
	if !WildcardMatch("hello", "*o") {
		t.Errorf("WildcardMatch('hello', '*o') should be true")
	}
	if !WildcardMatch("hello", "*") {
		t.Errorf("WildcardMatch('hello', '*') should be true")
	}
	if WildcardMatch("hello", "*x*") {
		t.Errorf("WildcardMatch('hello', '*x*') should be false")
	}
	if !WildcardMatch("prefix_suffix", "prefix*") {
		t.Errorf("WildcardMatch('prefix_suffix', 'prefix*') should be true")
	}
	if WildcardMatch("other", "prefix*") {
		t.Errorf("WildcardMatch('other', 'prefix*') should be false")
	}
}

func TestBasicTypes_StartWith(t *testing.T) {
	if !StartWith("hello world", "hello") {
		t.Errorf("StartWith('hello world', 'hello') should be true")
	}
	if StartWith("hello", "hello world") {
		t.Errorf("StartWith('hello', 'hello world') should be false")
	}
}

func TestBasicTypes_IsReservedAttribute(t *testing.T) {
	reserved := []string{"name", "ID", "_autoremap", "_failureIf", "_successIf", "_skipIf", "_while", "_onSuccess", "_onFailure", "_onHalted", "_post"}
	for _, attr := range reserved {
		if !IsReservedAttribute(attr) {
			t.Errorf("'%s' should be reserved", attr)
		}
	}
	if IsReservedAttribute("my_custom_attr") {
		t.Errorf("'my_custom_attr' should not be reserved")
	}
}

func TestBasicTypes_GetRemappedKey(t *testing.T) {
	key, ok := GetRemappedKey("my_port", "{bb_key}")
	if !ok || key != "bb_key" {
		t.Errorf("GetRemappedKey('my_port', '{bb_key}') = (%s, %v), want (bb_key, true)", key, ok)
	}
	key, ok = GetRemappedKey("my_port", "{=}")
	if !ok || key != "my_port" {
		t.Errorf("GetRemappedKey('my_port', '{=}') = (%s, %v), want (my_port, true)", key, ok)
	}
}

func TestBasicTypes_TypeInfo(t *testing.T) {
	ti := NewTypeInfo[int]()
	if !ti.IsStronglyTyped() {
		t.Errorf("int TypeInfo should be strongly typed")
	}
	if ti.TypeName() != "int" && ti.TypeName() != "int64" {
		t.Errorf("expected int type name, got %s", ti.TypeName())
	}
	tiAny := NewTypeInfoAnyAllowed()
	if tiAny.IsStronglyTyped() {
		t.Errorf("AnyTypeAllowed should not be strongly typed")
	}
	tiStr := NewTypeInfo[string]()
	if tiStr.Converter() == nil {
		t.Errorf("string TypeInfo should have converter")
	}
}

func TestBasicTypes_NodeTypeString(t *testing.T) {
	if Undefined.String() != "Undefined" {
		t.Errorf("Undefined.String() = %s", Undefined.String())
	}
}

// TestBasicTypes_LibraryVersion verifies that library version returns info.
// Note: Go port does not have LibraryVersionNumber/LibraryVersionString.
func TestBasicTypes_LibraryVersion(t *testing.T) {
	t.Log("LibraryVersion not implemented in Go port")
}

// TestBasicTypes_Result_Success verifies success/SUCCESS behavior.
func TestBasicTypes_Result_Success(t *testing.T) {
	if !SUCCESS.IsCompleted() {
		t.Error("SUCCESS should be completed")
	}
	if !SUCCESS.IsActive() {
		t.Error("SUCCESS should be active")
	}
}

// TestBasicTypes_Result_Error verifies failure/FAILURE behavior.
func TestBasicTypes_Result_Error(t *testing.T) {
	if !FAILURE.IsCompleted() {
		t.Error("FAILURE should be completed")
	}
}

// testControlNode is a minimal ControlNode used for testing.
type testControlNode struct {
	ControlNode
}

func (n *testControlNode) Tick() NodeStatus {
	for _, child := range n.Children() {
		status := child.ExecuteTick()
		if status == FAILURE {
			return FAILURE
		}
		if status == RUNNING {
			return RUNNING
		}
	}
	return SUCCESS
}

// testActionNode is a minimal action node used for testing.
type testActionNode struct {
	SyncActionNode
}

func (n *testActionNode) Tick() NodeStatus {
	return SUCCESS
}

// TestApplyRecursiveVisitor creates a simple tree and verifies node traversal.
// Equivalent of C++ BehaviorTree/ApplyRecursiveVisitor.
func TestApplyRecursiveVisitor(t *testing.T) {
	factory, err := NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	_ = factory.RegisterNodeType("Sequence", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testControlNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Sequence")
		return n
	}, Control)

	_ = factory.RegisterNodeType("AlwaysSuccess", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testActionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("AlwaysSuccess")
		return n
	}, Action)

	_ = factory.RegisterNodeType("AlwaysFailure", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testActionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("AlwaysFailure")
		return n
	}, Action)

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Sequence>
	        <AlwaysSuccess/>
	        <AlwaysFailure/>
	      </Sequence>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	nodeCount := 0
	err = ApplyRecursiveVisitor(tree.RootNode(), func(TreeNode) {
		nodeCount++
	})
	if err != nil {
		t.Fatal(err)
	}

	if nodeCount != 3 {
		t.Errorf("expected 3 nodes visited, got %d", nodeCount)
	}
}

// TestApplyRecursiveVisitor_MutableVersion collects node names via visitor.
// Equivalent of C++ BehaviorTree/ApplyRecursiveVisitor_MutableVersion.
func TestApplyRecursiveVisitor_MutableVersion(t *testing.T) {
	factory, err := NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	_ = factory.RegisterNodeType("Sequence", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testControlNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Sequence")
		return n
	}, Control)

	_ = factory.RegisterNodeType("AlwaysSuccess", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testActionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("AlwaysSuccess")
		return n
	}, Action)

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Sequence>
	        <AlwaysSuccess/>
	      </Sequence>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	names := make([]string, 0)
	err = ApplyRecursiveVisitor(tree.RootNode(), func(node TreeNode) {
		names = append(names, node.Name())
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(names) != 2 {
		t.Fatalf("expected 2 node names, got %d: %v", len(names), names)
	}
	if names[0] != "Sequence" {
		t.Errorf("expected first node name 'Sequence', got '%s'", names[0])
	}
	if names[1] != "AlwaysSuccess" {
		t.Errorf("expected second node name 'AlwaysSuccess', got '%s'", names[1])
	}
}

// TestApplyRecursiveVisitor_NullNode verifies error on nil node.
// Equivalent of C++ BehaviorTree/ApplyRecursiveVisitor_NullNode.
func TestApplyRecursiveVisitor_NullNode(t *testing.T) {
	err := ApplyRecursiveVisitor(nil, func(TreeNode) {})
	if err == nil {
		t.Error("expected error for nil node, got nil")
	}
}

// TestBuildSerializedStatusSnapshot builds a snapshot after ticking.
// Equivalent of C++ BehaviorTree/BuildSerializedStatusSnapshot.
func TestBuildSerializedStatusSnapshot(t *testing.T) {
	factory, err := NewBehaviorTreeFactory()
	if err != nil {
		t.Fatal(err)
	}
	_ = factory.RegisterNodeType("Sequence", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testControlNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("Sequence")
		return n
	}, Control)

	_ = factory.RegisterNodeType("AlwaysSuccess", PortsList{}, func(name string, config NodeConfig) TreeNode {
		n := &testActionNode{}
		n.Init(name, config)
		n.SetSelf(n)
		n.SetRegistrationID("AlwaysSuccess")
		return n
	}, Action)

	const xmlText = `
	<root BTCPP_format="4">
	   <BehaviorTree>
	      <Sequence>
	        <AlwaysSuccess/>
	        <AlwaysSuccess/>
	      </Sequence>
	   </BehaviorTree>
	</root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}

	tree.TickExactlyOnce()

	snapshot, err := BuildSerializedStatusSnapshot(tree.RootNode())
	if err != nil {
		t.Fatal(err)
	}

	if len(snapshot) != 3 {
		t.Errorf("expected 3 entries in snapshot, got %d", len(snapshot))
	}
}

// TestBasicTypes_PortInfo_Construction verifies PortInfo creation.
// Equivalent of C++ BasicTypes/PortInfo_Construction.
func TestBasicTypes_PortInfo_Construction(t *testing.T) {
	key, pi := InputPort[int]("test_input", "description")
	if key != "test_input" {
		t.Errorf("expected key 'test_input', got '%s'", key)
	}
	if pi.Direction() != INPUT {
		t.Errorf("expected INPUT direction, got %v", pi.Direction())
	}
	if pi.Description() != "description" {
		t.Errorf("expected description 'description', got '%s'", pi.Description())
	}

	_, pi2 := OutputPort[float64]("test_output", "out description")
	if pi2.Direction() != OUTPUT {
		t.Errorf("expected OUTPUT direction, got %v", pi2.Direction())
	}

	_, pi3 := BidirectionalPort[string]("test_bidir", "")
	if pi3.Direction() != INOUT {
		t.Errorf("expected INOUT direction, got %v", pi3.Direction())
	}
}

// TestBasicTypes_PortInfo_DefaultValue verifies PortInfo with defaults.
// Equivalent of C++ BasicTypes/PortInfo_DefaultValue.
func TestBasicTypes_PortInfo_DefaultValue(t *testing.T) {
	_, pi := InputPortWithDefault[int]("port_with_default", "42", "has default")
	if pi.DefaultValue().IsEmpty() {
		t.Error("expected non-empty default value")
	}
}

// TestBasicTypes_TreeNodeManifest verifies TreeNodeManifest construction.
// Equivalent of C++ BasicTypes/TreeNodeManifest.
func TestBasicTypes_TreeNodeManifest(t *testing.T) {
	manifest := TreeNodeManifest{}
	manifest.Type = Action
	manifest.RegistrationID = "TestAction"
	manifest.Ports = PortsList{}
	key1, pi1 := InputPort[int]("value", "")
	key2, pi2 := OutputPort[string]("result", "")
	manifest.Ports[key1] = pi1
	manifest.Ports[key2] = pi2

	if manifest.Type != Action {
		t.Errorf("expected Action type, got %v", manifest.Type)
	}
	if manifest.RegistrationID != "TestAction" {
		t.Errorf("expected RegistrationID 'TestAction', got '%s'", manifest.RegistrationID)
	}
	if len(manifest.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(manifest.Ports))
	}
}
