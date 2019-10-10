package trace

import (
	"net/http"

	"topcrown.com/centerserver/misc/expvar"

	_ "topcrown.com/centerserver/misc/pprof"

	"topcrown.com/centerserver/misc/iptable"

	"topcrown.com/centerserver/misc/golangtrace"
)

func Init(enableTracing bool, checkip bool) {
	iptable.CHECK_IPTRUSTED = checkip
	golangtrace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		w := iptable.IsTrustedIP1(iptable.RequestIP(req))
		return w, w
	}
	EnableTracing = enableTracing
}

// EnableTracing controls whether to trace using the golang.org/x/net/trace package.
var EnableTracing = true

//ProxyTrace 跟踪
type ProxyTrace struct {
	tr golangtrace.Trace
}

//TraceStart 开始跟踪
func TraceStart(family, title string, expvar bool) *ProxyTrace {
	if EnableTracing {
		pt := &ProxyTrace{tr: golangtrace.New(family, title, expvar)}
		return pt
	}
	return nil
}

func TraceFinish(pt *ProxyTrace) {
	if pt != nil {
		if pt.tr != nil {
			pt.tr.Finish()
		}
	}
}

func TraceFinishWithExpvar(pt *ProxyTrace, tracedefer func(*expvar.Map, int64)) {
	if pt != nil {
		if pt.tr != nil {
			pt.tr.Finish()
			if tracedefer != nil {
				family := pt.tr.GetFamily()
				req := expvar.Get(family)
				if req == nil {
					req = expvar.NewMap(family)
				}
				tracedefer(req.(*expvar.Map), pt.tr.GetElapsedTime())
			}
		}
	}
}

func TracePrintf(pt *ProxyTrace, format string, a ...interface{}) {
	if pt != nil {
		if pt.tr != nil {
			pt.tr.LazyPrintf(format, a...)
		}
	}
}

func TraceErrorf(pt *ProxyTrace, format string, a ...interface{}) {
	if pt != nil {
		if pt.tr != nil {
			pt.tr.LazyPrintf(format, a...)
			pt.tr.SetError()
		}
	}
}

func GetFamilyTotalString(family string) string {
	return golangtrace.GetFamilyTotalString(family)
}

func GetFamilyDetailString(family string, bucket int) string {
	return golangtrace.GetFamilyDetailString(family, bucket)
}
