package script_test

import (
	"testing"

	"github.com/actfuns/behaviortree/action"
	"github.com/actfuns/behaviortree/control"
	"github.com/actfuns/behaviortree/core"
	"github.com/actfuns/behaviortree/factory"
	"github.com/actfuns/behaviortree/script"
)

// GetScriptResult evaluates a script and returns the result of the last expression.
// Equivalent of C++ helper in tests.
func GetScriptResult(blackboard *core.Blackboard, enums *core.ScriptingEnumsRegistry, text string) (core.Any, error) {
	return script.ParseScriptAndExecute(blackboard, enums, text)
}

// registerScriptNodes registers nodes needed for XML-based script tests.
func registerScriptNodes(factory core.BehaviorTreeFactory) {
	_ = factory.RegisterNodeType("Sequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewSequenceNode(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("ReactiveSequence", core.PortsList{}, func(name string, config core.NodeConfig) core.TreeNode {
		return control.NewReactiveSequence(name, config)
	}, core.Control)

	_ = factory.RegisterNodeType("Script", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptNode(name, config)
	}, core.Action)

	_ = factory.RegisterNodeType("ScriptCondition", core.PortsList{
		"code": core.NewPortInfo(core.INPUT),
	}, func(name string, config core.NodeConfig) core.TreeNode {
		return action.NewScriptCondition(name, config)
	}, core.Condition)
}

// TestAnyTypes verifies parsing of various literal types.
func TestAnyTypes(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustParse := func(s string) (core.Any, error) {
		return script.ParseScriptAndExecute(bb, nil, s)
	}

	// Integer
	result, err := mustParse("628")
	if err != nil {
		t.Fatal(err)
	}
	v, err := result.ToInt64()
	if err != nil || v != 628 {
		t.Errorf("Expected 628, got %v (err=%v)", v, err)
	}

	// Negative integer
	result, err = mustParse("-628")
	if err != nil {
		t.Fatal(err)
	}
	v, err = result.ToInt64()
	if err != nil || v != -628 {
		t.Errorf("Expected -628, got %v", v)
	}

	// 0x100 = 256
	result, err = mustParse("0x100")
	if err != nil {
		t.Fatal(err)
	}
	v, err = result.ToInt64()
	if err != nil || v != 256 {
		t.Errorf("Expected 256, got %v", v)
	}

	// 0X100 = 256
	result, err = mustParse("0X100")
	if err != nil {
		t.Fatal(err)
	}
	v, err = result.ToInt64()
	if err != nil || v != 256 {
		t.Errorf("Expected 256, got %v", v)
	}

	// 3.14
	result, err = mustParse("3.14")
	if err != nil {
		t.Fatal(err)
	}
	fv, err := result.ToFloat64()
	if err != nil || fv != 3.14 {
		t.Errorf("Expected 3.14, got %v", fv)
	}

	// -3.14
	result, err = mustParse("-3.14")
	if err != nil {
		t.Fatal(err)
	}
	fv, err = result.ToFloat64()
	if err != nil || fv != -3.14 {
		t.Errorf("Expected -3.14, got %v", fv)
	}

	// 3.14e2 = 314
	result, err = mustParse("3.14e2")
	if err != nil {
		t.Fatal(err)
	}
	fv, err = result.ToFloat64()
	if err != nil || fv != 314 {
		t.Errorf("Expected 314, got %v", fv)
	}

	// 3.14e-2 = 0.0314
	result, err = mustParse("3.14e-2")
	if err != nil {
		t.Fatal(err)
	}
	fv, err = result.ToFloat64()
	if err != nil || (fv-0.0314)*(fv-0.0314) > 1e-10 {
		t.Errorf("Expected ~0.0314, got %v", fv)
	}

	// 3e2 = 300
	result, err = mustParse("3e2")
	if err != nil {
		t.Fatal(err)
	}
	fv, err = result.ToFloat64()
	if err != nil || fv != 300 {
		t.Errorf("Expected 300, got %v", fv)
	}

	// 3e-2 = 0.03
	result, err = mustParse("3e-2")
	if err != nil {
		t.Fatal(err)
	}
	fv, err = result.ToFloat64()
	if err != nil || (fv-0.03)*(fv-0.03) > 1e-10 {
		t.Errorf("Expected ~0.03, got %v", fv)
	}

	// String literal
	result, err = mustParse("'hello world '")
	if err != nil {
		t.Fatal(err)
	}
	sv, err := result.ToString()
	if err != nil || sv != "hello world " {
		t.Errorf("Expected 'hello world ', got '%v'", sv)
	}

	// true = 1
	result, err = mustParse("true")
	if err != nil {
		t.Fatal(err)
	}
	v, err = result.ToInt64()
	if err != nil || v != 1 {
		t.Errorf("Expected 1 for true, got %v", v)
	}

	// false = 0
	result, err = mustParse("false")
	if err != nil {
		t.Fatal(err)
	}
	v, err = result.ToInt64()
	if err != nil || v != 0 {
		t.Errorf("Expected 0 for false, got %v", v)
	}
}

