cmd := ctx.ParamStr("cmd")
sessionId := netmsg.NewSession()
netmsg.SendMsg(sessionId, cmd, func(param interface{}) interface{} { //params=SendMsg第二个参数
	return gm_module.HandleCMD(param.(string)) //异步返回等待消息管道
})
ret := netmsg.RecMsg(sessionId).(*netmsg.PipeMsg) //阻塞等待消息返回