package taskexcutor

import (
	"container/list"
	"errors"
	"time"

	"sync"

	"github.com/zxfonline/misc/chanutil"
	"github.com/zxfonline/misc/expvar"
	"github.com/zxfonline/misc/log"

	"github.com/zxfonline/misc/gerror"
)

var (
	PoolStopedError = errors.New("task pool stoped")
	PoolFullError   = errors.New("task pool full")
)

var (
	_GExcutor Excutor
	onceInit  sync.Once
)

func GExcutor() Excutor {
	if _GExcutor == nil {
		onceInit.Do(func() {
			SetGExcutor(NewTaskPoolExcutor(1, 0x10000, false, 0))
		})
	}
	return _GExcutor
}

func SetGExcutor(excutor Excutor) {
	if _GExcutor != nil {
		panic(errors.New("_GExcutor has been inited"))
	}
	_GExcutor = excutor
	expvar.RegistChanMonitor("chanGTaskExcutor", _GExcutor)
}
func NewTaskExcutor(chanSize int) TaskExcutor {
	return make(chan *TaskService, chanSize)
}

type TaskExcutor chan *TaskService

func (c TaskExcutor) Close() {
	defer func() { recover() }()
	close(c)
}
func (c TaskExcutor) Excute(task *TaskService) (err error) {
	defer gerror.PanicToErr(&err)
	c <- task
	if wait := len(c); wait > cap(c)/10*5 && wait%100 == 0 {
		log.Warnf("task excutor taskchan process,waitchan:%d/%d", wait, cap(c))
	}
	return
}

//事件回调
type CallBack func(...interface{})

type TaskService struct {
	callback CallBack
	args     []interface{}
	Cancel   bool //是否取消回调
	ID       interface{}
}

//代理执行
func (t *TaskService) Call() {
	if t.Cancel {
		return
	}
	defer func() {
		if e := recover(); e != nil {
			log.Errorf("recover task service err:%v,stack:%s", e, log.DumpStack())
		}
	}()
	t.callback(t.args...)
}

//重置参数
func (t *TaskService) SetArgs(args ...interface{}) *TaskService {
	t.args = args
	return t
}

//重置指定下标的参数
func (t *TaskService) SetArg(index int, arg interface{}) {
	if index < 0 || index+1 >= len(t.args) {
		return
	}
	t.args[index] = arg
}

//获取指定下标的参数
func (t *TaskService) GetArg(index int) interface{} {
	if index < 0 || index+1 >= len(t.args) {
		return nil
	}
	return t.args[index]
}

//添加回调函数参数,startIndex<0表示顺序添加,startIndex>=0表示将参数从指定位置开始添加，原来位置的参数依次后移
func (t *TaskService) AddArgs(startIndex int, args ...interface{}) *TaskService {
	length := len(args)
	if length > 0 {
		stmp := t.args
		slenth := len(stmp)
		if startIndex < 0 {
			t.args = append(stmp, args...)
		} else if startIndex >= slenth {
			tl := startIndex + length
			temp := make([]interface{}, tl, tl)
			if slenth > 0 {
				copy(temp, stmp[0:slenth])
			}
			copy(temp[startIndex:], args)
			t.args = temp
		} else {
			tl := slenth + length
			temp := make([]interface{}, tl, tl)
			mv := stmp[startIndex:slenth]
			if startIndex > 0 {
				copy(temp, stmp[0:startIndex])
			}
			copy(temp[startIndex:startIndex+length], args)
			copy(temp[startIndex+length:], mv)
			t.args = temp
		}
	}
	return t
}

func NewTaskService(callback CallBack, params ...interface{}) *TaskService {
	length := len(params)
	temp := make([]interface{}, 0, length)
	temp = append(temp, params...)
	return &TaskService{callback: callback, args: temp}
}

//任务执行器
type Excutor interface {
	//执行方法
	Excute(task *TaskService) error
	//执行器关闭方法
	Close()
}

//任务执行器
type poolexcutor struct {
	closeD chanutil.DoneChan
	wg     *sync.WaitGroup
}

func (c *poolexcutor) stop() {
	c.closeD.SetDone()
}

func (c *poolexcutor) excute(taskChan chan *TaskService, waitD chanutil.DoneChan) {
	defer func() {
		c.wg.Done()
	}()
	for q := false; !q; {
		select {
		case <-c.closeD:
			q = true
		case task := <-taskChan:
			task.Call()
		case <-waitD:
			select { //等待线程池剩余任务完成
			case task := <-taskChan:
				task.Call()
			default:
				q = true
			}
		}
	}
}

//并发执行器
type MultiplePoolExcutor struct {
	//任务缓冲池
	taskchan chan *TaskService
	//线程池执行关闭后是否立刻关闭子线程执行器
	shutdownNow bool
	//当 shutdownNow=false 时使用该变量 线程池执行关闭后等待子线程执行的时间，时间到后立刻关闭，当值为0时表示一直等待所有任务执行完毕
	shutdownWait time.Duration

	wgExcutor *sync.WaitGroup
	closeD    chanutil.DoneChan
	waitD     chanutil.DoneChan
	closeOnce sync.Once
	//当前运行中的执行器 不用改队列
	excutors *list.List
}

func NewTaskPoolExcutor(poolSize, chanSize uint, shutdownNow bool, shutdownWait time.Duration) Excutor {
	wgExcutor := &sync.WaitGroup{}
	taskchan := make(chan *TaskService, chanSize)
	expvar.RegistChanMonitor("chanMultiExcutor", taskchan)
	waitD := chanutil.NewDoneChan()
	p := &MultiplePoolExcutor{
		excutors:     list.New(),
		taskchan:     taskchan,
		shutdownNow:  shutdownNow,
		shutdownWait: shutdownWait,
		waitD:        waitD,
		closeD:       chanutil.NewDoneChan(),
		wgExcutor:    wgExcutor,
	}

	if poolSize < 1 {
		poolSize = 1
	}
	for i := 0; i < int(poolSize); i++ {
		ex := &poolexcutor{
			closeD: chanutil.NewDoneChan(),
			wg:     wgExcutor,
		}
		p.excutors.PushBack(ex)
		wgExcutor.Add(1)
		go ex.excute(taskchan, waitD)
	}
	return p
}

//Excutor.Excute()
func (p *MultiplePoolExcutor) Excute(task *TaskService) error {
	select {
	case <-p.closeD:
		return PoolStopedError
	default:
		p.taskchan <- task //阻塞等待
		if wait := len(p.taskchan); wait > cap(p.taskchan)/10*5 && wait%100 == 0 {
			log.Warnf("taskpool excutor taskchan process,waitchan:%d/%d", wait, cap(p.taskchan))
		}
		return nil
	}
}

func (p *MultiplePoolExcutor) waitdone() {
	//等待所有子任务执行执行完成
	p.wgExcutor.Wait()
	p.excutors.Init()
}
func (p *MultiplePoolExcutor) clear() {
	for ex := p.excutors.Front(); ex != nil; ex = ex.Next() {
		v := ex.Value.(*poolexcutor)
		v.stop()
	}
	p.waitdone()
}

//Excutor.Close()
func (p *MultiplePoolExcutor) Close() {
	p.closeOnce.Do(func() {
		p.closeD.SetDone()
		if p.shutdownNow {
			p.clear()
		} else {
			if p.shutdownWait > 0 {
				time.AfterFunc(p.shutdownWait, func() {
					p.clear()
				})
			} else {
				p.waitD.SetDone()
				p.waitdone()
			}
		}
	})
}