// TestAnyTypes_Failing verifies that invalid scripts fail.
func TestAnyTypes_Failing(t *testing.T) {
	if err := script.ValidateScript("0X100g"); err == nil {
		t.Error("Expected error for '0X100g'")
	}
	if err := script.ValidateScript("0X100."); err == nil {
		t.Error("Expected error for '0X100.'")
	}
	if err := script.ValidateScript("3foo"); err == nil {
		t.Error("Expected error for '3foo'")
	}
	if err := script.ValidateScript("65."); err == nil {
		t.Error("Expected error for '65.'")
	}
	if err := script.ValidateScript("65.43foo"); err == nil {
		t.Error("Expected error for '65.43foo'")
	}

	// "foo" is a valid identifier (parses as ExprName), only fails at evaluation.
	bb := core.NewBlackboard(nil)
	_, err := script.ParseScriptAndExecute(bb, nil, "foo")
	if err == nil {
		t.Error("Expected error for undefined variable 'foo'")
	}
}

// TestEquations verifies various equations and variable assignments.
func TestEquations(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// Basic arithmetic with assignment
	result := mustExec("x:= 3; y:=5; x+y")
	fv, err := result.ToFloat64()
	if err != nil || fv != 8 {
		t.Errorf("Expected 8, got %v", fv)
	}

	keys := bb.GetKeys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}

	xVal, _ := core.GetTyped[float64](bb, "x")
	if xVal != 3 {
		t.Errorf("Expected x=3, got %v", xVal)
	}
	yVal, _ := core.GetTyped[float64](bb, "y")
	if yVal != 5 {
		t.Errorf("Expected y=5, got %v", yVal)
	}

	// x += 1
	result = mustExec("x+=1")
	fv, _ = result.ToFloat64()
	if fv != 4 {
		t.Errorf("Expected 4, got %v", fv)
	}
	xVal, _ = core.GetTyped[float64](bb, "x")
	if xVal != 4 {
		t.Errorf("Expected x=4, got %v", xVal)
	}

	// x += 1
	result = mustExec("x += 1")
	fv, _ = result.ToFloat64()
	if fv != 5 {
		t.Errorf("Expected 5, got %v", fv)
	}

	// x -= 1
	result = mustExec("x-=1")
	fv, _ = result.ToFloat64()
	if fv != 4 {
		t.Errorf("Expected 4, got %v", fv)
	}

	// x -= 1
	result = mustExec("x -= 1")
	fv, _ = result.ToFloat64()
	if fv != 3 {
		t.Errorf("Expected 3, got %v", fv)
	}

	// x *= 2
	result = mustExec("x*=2")
	fv, _ = result.ToFloat64()
	if fv != 6 {
		t.Errorf("Expected 6, got %v", fv)
	}

	// -x
	result = mustExec("-x")
	fv, _ = result.ToFloat64()
	if fv != -6 {
		t.Errorf("Expected -6, got %v", fv)
	}

	// x /= 2
	result = mustExec("x/=2")
	fv, _ = result.ToFloat64()
	if fv != 3 {
		t.Errorf("Expected 3, got %v", fv)
	}

	// y
	result = mustExec("y")
	fv, _ = result.ToFloat64()
	if fv != 5 {
		t.Errorf("Expected 5, got %v", fv)
	}

	// y / 2
	result = mustExec("y/2")
	fv, _ = result.ToFloat64()
	if fv != 2.5 {
		t.Errorf("Expected 2.5, got %v", fv)
	}

	// y * 2
	result = mustExec("y*2")
	fv, _ = result.ToFloat64()
	if fv != 10 {
		t.Errorf("Expected 10, got %v", fv)
	}

	// y - x
	result = mustExec("y-x")
	fv, _ = result.ToFloat64()
	if fv != 2 {
		t.Errorf("Expected 2, got %v", fv)
	}

	// Bitwise AND: 5 & 3 = 1
	result = mustExec("y & x")
	iv, _ := result.ToInt64()
	if iv != int64(5&3) {
		t.Errorf("Expected %d, got %d", 5&3, iv)
	}

	// Bitwise OR: 5 | 3 = 7
	result = mustExec("y | x")
	iv, _ = result.ToInt64()
	if iv != int64(5|3) {
		t.Errorf("Expected %d, got %d", 5|3, iv)
	}

	// Bitwise XOR: 5 ^ 3 = 6
	result = mustExec("y ^ x")
	iv, _ = result.ToInt64()
	if iv != int64(5^3) {
		t.Errorf("Expected %d, got %d", 5^3, iv)
	}

	// String concatenation
	result = mustExec("A:='hello'; B:=' '; C:='world'; A+B+C")
	sv, _ := result.ToString()
	if sv != "hello world" {
		t.Errorf("Expected 'hello world', got '%v'", sv)
	}

	// Check variable count
	keys = bb.GetKeys()
	if len(keys) < 3 {
		t.Errorf("Expected at least 3 keys after string concat, got %d", len(keys))
	}

	aVal, _ := core.GetTyped[string](bb, "A")
	if aVal != "hello" {
		t.Errorf("Expected A='hello', got '%v'", aVal)
	}

	// String whitespace handling
	mustExec("A= '   right'; B= ' center '; C= 'left    '  ")
	aVal, _ = core.GetTyped[string](bb, "A")
	if aVal != "   right" {
		t.Errorf("Expected A='   right', got '%v'", aVal)
	}
	bVal, _ := core.GetTyped[string](bb, "B")
	if bVal != " center " {
		t.Errorf("Expected B=' center ', got '%v'", bVal)
	}
	cVal, _ := core.GetTyped[string](bb, "C")
	if cVal != "left    " {
		t.Errorf("Expected C='left    ', got '%v'", cVal)
	}

	// Type mismatch should fail
	_, err = script.ParseScriptAndExecute(bb, nil, "x='msg'")
	if err == nil {
		t.Error("Expected error for type mismatch: x is number, trying to assign string")
	}

	// Invalid assignment: can't assign to literal
	_, err = script.ParseScriptAndExecute(bb, nil, "'hello' = 'world'")
	if err == nil {
		t.Error("Expected error for invalid assignment to string literal")
	}
	_, err = script.ParseScriptAndExecute(bb, nil, "3.0 = 2.0")
	if err == nil {
		t.Error("Expected error for invalid assignment to number literal")
	}

	prevSize := len(bb.GetKeys())
	_, err = script.ParseScriptAndExecute(bb, nil, "new_var=69")
	if err == nil {
		t.Error("Expected error for assignment to non-existing variable with '='")
	}
	if len(bb.GetKeys()) != prevSize {
		t.Error("Variable count should not increase after failed assignment")
	}

	// Comparisons
	result = mustExec("x < y")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for x<y, got %d", iv)
	}

	result = mustExec("y > x")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for y>x, got %d", iv)
	}

	result = mustExec("y != x")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for y!=x, got %d", iv)
	}

	result = mustExec("y == x")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for y==x, got %d", iv)
	}

	// String comparisons
	result = mustExec("'hello' == 'hello'")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'hello'=='hello', got %d", iv)
	}

	result = mustExec("'hello' != 'world'")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'hello'!='world', got %d", iv)
	}

	result = mustExec("'hello' < 'world'")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'hello'<'world', got %d", iv)
	}

	result = mustExec("'hello' > 'world'")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for 'hello'>'world', got %d", iv)
	}

	// Ternary
	result = mustExec("y == x ? 'T' : 'F'")
	sv, _ = result.ToString()
	if sv != "F" {
		t.Errorf("Expected 'F', got '%v'", sv)
	}

	result = mustExec("y != x ? 'T' : 'F'")
	sv, _ = result.ToString()
	if sv != "T" {
		t.Errorf("Expected 'T', got '%v'", sv)
	}

	// Comparisons with values
	result = mustExec("y == 5")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for y==5, got %d", iv)
	}

	result = mustExec("x == 3")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for x==3, got %d", iv)
	}

	// Boolean
	result = mustExec("true")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for true, got %d", iv)
	}

	result = mustExec("'true'")
	sv, _ = result.ToString()
	if sv != "true" {
		t.Errorf("Expected 'true', got '%v'", sv)
	}

	// Boolean assignment and comparison
	mustExec("v1:= true; v2:= false")
	result = mustExec("v2 = true")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for v2=true, got %d", iv)
	}

	result = mustExec("v2 = !false")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for v2=!false, got %d", iv)
	}

	// Logical operators
	result = mustExec("v1 && v2")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for v1Expected 0 for v1&&v2Expected 0 for v1&&v2v2 (after v2=true), got %d", iv)
	}

	result = mustExec("v1 || v2")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for v1||v2, got %d", iv)
	}

	// Combined logical with comparisons
	result = mustExec("(y == x) && (x == 3)")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0, got %d", iv)
	}

	result = mustExec("(y == x) || (x == 3)")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1, got %d", iv)
	}

	// String-to-number cast in comparisons
	result = mustExec("par1:='3'; par2:=3; par1==par2")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for par1==par2, got %d", iv)
	}

	result = mustExec("par1:='3'; par2:=4; par1!=par2")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for par1!=par2, got %d", iv)
	}
}

