package core_test

import (
	"testing"

	"github.com/actfuns/behaviortree/core"
)

// --------------------------------------------------------------------
// SimpleString tests
// The Go port uses native Go string instead of the C++ SimpleString (SOO) type.
// These tests verify that Go's string type handles the same use cases correctly.
// --------------------------------------------------------------------

// TestSimpleString_Basic verifies basic string operations.
func TestSimpleString_Basic(t *testing.T) {
	// Default/empty string
	var s string
	if len(s) != 0 {
		t.Errorf("expected empty string")
	}

	// Empty string literal
	s = ""
	if len(s) != 0 {
		t.Errorf("expected empty string, got len %d", len(s))
	}

	// Non-empty string
	s = "hello"
	if len(s) != 5 {
		t.Errorf("expected len 5, got %d", len(s))
	}
	if s != "hello" {
		t.Errorf("expected 'hello', got '%s'", s)
	}

	// Long string
	longStr := ""
	for i := 0; i < 100; i++ {
		longStr += "x"
	}
	if len(longStr) != 100 {
		t.Errorf("expected len 100, got %d", len(longStr))
	}
}

// TestSimpleString_Construction verifies construction from various sources.
func TestSimpleString_Construction(t *testing.T) {
	// From string literal
	s := "testing"
	if len(s) != 7 {
		t.Errorf("expected len 7, got %d", len(s))
	}
	if s != "testing" {
		t.Errorf("expected 'testing', got '%s'", s)
	}

	// From substring
	text := "hello world"
	sub := text[:5]
	if sub != "hello" {
		t.Errorf("expected 'hello', got '%s'", sub)
	}

	// From single character
	c := "a"
	if len(c) != 1 {
		t.Errorf("expected len 1, got %d", len(c))
	}
	if c != "a" {
		t.Errorf("expected 'a', got '%s'", c)
	}

	// Very long string
	veryLong := ""
	for i := 0; i < 10000; i++ {
		veryLong += "z"
	}
	if len(veryLong) != 10000 {
		t.Errorf("expected len 10000, got %d", len(veryLong))
	}
}

// TestSimpleString_Copy verifies copy behavior.
func TestSimpleString_Copy(t *testing.T) {
	// Copy
	s1 := "hello"
	s2 := s1
	if s1 != s2 {
		t.Errorf("expected equal strings")
	}
	if len(s1) != len(s2) {
		t.Errorf("expected same length")
	}

	// Copy of long string
	longStr := ""
	for i := 0; i < 50; i++ {
		longStr += "a"
	}
	s1 = longStr
	s2 = s1
	if s1 != s2 {
		t.Errorf("expected equal long strings")
	}

	// Assignment
	s1 = "hello"
	s2 = "world"
	s2 = s1
	if s1 != s2 {
		t.Errorf("expected equal after assignment")
	}

	// Self-assignment (no-op in Go)
	s := "test"
	s = s
	if s != "test" {
		t.Errorf("expected 'test' after self-assignment")
	}

	// Copy of empty string
	var empty1 string
	empty2 := empty1
	if len(empty2) != 0 {
		t.Errorf("expected empty string after copy")
	}
}

// TestSimpleString_Conversion verifies conversion between string representations.
func TestSimpleString_Conversion(t *testing.T) {
	// String can be used directly (no conversion needed in Go)
	s := "convert me"
	if s != "convert me" {
		t.Errorf("unexpected value")
	}

	// Empty string
	var empty string
	if empty != "" {
		t.Errorf("expected empty string")
	}
}

// TestSimpleString_Equality verifies comparison operators.
func TestSimpleString_Equality(t *testing.T) {
	s1 := "hello"
	s2 := "hello"
	s3 := "world"
	s4 := "hell"

	if s1 != s2 {
		t.Errorf("expected equal")
	}
	if s1 == s3 {
		t.Errorf("expected not equal")
	}
	if s1 == s4 {
		t.Errorf("expected not equal")
	}
}

