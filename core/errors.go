package core

import (
	"fmt"
)

// BehaviorTreeError is the base error type for the behavior tree library.
type BehaviorTreeError struct {
	Message string
}

func (e *BehaviorTreeError) Error() string {
	return e.Message
}

func NewBehaviorTreeError(format string, args ...interface{}) *BehaviorTreeError {
	return &BehaviorTreeError{Message: fmt.Sprintf(format, args...)}
}

// LogicError indicates errors that require code refactoring to fix.
type LogicError struct {
	BehaviorTreeError
}

func NewLogicError(format string, args ...interface{}) *LogicError {
	return &LogicError{BehaviorTreeError{Message: fmt.Sprintf(format, args...)}}
}

// RuntimeError indicates errors related to runtime data or conditions.
type RuntimeError struct {
	BehaviorTreeError
}

func NewRuntimeError(format string, args ...interface{}) *RuntimeError {
	return &RuntimeError{BehaviorTreeError{Message: fmt.Sprintf(format, args...)}}
}

// TickBacktraceEntry contains info about a node in the tick backtrace.
type TickBacktraceEntry struct {
	NodeName         string
	NodePath         string
	RegistrationName string
}

// NodeExecutionError wraps errors from node tick() with context.
type NodeExecutionError struct {
	RuntimeError
	failedNode      TickBacktraceEntry
	originalMessage string
	backtrace       []TickBacktraceEntry
}

func NewNodeExecutionError(failedNode TickBacktraceEntry, originalMsg string) *NodeExecutionError {
	msg := fmt.Sprintf("Exception in node '%s' [%s]: %s",
		failedNode.NodePath, failedNode.RegistrationName, originalMsg)
	return &NodeExecutionError{
		RuntimeError:    RuntimeError{BehaviorTreeError{Message: msg}},
		failedNode:      failedNode,
		originalMessage: originalMsg,
		backtrace:       []TickBacktraceEntry{failedNode},
	}
}

func (e *NodeExecutionError) FailedNode() TickBacktraceEntry {
	return e.failedNode
}

func (e *NodeExecutionError) OriginalMessage() string {
	return e.originalMessage
}

func (e *NodeExecutionError) Backtrace() []TickBacktraceEntry {
	return e.backtrace
}

func (e *NodeExecutionError) PushBacktrace(entry TickBacktraceEntry) {
	e.backtrace = append(e.backtrace, entry)
}