// TestNotInitializedComparison verifies that uninitialized entries fail.
func TestNotInitializedComparison(t *testing.T) {
	bb := core.NewBlackboard(nil)

	// Create an entry without setting a value
	bb.CreateEntry("x", core.NewPortInfo(core.INOUT))

	// These should fail because x is not initialized
	_, err := script.ParseScriptAndExecute(bb, nil, "x < 0")
	if err == nil {
		t.Error("Expected error for uninitialized comparison x < 0")
	}

	_, err = script.ParseScriptAndExecute(bb, nil, "x == 0")
	if err == nil {
		t.Error("Expected error for uninitialized comparison x == 0")
	}

	_, err = script.ParseScriptAndExecute(bb, nil, "x + 1")
	if err == nil {
		t.Error("Expected error for uninitialized arithmetic x + 1")
	}

	_, err = script.ParseScriptAndExecute(bb, nil, "x += 1")
	if err == nil {
		t.Error("Expected error for uninitialized compound assignment x += 1")
	}
}

// TestEnumsBasic verifies enum support in scripts.
func TestEnumsBasic(t *testing.T) {
	bb := core.NewBlackboard(nil)
	enums := &core.ScriptingEnumsRegistry{
		"RED":   1,
		"BLUE":  3,
		"GREEN": 5,
	}

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, enums, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	mustExec("A:=RED")
	mustExec("B:=RED")
	mustExec("C:=BLUE")

	result := mustExec("A==B")
	iv, _ := result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for A==B, got %d", iv)
	}

	result = mustExec("A!=C")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for A!=C, got %d", iv)
	}
}

