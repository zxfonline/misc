package web

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zxfonline/misc/chanutil"
	"github.com/zxfonline/misc/fileutil"
	"github.com/zxfonline/misc/gerror"
	"github.com/zxfonline/misc/golangtrace"
	"github.com/zxfonline/misc/json"
	"github.com/zxfonline/misc/log"

	//	"golang.org/x/net/websocket"
	. "github.com/zxfonline/misc/trace"
)

var (
	// run mode, "debug" or "release"
	RunMode         string
	CopyRequestBody bool
	HttpHead             = "Backend WebServer"
	IndentJson      bool = true
)

const MAXN_RETRY_TIMES = 60

// ServerConfig is configuration for server objects.
type ServerConfig struct {
	StaticDir      string
	CookieSecret   string
	RecoverPanic   bool
	KeepAlive      bool
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	MaxHeaderBytes int
	MaxMemory      int64
}

// Server represents a web.go server.
type Server struct {
	Config *ServerConfig
	routes []route
	Env    map[string]interface{}
	//save the listener so it can be closed
	l        net.Listener
	stopD    chanutil.DoneChan
	stopOnce sync.Once
}

func SetMaxMemory(maxMemory int64) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.MaxMemory = maxMemory
	}
}

func SetKeepAlive(keepAlive bool) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.KeepAlive = keepAlive
	}
}
func SetReadTimeout(readTimeout time.Duration) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.ReadTimeout = readTimeout
	}
}
func SetWriteTimeout(writeTimeout time.Duration) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.WriteTimeout = writeTimeout
	}
}

func SetMaxHeaderBytes(maxHeaderBytes int) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.MaxHeaderBytes = maxHeaderBytes
	}
}

func SetStaticDir(staticDir string) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		staticDir = strings.Replace(staticDir, "\\", "/", -1)
		cfg.StaticDir = staticDir
	}
}
func SetCookieSecret(cookieSecret string) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.CookieSecret = cookieSecret
	}
}
func SetRecoverPanic(recoverPanic bool) func(*ServerConfig) {
	return func(cfg *ServerConfig) {
		cfg.RecoverPanic = recoverPanic
	}
}

func NewServerConfig(options ...func(*ServerConfig)) *ServerConfig {
	cfg := &ServerConfig{
		MaxHeaderBytes: 1 << 20, //1M
		WriteTimeout:   15 * time.Second,
		ReadTimeout:    15 * time.Second,
		RecoverPanic:   true,
		KeepAlive:      false,
		MaxMemory:      1 << 26, //64M
	}
	for _, option := range options {
		option(cfg)
	}
	return cfg
}

func SetServerConfig(cfg *ServerConfig) func(*Server) {
	return func(s *Server) {
		s.Config = cfg
	}
}

func NewServer(options ...func(*Server)) *Server {
	server := new(Server)
	server.Env = map[string]interface{}{}
	server.stopD = chanutil.NewDoneChan()
	for _, option := range options {
		option(server)
	}
	return server
}

func (s *Server) initServer() {
	if s.Config == nil {
		s.Config = NewServerConfig()
	}
}

type ContextHandler interface {
	ContextHandle(*Context) (interface{}, error)
}

type route struct {
	r              string
	cr             *regexp.Regexp
	method         string
	handler        reflect.Value
	httpHandler    http.Handler
	contextHandler ContextHandler
	svc            *Service
}

