package core

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
)

// Any is a type-erased value container that stores an interface{} along
// with type metadata. It mirrors C++ BT::Any.
//
// Integral types are normalized to int64 internally; floating-point types
// are normalized to float64. The original type is tracked via reflection
// so that type identity is preserved.
type Any struct {
	value        interface{}
	originalType reflect.Type
}

// AnyOf creates an Any from any value.
func AnyOf(v interface{}) Any {
	if v == nil {
		return Any{}
	}
	switch val := v.(type) {
	case int:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case int8:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case int16:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case int32:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case int64:
		return Any{value: val, originalType: reflect.TypeOf(val)}
	case uint:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case uint8:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case uint16:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case uint32:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case uint64:
		return Any{value: int64(val), originalType: reflect.TypeOf(val)}
	case float32:
		return Any{value: float64(val), originalType: reflect.TypeOf(val)}
	case float64:
		return Any{value: val, originalType: reflect.TypeOf(val)}
	case string:
		return Any{value: val, originalType: reflect.TypeOf("")}
	case bool:
		return Any{value: val, originalType: reflect.TypeOf(true)}
	default:
		return Any{value: val, originalType: reflect.TypeOf(val)}
	}
}

// AnyOfType creates an empty Any with a given type.
func AnyOfType(t reflect.Type) Any {
	return Any{originalType: t}
}

// IsString returns true if the stored value is a string.
func (a Any) IsString() bool {
	_, ok := a.value.(string)
	return ok
}

// IsNumber returns true if the stored value is a number.
func (a Any) IsNumber() bool {
	switch a.value.(type) {
	case int64, uint64, float64:
		return true
	}
	return false
}

// IsIntegral returns true if the stored value is an integer.
func (a Any) IsIntegral() bool {
	switch a.value.(type) {
	case int64, uint64:
		return true
	}
	return false
}

// IsEmpty returns true if the Any holds no value.
func (a Any) IsEmpty() bool {
	return a.value == nil
}

// IsBool returns true if the stored value is a bool.
func (a Any) IsBool() bool {
	_, ok := a.value.(bool)
	return ok
}

// Type returns the original type (before normalization).
func (a Any) Type() reflect.Type {
	return a.originalType
}

// Interface returns the stored value as interface{}.
func (a Any) Interface() interface{} {
	return a.value
}

// CastType returns the actual stored type (after normalization).
func (a Any) CastType() reflect.Type {
	if a.value == nil {
		return nil
	}
	return reflect.TypeOf(a.value)
}

// CopyInto copies the value into another Any, preserving destination type.
func (a Any) CopyInto(dst *Any) error {
	if dst == nil {
		return fmt.Errorf("Any::CopyInto: destination is nil")
	}
	if dst.value == nil {
		*dst = a
		return nil
	}

	dstCt := reflect.TypeOf(dst.value)
	srcCt := reflect.TypeOf(a.value)

	if srcCt == dstCt || (a.IsString() && dst.IsString()) {
		dst.value = a.value
		return nil
	}

	if a.IsNumber() && dst.IsNumber() {
		switch dstCt.Kind() {
		case reflect.Int64:
			v, err := a.ToInt64()
			if err != nil {
				return err
			}
			dst.value = v
		case reflect.Float64:
			v, err := a.ToFloat64()
			if err != nil {
				return err
			}
			dst.value = v
		default:
			return fmt.Errorf("Any::CopyInto: unexpected destination numeric type")
		}
		return nil
	}

	// numeric → bool: 0/1 converts to false/true (C++ compatible)
	if a.IsNumber() && dst.IsBool() {
		f, err := a.ToFloat64()
		if err != nil {
			return err
		}
		dst.value = f != 0.0
		return nil
	}

	// bool → numeric: false/true converts to 0/1 (C++ compatible)
	if a.IsBool() && dst.IsNumber() {
		if a.value.(bool) {
			switch dstCt.Kind() {
			case reflect.Int64:
				dst.value = int64(1)
			case reflect.Float64:
				dst.value = float64(1)
			default:
				return fmt.Errorf("Any::CopyInto: unexpected destination numeric type")
			}
		} else {
			switch dstCt.Kind() {
			case reflect.Int64:
				dst.value = int64(0)
			case reflect.Float64:
				dst.value = float64(0)
			default:
				return fmt.Errorf("Any::CopyInto: unexpected destination numeric type")
			}
		}
		return nil
	}

	return fmt.Errorf("Any::CopyInto: cannot copy between incompatible types")
}

// ToInt64 converts the stored value to int64.
func (a Any) ToInt64() (int64, error) {
	switch v := a.value.(type) {
	case int64:
		return v, nil
	case uint64:
		return int64(v), nil
	case float64:
		if v != float64(int64(v)) {
			return 0, fmt.Errorf("Any: cannot convert non-integer float %v to int64", v)
		}
		return int64(v), nil
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			f, err2 := strconv.ParseFloat(v, 64)
			if err2 != nil {
				return 0, fmt.Errorf("Any: cannot convert string to int64: %s", v)
			}
			return int64(f), nil
		}
		return n, nil
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	}
	return 0, fmt.Errorf("Any: cannot convert %T to int64", a.value)
}

// ToUint64 converts the stored value to uint64.
func (a Any) ToUint64() (uint64, error) {
	switch v := a.value.(type) {
	case uint64:
		return v, nil
	case int64:
		return uint64(v), nil
	case float64:
		if v != float64(int64(v)) {
			return 0, fmt.Errorf("Any: cannot convert non-integer float %v to uint64", v)
		}
		return uint64(v), nil
	case string:
		n, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("Any: cannot convert string to uint64: %s", v)
		}
		return n, nil
	}
	return 0, fmt.Errorf("Any: cannot convert %T to uint64", a.value)
}

