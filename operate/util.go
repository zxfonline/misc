package operate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var RecoverPanicToErr = true

func PanicValToErr(panicVal interface{}, err *error) {
	if panicVal == nil {
		return
	}
	// case nil
	switch xerr := panicVal.(type) {
	case error:
		*err = xerr
	case string:
		*err = errors.New(xerr)
	default:
		*err = fmt.Errorf("%v", panicVal)
	}
	return
}

func PanicToErr(err *error) {
	if RecoverPanicToErr {
		if x := recover(); x != nil {
			//debug.PrintStack()
			PanicValToErr(x, err)
		}
	}
}

func M2float64(i interface{}) float64 {
	if i == nil {
		panic(errors.New("invalid args,nil interface to float64"))
	}
	switch t := i.(type) {
	case float64:
		return t
	case float32:
		return float64(t)
	case int64:
		return float64(t)
	case int32:
		return float64(t)
	case int16:
		return float64(t)
	case int8:
		return float64(t)
	case int:
		return float64(t)
	case byte:
		return float64(t)
	default:
		return Stof64(M2string(i))
	}
}

func M2int64(i interface{}) int64 {
	if i == nil {
		panic(errors.New("invalid args,nil interface to int64"))
	}
	switch t := i.(type) {
	case int64:
		return i.(int64)
	case int32:
		return int64(t)
	case int16:
		return int64(t)
	case int8:
		return int64(t)
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case float32:
		return int64(t)
	case byte:
		return int64(t)
	default:
		return Stoi64(M2string(i))
	}
}

func M2string(i interface{}) string {
	if i == nil {
		panic(errors.New("invalid args,nil interface to string"))
	}
	switch t := i.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", i)
	}
}

//没找到并且没有默认值则 panic抛错
func Stof64(v string, def ...float64) float64 {
	if val, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
		return val
	} else if len(def) > 0 {
		return def[0]
	} else {
		panic(fmt.Errorf("invalid float64 param:%v ,err:%v", v, err))
	}
}

//没找到并且没有默认值则 panic抛错
func Stoi64(v string, def ...int64) int64 {
	if n, err := strconv.ParseInt(strings.TrimSpace(v), 0, 0); err == nil {
		return n
	} else if len(def) > 0 {
		return def[0]
	} else {
		panic(fmt.Errorf("invalid int64 param:%v ,err:%v", v, err))
	}
}