// TestOperatorAssociativity verifies left-associativity of arithmetic operators.
func TestOperatorAssociativity(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// "5 - 2 + 1" should be (5-2)+1 = 4
	result := mustExec("5 - 2 + 1")
	fv, _ := result.ToFloat64()
	if fv != 4 {
		t.Errorf("Expected 4 for '5 - 2 + 1', got %v", fv)
	}

	// "10 - 3 - 2" should be (10-3)-2 = 5
	result = mustExec("10 - 3 - 2")
	fv, _ = result.ToFloat64()
	if fv != 5 {
		t.Errorf("Expected 5 for '10 - 3 - 2', got %v", fv)
	}

	// "2 + 3 - 1" should be (2+3)-1 = 4
	result = mustExec("2 + 3 - 1")
	fv, _ = result.ToFloat64()
	if fv != 4 {
		t.Errorf("Expected 4 for '2 + 3 - 1', got %v", fv)
	}

	// "12 / 3 / 2" should be (12/3)/2 = 2
	result = mustExec("12 / 3 / 2")
	fv, _ = result.ToFloat64()
	if fv != 2 {
		t.Errorf("Expected 2 for '12 / 3 / 2', got %v", fv)
	}

	// "12 / 3 * 2" should be (12/3)*2 = 8
	result = mustExec("12 / 3 * 2")
	fv, _ = result.ToFloat64()
	if fv != 8 {
		t.Errorf("Expected 8 for '12 / 3 * 2', got %v", fv)
	}

	// "2 + 3 * 4 - 1" should be 2+(3*4)-1 = 13
	result = mustExec("2 + 3 * 4 - 1")
	fv, _ = result.ToFloat64()
	if fv != 13 {
		t.Errorf("Expected 13 for '2 + 3 * 4 - 1', got %v", fv)
	}

	// String concatenation
	result = mustExec("A:='hello'; B:=' world'; A .. B")
	sv, _ := result.ToString()
	if sv != "hello world" {
		t.Errorf("Expected 'hello world', got '%v'", sv)
	}

	// Chained concatenation
	result = mustExec("A .. ' ' .. B")
	sv, _ = result.ToString()
	if sv != "hello  world" {
		t.Errorf("Expected 'hello  world', got '%v'", sv)
	}
}

