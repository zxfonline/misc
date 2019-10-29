package expvar

import (
	"bytes"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/zxfonline/misc/golangtrace"
)

var (
	startTime      = time.Now().UTC()
	monitorChanMap = make(map[string]interface{}, 16)
	monitorLock    sync.RWMutex
)

func GetExpvarString() string {
	var buf bytes.Buffer
	buf.WriteString("{")
	first := true
	DeBarDo(func(kv KeyValue) {
		if !first {
			buf.WriteString(",")
		}
		first = false
		buf.WriteString(fmt.Sprintf("%q: %s", kv.Key, kv.Value))
	})

	buf.WriteString("}")

	return buf.String()
}

// func gcstats() interface{} {
// 	w := new(bytes.Buffer)
// 	toolbox.PrintGCSummary(w)
// 	return w.String()
// }

func goroutines() interface{} {
	return runtime.NumGoroutine()
}

// uptime is an expvar.Func compliant wrapper for uptime info.
func uptime() interface{} {
	uptime := time.Since(startTime)
	return int64(uptime)
}

func traceTotal() interface{} {
	return golangtrace.GetAllExpvarFamily(2)
}
func traceHour() interface{} {
	return golangtrace.GetAllExpvarFamily(1)
}
func traceMinute() interface{} {
	return golangtrace.GetAllExpvarFamily(0)
}

//RegistChanMonitor 注册管道监控
func RegistChanMonitor(name string, chanptr interface{}) bool {
	if !isChan(chanptr) {
		return false
	}
	monitorLock.Lock()
	defer monitorLock.Unlock()
	monitorChanMap[name] = chanptr
	return true
}
func isChan(a interface{}) bool {
	if a == nil {
		return false
	}

	v := reflect.ValueOf(a)
	if v.Kind() != reflect.Chan {
		return false
	}
	if v.IsNil() {
		return false
	}
	return true
}

func chanInfo(a interface{}) (bool, int, int) {
	if a == nil {
		return false, 0, 0
	}

	v := reflect.ValueOf(a)
	if v.Kind() != reflect.Chan {
		return false, 0, 0
	}
	if v.IsNil() {
		return false, 0, 0
	}
	return true, v.Cap(), v.Len()
}

type ChanInfo struct {
	Cap  int
	Len  int
	Rate float64
}

func chanstats() interface{} {
	monitorLock.RLock()
	defer monitorLock.RUnlock()
	mp := make(map[string]ChanInfo)
	for k, v := range monitorChanMap {
		_, mcap, mlen := chanInfo(v)
		mp[k] = ChanInfo{Cap: mcap, Len: mlen, Rate: float64(int64((float64(mlen) / float64(mcap) * 10000.0))) / 10000.0}
	}
	return mp
}

func init() {
	http.HandleFunc("/debug/vars", expvarHandler)
	Publish("cmdline", Func(cmdline))
	Publish("memstats", Func(memstats))

	// Publish("gcsummary", Func(gcstats))

	Publish("Goroutines", Func(goroutines))
	Publish("Uptime", Func(uptime))

	Publish("tracetotal", Func(traceTotal))
	Publish("tracehour", Func(traceHour))
	Publish("traceminute", Func(traceMinute))

	Publish("chanstats", Func(chanstats))
}

func isIgnoreVar(k string) bool {
	if k == "cmdline" || k == "memstats" ||
		//		k == "gcsummary" ||
		k == "chanstats" ||
		k == "Goroutines" || k == "Uptime" ||
		k == "tracetotal" || k == "tracehour" || k == "traceminute" {
		return true
	}
	return false
}
