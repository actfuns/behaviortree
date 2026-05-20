// Package script provides a full BT script tokenizer, parser, and evaluator.
// It translates C++ BehaviorTree.CPP src/script_tokenizer.cpp + src/script_parser.cpp
// + include/behaviortree_cpp/scripting/operators.hpp to Go.
package script

import (
	"fmt"
	"math"
	"strconv"

	"github.com/actfuns/behaviortree/core"
)

// ---------------------------------------------------------------------------
// Token types
// ---------------------------------------------------------------------------

type tokenType int

const (
	tokInteger    tokenType = iota // 0
	tokReal                        // 1
	tokString                      // 2
	tokBoolean                     // 3
	tokIdentifier                  // 4
	// Arithmetic
	tokPlus   // 5
	tokMinus  // 6
	tokStar   // 7
	tokSlash  // 8
	tokDotDot // 9
	// Bitwise
	tokAmpersand // 10
	tokPipe      // 11
	tokCaret     // 12
	tokTilde     // 13
	// Logical
	tokAmpAmp   // 14
	tokPipePipe // 15
	tokBang     // 16
	// Comparison
	tokEqualEqual   // 17
	tokBangEqual    // 18
	tokLess         // 19
	tokGreater      // 20
	tokLessEqual    // 21
	tokGreaterEqual // 22
	// Assignment
	tokColonEqual // 23
	tokEqual      // 24
	tokPlusEqual  // 25
	tokMinusEqual // 26
	tokStarEqual  // 27
	tokSlashEqual // 28
	// Ternary
	tokQuestion // 29
	tokColon    // 30
	// Delimiters
	tokLeftParen  // 31
	tokRightParen // 32
	tokSemicolon  // 33
	// Control
	tokEndOfInput // 34
	tokError      // 35
)

type token struct {
	typ  tokenType
	text string
	pos  int
}

// ---------------------------------------------------------------------------
// Tokenizer
// ---------------------------------------------------------------------------

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '@'
}

func isIdentChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func consumeTrailingGarbage(source string, length int, i *int) {
	for *i < length && (isIdentChar(source[*i]) || source[*i] == '.') {
		*i++
	}
}

type numberResult struct {
	isReal   bool
	hasError bool
}

func scanHexNumber(source string, length int, i *int) numberResult {
	var res numberResult
	*i += 2 // skip "0x"/"0X"
	if *i >= length || !isHexDigit(source[*i]) {
		res.hasError = true
	} else {
		for *i < length && isHexDigit(source[*i]) {
			*i++
		}
	}
	if *i < length && (source[*i] == '.' || isIdentChar(source[*i])) {
		res.hasError = true
		consumeTrailingGarbage(source, length, i)
	}
	return res
}

func scanDecimalNumber(source string, length int, i *int) numberResult {
	var res numberResult
	// Integer part
	for *i < length && isDigit(source[*i]) {
		*i++
	}
	// Fractional part
	if *i < length && source[*i] == '.' {
		if *i+1 < length && source[*i+1] == '.' {
			// "65.." is Integer("65") + DotDot
		} else if *i+1 < length && isDigit(source[*i+1]) {
			res.isReal = true
			*i++ // consume '.'
			for *i < length && isDigit(source[*i]) {
				*i++
			}
		} else {
			// "65." or "65.x" -- incomplete real
			res.hasError = true
			*i++ // consume the dot
			consumeTrailingGarbage(source, length, i)
		}
	}
	// Exponent
	if !res.hasError && *i < length && (source[*i] == 'e' || source[*i] == 'E') {
		res.isReal = true
		*i++ // consume 'e'/'E'
		if *i < length && (source[*i] == '+' || source[*i] == '-') {
			*i++
		}
		if *i >= length || !isDigit(source[*i]) {
			res.hasError = true
		} else {
			for *i < length && isDigit(source[*i]) {
				*i++
			}
		}
	}
	// Trailing alpha
	if !res.hasError && *i < length && isIdentStart(source[*i]) {
		res.hasError = true
		for *i < length && isIdentChar(source[*i]) {
			*i++
		}
	}
	return res
}