type Service struct {
	method reflect.Value
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func isNil(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func (s *Service) ServiceCall(ctx *Context, args ...reflect.Value) (robj interface{}, err error) {
	ret := s.call(ctx, args...)
	if len(ret) == 0 {
		return
	}
	//默认支持两个返回参数(interface{},error)
	if len(ret) > 1 { //默认判定最后一个返回值为error类型
		var ok bool
		if err, ok = ret[len(ret)-1].Interface().(error); ok {
			return
		}
	}
	//默认第一个返回值为结果
	robj = ret[0].Interface()
	return
}

func (s *Service) call(ctx *Context, args ...reflect.Value) []reflect.Value {
	var targs []reflect.Value
	if requiresContext(s.method.Type()) {
		targs = append(targs, reflect.ValueOf(ctx))
	}
	if len(args) > 0 {
		targs = append(targs, args...)
	}
	return s.method.Call(targs)
}

//根据结构体的方法名注册为http服务方法 handler's kind must be ptr
func InvokeService(handler interface{}, methodName string) *Service {
	rv := reflect.ValueOf(handler)
	if rv.Kind() != reflect.Ptr || rv.IsNil() || !rv.IsValid() {
		panic(fmt.Errorf(`invoke service %s'%s err,handler's kind must be ptr`, reflect.TypeOf(handler).Name(), methodName))
	}
	rtm := rv.MethodByName(methodName)
	if rtm.Kind() != reflect.Func {
		panic(fmt.Errorf(`invoke service %s'%s err,method can not access`, reflect.Indirect(rv).Type().Name(), methodName))
	}
	return &Service{
		method: rtm,
	}
}

func (s *Server) addRoute(r string, method string, handler interface{}) {
	cr, err := regexp.Compile(r)
	if err != nil {
		log.Errorf("add route err,regex:%q,err:%v", r, err)
		return
	}
	switch v := handler.(type) {
	case http.Handler:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, httpHandler: v})
	case ContextHandler:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, contextHandler: v})
	case *Service:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, svc: v})
	case reflect.Value:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: v})
	default:
		s.routes = append(s.routes, route{r: r, cr: cr, method: method, handler: reflect.ValueOf(handler)})
	}
	log.Infof("register http service handler:%s,method:%s", r, method)
}

// ServeHTTP is the interface method for Go's http server package
func (s *Server) ServeHTTP(c http.ResponseWriter, req *http.Request) {
	s.Process(c, req)
}

// Process invokes the routing system for server s
func (s *Server) Process(c http.ResponseWriter, req *http.Request) {
	route := s.routeHandler(req, c)
	if route != nil {
		route.httpHandler.ServeHTTP(c, req)
	}
}

// Get adds a handler for the 'GET' http method for server s.
func (s *Server) Get(route string, handler interface{}) {
	s.addRoute(route, "GET", handler)
}

// Post adds a handler for the 'POST' http method for server s.
func (s *Server) Post(route string, handler interface{}) {
	s.addRoute(route, "POST", handler)
}

// Put adds a handler for the 'PUT' http method for server s.
func (s *Server) Put(route string, handler interface{}) {
	s.addRoute(route, "PUT", handler)
}

// Delete adds a handler for the 'DELETE' http method for server s.
func (s *Server) Delete(route string, handler interface{}) {
	s.addRoute(route, "DELETE", handler)
}

// Match adds a handler for an arbitrary http method for server s.
func (s *Server) Match(route string, handler interface{}, method string) {
	s.addRoute(route, method, handler)
}

// Match adds a handler for an arbitrary http method for server s.
func (s *Server) Matches(route string, handler interface{}, methods ...string) {
	for _, method := range methods {
		s.addRoute(route, method, handler)
	}
}

//Adds a custom handler. Only for webserver mode. Will have no effect when running as FCGI or SCGI.
func (s *Server) Handler(route string, method string, httpHandler http.Handler) {
	s.addRoute(route, method, httpHandler)
}

//Adds a handler for websockets. Only for webserver mode. Will have no effect when running as FCGI or SCGI.
//func (s *Server) Websocket(route string, httpHandler websocket.Handler) {
//	s.addRoute(route, "GET", httpHandler)
//}

// Run starts the web application and serves HTTP requests for s
func (s *Server) Run(addr string) {
	s.initServer()

	mux := http.DefaultServeMux
	mux.Handle("/", s)

	log.Infof("http serving %s", addr)

	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	s.l = l

	srv := &http.Server{
		Handler:        mux,
		ReadTimeout:    s.Config.ReadTimeout,
		WriteTimeout:   s.Config.WriteTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
	}
	srv.SetKeepAlivesEnabled(s.Config.KeepAlive)
	err = srv.Serve(s.l)
	s.l = nil
	//	err = http.Serve(s.l, mux)
	if err != nil {
		panic(err)
	}
}