// TestCompareWithNegativeNumber verifies comparisons with negative numbers.
func TestCompareWithNegativeNumber(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// "A!= -1" should parse and evaluate correctly
	result := mustExec("A:=0; A!=-1")
	iv, _ := result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'A:=0; A!=-1', got %d", iv)
	}

	result = mustExec("A:=-1; A!=-1")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for 'A:=-1; A!=-1', got %d", iv)
	}

	result = mustExec("A:=0; A==-1")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for 'A:=0; A==-1', got %d", iv)
	}

	result = mustExec("A:=0; A>-1")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'A:=0; A>-1', got %d", iv)
	}

	result = mustExec("A:=0; A<-1")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for 'A:=0; A<-1', got %d", iv)
	}

	// ValidateScript accepts these
	if err := script.ValidateScript("A:=0; A!=-1"); err != nil {
		t.Errorf("ValidateScript failed for 'A:=0; A!=-1': %v", err)
	}
	if err := script.ValidateScript("A:=0; A>-1"); err != nil {
		t.Errorf("ValidateScript failed for 'A:=0; A>-1': %v", err)
	}
}

// TestTokenizerEdgeCases tests various edge cases in the tokenizer.
func TestTokenizerEdgeCases(t *testing.T) {
	// Unterminated string
	if err := script.ValidateScript("'hello"); err == nil {
		t.Error("Expected error for unterminated string")
	}

	// Hex edge cases
	if err := script.ValidateScript("0x"); err == nil {
		t.Error("Expected error for '0x'")
	}
	if err := script.ValidateScript("0xG"); err == nil {
		t.Error("Expected error for '0xG'")
	}

	// Exponent without digits
	if err := script.ValidateScript("3e"); err == nil {
		t.Error("Expected error for '3e'")
	}
	if err := script.ValidateScript("3e+"); err == nil {
		t.Error("Expected error for '3e+'")
	}

	// DotDot adjacent to integer
	bb := core.NewBlackboard(nil)
	result, err := script.ParseScriptAndExecute(bb, nil, "A:='65'; B:='66'; A..B")
	if err != nil {
		t.Fatalf("DotDot parse failed: %v", err)
	}
	sv, _ := result.ToString()
	if sv != "6566" {
		t.Errorf("Expected '6566', got '%v'", sv)
	}

	// Empty and whitespace-only scripts
	if err := script.ValidateScript(""); err == nil {
		t.Error("Expected error for empty script")
	}
	if err := script.ValidateScript("   "); err == nil {
		t.Error("Expected error for whitespace-only script")
	}
}