func matchTwoCharOp(c, next byte) tokenType {
	if c == '.' && next == '.' {
		return tokDotDot
	}
	if c == '&' && next == '&' {
		return tokAmpAmp
	}
	if c == '|' && next == '|' {
		return tokPipePipe
	}
	if c == '=' && next == '=' {
		return tokEqualEqual
	}
	if c == '!' && next == '=' {
		return tokBangEqual
	}
	if c == '<' && next == '=' {
		return tokLessEqual
	}
	if c == '>' && next == '=' {
		return tokGreaterEqual
	}
	if c == ':' && next == '=' {
		return tokColonEqual
	}
	if c == '+' && next == '=' {
		return tokPlusEqual
	}
	if c == '-' && next == '=' {
		return tokMinusEqual
	}
	if c == '*' && next == '=' {
		return tokStarEqual
	}
	if c == '/' && next == '=' {
		return tokSlashEqual
	}
	return tokError
}

func matchSingleCharOp(c byte) tokenType {
	switch c {
	case '+':
		return tokPlus
	case '-':
		return tokMinus
	case '*':
		return tokStar
	case '/':
		return tokSlash
	case '&':
		return tokAmpersand
	case '|':
		return tokPipe
	case '^':
		return tokCaret
	case '~':
		return tokTilde
	case '!':
		return tokBang
	case '<':
		return tokLess
	case '>':
		return tokGreater
	case '=':
		return tokEqual
	case '?':
		return tokQuestion
	case ':':
		return tokColon
	case '(':
		return tokLeftParen
	case ')':
		return tokRightParen
	case ';':
		return tokSemicolon
	default:
		return tokError
	}
}

func tokenize(source string) []token {
	var tokens []token
	length := len(source)
	i := 0

	for i < length {
		c := source[i]

		// Skip whitespace
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		start := i

		// Single-quoted string literal
		if c == '\'' {
			i++
			for i < length && source[i] != '\'' {
				i++
			}
			if i < length {
				text := source[start+1 : i]
				tokens = append(tokens, token{typ: tokString, text: text, pos: start})
				i++ // skip closing quote
			} else {
				text := source[start:i]
				tokens = append(tokens, token{typ: tokError, text: text, pos: start})
			}
			continue
		}

		// Number literal
		if isDigit(c) {
			var nr numberResult
			isHex := c == '0' && i+1 < length && (source[i+1] == 'x' || source[i+1] == 'X')
			if isHex {
				nr = scanHexNumber(source, length, &i)
			} else {
				nr = scanDecimalNumber(source, length, &i)
			}
			text := source[start:i]
			if nr.hasError {
				tokens = append(tokens, token{typ: tokError, text: text, pos: start})
			} else if nr.isReal {
				tokens = append(tokens, token{typ: tokReal, text: text, pos: start})
			} else {
				tokens = append(tokens, token{typ: tokInteger, text: text, pos: start})
			}
			continue
		}

		// Identifier or keyword (true/false)
		if isIdentStart(c) {
			i++ // consume start character
			for i < length && isIdentChar(source[i]) {
				i++
			}
			text := source[start:i]
			if text == "true" || text == "false" {
				tokens = append(tokens, token{typ: tokBoolean, text: text, pos: start})
			} else {
				tokens = append(tokens, token{typ: tokIdentifier, text: text, pos: start})
			}
			continue
		}

		// Two-character operators
		if i+1 < length {
			tt := matchTwoCharOp(c, source[i+1])
			if tt != tokError {
				tokens = append(tokens, token{typ: tt, text: source[start : start+2], pos: start})
				i += 2
				continue
			}
		}

		// Single-character operators and delimiters
		tt := matchSingleCharOp(c)
		tokens = append(tokens, token{typ: tt, text: source[start : start+1], pos: start})
		i++
	}

	// Sentinel
	tokens = append(tokens, token{typ: tokEndOfInput, text: "", pos: i})
	return tokens
}

// ---------------------------------------------------------------------------
// AST node types
// ---------------------------------------------------------------------------

type exprBase interface {
	evaluate(env *scriptEnv) core.Any
}

type exprLiteral struct {
	value core.Any
}

func (e *exprLiteral) evaluate(env *scriptEnv) core.Any {
	return e.value
}

type exprName struct {
	name string
}

