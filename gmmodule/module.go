package gmmodule

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strconv"

	"github.com/zxfonline/misc/log"

	"github.com/zxfonline/misc/gerror"
)

var (
	h reflect.Value
)

//CastParam 数据格式转换
func CastParam(kind reflect.Kind, param string) (reflect.Value, error) {
	switch kind {
	case reflect.Int:
		val, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid int param:%s", param)
		}
		return reflect.ValueOf(int(val)), nil
	case reflect.Uint:
		val, err := strconv.ParseUint(param, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid uint param:%s", param)
		}
		return reflect.ValueOf(uint(val)), nil
	case reflect.Int8:
		val, err := strconv.ParseInt(param, 10, 8)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid int8 param:%s", param)
		}
		return reflect.ValueOf(int8(val)), nil
	case reflect.Uint8:
		val, err := strconv.ParseUint(param, 10, 8)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid uint8 param:%s", param)
		}
		return reflect.ValueOf(uint8(val)), nil
	case reflect.Int16:
		val, err := strconv.ParseInt(param, 10, 16)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid int16 param:%s", param)
		}
		return reflect.ValueOf(int16(val)), nil
	case reflect.Uint16:
		val, err := strconv.ParseUint(param, 10, 16)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid uint16 param:%s", param)
		}
		return reflect.ValueOf(uint16(val)), nil
	case reflect.Int32:
		val, err := strconv.ParseInt(param, 10, 32)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid int32 param:%s", param)
		}
		return reflect.ValueOf(int32(val)), nil
	case reflect.Uint32:
		val, err := strconv.ParseUint(param, 10, 32)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid uint32 param:%s", param)
		}
		return reflect.ValueOf(uint32(val)), nil
	case reflect.Int64:
		val, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid int64 param:%s", param)
		}
		return reflect.ValueOf(int64(val)), nil
	case reflect.Uint64:
		val, err := strconv.ParseUint(param, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid uint64 param:%s", param)
		}
		return reflect.ValueOf(uint64(val)), nil
	case reflect.Float32:
		val, err := strconv.ParseFloat(param, 32)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid float32 param:%s", param)
		}
		return reflect.ValueOf(float32(val)), nil
	case reflect.Float64:
		val, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid float64 param:%s", param)
		}
		return reflect.ValueOf(val), nil
	case reflect.Bool:
		val, err := strconv.ParseBool(param)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("invalid bool param:%s", param)
		}
		return reflect.ValueOf(val), nil
	case reflect.String:
		return reflect.ValueOf(param), nil
	default:
		return reflect.Value{}, fmt.Errorf("invalid kind param:%s", param)
	}
}

func call(f ast.Expr) (string, interface{}, error) {
	funcname := ""
	params := make([]string, 0)
	unaryExprs := make(map[token.Pos]token.Token)
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			val := x.Value
			if nVal, err := strconv.Unquote(val); err == nil {
				val = nVal
			}
			if op, present := unaryExprs[x.ValuePos]; present {
				switch op {
				case token.SUB:
					val = "-" + val
				}
			}
			params = append(params, val)
		case *ast.Ident:
			funcname = x.Name
		case *ast.UnaryExpr:
			v := n.(*ast.UnaryExpr)
			unaryExprs[v.X.Pos()] = v.Op
		}
		return true
	})
	fn := h.MethodByName(funcname)
	if fn.Kind() != reflect.Func {
		return funcname, nil, errors.New("cmd no found")
	}
	if fn.Type().NumIn() > len(params) {
		return funcname, nil, errors.New("cmd param not enough")
	}
	in := make([]reflect.Value, fn.Type().NumIn())
	for i := 0; i < fn.Type().NumIn(); i++ {
		val, err := CastParam(fn.Type().In(i).Kind(), params[i])
		if err == nil {
			in[i] = val
		} else {
			return funcname, nil, err
		}
	}
	ret := fn.Call(in)
	if len(ret) == 0 {
		return funcname, nil, nil
	}
	//默认支持两个返回参数(interface{},error)
	if len(ret) > 1 { //默认判定最后一个返回值为error类型
		if err, ok := ret[len(ret)-1].Interface().(error); ok {
			return funcname, nil, err
		}
	}
	//默认第一个返回值为结果
	return funcname, ret[0].Interface(), nil
}

//RegistHander 注册gm工具类
func RegistHander(r reflect.Value) {
	h = r
}

//HandleCMD 指令处理
func HandleCMD(exp string, ip string) (rs interface{}) {
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("HandleGM[exp=%s],ip:%v,err:%v,stack:%s", exp, ip, e, log.DumpStack())
			rs = gerror.NewError(gerror.SERVER_CMSG_ERROR, fmt.Sprintf("%v", e))
		}
	}()
	if f, perr := parser.ParseExpr(exp); perr == nil {
		funcn, resp, err := call(f)
		if err != nil {
			log.Warnf("HandleGM[exp=%s],ip:%v func:%s err:%+v.", exp, ip, funcn, err)
			switch err.(type) {
			case *gerror.SysError:
				rs = err.(*gerror.SysError)
			case error:
				rs = gerror.New(gerror.SERVER_CMSG_ERROR, err.(error))
			default:
				rs = gerror.NewError(gerror.SERVER_CMSG_ERROR, fmt.Sprintf("%v", err))
			}
		} else {
			log.Infof("HandleGM[exp=%s],ip:%v func:%s.", exp, ip, funcn)
			switch resp.(type) {
			case *gerror.SysError:
				rs = resp
			case error:
				rs = gerror.New(gerror.SERVER_CMSG_ERROR, resp.(error))
			default:
				if resp != nil {
					rs = &gerror.SysError{Code: gerror.OK, Data: resp}
				} else {
					rs = &gerror.SysError{Code: gerror.OK}
				}
			}
		}
	} else {
		log.Warnf("HandleGM([exp=%s],ip:%v,err:%+v.", exp, ip, perr)
		rs = gerror.NewError(gerror.SERVER_CMSG_ERROR, "parse exp error")
	}
	return
}