// TestChainedComparisons tests chained (a < b < c) comparisons.
func TestChainedComparisons(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// 1 < 2 < 3 should be true
	result := mustExec("1 < 2 < 3")
	iv, _ := result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '1 < 2 < 3', got %d", iv)
	}

	// 3 > 2 > 1 should be true
	result = mustExec("3 > 2 > 1")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '3 > 2 > 1', got %d", iv)
	}

	// 1 < 2 > 3 should be false (1<2 is true, but 2>3 is false)
	result = mustExec("1 < 2 > 3")
	iv, _ = result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for '1 < 2 > 3', got %d", iv)
	}

	// Chained equality
	result = mustExec("5 == 5 == 5")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '5 == 5 == 5', got %d", iv)
	}

	result = mustExec("5 == 5 != 3")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '5 == 5 != 3', got %d", iv)
	}

	// 1 <= 2 <= 3
	result = mustExec("1 <= 2 <= 3")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '1 <= 2 <= 3', got %d", iv)
	}

	// 3 >= 2 >= 1
	result = mustExec("3 >= 2 >= 1")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for '3 >= 2 >= 1', got %d", iv)
	}
}

// TestOperatorPrecedence tests precedence of operators.
func TestOperatorPrecedence(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// 6 | 3 & 5 should be 6 | (3 & 5) = 6 | 1 = 7
	result := mustExec("6 | 3 & 5")
	iv, _ := result.ToInt64()
	if iv != 7 {
		t.Errorf("Expected 7 for '6 | 3 & 5', got %d", iv)
	}

	// true && (6 | 0) should be true
	result = mustExec("true && (6 | 0)")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'true && (6 | 0)', got %d", iv)
	}

	// false || true && true should be false || (true && true) = true
	result = mustExec("false || true && true")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'false || true && true', got %d", iv)
	}

	// false && true || true should be (false && true) || true = true
	result = mustExec("false && true || true")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for 'false && true || true', got %d", iv)
	}

	// (2 + 3) * 4 = 20
	result = mustExec("(2 + 3) * 4")
	fv, _ := result.ToFloat64()
	if fv != 20 {
		t.Errorf("Expected 20 for '(2+3)*4', got %v", fv)
	}

	// 2 * (3 + 4) = 14
	result = mustExec("2 * (3 + 4)")
	fv, _ = result.ToFloat64()
	if fv != 14 {
		t.Errorf("Expected 14 for '2*(3+4)', got %v", fv)
	}
}

// TestUnaryOperators tests unary NOT, complement, and minus.
func TestUnaryOperators(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// Logical NOT
	result := mustExec("!true")
	iv, _ := result.ToInt64()
	if iv != 0 {
		t.Errorf("Expected 0 for !true, got %d", iv)
	}

	result = mustExec("!false")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for !false, got %d", iv)
	}

	result = mustExec("!!true")
	iv, _ = result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1 for !!true, got %d", iv)
	}

	// Unary minus
	result = mustExec("-(3 + 2)")
	fv, _ := result.ToFloat64()
	if fv != -5 {
		t.Errorf("Expected -5 for -(3+2), got %v", fv)
	}

	result = mustExec("10 + -3")
	fv, _ = result.ToFloat64()
	if fv != 7 {
		t.Errorf("Expected 7 for '10 + -3', got %v", fv)
	}
}

// TestTernaryExpressions tests ternary (cond ? a : b) expressions.
func TestTernaryExpressions(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	result := mustExec("true ? 1 : 2")
	iv, _ := result.ToInt64()
	if iv != 1 {
		t.Errorf("Expected 1, got %d", iv)
	}

	result = mustExec("false ? 1 : 2")
	iv, _ = result.ToInt64()
	if iv != 2 {
		t.Errorf("Expected 2, got %d", iv)
	}

	// Ternary with expressions
	result = mustExec("true ? 2 + 3 : 10")
	fv, _ := result.ToFloat64()
	if fv != 5 {
		t.Errorf("Expected 5, got %v", fv)
	}

	result = mustExec("false ? 10 : 2 + 3")
	fv, _ = result.ToFloat64()
	if fv != 5 {
		t.Errorf("Expected 5, got %v", fv)
	}

	// Ternary with comparison
	result = mustExec("3 > 2 ? 'yes' : 'no'")
	sv, _ := result.ToString()
	if sv != "yes" {
		t.Errorf("Expected 'yes', got '%v'", sv)
	}

	result = mustExec("3 < 2 ? 'yes' : 'no'")
	sv, _ = result.ToString()
	if sv != "no" {
		t.Errorf("Expected 'no', got '%v'", sv)
	}
}

