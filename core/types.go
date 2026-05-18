package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// NodeType enumerates the possible types of nodes.
type NodeType int

const (
	Undefined NodeType = iota
	Action
	Condition
	Control
	Decorator
	Subtree
)

func (t NodeType) String() string {
	switch t {
	case Action:
		return "Action"
	case Condition:
		return "Condition"
	case Control:
		return "Control"
	case Decorator:
		return "Decorator"
	case Subtree:
		return "SubTree"
	default:
		return "Undefined"
	}
}

// NodeStatus enumerates the states every node can be in after execution.
type NodeStatus int

const (
	IDLE    NodeStatus = 0
	RUNNING NodeStatus = 1
	SUCCESS NodeStatus = 2
	FAILURE NodeStatus = 3
	SKIPPED NodeStatus = 4
)

func (s NodeStatus) String() string {
	switch s {
	case RUNNING:
		return "RUNNING"
	case SUCCESS:
		return "SUCCESS"
	case FAILURE:
		return "FAILURE"
	case SKIPPED:
		return "SKIPPED"
	default:
		return "IDLE"
	}
}

// IsActive returns true if the status is not IDLE and not SKIPPED.
func (s NodeStatus) IsActive() bool {
	return s != IDLE && s != SKIPPED
}

// IsCompleted returns true if the status is SUCCESS or FAILURE.
func (s NodeStatus) IsCompleted() bool {
	return s == SUCCESS || s == FAILURE
}

// PortDirection enumerates the direction of a port.
type PortDirection int

const (
	INPUT  PortDirection = 0
	OUTPUT PortDirection = 1
	INOUT  PortDirection = 2
)

func (d PortDirection) String() string {
	switch d {
	case INPUT:
		return "INPUT"
	case OUTPUT:
		return "OUTPUT"
	case INOUT:
		return "INOUT"
	default:
		return "UNKNOWN"
	}
}

// KeyValueVector is a vector of key/value pairs.
type KeyValueVector []KeyValuePair

type KeyValuePair struct {
	Key   string
	Value string
}

// Timestamp contains sequence number and time point for blackboard entries.
type Timestamp struct {
	Seq   uint64
	Stamp int64 // nanoseconds since epoch
}

// PreCond enumerates types of pre-conditions that can be attached to nodes.
type PreCond int

const (
	FailureIf PreCond = iota
	SuccessIf
	SkipIf
	WhileTrue
	PreCondCount
)

var PreCondNames = [4]string{"_failureIf", "_successIf", "_skipIf", "_while"}

// PostCond enumerates types of post-conditions.
type PostCond int

const (
	OnHalted PostCond = iota
	OnFailure
	OnSuccess
	Always
	PostCondCount
)

var PostCondNames = [4]string{"_onHalted", "_onFailure", "_onSuccess", "_post"}

// StringConverter converts a string to an Any value.
type StringConverter func(string) (Any, error)

// TypeInfo stores type metadata for ports and blackboard entries.
type TypeInfo struct {
	typeStr string
	typeRfl reflect.Type
	conv    StringConverter
}

func NewTypeInfo[T any]() TypeInfo {
	var zero T
	t := reflect.TypeOf(zero)
	return TypeInfo{
		typeRfl: t,
		typeStr: t.String(),
		conv:    getStringConverter[T](),
	}
}

func NewTypeInfoFromReflect(t reflect.Type) TypeInfo {
	return TypeInfo{
		typeRfl: t,
		typeStr: t.String(),
	}
}

func NewTypeInfoAnyAllowed() TypeInfo {
	return TypeInfo{
		typeStr: "AnyTypeAllowed",
	}
}

func (ti TypeInfo) Type() reflect.Type {
	return ti.typeRfl
}

func (ti TypeInfo) TypeName() string {
	return ti.typeStr
}

func (ti TypeInfo) ParseString(str string) (Any, error) {
	if ti.conv != nil {
		val, err := ti.conv(str)
		if err != nil {
			return Any{}, err
		}
		return val, nil
	}
	return Any{}, nil
}

func (ti TypeInfo) IsStronglyTyped() bool {
	if ti.typeStr == "AnyTypeAllowed" {
		return false
	}
	if ti.typeRfl != nil {
		return ti.typeRfl != reflect.TypeOf(Any{})
	}
	return ti.typeStr != "Any"
}

func (ti TypeInfo) Converter() StringConverter {
	return ti.conv
}

