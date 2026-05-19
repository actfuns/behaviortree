package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// AnyPtrLocked holds a locked reference to a Blackboard entry's value.
type AnyPtrLocked struct {
	entry  *BlackboardEntry
	unlock func()
	locked bool
}

// Get returns the Any value (must call Release when done).
func (l *AnyPtrLocked) Get() *Any {
	if l.entry != nil {
		l.locked = true
		return &l.entry.value
	}
	return nil
}

// Release unlocks the entry.
func (l *AnyPtrLocked) Release() {
	if l.locked && l.entry != nil {
		l.locked = false
	}
}

// StampedValue holds a value with its timestamp.
type StampedValue struct {
	Value Any
	Stamp Timestamp
}

// BlackboardEntry is a single entry in the blackboard.
type BlackboardEntry struct {
	value      Any
	info       TypeInfo
	stringConv StringConverter
	sequenceID uint64
	stamp      time.Time
}

func (e *BlackboardEntry) SequenceID() uint64 {
	return e.sequenceID
}

func (e *BlackboardEntry) SetSequenceID(id uint64) {
	e.sequenceID = id
}

func (e *BlackboardEntry) Stamp() time.Time {
	return e.stamp
}

// Info returns the TypeInfo of the entry.
func (e *BlackboardEntry) Info() TypeInfo {
	return e.info
}

// StringConv returns the string converter of the entry.
func (e *BlackboardEntry) StringConv() StringConverter {
	return e.stringConv
}

// GetValue returns a copy of the entry's value (must be locked).
func (e *BlackboardEntry) GetValue() *Any {
	if e.value.IsEmpty() {
		return nil
	}
	val := e.value
	return &val
}

// SetValue sets the entry's value (must be locked).
func (e *BlackboardEntry) SetValue(val Any) {
	e.value = val
	e.sequenceID++
	e.stamp = time.Now()
}

// Blackboard is the mechanism used to exchange typed data between nodes.
type Blackboard struct {
	storage            map[string]*BlackboardEntry
	parent             *Blackboard
	internalToExternal map[string]string
	autoRemapping      bool
}

// NewBlackboard creates a new Blackboard with an optional parent.
func NewBlackboard(parent *Blackboard) *Blackboard {
	return &Blackboard{
		storage:            make(map[string]*BlackboardEntry),
		parent:             parent,
		internalToExternal: make(map[string]string),
	}
}

// EnableAutoRemapping enables automatic remapping to parent blackboard.
func (bb *Blackboard) EnableAutoRemapping(remapping bool) {
	bb.autoRemapping = remapping
}

// GetEntry returns the entry for a given key.
func (bb *Blackboard) GetEntry(key string) *BlackboardEntry {
	// Special syntax: "@" refers to root BB
	if len(key) > 0 && key[0] == '@' {
		return bb.RootBlackboard().GetEntry(key[1:])
	}

	entry, ok := bb.storage[key]
	if ok {
		return entry
	}

	// Try remapping
	if bb.parent != nil {
		if remappedKey, ok := bb.internalToExternal[key]; ok {
			return bb.parent.GetEntry(remappedKey)
		}
		if bb.autoRemapping && !isPrivateKey(key) {
			return bb.parent.GetEntry(key)
		}
	}

	return nil
}

// isPrivateKey checks if a key starts with '_'.
func isPrivateKey(key string) bool {
	return len(key) > 0 && key[0] == '_'
}

// GetAnyLocked returns a locked reference to an entry's value.
func (bb *Blackboard) GetAnyLocked(key string) *AnyPtrLocked {
	entry := bb.GetEntry(key)
	if entry == nil {
		return nil
	}
	return &AnyPtrLocked{entry: entry}
}

// Get retrieves a value from the blackboard.
func (bb *Blackboard) Get(key string, dest interface{}) (bool, error) {
	entry := bb.GetEntry(key)
	if entry == nil {
		return false, nil
	}


	if entry.value.IsEmpty() {
		return false, nil
	}

	return true, assignValue(entry.value, dest)
}