func (e *exprName) evaluate(env *scriptEnv) core.Any {
	// Search first in the enums table
	if env.enums != nil {
		if val, ok := (*env.enums)[e.name]; ok {
			return core.AnyOf(float64(val))
		}
	}
	// Search in the blackboard
	entry := env.blackboard.GetEntry(e.name)
	if entry == nil {
		panic(fmt.Sprintf("Variable not found: %s", e.name))
	}
	val := entry.GetValue()
	if val == nil {
		panic(fmt.Sprintf("Variable not found: %s", e.name))
	}
	return *val
}

type exprUnaryArithmetic struct {
	op  unaryOp
	rhs exprBase
}

type unaryOp int

const (
	unaryNegate unaryOp = iota
	unaryComplement
	unaryLogicalNot
)

func (e *exprUnaryArithmetic) evaluate(env *scriptEnv) core.Any {
	rhsV := e.rhs.evaluate(env)
	if rhsV.IsNumber() {
		rv, _ := rhsV.ToFloat64()
		switch e.op {
		case unaryNegate:
			return core.AnyOf(-rv)
		case unaryComplement:
			if rv > float64(math.MaxInt64) || rv < float64(math.MinInt64) {
				panic("Number out of range for bitwise operation")
			}
			return core.AnyOf(float64(^int64(rv)))
		case unaryLogicalNot:
			b, _ := rhsV.ToBool()
			if b {
				return core.AnyOf(0.0)
			}
			return core.AnyOf(1.0)
		}
	} else if rhsV.IsString() {
		panic("Invalid operator for std::string")
	} else if e.op == unaryLogicalNot {
		// Handle non-number types (bool, etc.) for logical NOT
		b, err := rhsV.ToBool()
		if err == nil {
			if b {
				return core.AnyOf(0.0)
			}
			return core.AnyOf(1.0)
		}
	}
	panic("ExprUnaryArithmetic: undefined")
}

type binaryOp int

const (
	binaryPlus binaryOp = iota
	binaryMinus
	binaryTimes
	binaryDiv
	binaryConcat
	binaryBitAnd
	binaryBitOr
	binaryBitXor
	binaryLogicAnd
	binaryLogicOr
)

type exprBinaryArithmetic struct {
	op  binaryOp
	lhs exprBase
	rhs exprBase
}

func binaryOpStr(op binaryOp) string {
	switch op {
	case binaryPlus:
		return "+"
	case binaryMinus:
		return "-"
	case binaryTimes:
		return "*"
	case binaryDiv:
		return "/"
	case binaryConcat:
		return ".."
	case binaryBitAnd:
		return "&"
	case binaryBitOr:
		return "|"
	case binaryBitXor:
		return "^"
	case binaryLogicAnd:
		return "&&"
	case binaryLogicOr:
		return "||"
	}
	return ""
}

