package gmmodule

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strconv"

	"github.com/zxfonline/misc/log"

	"github.com/zxfonline/misc/fileutil"
	"github.com/zxfonline/misc/gerror"
	"github.com/zxfonline/misc/timefix"

	lua "github.com/yuin/gopher-lua"
	luajson "layeh.com/gopher-json"
	luar "layeh.com/gopher-luar"
)

//GMHandler gm Lua指令实现接口
type GMHandler interface {
	LuaScript(fname string, params string) interface{}
	Lua(cmd string) interface{}
}

var (
	//GlobalLuaState 全局lua域
	GlobalLuaState *lua.LState

	_GmHandler GMHandler
	//lua文件目录
	_luapath string
)

//LuaLogf 打印日志文件
func LuaLogf(format string, v ...interface{}) {
	log.Infof(format, v...)
}

//GMInit 初始化gm指令实现类
func GMInit(H GMHandler, globalVars map[string]interface{}, luapath string) {
	_luapath = luapath
	L := lua.NewState(lua.Options{SkipOpenLibs: true, IncludeGoStackTrace: true})
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.LoadLibName, lua.OpenPackage}, // Must be first
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
		{lua.OsLibName, lua.OpenOs},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			panic(err)
		}
	}
	luajson.Preload(L)

	GlobalLuaState = L
	_GmHandler = H
	RegistHander(reflect.ValueOf(H))

	initGlobalVariable(L, globalVars)
}

func initGlobalVariable(L *lua.LState, globalVars map[string]interface{}) {
	for k, v := range globalVars {
		L.SetGlobal(k, luar.New(L, v))
	}
	//日志方法
	L.SetGlobal("Logf", luar.New(L, LuaLogf))
	//gm工具类
	L.SetGlobal("GM", luar.New(L, _GmHandler))
}

func setOnceGlobal(L *lua.LState, now int64, params map[int]interface{}) {
	L.SetGlobal("CUR_FRAME_TIME", luar.New(L, now))
	L.SetGlobal("Params", luar.New(L, params))
}

func doLuaString(cmd string, params map[int]interface{}) interface{} {
	if GlobalLuaState == nil {
		return gerror.NewError(gerror.SERVER_ACCESS_REFUSED, "Lua module not initialize")
	}
	L := GlobalLuaState
	setOnceGlobal(L, timefix.CurrentMS(), params)
	ret := L.DoString(cmd)
	if ret != nil {
		return gerror.New(gerror.SERVER_FILE_NOT_FOUND, ret)
	}
	return gerror.NewErrorByCode(gerror.OK)
}

func parseLuaParams(params string) (map[int]interface{}, bool) {
	exp := "t(" + params + ")"
	paramsMap := make(map[int]interface{})
	f, err := parser.ParseExpr(exp)
	if err != nil {
		return paramsMap, false
	}
	count := 1
	result := true
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.BasicLit:
			val := x.Value
			if nVal, err := strconv.Unquote(x.Value); err == nil {
				val = nVal
			}
			paramVal, ok := getLuaParamValue(x.Kind, val)
			if ok == false {
				result = false
				return false
			}
			paramsMap[count] = paramVal
			count++
		}
		return true
	})
	return paramsMap, result
}

func getLuaParamValue(kind token.Token, param string) (interface{}, bool) {
	switch kind {
	case token.INT:
		val, err := strconv.ParseInt(param, 10, 64)
		if err != nil {
			return nil, false
		}
		return val, true
	case token.FLOAT:
		val, err := strconv.ParseFloat(param, 64)
		if err != nil {
			return nil, false
		}
		return val, true
	case token.STRING:
		return param, true
	}
	return nil, false
}

func fileFullPath(fname string) (string, error) {
	return fileutil.FindFullFilePath(fileutil.PathJoin(_luapath, fname))
}

func loadFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return "", err
	}

	b := make([]byte, int(fi.Size()))
	f.Read(b)
	return string(b), nil
}