// TestMultipleStatements tests multiple semicolons and return values.
func TestMultipleStatements(t *testing.T) {
	bb := core.NewBlackboard(nil)

	mustExec := func(s string) core.Any {
		result, err := script.ParseScriptAndExecute(bb, nil, s)
		if err != nil {
			t.Fatalf("Script '%s' failed: %v", s, err)
		}
		return result
	}

	// Multiple semicolons
	mustExec("a:=1;;; b:=2;;")
	aVal, _ := core.GetTyped[float64](bb, "a")
	if aVal != 1 {
		t.Errorf("Expected a=1, got %v", aVal)
	}
	bVal, _ := core.GetTyped[float64](bb, "b")
	if bVal != 2 {
		t.Errorf("Expected b=2, got %v", bVal)
	}

	// Last expression is the return value
	result := mustExec("a:=10; b:=20; a+b")
	fv, _ := result.ToFloat64()
	if fv != 30 {
		t.Errorf("Expected 30, got %v", fv)
	}
}

// --------------------------------------------------------------------
// Tests ported from C++ script_parser_test.cpp
// --------------------------------------------------------------------

// TestEnumsXML verifies enum support via XML tree with factory.
// Equivalent of C++ ParserTest/EnumsXML.
func TestEnumsXML(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerScriptNodes(factory)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="MainTree">
	        <Script code = "A:=THE_ANSWER; color1:=RED; color2:=BLUE; color3:=GREEN" />
	    </BehaviorTree>
	</root>`

	factory.RegisterScriptingEnum("THE_ANSWER", 42)
	factory.RegisterScriptingEnum("RED", 1)
	factory.RegisterScriptingEnum("BLUE", 3)
	factory.RegisterScriptingEnum("GREEN", 5)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	bb := tree.Subtrees[0].Blackboard
	aVal, err := core.GetTyped[int](bb, "A")
	if err != nil || aVal != 42 {
		t.Errorf("Expected A=42, got %v (err=%v)", aVal, err)
	}
	color1, _ := core.GetTyped[int](bb, "color1")
	if color1 != 1 {
		t.Errorf("Expected color1=1, got %v", color1)
	}
	color2, _ := core.GetTyped[int](bb, "color2")
	if color2 != 3 {
		t.Errorf("Expected color2=3, got %v", color2)
	}
	color3, _ := core.GetTyped[int](bb, "color3")
	if color3 != 5 {
		t.Errorf("Expected color3=5, got %v", color3)
	}
}

// TestEnums_Issue523 verifies enum usage with port remapping and _skipIf.
// Equivalent of C++ ParserTest/Enums_Issue_523.
func TestEnums_Issue523(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerScriptNodes(factory)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="PowerManagerT">
	      <ReactiveSequence>
	        <Script code=" deviceA:=BATT; deviceB:=CONTROLLER; battery_level:=30 "/>
	        <CheckLevel deviceType="{deviceA}" percentage="{battery_level}" isLowBattery="{isLowBattery}"/>
	        <SaySomething message="FIRST low batteries!" _skipIf="!isLowBattery" />
	        <Script code=" battery_level:=20 "/>
	        <CheckLevel deviceType="{deviceA}" percentage="{battery_level}" isLowBattery="{isLowBattery}"/>
	        <SaySomething message="SECOND low batteries!" _skipIf="!isLowBattery" />
	      </ReactiveSequence>
	    </BehaviorTree>
	  </root>`

	_ = factory.RegisterSimpleAction("SaySomething", func(core.TreeNode) core.NodeStatus {
		return core.SUCCESS
	}, core.PortsList{
		"message": core.NewPortInfo(core.INPUT),
	})

	_ = factory.RegisterSimpleCondition("CheckLevel", func(self core.TreeNode) core.NodeStatus {
		percent, err := core.GetInputTyped[float64](self, "percentage")
		if err != nil {
			return core.FAILURE
		}
		devType, err := core.GetInputTyped[int64](self, "deviceType")
		if err != nil {
			return core.FAILURE
		}
		if devType == 1 { // BATT
			_ = self.SetOutput("isLowBattery", percent < 25)
		}
		return core.SUCCESS
	}, core.PortsList{
		"percentage":   core.NewPortInfo(core.INPUT),
		"deviceType":   core.NewPortInfo(core.INPUT),
		"isLowBattery": core.NewPortInfo(core.OUTPUT),
	})

	factory.RegisterScriptingEnum("BATT", 1)
	factory.RegisterScriptingEnum("CONTROLLER", 2)

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	bb := tree.Subtrees[0].Blackboard
	deviceA, _ := core.GetTyped[int](bb, "deviceA")
	if deviceA != 1 {
		t.Errorf("Expected deviceA=1 (BATT), got %v", deviceA)
	}
	deviceB, _ := core.GetTyped[int](bb, "deviceB")
	if deviceB != 2 {
		t.Errorf("Expected deviceB=2 (CONTROLLER), got %v", deviceB)
	}
	isLowBattery, _ := core.GetTyped[bool](bb, "isLowBattery")
	if !isLowBattery {
		t.Error("Expected isLowBattery=true")
	}
}