func (e *exprBinaryArithmetic) evaluate(env *scriptEnv) core.Any {
	lhsV := e.lhs.evaluate(env)
	rhsV := e.rhs.evaluate(env)

	if lhsV.IsEmpty() {
		panic(fmt.Sprintf("The left operand of the operator [%s] is not initialized", binaryOpStr(e.op)))
	}
	if rhsV.IsEmpty() {
		panic(fmt.Sprintf("The right operand of the operator [%s] is not initialized", binaryOpStr(e.op)))
	}

	if lhsV.IsNumber() && rhsV.IsNumber() {
		lv, _ := lhsV.ToFloat64()
		rv, _ := rhsV.ToFloat64()

		switch e.op {
		case binaryPlus:
			return core.AnyOf(lv + rv)
		case binaryMinus:
			return core.AnyOf(lv - rv)
		case binaryTimes:
			return core.AnyOf(lv * rv)
		case binaryDiv:
			return core.AnyOf(lv / rv)
		}

		if e.op == binaryBitAnd || e.op == binaryBitOr || e.op == binaryBitXor {
			li, err := lhsV.ToInt64()
			if err != nil {
				panic("Binary operators are not allowed if one of the operands is not an integer")
			}
			ri, err := rhsV.ToInt64()
			if err != nil {
				panic("Binary operators are not allowed if one of the operands is not an integer")
			}
			switch e.op {
			case binaryBitAnd:
				return core.AnyOf(float64(li & ri))
			case binaryBitOr:
				return core.AnyOf(float64(li | ri))
			case binaryBitXor:
				return core.AnyOf(float64(li ^ ri))
			}
		}

		if e.op == binaryLogicOr || e.op == binaryLogicAnd {
			lb, err := lhsV.ToBool()
			if err != nil {
				panic("Logic operators are not allowed if one of the operands is not castable to bool")
			}
			rb, err := rhsV.ToBool()
			if err != nil {
				panic("Logic operators are not allowed if one of the operands is not castable to bool")
			}
			switch e.op {
			case binaryLogicOr:
				if lb || rb {
					return core.AnyOf(1.0)
				}
				return core.AnyOf(0.0)
			case binaryLogicAnd:
				if lb && rb {
					return core.AnyOf(1.0)
				}
				return core.AnyOf(0.0)
			}
		}
	} else if lhsV.IsString() && rhsV.IsString() && e.op == binaryPlus {
		ls, _ := lhsV.ToString()
		rs, _ := rhsV.ToString()
		return core.AnyOf(ls + rs)
	} else if e.op == binaryConcat && ((lhsV.IsString() && rhsV.IsString()) ||
		(lhsV.IsString() && rhsV.IsNumber()) ||
		(lhsV.IsNumber() && rhsV.IsString())) {
		ls, _ := lhsV.ToString()
		rs, _ := rhsV.ToString()
		return core.AnyOf(ls + rs)
	} else {
		panic("Operation not permitted")
	}

	panic("unreachable")
}

type comparisonOp int

const (
	cmpEqual comparisonOp = iota
	cmpNotEqual
	cmpLess
	cmpGreater
	cmpLessEqual
	cmpGreaterEqual
)

type exprComparison struct {
	ops      []comparisonOp
	operands []exprBase
}

func cmpOpStr(op comparisonOp) string {
	switch op {
	case cmpEqual:
		return "=="
	case cmpNotEqual:
		return "!="
	case cmpLess:
		return "<"
	case cmpGreater:
		return ">"
	case cmpLessEqual:
		return "<="
	case cmpGreaterEqual:
		return ">="
	}
	return ""
}

func isSameFloat(lv, rv float64) bool {
	const eps = 1e-7 // float epsilon approximation
	return math.Abs(lv-rv) <= eps
}

func switchImplFloat(lv, rv float64, op comparisonOp) bool {
	switch op {
	case cmpEqual:
		return isSameFloat(lv, rv)
	case cmpNotEqual:
		return !isSameFloat(lv, rv)
	case cmpLess:
		return lv < rv
	case cmpGreater:
		return lv > rv
	case cmpLessEqual:
		return lv <= rv
	case cmpGreaterEqual:
		return lv >= rv
	}
	return true
}

func switchImplString(lv, rv string, op comparisonOp) bool {
	switch op {
	case cmpEqual:
		return lv == rv
	case cmpNotEqual:
		return lv != rv
	case cmpLess:
		return lv < rv
	case cmpGreater:
		return lv > rv
	case cmpLessEqual:
		return lv <= rv
	case cmpGreaterEqual:
		return lv >= rv
	}
	return true
}

func stringToDouble(value core.Any, env *scriptEnv) float64 {
	s, _ := value.ToString()
	if s == "true" {
		return 1.0
	}
	if s == "false" {
		return 0.0
	}
	if env.enums != nil {
		if val, ok := (*env.enums)[s]; ok {
			return float64(val)
		}
	}
	f, err := value.ToFloat64()
	if err != nil {
		panic(fmt.Sprintf("Cannot convert string '%s' to number", s))
	}
	return f
}

