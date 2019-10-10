package reflects

import (
	"fmt"
	"reflect"
)

func IsPtr(a interface{}) bool {
	return reflect.TypeOf(a).Kind() == reflect.Ptr
}

func IsBool(a interface{}) bool {
	return reflect.TypeOf(a).Kind() == reflect.Bool
}

func IsNumber(a interface{}) bool {
	if a == nil {
		return false
	}
	kind := reflect.TypeOf(a).Kind()
	return reflect.Int <= kind && kind <= reflect.Float64
}

func IsInteger(a interface{}) bool {
	kind := reflect.TypeOf(a).Kind()
	return reflect.Int <= kind && kind <= reflect.Int64
}

func IsUnsignedInteger(a interface{}) bool {
	kind := reflect.TypeOf(a).Kind()
	return reflect.Uint <= kind && kind <= reflect.Uint64
}

func IsFloat(a interface{}) bool {
	kind := reflect.TypeOf(a).Kind()
	return reflect.Float32 <= kind && kind <= reflect.Float64
}

func ToInteger(a interface{}) int64 {
	if IsInteger(a) {
		return reflect.ValueOf(a).Int()
	} else if IsUnsignedInteger(a) {
		return int64(reflect.ValueOf(a).Uint())
	} else if IsFloat(a) {
		return int64(reflect.ValueOf(a).Float())
	} else {
		panic(fmt.Errorf("Expected a number!  Got <%T> %#v", a, a))
	}
}

func ToUnsignedInteger(a interface{}) uint64 {
	if IsInteger(a) {
		return uint64(reflect.ValueOf(a).Int())
	} else if IsUnsignedInteger(a) {
		return reflect.ValueOf(a).Uint()
	} else if IsFloat(a) {
		return uint64(reflect.ValueOf(a).Float())
	} else {
		panic(fmt.Errorf("Expected a number!  Got <%T> %#v", a, a))
	}
}

func ToFloat(a interface{}) float64 {
	if IsInteger(a) {
		return float64(reflect.ValueOf(a).Int())
	} else if IsUnsignedInteger(a) {
		return float64(reflect.ValueOf(a).Uint())
	} else if IsFloat(a) {
		return reflect.ValueOf(a).Float()
	} else {
		panic(fmt.Errorf("Expected a number!  Got <%T> %#v", a, a))
	}
}

func IsError(a interface{}) bool {
	_, ok := a.(error)
	return ok
}

func IsChan(a interface{}) bool {
	if IsNil(a) {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.Chan
}

func IsMap(a interface{}) bool {
	if a == nil {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.Map
}

func IsArrayOrSlice(a interface{}) bool {
	if a == nil {
		return false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Array, reflect.Slice:
		return true
	default:
		return false
	}
}

func IsString(a interface{}) bool {
	if a == nil {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.String
}

func IsFunc(a interface{}) bool {
	if IsNil(a) {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.Func
}

func ToString(a interface{}) (string, bool) {
	aString, isString := a.(string)
	if isString {
		return aString, true
	}

	aBytes, isBytes := a.([]byte)
	if isBytes {
		return string(aBytes), true
	}

	aStringer, isStringer := a.(fmt.Stringer)
	if isStringer {
		return aStringer.String(), true
	}

	return "", false
}

func LengthOf(a interface{}) (int, bool) {
	if a == nil {
		return 0, false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Map, reflect.Array, reflect.String, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(a).Len(), true
	default:
		return 0, false
	}
}
func CapOf(a interface{}) (int, bool) {
	if a == nil {
		return 0, false
	}
	switch reflect.TypeOf(a).Kind() {
	case reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(a).Cap(), true
	default:
		return 0, false
	}
}

func IsNil(a interface{}) bool {
	if a == nil {
		return true
	}

	switch reflect.TypeOf(a).Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Interface:
		return reflect.ValueOf(a).IsNil()
	}

	return false
}
