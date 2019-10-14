package gmmodule

import (
	"fmt"

	"github.com/zxfonline/misc/timefix"

	"github.com/zxfonline/misc/gerror"
)

const (
	//PublicStatusKey 状态主键
	PublicStatusKey = "status"
	//PublicStatusSuccess 成功
	PublicStatusSuccess = "success"
	//PublicStatusFailed 失败
	PublicStatusFailed = "fail"
)

//BaseGMHandler GM基础工具实现类
type BaseGMHandler struct {
}

//Lua 执行GM指令
func (h BaseGMHandler) Lua(cmd string) interface{} {
	return doLuaString(cmd, nil)
}

//LuaScript 执行GM脚本文件
func (h BaseGMHandler) LuaScript(fname string, params string) interface{} {
	if GlobalLuaState == nil {
		return gerror.NewError(gerror.SERVER_ACCESS_REFUSED, "Lua module not initialize")
	}
	L := GlobalLuaState
	fname, err := fileFullPath(fname)
	if err != nil {
		return gerror.New(gerror.SERVER_FILE_NOT_FOUND, err)
	}
	paramsMap, ok := parseLuaParams(params)
	if !ok {
		return gerror.NewError(gerror.SERVER_CMSG_ERROR, fmt.Sprintf("param invalid:%s", params))
	}

	//func 1
	//	cmd, err := loadFile(fname)
	//	if err != nil {
	//		return nil, gerror.New(gerror.SERVER_FILE_NOT_FOUND, err)
	//	}
	//	return doLuaString(cmd, paramsMap)

	//func 2
	setOnceGlobal(L, timefix.CurrentMS(), paramsMap)
	ret := L.DoFile(fname)
	if ret != nil {
		return gerror.New(gerror.SERVER_FILE_NOT_FOUND, ret)
	}
	return gerror.NewErrorByCode(gerror.OK)
}