func (e *exprComparison) evaluate(env *scriptEnv) core.Any {
	lhsV := e.operands[0].evaluate(env)
	for i, op := range e.ops {
		rhsV := e.operands[i+1].evaluate(env)

		if lhsV.IsEmpty() {
			panic(fmt.Sprintf("The left operand of the operator [%s] is not initialized", cmpOpStr(op)))
		}
		if rhsV.IsEmpty() {
			panic(fmt.Sprintf("The right operand of the operator [%s] is not initialized", cmpOpStr(op)))
		}

		falseVal := core.AnyOf(0.0)

		if lhsV.IsNumber() && rhsV.IsNumber() {
			lv, _ := lhsV.ToFloat64()
			rv, _ := rhsV.ToFloat64()
			if !switchImplFloat(lv, rv, op) {
				return falseVal
			}
		} else if lhsV.IsString() && rhsV.IsString() {
			lv, _ := lhsV.ToString()
			rv, _ := rhsV.ToString()
			if !switchImplString(lv, rv, op) {
				return falseVal
			}
		} else if lhsV.IsString() && rhsV.IsNumber() {
			lv := stringToDouble(lhsV, env)
			rv, _ := rhsV.ToFloat64()
			if !switchImplFloat(lv, rv, op) {
				return falseVal
			}
		} else if lhsV.IsNumber() && rhsV.IsString() {
			lv, _ := lhsV.ToFloat64()
			rv := stringToDouble(rhsV, env)
			if !switchImplFloat(lv, rv, op) {
				return falseVal
			}
		} else {
			panic("Can't mix different types in Comparison.")
		}
		lhsV = rhsV
	}
	return core.AnyOf(1.0)
}

type exprIf struct {
	condition exprBase
	then      exprBase
	else_     exprBase
}

func (e *exprIf) evaluate(env *scriptEnv) core.Any {
	v := e.condition.evaluate(env)
	var valid bool
	if v.IsString() {
		s, _ := v.ToString()
		valid = len(s) > 0
	} else {
		f, err := v.ToFloat64()
		if err != nil {
			panic("Cannot evaluate ternary condition")
		}
		valid = f != 0.0
	}
	if valid {
		return e.then.evaluate(env)
	}
	return e.else_.evaluate(env)
}

type assignmentOp int

const (
	assignCreate assignmentOp = iota
	assignExisting
	assignPlus
	assignMinus
	assignTimes
	assignDiv
)

type exprAssignment struct {
	op  assignmentOp
	lhs exprBase
	rhs exprBase
}

func assignOpStr(op assignmentOp) string {
	switch op {
	case assignCreate:
		return ":="
	case assignExisting:
		return "="
	case assignPlus:
		return "+="
	case assignMinus:
		return "-="
	case assignTimes:
		return "*="
	case assignDiv:
		return "/="
	}
	return ""
}