// TestSimpleString_Comparison verifies lexicographic ordering.
func TestSimpleString_Comparison(t *testing.T) {
	s1 := "apple"
	s2 := "banana"
	s3 := "apple"
	s4 := "app"

	if !(s1 < s2) {
		t.Errorf("expected apple < banana")
	}
	if s2 < s1 {
		t.Errorf("expected not banana < apple")
	}
	if s1 < s3 {
		t.Errorf("expected not apple < apple")
	}
	if s1 < s4 {
		t.Errorf("expected not apple < app")
	}
	if !(s4 < s1) {
		t.Errorf("expected app < apple")
	}
	if !(s1 > s4) {
		t.Errorf("expected apple > app")
	}
}

// TestSimpleString_NullTerminated verifies that Go strings behave as expected.
func TestSimpleString_NullTerminated(t *testing.T) {
	// Go strings are not null-terminated but contain the raw bytes.
	// Verify the content is correct.
	s := "test"
	if len(s) != 4 {
		t.Errorf("expected len 4, got %d", len(s))
	}
	if s[0] != 't' || s[1] != 'e' || s[2] != 's' || s[3] != 't' {
		t.Errorf("unexpected content")
	}

	// Access by index
	longStr := ""
	for i := 0; i < 50; i++ {
		longStr += "x"
	}
	if longStr[49] != 'x' {
		t.Errorf("expected 'x' at position 49")
	}
	if len(longStr) != 50 {
		t.Errorf("expected len 50")
	}

	// Verify through Any
	a := core.AnyOf("test")
	v, err := core.Cast[string](a)
	if err != nil {
		t.Fatalf("Cast failed: %v", err)
	}
	if v != "test" {
		t.Errorf("expected 'test', got '%s'", v)
	}
}

// TestSimpleString_SizeOfString verifies basic type behavior.
func TestSimpleString_SizeOfString(t *testing.T) {
	// In Go, we verify that the Any type correctly handles strings
	s := "hello"
	a := core.AnyOf(s)
	if a.IsEmpty() {
		t.Errorf("expected non-empty Any")
	}
	if !a.IsString() {
		t.Errorf("expected IsString()=true")
	}
	if v, _ := core.Cast[string](a); v != "hello" {
		t.Errorf("expected 'hello', got '%s'", v)
	}
}

// TestSimpleString_EmptyString verifies detailed empty string behavior.
// Equivalent of C++ SimpleStringTest/EmptyString.
func TestSimpleString_EmptyString(t *testing.T) {
	// Default empty string
	var s string
	if len(s) != 0 {
		t.Errorf("expected empty string length 0, got %d", len(s))
	}

	// Empty literal
	s = ""
	if len(s) != 0 {
		t.Errorf("expected empty string length 0, got %d", len(s))
	}

	// Empty string can be compared
	if s != "" {
		t.Error("empty string should equal ''")
	}

	// Empty string data access (no crash)
	if len(s) != 0 {
		t.Error("expected empty string")
	}
}

// TestSimpleString_LongString verifies long string handling.
// Equivalent of C++ SimpleStringTest/LongString.
func TestSimpleString_LongString(t *testing.T) {
	longStr := make([]byte, 100)
	for i := 0; i < 100; i++ {
		longStr[i] = 'x'
	}
	s := string(longStr)
	if len(s) != 100 {
		t.Errorf("expected len 100, got %d", len(s))
	}
	for i := 0; i < 100; i++ {
		if s[i] != 'x' {
			t.Errorf("expected 'x' at position %d, got '%c'", i, s[i])
			break
		}
	}

	// Verify via Any type - equivalent to toStdString()
	a := core.AnyOf(s)
	v, err := core.Cast[string](a)
	if err != nil {
		t.Fatalf("Cast failed: %v", err)
	}
	if v != s {
		t.Errorf("Any round-trip changed the string")
	}
}

// TestSimpleString_SingleCharacter verifies single character string handling.
// Equivalent of C++ SimpleStringTest/SingleCharacter.
func TestSimpleString_SingleCharacter(t *testing.T) {
	s := "a"
	if len(s) != 1 {
		t.Errorf("expected len 1, got %d", len(s))
	}
	if s != "a" {
		t.Errorf("expected 'a', got '%s'", s)
	}

	// Access via index
	if s[0] != 'a' {
		t.Errorf("expected s[0] == 'a', got '%c'", s[0])
	}
}