// RunFCGI starts the web application and serves FastCGI requests for s.
func (s *Server) RunFCGI(addr string) {
	s.initServer()
	log.Infof("http serving fcgi %s", addr)
	err := s.listenAndServeFCGI(addr)
	if err != nil {
		panic(err)
	}
}

// RunSCGI starts the web application and serves SCGI requests for s.
func (s *Server) RunSCGI(addr string) {
	s.initServer()
	log.Infof("http serving scgi %s", addr)
	err := s.listenAndServeSCGI(addr)
	if err != nil {
		panic(err)
	}
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLS(addr string, config *tls.Config) {
	s.initServer()
	mux := http.DefaultServeMux
	mux.Handle("/", s)
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		panic(err)
	}
	s.l = l
	srv := &http.Server{
		Handler:        mux,
		ReadTimeout:    s.Config.ReadTimeout,
		WriteTimeout:   s.Config.WriteTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
	}
	srv.SetKeepAlivesEnabled(s.Config.KeepAlive)
	err = srv.Serve(s.l)
	//	err = http.Serve(s.l, mux)
	if err != nil {
		panic(err)
	}
}

func (s *Server) RunMux(basePattern, addr string) {
	s.initServer()
	err := s.startListen(1, addr)
	if err != nil {
		panic(err)
	}
	go s.working(basePattern, addr)
}

func (s *Server) working(basePattern, addr string) {
	defer func() {
		if !s.Closed() {
			if e := recover(); e != nil {
				log.Errorf("recover http err:%v,stack:%s", e, log.DumpStack())
			}
			//尝试重连
			err := s.startListen(MAXN_RETRY_TIMES, addr)
			if err != nil { //重连失败
				s.Close()
			} else { //重连成功，继续工作
				go s.working(basePattern, addr)
			}
		} else {
			if e := recover(); e != nil {
				//log.Errorf("recover http err:%+v",e)
			}
		}
	}()
	mux := http.DefaultServeMux
	mux.Handle(basePattern, s)
	srv := &http.Server{
		Handler:        mux,
		ReadTimeout:    s.Config.ReadTimeout,
		WriteTimeout:   s.Config.WriteTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
	}
	srv.SetKeepAlivesEnabled(s.Config.KeepAlive)
	err := srv.Serve(s.l)
	s.l = nil
	//	err = http.Serve(s.l, mux)
	if err != nil {
		panic(err)
	}
}

func (s *Server) startListen(tryTimes int, addr string) error {
	log.Infof("http serving %s", addr)
	tryTimes--
	err := func() error {
		if s.l != nil {
			s.l.Close()
			s.l = nil
		}
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		s.l = l
		return nil
	}()
	if err != nil {
		if tryTimes > 0 {
			time.Sleep(1 * time.Second)
			log.Errorf("tcp listen err:%v,retrying %d", err, tryTimes+1)
			return s.startListen(tryTimes, addr)
		} else {
			return err
		}
	}
	return nil
}

// RunTLS starts the web application and serves HTTPS requests for s.
func (s *Server) RunTLSMux(addr string, config *tls.Config) {
	s.initServer()
	err := s.startListenTLS(1, addr, config)
	if err != nil {
		panic(err)
	}
	go s.workingTls(addr, config)
}

func (s *Server) workingTls(addr string, config *tls.Config) {
	defer func() {
		if !s.Closed() {
			if e := recover(); e != nil {
				log.Errorf("recover http err:%v,stack:%s", e, log.DumpStack())
			}
			//尝试重连
			err := s.startListenTLS(MAXN_RETRY_TIMES, addr, config)
			if err != nil { //重连失败
				s.Close()
			} else { //重连成功，继续工作
				go s.workingTls(addr, config)
			}
		} else {
			if e := recover(); e != nil {
				//				s.Logger.Printf("recover http error:%s\n", e)
			}
		}
	}()
	mux := http.DefaultServeMux
	mux.Handle("/", s)

	srv := &http.Server{
		Handler:        mux,
		ReadTimeout:    s.Config.ReadTimeout,
		WriteTimeout:   s.Config.WriteTimeout,
		MaxHeaderBytes: s.Config.MaxHeaderBytes,
	}
	srv.SetKeepAlivesEnabled(s.Config.KeepAlive)
	err := srv.Serve(s.l)
	s.l = nil
	//	err = http.Serve(s.l, mux)
	if err != nil {
		panic(err)
	}
}