// ToFloat64 converts the stored value to float64.
func (a Any) ToFloat64() (float64, error) {
	switch v := a.value.(type) {
	case float64:
		return v, nil
	case int64:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("Any: cannot convert string to float64: %s", v)
		}
		return f, nil
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	}
	return 0, fmt.Errorf("Any: cannot convert %T to float64", a.value)
}

// ToString converts the stored value to string.
func (a Any) ToString() (string, error) {
	switch v := a.value.(type) {
	case string:
		return v, nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case uint64:
		return strconv.FormatUint(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		if v {
			return "1", nil
		}
		return "0", nil
	}
	return "", fmt.Errorf("Any: cannot convert %T to string", a.value)
}

// ToBool converts the stored value to bool.
func (a Any) ToBool() (bool, error) {
	switch v := a.value.(type) {
	case bool:
		return v, nil
	case string:
		switch v {
		case "true", "TRUE", "1":
			return true, nil
		case "false", "FALSE", "0":
			return false, nil
		}
		return false, fmt.Errorf("Any: cannot convert string to bool: %s", v)
	case int64:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case float64:
		return v != 0.0, nil
	}
	return false, fmt.Errorf("Any: cannot convert %T to bool", a.value)
}

// Cast tries to cast the stored value to the destination type.
// It mirrors C++ Any::cast<T>().
func Cast[T any](a Any) (T, error) {
	var zero T
	targetType := reflect.TypeOf(zero)

	// Direct type match
	if a.originalType != nil && a.originalType == targetType {
		if v, ok := a.value.(T); ok {
			return v, nil
		}
	}

	// Try direct cast first
	if v, ok := a.value.(T); ok {
		return v, nil
	}

	// Handle type conversions
	switch targetType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := a.ToInt64()
		if err != nil {
			return zero, err
		}
		rv := reflect.ValueOf(v).Convert(targetType)
		return rv.Interface().(T), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := a.ToUint64()
		if err != nil {
			return zero, err
		}
		rv := reflect.ValueOf(v).Convert(targetType)
		return rv.Interface().(T), nil

	case reflect.Float32, reflect.Float64:
		v, err := a.ToFloat64()
		if err != nil {
			return zero, err
		}
		rv := reflect.ValueOf(v).Convert(targetType)
		return rv.Interface().(T), nil

	case reflect.String:
		v, err := a.ToString()
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil

	case reflect.Bool:
		v, err := a.ToBool()
		if err != nil {
			return zero, err
		}
		return any(v).(T), nil
	}

	return zero, fmt.Errorf("Any::Cast: cannot cast from %v to %v", a.originalType, targetType)
}

// MustCast casts and panics on failure.
func MustCast[T any](a Any) T {
	v, err := Cast[T](a)
	if err != nil {
		panic(err)
	}
	return v
}

// IsCastingSafe checks if a value can be safely cast to a given type.
func IsCastingSafe(targetType reflect.Type, value interface{}) bool {
	if targetType == nil {
		return false
	}
	val := reflect.ValueOf(value)
	switch targetType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if !val.CanInt() && !val.CanFloat() {
			return false
		}
		var v int64
		if val.CanInt() {
			v = val.Int()
		} else {
			v = int64(val.Float())
		}
		// Check range
		converted := reflect.ValueOf(v).Convert(targetType)
		back := converted.Int()
		return back == v
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if !val.CanInt() && !val.CanFloat() {
			return false
		}
		var v uint64
		if val.CanUint() {
			v = val.Uint()
		} else if val.CanInt() {
			if val.Int() < 0 {
				return false
			}
			v = uint64(val.Int())
		} else {
			f := val.Float()
			if f < 0 || f != math.Trunc(f) {
				return false
			}
			v = uint64(f)
		}
		converted := reflect.ValueOf(v).Convert(targetType)
		back := converted.Uint()
		return back == v
	case reflect.Float32, reflect.Float64:
		if val.CanFloat() {
			f := val.Float()
			converted := reflect.ValueOf(f).Convert(targetType)
			back := converted.Float()
			// Check precision
			if targetType.Kind() == reflect.Float32 {
				back32 := float32(back)
				return float64(back32) == f || math.Abs(f-float64(back32))/math.Max(1, math.Abs(f)) < 1e-6
			}
			return true
		}
		if val.CanInt() {
			return true
		}
		return false
	}
	return true
}

// ValidCast checks if a value can be safely converted between arithmetic types.
func ValidCast[SRC, TO any](val SRC) bool {
	var srcZero SRC
	var toZero TO
	srcType := reflect.TypeOf(srcZero)
	toType := reflect.TypeOf(toZero)

	if srcType.Kind() == reflect.Float32 || srcType.Kind() == reflect.Float64 {
		fv := reflect.ValueOf(val).Float()
		if toType.Kind() == reflect.Float32 || toType.Kind() == reflect.Float64 {
			return true
		}
		// Float to integral: check truncation
		if fv != math.Trunc(fv) {
			return false
		}
	}

	if srcType.Kind() == reflect.Int64 || srcType.Kind() == reflect.Int {
		sv := reflect.ValueOf(val).Int()
		if toType.Kind() == reflect.Float32 || toType.Kind() == reflect.Float64 {
			// Check if integral value can be represented exactly
			fv := float64(sv)
			back := int64(fv)
			return back == sv
		}
	}

	return true
}