func (e *exprAssignment) evaluate(env *scriptEnv) core.Any {
	nameExpr, ok := e.lhs.(*exprName)
	if !ok {
		panic("Assignment left operand not a blackboard entry")
	}
	key := nameExpr.name

	entry := env.blackboard.GetEntry(key)
	if entry == nil {
		// Variable doesn't exist
		if e.op == assignCreate {
			if _, err := env.blackboard.CreateEntry(key, core.NewPortInfo(core.INOUT)); err != nil {
				panic(err.Error())
			}
			entry = env.blackboard.GetEntry(key)
			if entry == nil {
				panic("Bug: report")
			}
		} else {
			panic(fmt.Sprintf("The blackboard entry [%s] doesn't exist, yet.\n"+
				"If you want to create a new one, use the operator [:=] instead of [=]", key))
		}
	}

	value := e.rhs.evaluate(env)
	if value.IsEmpty() {
		panic(fmt.Sprintf("The right operand of the operator [%s] is not initialized", assignOpStr(e.op)))
	}

	if e.op == assignCreate || e.op == assignExisting {
		// For simple assignment, validate type compatibility
		if entry != nil {
			existingVal := entry.GetValue()
			if existingVal != nil && !existingVal.IsEmpty() {
				// Check type compatibility — C++ behavior: once a variable has a type,
				// incompatible assignments must fail
				if value.IsString() && !existingVal.IsString() {
					// String to non-string: try to use string converter or check if target is number
					if existingVal.IsNumber() {
						// Allow string-to-number conversion via stringToDouble
						numVal := stringToDouble(value, env)
						if err := env.blackboard.Set(key, core.AnyOf(numVal).Interface()); err != nil {
							panic(err.Error())
						}
					} else {
						panic(fmt.Sprintf("Error assigning a value to entry [%s]. "+
							"The right operand is a string, can't convert.", key))
					}
				} else if existingVal.IsString() && !value.IsString() {
					panic(fmt.Sprintf("Error assigning a value to entry [%s]. "+
						"Can't assign non-string to string entry.", key))
				} else if existingVal.IsNumber() && value.IsNumber() {
					if err := env.blackboard.Set(key, value.Interface()); err != nil {
						panic(err.Error())
					}
				} else if existingVal.IsString() && value.IsString() {
					if err := env.blackboard.Set(key, value.Interface()); err != nil {
						panic(err.Error())
					}
				} else {
					if err := env.blackboard.Set(key, value.Interface()); err != nil {
						panic(err.Error())
					}
				}
			} else {
				if err := env.blackboard.Set(key, value.Interface()); err != nil {
					panic(err.Error())
				}
			}
		} else {
			if err := env.blackboard.Set(key, value.Interface()); err != nil {
				panic(err.Error())
			}
		}
		// Re-read to return
		result := env.blackboard.GetEntry(key)
		if result != nil {
			v := result.GetValue()
			if v != nil {
				return *v
			}
		}
		return value
	}

	// Compound assignment (+=, -=, *=, /=)
	existing := env.blackboard.GetEntry(key)
	existingV := existing.GetValue()
	if existingV == nil {
		panic(fmt.Sprintf("The left operand of the operator [%s] is not initialized", assignOpStr(e.op)))
	}

	if value.IsNumber() {
		ev, _ := existingV.ToFloat64()
		rv, _ := value.ToFloat64()
		var result float64
		switch e.op {
		case assignPlus:
			result = ev + rv
		case assignMinus:
			result = ev - rv
		case assignTimes:
			result = ev * rv
		case assignDiv:
			result = ev / rv
		default:
			panic("unreachable")
		}
		if err := env.blackboard.Set(key, result); err != nil {
			panic(err.Error())
		}
	} else if value.IsString() {
		if e.op == assignPlus {
			ev, _ := existingV.ToString()
			rv, _ := value.ToString()
			if err := env.blackboard.Set(key, ev+rv); err != nil {
				panic(err.Error())
			}
		} else {
			panic("Operator not supported for strings")
		}
	} else {
		panic("Assignment operator not supported for this type")
	}

	result := env.blackboard.GetEntry(key)
	if result != nil {
		v := result.GetValue()
		if v != nil {
			return *v
		}
	}
	return value
}

// ---------------------------------------------------------------------------
// Parser (Pratt parser)
// ---------------------------------------------------------------------------

const (
	kAssignmentBP = 2
	kTernaryBP    = 4
	kComparisonBP = 10
	kMulDivBP     = 18
	kPrefixBP     = 20
)

type parser struct {
	tokens  []token
	current int
}

func (p *parser) peek() token {
	return p.tokens[p.current]
}

func (p *parser) advance() token {
	tok := p.tokens[p.current]
	if p.peek().typ != tokEndOfInput {
		p.current++
	}
	return tok
}

func (p *parser) atEnd() bool {
	return p.peek().typ == tokEndOfInput
}

func (p *parser) check(tt tokenType) bool {
	return p.peek().typ == tt
}

func (p *parser) expect(tt tokenType, msg string) token {
	if !p.check(tt) {
		panic(fmt.Sprintf("Parse error at position %d: %s (got '%s')", p.peek().pos, msg, p.peek().text))
	}
	return p.advance()
}

func leftBP(tt tokenType) int {
	switch tt {
	case tokColonEqual, tokEqual, tokPlusEqual, tokMinusEqual, tokStarEqual, tokSlashEqual:
		return kAssignmentBP
	case tokQuestion:
		return kTernaryBP
	case tokPipePipe:
		return 6
	case tokAmpAmp:
		return 8
	case tokEqualEqual, tokBangEqual, tokLess, tokGreater, tokLessEqual, tokGreaterEqual:
		return kComparisonBP
	case tokPipe, tokCaret:
		return 12
	case tokAmpersand:
		return 14
	case tokPlus, tokMinus, tokDotDot:
		return 16
	case tokStar, tokSlash:
		return kMulDivBP
	default:
		return -1
	}
}

func isComparison(tt tokenType) bool {
	return tt == tokEqualEqual || tt == tokBangEqual || tt == tokLess ||
		tt == tokGreater || tt == tokLessEqual || tt == tokGreaterEqual
}