func (s *Server) startListenTLS(trytime int, addr string, config *tls.Config) error {
	log.Infof("http serving %s", addr)
	trytime--
	err := func() error {
		if s.l != nil {
			s.l.Close()
			s.l = nil
		}
		l, err := tls.Listen("tcp", addr, config)
		if err != nil {
			return err
		}
		s.l = l
		return nil
	}()
	if err != nil {
		if trytime > 0 {
			time.Sleep(1 * time.Second)
			log.Errorf("tcp listen err:%v,retrying %d", err, trytime+1)
			return s.startListenTLS(trytime, addr, config)
		} else {
			return err
		}
	}
	return nil
}

// Close stops server s.
func (s *Server) Close() {
	s.stopOnce.Do(func() {
		defer func() { recover() }()
		s.stopD.SetDone()
		if s.l != nil {
			s.l.Close()
			s.l = nil
		}
	})
}

func (s *Server) Closed() bool {
	return s.stopD.R().Done()
}

// safelyCall invokes `function` in recover block
func (s *Server) safelyCall(function reflect.Value, args []reflect.Value) (resp []reflect.Value, err interface{}) {
	defer func() {
		if e := recover(); e != nil {
			if !s.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				err = e
				resp = nil
				switch e.(type) {
				case *gerror.SysError:
					log.Errorf("handler crashed with content:%+v", e)
				default:
					log.Errorf("handler crashed with content:%+v", e)
				}
			}
		}
	}()
	resp = function.Call(args)
	return
}

func (s *Server) safelyServiceCall(svc *Service, ctx *Context, args ...reflect.Value) (resp []reflect.Value, err interface{}) {
	defer func() {
		if e := recover(); e != nil {
			if !s.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				err = e
				resp = nil
				switch e.(type) {
				case *gerror.SysError:
					log.Errorf("handler crashed with content:%+v", e)
				default:
					log.Errorf("handler crashed with content:%+v", e)
				}
			}
		}
	}()
	resp = svc.call(ctx, args...)
	return
}

func (s *Server) safelyCtxHandler(handler ContextHandler, ctx *Context) (resp []reflect.Value, err interface{}) {
	defer func() {
		if e := recover(); e != nil {
			if !s.Config.RecoverPanic {
				// go back to panic
				panic(e)
			} else {
				err = e
				switch e.(type) {
				case *gerror.SysError:
					log.Errorf("handler crashed with content:%+v", e)
				default:
					log.Errorf("handler crashed with content:%+v", e)
				}
			}
		}
	}()
	ret, err1 := handler.ContextHandle(ctx)
	if ret == nil && err1 != nil {
		ret = err1
	}
	if ret == nil {
		return
	}
	var rv reflect.Value
	if reflect.TypeOf(ret).Kind() == reflect.Ptr {
		rv = reflect.ValueOf(ret)
	} else {
		rv = reflect.ValueOf(&ret)
	}
	resp = []reflect.Value{rv}
	return
}

// requiresContext determines whether 'handlerType' contains
// an argument to 'web.Ctx' as its first argument
func requiresContext(handlerType reflect.Type) bool {
	//if the method doesn't take arguments, no
	if handlerType.NumIn() == 0 {
		return false
	}

	//if the first argument is not a pointer, no
	a0 := handlerType.In(0)
	if a0.Kind() != reflect.Ptr {
		return false
	}
	//if the first argument is a context, yes
	if a0.Elem() == contextType {
		return true
	}

	return false
}