// GetTyped retrieves a typed value from the blackboard.
func GetTyped[T any](bb *Blackboard, key string) (T, error) {
	var result T
	found, err := bb.Get(key, &result)
	if err != nil {
		return result, err
	}
	if !found {
		return result, fmt.Errorf("Blackboard::get() error. Missing key [%s]", key)
	}
	return result, nil
}

// GetInto retrieves a value into an existing destination pointer.
func (bb *Blackboard) GetInto(key string, dest interface{}) error {
	entry := bb.GetEntry(key)
	if entry == nil {
		return fmt.Errorf("Blackboard::get() error. Missing key [%s]", key)
	}


	if entry.value.IsEmpty() {
		return fmt.Errorf("Blackboard::get() error. Entry [%s] hasn't been initialized, yet", key)
	}

	return assignValue(entry.value, dest)
}

// GetStamped retrieves a value with its timestamp.
func (bb *Blackboard) GetStamped(key string, dest interface{}) (*Timestamp, error) {
	entry := bb.GetEntry(key)
	if entry == nil {
		return nil, fmt.Errorf("Blackboard::getStamped() error. Missing key [%s]", key)
	}


	if entry.value.IsEmpty() {
		return nil, fmt.Errorf("Blackboard::getStamped() error. Entry [%s] hasn't been initialized, yet", key)
	}

	err := assignValue(entry.value, dest)
	if err != nil {
		return nil, err
	}
	return &Timestamp{
		Seq:   entry.sequenceID,
		Stamp: entry.stamp.UnixNano(),
	}, nil
}

// Set stores a value in the blackboard.
func (bb *Blackboard) Set(key string, value interface{}) error {
	// Handle "@" prefix for root blackboard access
	if len(key) > 0 && key[0] == '@' {
		return bb.RootBlackboard().Set(key[1:], value)
	}

	// Detect if value is already an Any (matches C++ set<BT::Any> special case)
	var anyVal Any
	var valueType reflect.Type
	isAnyValue := false
	if existingAny, ok := value.(Any); ok {
		anyVal = existingAny
		isAnyValue = true
		valueType = reflect.TypeOf(Any{})
	} else {
		anyVal = AnyOf(value)
		valueType = reflect.TypeOf(value)
	}

	existingEntry, ok := bb.storage[key]

	if !ok {
		// Create new entry
		var portInfo PortInfo
		if valueType.Kind() == reflect.String && !isAnyValue {
			portInfo = NewPortInfo(INOUT)
		} else {
			portInfo = NewPortInfoTyped(INOUT, newTypeInfoFromReflectWithConv(valueType))
		}
		entry, err := bb.createEntryImpl(key, portInfo)
		if err != nil {
			return err
		}

		entry.value = anyVal
		entry.sequenceID++
		entry.stamp = time.Now()
		return nil
	} else {
		// Update existing entry — matches C++ set<T>() update path

		previousAny := &existingEntry.value

		// Special case: entry exists but is not strongly typed yet
		if !existingEntry.info.IsStronglyTyped() {
			if !isAnyValue {
				existingEntry.info = newTypeInfoFromReflectWithConv(valueType)
			}
			existingEntry.sequenceID++
			existingEntry.stamp = time.Now()
			*previousAny = anyVal
			return nil
		}

		previousType := existingEntry.info.Type()

		// Check type mismatch
		anyCastType := anyVal.CastType()
		if previousType != valueType && (anyCastType == nil || previousType != anyCastType) {
			mismatching := true

			// If value is a string, try parsing it to the entry's type
			if str, ok := value.(string); ok {
				if existingEntry.info.Converter() != nil {
					parsed, parseErr := existingEntry.info.ParseString(str)
					if parseErr == nil && !parsed.IsEmpty() {
						mismatching = false
						anyVal = parsed
					}
				}
			}

			// Check safe numeric cast between arithmetic types
			if mismatching && isArithmeticType(valueType) {
				if IsCastingSafe(previousType, value) {
					mismatching = false
				}
			}

			if mismatching {
				return fmt.Errorf("Blackboard::set(%s): once declared, "+
					"the type of a port shall not change. "+
					"Previously declared type [%s], current type [%s]",
					key, previousType.String(), valueType.String())
			}
		}

		// Set the value
		if isAnyValue {
			*previousAny = anyVal
		} else {
			if err := anyVal.CopyInto(previousAny); err != nil {
				return fmt.Errorf("Blackboard::set(%s): %s", key, err.Error())
			}
		}
		existingEntry.sequenceID++
		existingEntry.stamp = time.Now()
		return nil
	}
}