func isAssignment(tt tokenType) bool {
	return tt == tokColonEqual || tt == tokEqual || tt == tokPlusEqual ||
		tt == tokMinusEqual || tt == tokStarEqual || tt == tokSlashEqual
}

func (p *parser) parsePrefix() exprBase {
	tok := p.peek()

	// Unary minus
	if tok.typ == tokMinus {
		p.advance()
		operand := p.parseExpr(kPrefixBP)
		return &exprUnaryArithmetic{op: unaryNegate, rhs: operand}
	}
	// Bitwise complement
	if tok.typ == tokTilde {
		p.advance()
		operand := p.parseExpr(kPrefixBP)
		return &exprUnaryArithmetic{op: unaryComplement, rhs: operand}
	}
	// Logical NOT
	if tok.typ == tokBang {
		p.advance()
		operand := p.parseExpr(kPrefixBP)
		return &exprUnaryArithmetic{op: unaryLogicalNot, rhs: operand}
	}
	// Parenthesized expression
	if tok.typ == tokLeftParen {
		p.advance()
		expr := p.parseExpr(0)
		p.expect(tokRightParen, "expected ')'")
		return expr
	}
	// Boolean literal
	if tok.typ == tokBoolean {
		p.advance()
		val := 0.0
		if tok.text == "true" {
			val = 1.0
		}
		return &exprLiteral{value: core.AnyOf(val)}
	}
	// Integer literal
	if tok.typ == tokInteger {
		p.advance()
		var val int64
		if len(tok.text) > 2 && tok.text[0] == '0' && (tok.text[1] == 'x' || tok.text[1] == 'X') {
			val, _ = strconv.ParseInt(tok.text[2:], 16, 64)
		} else {
			val, _ = strconv.ParseInt(tok.text, 10, 64)
		}
		return &exprLiteral{value: core.AnyOf(val)}
	}
	// Real literal
	if tok.typ == tokReal {
		p.advance()
		val, _ := strconv.ParseFloat(tok.text, 64)
		return &exprLiteral{value: core.AnyOf(val)}
	}
	// String literal
	if tok.typ == tokString {
		p.advance()
		return &exprLiteral{value: core.AnyOf(tok.text)}
	}
	// Identifier
	if tok.typ == tokIdentifier {
		p.advance()
		return &exprName{name: tok.text}
	}
	// Error token
	if tok.typ == tokError {
		panic(fmt.Sprintf("Invalid token '%s' at position %d", tok.text, tok.pos))
	}

	panic(fmt.Sprintf("Expected operand at position %d (got '%s')", tok.pos, tok.text))
}

func (p *parser) parseExpr(minBP int) exprBase {
	left := p.parsePrefix()

	for {
		tokType := p.peek().typ
		lbp := leftBP(tokType)
		if lbp < 0 || lbp < minBP {
			break
		}

		// Assignment (non-associative)
		if isAssignment(tokType) {
			left = p.parseAssignment(left)
			break
		}

		// Ternary (non-associative)
		if tokType == tokQuestion {
			left = p.parseTernary(left)
			break
		}

		// Chained comparison
		if isComparison(tokType) {
			left = p.parseChainedComparison(left)
			continue
		}

		// Regular left-associative binary operator
		opTok := p.advance()
		right := p.parseExpr(lbp + 1)
		left = makeBinary(left, opTok.typ, right)
	}

	return left
}

func (p *parser) parseAssignment(left exprBase) exprBase {
	opTok := p.advance()
	var op assignmentOp
	switch opTok.typ {
	case tokColonEqual:
		op = assignCreate
	case tokEqual:
		op = assignExisting
	case tokPlusEqual:
		op = assignPlus
	case tokMinusEqual:
		op = assignMinus
	case tokStarEqual:
		op = assignTimes
	case tokSlashEqual:
		op = assignDiv
	default:
		panic("Internal error: unexpected assignment op")
	}
	right := p.parseExpr(0)
	return &exprAssignment{op: op, lhs: left, rhs: right}
}

func (p *parser) parseTernary(condition exprBase) exprBase {
	p.advance() // consume '?'
	thenExpr := p.parseExpr(0)
	p.expect(tokColon, "expected ':' in ternary expression")
	elseExpr := p.parseExpr(kTernaryBP)
	return &exprIf{condition: condition, then: thenExpr, else_: elseExpr}
}