// tryServingFile attempts to serve a static file, and returns
// whether or not the operation is successful.
// It checks the following directories for the file, in order:
// 1) Config.StaticDir
// 2) The 'static' directory in the parent directory of the executable.
// 3) The 'static' directory in the current working directory
func (s *Server) tryServingFile(name string, req *http.Request, w http.ResponseWriter) bool {
	//try to serve a static file
	if s.Config.StaticDir != "" {
		staticFile := fileutil.PathJoin(s.Config.StaticDir, name)
		if fileExists(staticFile) {
			http.ServeFile(w, req, staticFile)
			return true
		}
	} else {
		for _, staticDir := range defaultStaticDirs {
			staticFile := fileutil.PathJoin(staticDir, name)
			if fileExists(staticFile) {
				http.ServeFile(w, req, staticFile)
				return true
			}
		}
	}
	return false
}

func (s *Server) logRequest(ctx Context, sTime time.Time) {
	//log the request
	req := ctx.Request
	requestPath := req.URL.Path

	duration := time.Now().Sub(sTime)
	var client string
	// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
	// documentation at http://golang.org/pkg/net/http/#Request
	pos := strings.LastIndex(req.RemoteAddr, ":")
	if pos > 0 {
		client = req.RemoteAddr[0:pos]
	} else {
		client = req.RemoteAddr
	}

	if len(ctx.Params) > 0 {
		ctx.TracePrintf("%s %s %s - %v -Params: %+v", client, req.Method, requestPath, duration, ctx.Params)
	} else {
		ctx.TracePrintf("%s %s %s - %v", client, req.Method, requestPath, duration)
	}

}

