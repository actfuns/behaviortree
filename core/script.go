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