// TestSimpleString_CapacityMinus1 verifies strings just under the Go
// SSA/heap boundary. Note: Go strings don't have explicit SOO, but we
// test boundary-like lengths for parity with C++.
// Equivalent of C++ SimpleStringTest/CapacityMinus1 (14 chars).
func TestSimpleString_CapacityMinus1(t *testing.T) {
	// 14 characters (C++ SOO capacity is 15, so 14 = capacity - 1)
	s := "12345678901234"
	if len(s) != 14 {
		t.Errorf("expected len 14, got %d", len(s))
	}
	if s != "12345678901234" {
		t.Errorf("content mismatch: got '%s'", s)
	}
}

// TestSimpleString_CapacityPlus1 verifies strings just over the C++ SOO
// boundary (16 chars). Go handles this natively.
// Equivalent of C++ SimpleStringTest/CapacityPlus1 (16 chars).
func TestSimpleString_CapacityPlus1(t *testing.T) {
	// 16 characters (C++ SOO capacity is 15, so 16 = capacity + 1)
	s := "1234567890123456"
	if len(s) != 16 {
		t.Errorf("expected len 16, got %d", len(s))
	}
	if s != "1234567890123456" {
		t.Errorf("content mismatch: got '%s'", s)
	}
}

// TestSimpleString_CopyEmptyString verifies copy of an empty string.
// Equivalent of C++ SimpleStringTest/CopyEmptyString.
func TestSimpleString_CopyEmptyString(t *testing.T) {
	var s1 string
	s2 := s1
	if len(s2) != 0 {
		t.Errorf("expected empty string after copying empty, got len %d", len(s2))
	}
	if s2 != "" {
		t.Errorf("expected '' after copy, got '%s'", s2)
	}
}

// TestSimpleString_EmptyStringComparison verifies comparisons with empty strings.
// Equivalent of C++ SimpleStringTest/EmptyStringComparison.
func TestSimpleString_EmptyStringComparison(t *testing.T) {
	var empty1 string
	var empty2 string
	nonEmpty := "a"

	if empty1 != empty2 {
		t.Error("empty1 should == empty2")
	}
	if empty1 == nonEmpty {
		t.Error("empty should not == nonEmpty")
	}
	if !(empty1 < nonEmpty) {
		t.Error("empty should be < nonEmpty")
	}
	if !(nonEmpty > empty1) {
		t.Error("nonEmpty should be > empty")
	}
	if !(empty1 <= nonEmpty) {
		t.Error("empty should be <= nonEmpty")
	}
	if !(nonEmpty >= empty1) {
		t.Error("nonEmpty should be >= empty1")
	}
}

// TestSimpleString_ComparisonNonSOO verifies comparison of long (non-SOO in C++) strings.
// Equivalent of C++ SimpleStringTest/ComparisonNonSOO.
func TestSimpleString_ComparisonNonSOO(t *testing.T) {
	longStr1 := ""
	longStr2 := ""
	longStr3 := ""
	for i := 0; i < 50; i++ {
		longStr1 += "a"
		longStr2 += "b"
		longStr3 += "a"
	}

	if longStr1 != longStr3 {
		t.Error("longStr1 should == longStr3")
	}
	if longStr1 == longStr2 {
		t.Error("longStr1 should != longStr2")
	}
	if !(longStr1 < longStr2) {
		t.Error("longStr1 should be < longStr2")
	}
	if !(longStr2 > longStr1) {
		t.Error("longStr2 should be > longStr1")
	}
	if !(longStr1 <= longStr3) {
		t.Error("longStr1 should be <= longStr3")
	}
	if !(longStr1 >= longStr3) {
		t.Error("longStr1 should be >= longStr3")
	}
}

// TestSimpleString_VeryLongString verifies very long string handling.
// Equivalent of C++ SimpleStringTest/VeryLongString.
func TestSimpleString_VeryLongString(t *testing.T) {
	veryLong := ""
	for i := 0; i < 10000; i++ {
		veryLong += "z"
	}
	if len(veryLong) != 10000 {
		t.Errorf("expected len 10000, got %d", len(veryLong))
	}
	for i := 0; i < len(veryLong); i++ {
		if veryLong[i] != 'z' {
			t.Errorf("expected 'z' at position %d", i)
			break
		}
	}
}
