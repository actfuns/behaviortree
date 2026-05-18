package core

// ScriptFunction is a compiled script that can be evaluated in an environment.
type ScriptFunction func(env ScriptEnv) Any

// ScriptEnv provides the environment for script evaluation.
type ScriptEnv struct {
	Blackboard *Blackboard
	Enums      *ScriptingEnumsRegistry
}

// ScriptParseFunc is the signature for a full script parser.
type ScriptParseFunc func(script string) (ScriptFunction, error)

// scriptParser is a package-level function pointer that can be set by the bt/script package.
var scriptParser ScriptParseFunc

// RegisterScriptParser sets the script parsing function used by ParseScriptExpr.
// This is called by the bt/script package's init() function.
func RegisterScriptParser(fn ScriptParseFunc) {
	scriptParser = fn
}

// ParseScriptExpr parses a script expression and returns a ScriptFunction.
// If the full script parser (bt/script) is registered, it uses that.
// Otherwise returns a simple stub.
func ParseScriptExpr(script string) ScriptFunction {
	if scriptParser != nil {
		fn, err := scriptParser(script)
		if err == nil {
			return fn
		}
		// Return a function that panics with the parse error
		return func(env ScriptEnv) Any {
			panic("Script parse error: " + err.Error())
		}
	}
	return func(env ScriptEnv) Any {
		return AnyOf(true)
	}
}