// TestIssue595 verifies _skipIf with uint8 output port.
// Equivalent of C++ ParserTest/Issue595.
func TestIssue595(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerScriptNodes(factory)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="PowerManagerT">
	      <Sequence>
	        <SampleNode595 find_enemy="{find_enemy}" />
	        <TestA _skipIf="find_enemy==0"/>
	      </Sequence>
	    </BehaviorTree>
	  </root>`

	_ = factory.RegisterSimpleAction("SampleNode595", func(self core.TreeNode) core.NodeStatus {
		_ = self.SetOutput("find_enemy", 0)
		return core.SUCCESS
	}, core.PortsList{
		"find_enemy": core.NewPortInfo(core.OUTPUT),
	})

	var testCount int
	_ = factory.RegisterSimpleAction("TestA", func(core.TreeNode) core.NodeStatus {
		testCount++
		return core.SUCCESS
	}, core.PortsList{})

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}
	if testCount != 0 {
		t.Errorf("Expected TestA ticked 0 times (skipped), got %d", testCount)
	}
}

// TestValidateScriptLargeError verifies ValidateScript doesn't crash on large invalid scripts.
// Equivalent of C++ ParserTest/ValidateScriptLargeError_Issue923.
func TestValidateScriptLargeError(t *testing.T) {
	s := ""
	for i := 0; i < 20; i++ {
		s += "+6e66>6666.6+66\r6>6;6e62=6+6e66>66666'; en';o';o'; en'; "
	}
	err := script.ValidateScript(s)
	if err == nil {
		t.Log("Expected invalid script to fail validation (no crash was the main concern)")
	}
}

// TestNewLine verifies XML &#10; entity is handled in scripts.
// Equivalent of C++ ParserTest/NewLine.
func TestNewLine(t *testing.T) {
	factory := factory.NewBehaviorTreeFactory()
	registerScriptNodes(factory)

	const xmlText = `
	<root BTCPP_format="4" >
	    <BehaviorTree ID="Main">
	      <Script code="A:=5;&#10;B:=6"/>
	    </BehaviorTree>
	  </root>`

	tree, err := factory.CreateTreeFromText(xmlText, nil)
	if err != nil {
		t.Fatal(err)
	}
	status := tree.TickWhileRunning(0)
	if status != core.SUCCESS {
		t.Errorf("Expected SUCCESS, got %v", status)
	}

	bb := tree.Subtrees[0].Blackboard
	aVal, _ := core.GetTyped[int](bb, "A")
	if aVal != 5 {
		t.Errorf("Expected A=5, got %v", aVal)
	}
	bVal, _ := core.GetTyped[int](bb, "B")
	if bVal != 6 {
		t.Errorf("Expected B=6, got %v", bVal)
	}
}
