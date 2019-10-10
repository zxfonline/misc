package timer

import (
	"container/heap"
	"errors"
	"sync"
	"time"

	"topcrown.com/centerserver/misc/log"

	"reflect"

	"topcrown.com/centerserver/misc/taskexcutor"
)

const (
	forever      = 1 << 62
	maxSleepTime = 1e9
)

var (
	_GTimer  *Timer
	onceInit sync.Once
)

func nanoseconds() int64 {
	return time.Now().UnixNano()
}

type Timer struct {
	events         eventHeap
	excutor        taskexcutor.Excutor
	eventMutex     sync.Mutex
	currentSleeper int64
}

func GTimer() *Timer {
	if _GTimer == nil {
		onceInit.Do(func() {
			_GTimer = NewTimer(nil)
		})
	}
	return _GTimer
}

func SetGTimer(timer *Timer) {
	if _GTimer != nil {
		panic(errors.New("GTimer has been inited."))
	}
	_GTimer = timer
}

//添加定时事件，事件的回调函数默认第一个参数为 *TimerEvent
func AddOnceEvent(listener *taskexcutor.TaskService, name string, wait time.Duration) (e *TimerEvent) {
	return GTimer().AddTimerEvent(listener, name, wait, wait, 1, true)
}

//添加定时事件，事件的回调函数默认第一个参数为 *TimerEvent
func AddTimerEvent(listener *taskexcutor.TaskService, name string, initTime, interval time.Duration, count int, absolute bool) (e *TimerEvent) {
	return GTimer().AddTimerEvent(listener, name, initTime, interval, count, absolute)
}

func NewTimer(excutor taskexcutor.Excutor) *Timer {
	events := make([]*TimerEvent, 0, 128)
	events = append(events, &TimerEvent{nextTime: forever})
	if excutor == nil || reflect.ValueOf(excutor).IsNil() {
		excutor = taskexcutor.NewTaskPoolExcutor(1, 0x10000, false, 0)
	}
	t := &Timer{events: events, excutor: excutor}
	heap.Init(&t.events)
	return t
}

type TimerEvent struct {
	i         int    //所属排序队列的index
	Name      string //事件名称
	count     int    //执行次数,当为0表示永久执行
	costCount int    //已执行次数
	absolute  bool   //决对时间定时，线程工作的时间计算在定时时间内
	interval  int64  //定时执行时间
	nextTime  int64  //上次执行时间
	t         *Timer //所属定时器
	//事件执行方法
	listener *taskexcutor.TaskService
}

//定时器下次执行时间 毫秒
func (e *TimerEvent) NextTime() int64 {
	return e.nextTime / 1e6
}

func (e *TimerEvent) Closed() bool {
	return e.i < 0
}
func (e *TimerEvent) Close() {
	if e.i < 0 {
		return
	}
	if e.listener != nil {
		e.listener.Cancel = true
	}
	if e.i >= 0 {
		t := e.t
		t.eventMutex.Lock()
		heap.Remove(&e.t.events, e.i)
		t.eventMutex.Unlock()
	}
}

//添加定时事件，事件的回调函数默认第一个参数为 *TimerEvent
func (t *Timer) AddOnceEvent(listener *taskexcutor.TaskService, name string, wait time.Duration) (e *TimerEvent) {
	return t.AddTimerEvent(listener, name, wait, wait, 1, true)
}

//添加定时事件，事件的回调函数默认第一个参数为 *TimerEvent
func (t *Timer) AddTimerEvent(listener *taskexcutor.TaskService, name string, initTime, interval time.Duration, count int, absolute bool) (e *TimerEvent) {
	now := nanoseconds()
	init := initTime.Nanoseconds()
	if init < 0 {
		init = interval.Nanoseconds()
	}
	nextTime := now + init
	if nextTime < now {
		panic("timer: invalid timer start time")
	}
	if count < 0 {
		count = 0
	}
	if interval.Nanoseconds() <= 0 && count == 0 {
		panic("timer: invalid timer interval time")
	}
	t.eventMutex.Lock()
	t0 := t.events[0].nextTime
	e = &TimerEvent{
		nextTime: nextTime,
		interval: interval.Nanoseconds(),
		count:    count,
		absolute: absolute,
		listener: listener,
		t:        t,
		Name:     name,
		i:        -1,
	}
	//默认事件添加到第一个参数
	if listener != nil {
		listener.AddArgs(0, e)
	}
	heap.Push(&t.events, e)
	if t0 > e.nextTime && (t0 == forever || init < maxSleepTime) {
		t.currentSleeper++
		go t.sleeper(t.currentSleeper)
	}
	t.eventMutex.Unlock()
	return
}

func (t *Timer) sleeper(sleeperId int64) {
	t.eventMutex.Lock()
	e := t.events[0]
	tm := nanoseconds()
	for e.nextTime != forever {
		if dt := e.nextTime - tm; dt > 0 {
			if dt > maxSleepTime {
				dt = maxSleepTime
			}
			t.eventMutex.Unlock()
			time.Sleep(time.Duration(dt))
			t.eventMutex.Lock()
			if t.currentSleeper != sleeperId {
				break
			}
		}
		e = t.events[0]
		tm = nanoseconds()
		for tm >= e.nextTime {
			if e.listener != nil {
				if tm-e.nextTime >= 2*e.interval {
					log.Warnf("interval run timeout,event:%+v,current:%v.", e, tm)
				}
				if e.absolute {
					e.nextTime += e.interval
				} else {
					e.nextTime = tm + e.interval
				}
				err := e.t.excutor.Excute(e.listener)
				if e.count != 0 {
					e.costCount++
					if err != nil {
						log.Warnf("interval run err:%v,event:%+v.", err, e)
						//					} else {
						//						e.costCount++
					}
					if e.costCount >= e.count {
						heap.Pop(&t.events)
						e = t.events[0]
					} else {
						heap.Fix(&t.events, e.i)
						e = t.events[0]
					}
				} else {
					heap.Fix(&t.events, e.i)
					e = t.events[0]
				}
			} else {
				log.Warnf("run nil listener,event:%+v.", e)
				heap.Pop(&t.events)
				e = t.events[0]
			}
		}
	}
	t.eventMutex.Unlock()
}

type eventHeap []*TimerEvent

func (t *eventHeap) Len() int {
	return len(*t)
}

func (t *eventHeap) Less(i, j int) bool {
	return (*t)[i].nextTime < (*t)[j].nextTime
}

func (t *eventHeap) Swap(i, j int) {
	tp := *t
	tp[i], tp[j] = tp[j], tp[i]
	tp[i].i = i
	tp[j].i = j
}

func (t *eventHeap) Push(x interface{}) {
	e := x.(*TimerEvent)
	e.i = len(*t)
	*t = append(*t, e)
}

func (t *eventHeap) Pop() interface{} {
	old := *t
	n := len(old) - 1
	e := old[n]
	*t = old[0:n]
	e.i = -1
	//logger.LogDebug("closed:event=%+v.", e)
	e.listener = nil
	e.t = nil
	return e
}