// the main route handler in web.go
// Tries to handle the given request.
// Finds the route matching the request, and execute the callback associated
// with it.  In case of custom http handlers, this function returns an "unused"
// route. The caller is then responsible for calling the httpHandler associated
// with the returned route.
func (s *Server) routeHandler(req *http.Request, w http.ResponseWriter) (unused *route) {
	requestPath := req.URL.Path
	ctx := Context{req, []byte{}, map[string]string{}, s, w, nil}
	defer ctx.TraceFinish()
	//set some default headers

	ctx.SetHeader("Server", HttpHead, true)
	tm := time.Now()
	ctx.SetHeader("Date", webTime(tm), true)
	ctx.SetHeader("Access-Control-Allow-Origin", "*", true)             //允许访问所有域
	ctx.SetHeader("Access-Control-Allow-Headers", "Content-Type", true) //header的类型
	//	ctx.SetCacheControl(0)
	//	ctx.SetLastModified(tm)

	if req.Method == "GET" || req.Method == "HEAD" {
		req.ParseForm()
		if len(req.Form) > 0 {
			for k, v := range req.Form {
				ctx.Params[k] = v[0]
			}
		}
		defer s.logRequest(ctx, tm)

		if s.tryServingFile(requestPath, req, w) {
			return
		}
	} else {
		if CopyRequestBody && !ctx.IsUpload() {
			ctx.CopyBody()
		}
		ctx.ParseFormOrMutliForm(s.Config.MaxMemory)
		if len(req.Form) > 0 {
			for k, v := range req.Form {
				ctx.Params[k] = v[0]
			}
		}
		defer s.logRequest(ctx, tm)
	}

	//Set the default content-type
	ctx.SetHeader("Content-Type", "text/html; charset=utf-8", true)

	for i := 0; i < len(s.routes); i++ {
		route := s.routes[i]
		cr := route.cr
		//if the methods don't match, skip this handler (except HEAD can be used in place of GET)
		if req.Method != route.method && !(req.Method == "HEAD" && route.method == "GET") {
			continue
		}

		if !cr.MatchString(requestPath) {
			continue
		}
		match := cr.FindStringSubmatch(requestPath)

		if len(match[0]) != len(requestPath) {
			continue
		}

		if route.httpHandler != nil {
			unused = &route
			// We can not handle custom http handlers here, give back to the caller.
			return
		}
		if EnableTracing {
			ctx.tr = golangtrace.New("HttpService", route.r, true)
			var client string
			// We suppose RemoteAddr is of the form Ip:Port as specified in the Request
			// documentation at http://golang.org/pkg/net/http/#Request
			pos := strings.LastIndex(req.RemoteAddr, ":")
			if pos > 0 {
				client = req.RemoteAddr[0:pos]
			} else {
				client = req.RemoteAddr
			}
			ctx.TracePrintf("%s %s %s", client, req.Method, requestPath)
		}
		var ret []reflect.Value
		var err interface{}
		if route.contextHandler != nil {
			ret, err = s.safelyCtxHandler(route.contextHandler, &ctx)
		} else if route.svc != nil {
			var args []reflect.Value
			for _, arg := range match[1:] {
				args = append(args, reflect.ValueOf(arg))
			}
			ret, err = s.safelyServiceCall(route.svc, &ctx, args...)
		} else {
			var args []reflect.Value
			handlerType := route.handler.Type()
			if requiresContext(handlerType) {
				args = append(args, reflect.ValueOf(&ctx))
			}
			for _, arg := range match[1:] {
				args = append(args, reflect.ValueOf(arg))
			}
			ret, err = s.safelyCall(route.handler, args)
		}
		if err != nil {
			stt := http.StatusInternalServerError
			switch err.(type) {
			case *gerror.SysError:
				//				if err.(*gerror.SysError).Code == gerror.OK {
				stt = http.StatusOK
				//				}
			case error:
				err = gerror.New(gerror.SERVER_CMSG_ERROR, err.(error))
			default:
				err = gerror.NewError(gerror.SERVER_CMSG_ERROR, fmt.Sprintf("%v", err))
			}
			var bb []byte
			var err1 error
			if !IndentJson {
				bb, err1 = json.Marshal(err)
			} else {
				bb, err1 = json.MarshalIndent(err, "", " ")
			}
			if err1 == nil {
				ctx.SetHeader("Content-Type", "text/plain; charset=utf-8", true)
				ctx.SetHeader("Content-Length", strconv.Itoa(len(bb)), true)
				ctx.AbortBytes(stt, bb)
			} else {
				ctx.Abort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
			}
			return
		}
		if len(ret) == 0 {
			return
		}
		//默认支持两个返回参数(interface{},error)
		sval := ret[0]
		if len(ret) > 1 { //默认判定最后一个返回值为error类型
			if _, ok := ret[len(ret)-1].Interface().(error); ok {
				sval = ret[len(ret)-1]
			}
		}
		if isNil(sval) {
			return
		}
		content, ok := asBytes(nil, sval)
		if !ok {
			v := sval.Interface()
			switch v.(type) {
			case *gerror.SysError:
			case error:
				v = gerror.New(gerror.CUSTOM_ERROR, v.(error))
			default:
			}
			var bb []byte
			var err1 error
			if !IndentJson {
				bb, err1 = json.Marshal(v)
			} else {
				bb, err1 = json.MarshalIndent(v, "", " ")
			}
			if err1 == nil {
				ctx.SetHeader("Content-Type", "text/plain; charset=utf-8", true)
				content = bb
			} else {
				ctx.Abort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
				return
			}
		}

		ctx.SetHeader("Content-Length", strconv.Itoa(len(content)), true)
		ctx.WriteBytes(content)
		return
	}

	// try serving index.html or index.htm
	if req.Method == "GET" || req.Method == "HEAD" {
		if s.tryServingFile(path.Join(requestPath, "index.html"), req, w) {
			return
		} else if s.tryServingFile(path.Join(requestPath, "index.htm"), req, w) {
			return
		}
	}
	ctx.NotFound(http.StatusText(http.StatusNotFound))
	return
}

func asBytes(buf []byte, rv reflect.Value) (b []byte, ok bool) {
	switch rv.Kind() {
	case reflect.Ptr:
		return asBytes(buf, rv.Elem())
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return rv.Bytes(), true
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.AppendInt(buf, rv.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.AppendUint(buf, rv.Uint(), 10), true
	case reflect.Float32:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 32), true
	case reflect.Float64:
		return strconv.AppendFloat(buf, rv.Float(), 'g', -1, 64), true
	case reflect.Bool:
		return strconv.AppendBool(buf, rv.Bool()), true
	case reflect.String:
		s := rv.String()
		return append(buf, s...), true
	}
	return
}
