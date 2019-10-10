package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/zxfonline/misc/taskexcutor"
	"github.com/zxfonline/misc/timer"
)

func randInt(min int32, max int32) int32 {
	return min + rand.Int31n(max-min)
}

func main() {
	//	for i := 0; i < 1; i++ {
	//		//		go func() {
	//		obj := &Obj{
	//			name: fmt.Sprintf("name_%d", i+1),
	//			t1:   T1{name: fmt.Sprintf("t1_%d", i+1)},
	//			t2:   T2{name: fmt.Sprintf("t2_%d", i+1)},
	//		}
	//		ev := timer.AddTimerEvent(taskexcutor.NewTaskService(obj.t1.Processing), fmt.Sprintf("name1_%d", i+1), 0, time.Duration((100+int(randInt(1, 50))*i)*1e6), 10-i, true)
	//		obj.t1.event = ev
	//		ev = timer.AddTimerEvent(taskexcutor.NewTaskService(obj.t2.Processing), fmt.Sprintf("name2_%d", i+1), 0, time.Duration((100+20*i)*1e6), 10-i, true)
	//		obj.t2.event = ev
	//		//		}()
	//	}
	//	time.Sleep(5 * time.Second)
	t3 := &T3{name: "t3_1"}
	//	ev3 := timer.AddTimerEvent(taskexcutor.NewTaskService(t3.Processing), "name3_1", 0, time.Duration((100)*1e6), 3, true)
	ev3 := timer.AddTimerEvent(taskexcutor.NewTaskService(func(params ...interface{}) {
		event := (params[0])
		ev := (params[0]).(*timer.TimerEvent)
		if ev.Closed() {
			fmt.Println("closed")
			return
		}
		fmt.Printf("event3=%p,%+v\n", event, event)
		time.Sleep(1 * time.Second)
		ev.Close()
	}), "name3_1", 0, time.Duration((100)*1e6), 3, true)

	t3.event = ev3
	time.Sleep(5 * time.Second)
}

type Obj struct {
	name string
	t1   T1
	t2   T2
}
type T1 struct {
	name  string
	event *timer.TimerEvent
}

func (t *T1) Processing(params ...interface{}) {
	//	time.Sleep(2 * time.Second)
	fmt.Printf("1=%v ,e=%p,name=%p\n", t, t, &t.name)
}

type T2 struct {
	name  string
	event *timer.TimerEvent
}

func (t T2) Processing(params ...interface{}) {
	//	time.Sleep(1 * time.Second)
	lm := t.name
	fmt.Printf("2=%v ,e=%p,name=%s,lm=%s\n", t, t, t.name, lm)
}

type T3 struct {
	name  string
	event *timer.TimerEvent
}

func (t *T3) Processing(params ...interface{}) {
	//	time.Sleep(1 * time.Second)

	//	event := (params[0])
	//	fmt.Printf("event0=%p,%+v\n", event, event)
	//	ev := event.(*timer.TimerEvent)
	//	fmt.Printf("event1=%p,%+v\n", ev, ev)
	//	fmt.Printf("event2=%p,%+v\n", t.event, t.event)
	//	fmt.Println(t.event == ev)
	//	t.name = fmt.Sprintf("%d", randInt(1, 100))
	fmt.Printf("3=%v ,e=%p,name=%p\n", t, t, &t.name)
}
