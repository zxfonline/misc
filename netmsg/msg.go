package netmsg

import (
	"fmt"
	"time"

	"github.com/zxfonline/misc/log"
	"github.com/zxfonline/misc/taskexcutor"
)

type Msg struct {
	SessionId int64
	Data      interface{}
	CallBack  CallBackMsg
}

//事件回调
type CallBackMsg func(interface{}) interface{}

func SendMsg(excutor taskexcutor.Excutor, sessionid int64, data interface{}, callback CallBackMsg, taskid interface{}) error {
	task := taskexcutor.NewTaskService(func(params ...interface{}) {
		msg := (params[0]).(Msg)
		if s := GetSession(msg.SessionId); s != nil {
			defer func() {
				if err := recover(); err != nil {
					s.Write(&PipeMsg{Error: fmt.Errorf("%v", err)})
				}
			}()
			s.Write(&PipeMsg{Data: msg.CallBack(msg.Data)})
		}
	}, Msg{SessionId: sessionid, Data: data, CallBack: callback})
	task.ID = taskid
	return excutor.Excute(task)
}
func AsyncSendMsg(excutor taskexcutor.Excutor, data interface{}, callback CallBackMsg, taskid interface{}) error {
	task := taskexcutor.NewTaskService(func(params ...interface{}) {
		msg := (params[0]).(Msg)
		defer log.PrintPanicStack()
		msg.CallBack(msg.Data)
	}, Msg{Data: data, CallBack: callback})
	task.ID = taskid
	return excutor.Excute(task)
}

func RecMsg(sId int64) interface{} {
	if s := GetSession(sId); s != nil {
		rt := s.Read(0)
		DelSession(sId)
		return rt
	}
	return nil
}

func RecMsgWithTime(sId int64, timeout time.Duration) interface{} {
	if s := GetSession(sId); s != nil {
		rt := s.Read(timeout)
		DelSession(sId)
		return rt
	}
	return nil
}