func (p *parser) parseChainedComparison(first exprBase) exprBase {
	node := &exprComparison{}
	node.operands = append(node.operands, first)

	for isComparison(p.peek().typ) {
		node.ops = append(node.ops, mapComparisonOp(p.advance().typ))
		node.operands = append(node.operands, p.parseExpr(kComparisonBP+1))
	}
	return node
}

func mapComparisonOp(tt tokenType) comparisonOp {
	switch tt {
	case tokEqualEqual:
		return cmpEqual
	case tokBangEqual:
		return cmpNotEqual
	case tokLess:
		return cmpLess
	case tokGreater:
		return cmpGreater
	case tokLessEqual:
		return cmpLessEqual
	case tokGreaterEqual:
		return cmpGreaterEqual
	default:
		panic("Internal error: not a comparison op")
	}
}

func makeBinary(left exprBase, opType tokenType, right exprBase) exprBase {
	var op binaryOp
	switch opType {
	case tokPlus:
		op = binaryPlus
	case tokMinus:
		op = binaryMinus
	case tokStar:
		op = binaryTimes
	case tokSlash:
		op = binaryDiv
	case tokDotDot:
		op = binaryConcat
	case tokAmpersand:
		op = binaryBitAnd
	case tokPipe:
		op = binaryBitOr
	case tokCaret:
		op = binaryBitXor
	case tokAmpAmp:
		op = binaryLogicAnd
	case tokPipePipe:
		op = binaryLogicOr
	default:
		panic("Internal error: unknown binary operator")
	}
	return &exprBinaryArithmetic{op: op, lhs: left, rhs: right}
}

func (p *parser) parseAll() []exprBase {
	var stmts []exprBase
	for !p.atEnd() {
		stmts = append(stmts, p.parseExpr(0))
		// Consume optional semicolons
		for p.check(tokSemicolon) {
			p.advance()
		}
	}
	return stmts
}

// ---------------------------------------------------------------------------
// scriptEnv wraps the core types for evaluation
// ---------------------------------------------------------------------------

type scriptEnv struct {
	blackboard *core.Blackboard
	enums      *core.ScriptingEnumsRegistry
}

func newScriptEnv(blackboard *core.Blackboard, enums *core.ScriptingEnumsRegistry) *scriptEnv {
	return &scriptEnv{blackboard: blackboard, enums: enums}
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

// ParseScript parses a script string and returns a ScriptFunction.
func ParseScript(script string) (core.ScriptFunction, error) {
	exprs, err := parseStatements(script)
	if err != nil {
		return nil, err
	}
	if len(exprs) == 0 {
		return nil, fmt.Errorf("Empty Script")
	}
	fn := func(env core.ScriptEnv) core.Any {
		senv := newScriptEnv(env.Blackboard, env.Enums)
		for i := 0; i < len(exprs)-1; i++ {
			exprs[i].evaluate(senv)
		}
		return exprs[len(exprs)-1].evaluate(senv)
	}
	return fn, nil
}

func parseStatements(source string) (stmts []exprBase, err error) {
	toks := tokenize(source)
	par := &parser{tokens: toks}
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = fmt.Errorf("%s", s)
			} else if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
		}
	}()
	stmts = par.parseAll()
	return
}

// ParseScriptAndExecute parses and immediately executes a script.
func ParseScriptAndExecute(blackboard *core.Blackboard, enums *core.ScriptingEnumsRegistry, script string) (result core.Any, err error) {
	fn, err := ParseScript(script)
	if err != nil {
		return core.Any{}, err
	}
	env := core.ScriptEnv{Blackboard: blackboard, Enums: enums}
	defer func() {
		if r := recover(); r != nil {
			if s, ok := r.(string); ok {
				err = fmt.Errorf("%s", s)
			} else if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
			result = core.Any{}
		}
	}()
	result = fn(env)
	return
}

// ValidateScript checks if a script string is valid.
func ValidateScript(script string) error {
	exprs, err := parseStatements(script)
	if err != nil {
		return err
	}
	if len(exprs) == 0 {
		return fmt.Errorf("Empty Script")
	}
	return nil
}