func getStringConverter[T any]() StringConverter {
	var zero T
	switch any(zero).(type) {
	case string:
		return func(s string) (Any, error) { return AnyOf(s), nil }
	case int:
		return func(s string) (Any, error) {
			v, err := strconv.Atoi(s)
			if err != nil {
				return Any{}, NewRuntimeError("Can't convert string [%s] to integer", s)
			}
			return AnyOf(v), nil
		}
	case int64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return Any{}, NewRuntimeError("Can't convert string [%s] to integer", s)
			}
			return AnyOf(v), nil
		}
	case uint64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return Any{}, NewRuntimeError("Can't convert string [%s] to integer", s)
			}
			return AnyOf(v), nil
		}
	case float64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return Any{}, NewRuntimeError("Can't convert string [%s] to double", s)
			}
			return AnyOf(v), nil
		}
	case bool:
		return func(s string) (Any, error) {
			switch strings.ToLower(s) {
			case "true", "1":
				return AnyOf(true), nil
			case "false", "0":
				return AnyOf(false), nil
			default:
				return Any{}, NewRuntimeError("Can't convert string [%s] to bool", s)
			}
		}
	case NodeStatus:
		return func(s string) (Any, error) {
			for _, v := range []NodeStatus{IDLE, RUNNING, SUCCESS, FAILURE, SKIPPED} {
				if v.String() == s {
					return AnyOf(v), nil
				}
			}
			return Any{}, NewRuntimeError("Can't convert string [%s] to NodeStatus", s)
		}
	case NodeType:
		return func(s string) (Any, error) {
			for _, v := range []NodeType{Undefined, Action, Condition, Control, Decorator, Subtree} {
				if v.String() == s {
					return AnyOf(v), nil
				}
			}
			return Any{}, NewRuntimeError("Can't convert string [%s] to NodeType", s)
		}
	case PortDirection:
		return func(s string) (Any, error) {
			for _, v := range []PortDirection{INPUT, OUTPUT, INOUT} {
				if v.String() == s {
					return AnyOf(v), nil
				}
			}
			return Any{}, NewRuntimeError("Can't convert string [%s] to PortDirection", s)
		}
	default:
		return nil
	}
}

// PortInfo extends TypeInfo with direction and default value.
type PortInfo struct {
	TypeInfo
	direction       PortDirection
	description     string
	defaultValue    Any
	defaultValueStr string
}

func NewPortInfo(direction PortDirection) PortInfo {
	return PortInfo{
		TypeInfo:  NewTypeInfoAnyAllowed(),
		direction: direction,
	}
}

func NewPortInfoTyped(direction PortDirection, ti TypeInfo) PortInfo {
	return PortInfo{
		TypeInfo:  ti,
		direction: direction,
	}
}

func (pi PortInfo) Direction() PortDirection {
	return pi.direction
}

func (pi *PortInfo) SetDescription(desc string) {
	pi.description = desc
}

func (pi PortInfo) Description() string {
	return pi.description
}

func (pi *PortInfo) SetDefaultValue(v Any) {
	pi.defaultValue = v
	pi.defaultValueStr = fmt.Sprintf("%v", v.Interface())
}

func (pi PortInfo) DefaultValue() Any {
	return pi.defaultValue
}

func (pi PortInfo) DefaultValueString() string {
	return pi.defaultValueStr
}

// PortsList is a map of port name to PortInfo.
type PortsList map[string]PortInfo

// PortsRemapping maps port names to remapping expressions.
type PortsRemapping map[string]string

// NonPortAttributes maps attribute names to values that are not ports.
type NonPortAttributes map[string]string

// ScriptingEnumsRegistry stores enum name to integer value mappings.
type ScriptingEnumsRegistry map[string]int

// TreeNodeManifest contains type information for a registered node type.
type TreeNodeManifest struct {
	Type           NodeType
	RegistrationID string
	Ports          PortsList
	Metadata       KeyValueVector
}

// NodeConfig contains configuration passed to node constructors.
type NodeConfig struct {
	// Blackboard used by this node
	Blackboard *Blackboard
	// Enums available for scripting
	Enums *ScriptingEnumsRegistry
	// Input ports remapping
	InputPorts PortsRemapping
	// Output ports remapping
	OutputPorts PortsRemapping
	// Other attributes from XML not parsed as ports
	OtherAttributes NonPortAttributes
	// Pointer to the manifest
	Manifest *TreeNodeManifest
	// Numeric unique identifier
	UID uint16
	// Hierarchical path including subtrees
	Path string
	// Pre and post conditions
	PreConditions  map[PreCond]string
	PostConditions map[PostCond]string
	// FromXML is set to true when this config comes from XML parsing.
	// The factory uses this to decide whether to apply default port remapping.
	FromXML bool
}

// NewNodeConfig creates a new NodeConfig with initialized maps.
func NewNodeConfig() NodeConfig {
	return NodeConfig{
		InputPorts:      make(PortsRemapping),
		OutputPorts:     make(PortsRemapping),
		OtherAttributes: make(NonPortAttributes),
		PreConditions:   make(map[PreCond]string),
		PostConditions:  make(map[PostCond]string),
	}
}

// CreatePort creates a port with the given direction, name, and type.
func CreatePort[T any](direction PortDirection, name string, description string) (string, PortInfo) {
	ti := NewTypeInfo[T]()
	pi := NewPortInfoTyped(direction, ti)
	if description != "" {
		pi.SetDescription(description)
	}
	return name, pi
}

// InputPort creates an INPUT port.
func InputPort[T any](name string, description string) (string, PortInfo) {
	return CreatePort[T](INPUT, name, description)
}