// Unset removes a key from the blackboard.
func (bb *Blackboard) Unset(key string) {
	delete(bb.storage, key)
}

// HasKey returns true if the key exists in the blackboard.
func (bb *Blackboard) HasKey(key string) bool {
	_, ok := bb.storage[key]
	return ok
}

// GetKeys returns all keys in the blackboard.
func (bb *Blackboard) GetKeys() []string {
	keys := make([]string, 0, len(bb.storage))
	for k := range bb.storage {
		keys = append(keys, k)
	}
	return keys
}

// Clear removes all entries.
func (bb *Blackboard) Clear() {
	bb.storage = make(map[string]*BlackboardEntry)
}

// AddSubtreeRemapping adds a remapping from internal to external key.
func (bb *Blackboard) AddSubtreeRemapping(internal, external string) {
	bb.internalToExternal[internal] = external
}

// Parent returns the parent blackboard.
func (bb *Blackboard) Parent() *Blackboard {
	return bb.parent
}

// RootBlackboard returns the root blackboard in the hierarchy.
func (bb *Blackboard) RootBlackboard() *Blackboard {
	current := bb
	for current.parent != nil {
		current = current.parent
	}
	return current
}

// CreateEntry creates a new entry with the given type info.
func (bb *Blackboard) CreateEntry(key string, info PortInfo) (*BlackboardEntry, error) {
	if len(key) > 0 && key[0] == '@' {
		if strings.ContainsRune(key[1:], '@') {
			return nil, fmt.Errorf("Character '@' used multiple times in the key")
		}
		return bb.RootBlackboard().createEntryImpl(key[1:], info)
	} else {
		return bb.createEntryImpl(key, info)
	}
}

func (bb *Blackboard) createEntryImpl(key string, info PortInfo) (*BlackboardEntry, error) {

	// Check if already exists
	if entry, ok := bb.storage[key]; ok {
		prevInfo := entry.info
		if prevInfo.Type() != info.Type() && prevInfo.IsStronglyTyped() && info.IsStronglyTyped() {
			return nil, fmt.Errorf("Blackboard entry [%s]: once declared, the type of a port"+
				" shall not change. Previously declared type [%s], current type [%s]",
				key, prevInfo.TypeName(), info.TypeName())
		}
		return entry, nil
	}

	// Check remapping
	if remappedKey, ok := bb.internalToExternal[key]; ok {
		if bb.parent != nil {
			return bb.parent.createEntryImpl(remappedKey, info)
		}
		return nil, fmt.Errorf("Missing parent blackboard")
	}

	// Auto-remapping
	if bb.autoRemapping && !isPrivateKey(key) {
		if bb.parent != nil {
			return bb.parent.createEntryImpl(key, info)
		}
		return nil, fmt.Errorf("Missing parent blackboard")
	}

	// Create locally
	entry := &BlackboardEntry{
		info:  info.TypeInfo,
		value: AnyOfType(info.Type()),
	}
	bb.storage[key] = entry
	return entry, nil
}