// OutputPort creates an OUTPUT port.
func OutputPort[T any](name string, description string) (string, PortInfo) {
	return CreatePort[T](OUTPUT, name, description)
}

// BidirectionalPort creates an INOUT port.
func BidirectionalPort[T any](name string, description string) (string, PortInfo) {
	return CreatePort[T](INOUT, name, description)
}

// InputPortWithDefault creates an INPUT port with a default value.
func InputPortWithDefault[T any](name string, defaultValue string, description string) (string, PortInfo) {
	key, pi := CreatePort[T](INPUT, name, description)
	if defaultValue != "" {
		pi.SetDefaultValue(AnyOf(defaultValue))
	}
	return key, pi
}

// OutputPortWithDefault creates an OUTPUT port with a default blackboard entry.
func OutputPortWithDefault[T any](name string, defaultBlackboardEntry string, description string) (string, PortInfo) {
	if defaultBlackboardEntry == "" || (defaultBlackboardEntry[0] != '{' || defaultBlackboardEntry[len(defaultBlackboardEntry)-1] != '}') {
		panic("Output port can only refer to blackboard entries, i.e. use the syntax '{port_name}'")
	}
	key, pi := CreatePort[T](OUTPUT, name, description)
	pi.SetDefaultValue(AnyOf(defaultBlackboardEntry))
	return key, pi
}

// BidirectionalPortWithDefault creates an INOUT port with a default value.
func BidirectionalPortWithDefault[T any](name string, defaultValue string, description string) (string, PortInfo) {
	key, pi := CreatePort[T](INOUT, name, description)
	if defaultValue != "" {
		pi.SetDefaultValue(AnyOf(defaultValue))
	}
	return key, pi
}

// IsAllowedPortName checks if the port name is valid.
func IsAllowedPortName(name string) bool {
	if len(name) == 0 {
		return false
	}
	c := name[0]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_') {
		return false
	}
	if FindForbiddenChar(name) != 0 {
		return false
	}
	return !IsReservedAttribute(name)
}

// FindForbiddenChar returns the first forbidden character or 0 if valid.
func FindForbiddenChar(name string) byte {
	forbidden := []byte{' ', '\t', '\n', '\r', '<', '>', '&', '"', '\'', '/', '\\', ':', '*', '?', '|', '.'}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c <= 31 || c == 127 {
			return c
		}
		for _, f := range forbidden {
			if c == f {
				return c
			}
		}
	}
	return 0
}

// IsReservedAttribute checks if an attribute name is reserved.
func IsReservedAttribute(name string) bool {
	for _, pre := range PreCondNames {
		if name == pre {
			return true
		}
	}
	for _, post := range PostCondNames {
		if name == post {
			return true
		}
	}
	return name == "name" || name == "ID" || name == "_autoremap"
}

// StartWith checks if a string starts with a prefix.
func StartWith(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// StartWithByte checks if a string starts with a specific byte.
func StartWithByte(s string, prefix byte) bool {
	return len(s) > 0 && s[0] == prefix
}

// SplitString splits a string by a delimiter.
func SplitString(s string, delimiter byte) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == delimiter {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}

// hasStaticMethodProvidedPorts checks if a type has a static ProvidedPorts method.
// We use a runtime check via reflection.

// GetProvidedPorts returns the PortsList from a type that implements PortProvider.
func GetProvidedPorts[T PortProvider]() PortsList {
	var zero T
	return zero.ProvidedPorts()
}

// PortProvider is an interface for types that provide a static port list.
type PortProvider interface {
	ProvidedPorts() PortsList
}

// assignDefaultRemapping fills NodeConfig with default remappings from provided ports.
func AssignDefaultRemapping(config *NodeConfig, ports PortsList) {
	for portName, portInfo := range ports {
		direction := portInfo.Direction()
		if direction != OUTPUT {
			config.InputPorts[portName] = "{=}"
		}
		if direction != INPUT {
			config.OutputPorts[portName] = "{=}"
		}
	}
}

// IsBlackboardPointer checks if a string is a blackboard pointer like "{key}".
func IsBlackboardPointer(s string) (bool, string) {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return false, ""
	}
	if s[0] == '{' && s[len(s)-1] == '}' {
		return true, s[1 : len(s)-1]
	}
	return false, ""
}

// StripBlackboardPointer removes the curly braces from a blackboard pointer.
func StripBlackboardPointer(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 3 && s[0] == '{' && s[len(s)-1] == '}' {
		return s[1 : len(s)-1]
	}
	return ""
}

// GetRemappedKey resolves a remapped port to its actual blackboard key.
func GetRemappedKey(portName, remappedPort string) (string, bool) {
	remappedPort = strings.TrimSpace(remappedPort)
	if remappedPort == "{=}" || remappedPort == "=" {
		return portName, true
	}
	if ok, key := IsBlackboardPointer(remappedPort); ok {
		return key, true
	}
	return "", false
}