// isArithmeticType returns true if the reflect.Type is an arithmetic type.
func isArithmeticType(t reflect.Type) bool {
	if t == nil {
		return false
	}
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// newTypeInfoFromReflectWithConv creates a TypeInfo from reflect.Type with a string converter.
func newTypeInfoFromReflectWithConv(t reflect.Type) TypeInfo {
	return TypeInfo{
		typeRfl: t,
		typeStr: t.String(),
		conv:    stringConverterForKind(t.Kind()),
	}
}

// stringConverterForKind returns a StringConverter for a given reflect.Kind.
func stringConverterForKind(k reflect.Kind) StringConverter {
	switch k {
	case reflect.String:
		return func(s string) (Any, error) { return AnyOf(s), nil }
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return Any{}, fmt.Errorf("Can't convert string [%s] to integer", s)
			}
			return AnyOf(v), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return Any{}, fmt.Errorf("Can't convert string [%s] to unsigned integer", s)
			}
			return AnyOf(v), nil
		}
	case reflect.Float32, reflect.Float64:
		return func(s string) (Any, error) {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return Any{}, fmt.Errorf("Can't convert string [%s] to double", s)
			}
			return AnyOf(v), nil
		}
	case reflect.Bool:
		return func(s string) (Any, error) {
			switch strings.ToLower(s) {
			case "true", "1":
				return AnyOf(true), nil
			case "false", "0":
				return AnyOf(false), nil
			default:
				return Any{}, fmt.Errorf("Can't convert string [%s] to bool", s)
			}
		}
	default:
		return nil
	}
}

// DebugMessage prints debug information about the blackboard.
func (bb *Blackboard) DebugMessage() {
	for key, entry := range bb.storage {
		portType := entry.info.Type()
		typeName := "unknown"
		if portType != nil {
			typeName = portType.String()
		}
		fmt.Printf("%s (%s)\n", key, typeName)
	}
	for from, to := range bb.internalToExternal {
		fmt.Printf("[%s] remapped to port of parent tree [%s]\n", from, to)
	}
}

// CloneInto copies values from this blackboard into another.
// Known limitations: it doesn't update the remapping in dst, it doesn't change
// the parent blackboard of dst.
func (bb *Blackboard) CloneInto(dst *Blackboard) {
	type copyTask struct {
		key      string
		srcEntry *BlackboardEntry
		dstEntry *BlackboardEntry // nil when a new entry is needed
	}

	var tasks []copyTask
	var keysToRemove []string

	// Step 1: snapshot src/dst entries under both storage mutexes.
	{

		// Build set of dst keys
		dstKeys := make(map[string]bool)
		for key := range dst.storage {
			dstKeys[key] = true
		}

		for srcKey, srcEntry := range bb.storage {
			delete(dstKeys, srcKey)
			if dstEntry, ok := dst.storage[srcKey]; ok {
				tasks = append(tasks, copyTask{key: srcKey, srcEntry: srcEntry, dstEntry: dstEntry})
			} else {
				tasks = append(tasks, copyTask{key: srcKey, srcEntry: srcEntry, dstEntry: nil})
			}
		}

		for key := range dstKeys {
			keysToRemove = append(keysToRemove, key)
		}

	}
	// storage mutexes released here

	// Step 2: copy entry data under entry_mutex only.
	type newEntry struct {
		key   string
		entry *BlackboardEntry
	}
	var newEntries []newEntry

	for _, task := range tasks {
		if task.dstEntry != nil {
			// overwrite existing entry — lock both entry mutexes
			val := task.srcEntry.value
			info := task.srcEntry.info
			conv := task.srcEntry.stringConv

			task.dstEntry.value = val
			task.dstEntry.info = info
			task.dstEntry.stringConv = conv
			task.dstEntry.sequenceID++
			task.dstEntry.stamp = time.Now()
		} else {
			// create new entry from src
			newEnt := &BlackboardEntry{
				value:      task.srcEntry.value,
				info:       task.srcEntry.info,
				stringConv: task.srcEntry.stringConv,
			}
			newEntries = append(newEntries, newEntry{key: task.key, entry: newEnt})
		}
	}

	// Step 3: insert new entries and remove stale ones under dst storage mutex.
	if len(newEntries) > 0 || len(keysToRemove) > 0 {
		for _, ne := range newEntries {
			if _, exists := dst.storage[ne.key]; !exists {
				dst.storage[ne.key] = ne.entry
			}
		}
		for _, key := range keysToRemove {
			delete(dst.storage, key)
		}
	}
}
