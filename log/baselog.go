package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	log "github.com/sirupsen/logrus"
)

var Logger *log.Logger

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.TraceLevel)

	//log.SetReportCaller(true)
	Logger = log.StandardLogger()
}

func Trace(args ...interface{}) {
	Logger.Trace(args...)
}

func Debug(args ...interface{}) {
	Logger.Debug(args...)
}

func Print(args ...interface{}) {
	Logger.Print(args...)
}

func Info(args ...interface{}) {
	Logger.Info(args...)
}

func Warn(args ...interface{}) {
	Logger.Warn(args...)
}

func Warning(args ...interface{}) {
	Logger.Warning(args...)
}

func Error(args ...interface{}) {
	Logger.Error(args...)
}

//Not recommended
func Fatal(args ...interface{}) {
	Logger.Fatal(args...)
}

func Panic(args ...interface{}) {
	Logger.Panic(args...)
}

func Tracef(format string, args ...interface{}) {
	Logger.Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	Logger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	Logger.Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	Logger.Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Logger.Warnf(format, args...)
}

func Warningf(format string, args ...interface{}) {
	Logger.Warningf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	Logger.Errorf(format, args...)
}

//Not recommended
func Fatalf(format string, args ...interface{}) {
	Logger.Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	Logger.Panicf(format, args...)
}

func Traceln(args ...interface{}) {
	Logger.Traceln(args...)
}

func Debugln(args ...interface{}) {
	Logger.Debugln(args...)
}

func Infoln(args ...interface{}) {
	Logger.Infoln(args...)
}

func Println(args ...interface{}) {
	Logger.Println(args...)
}

func Warnln(args ...interface{}) {
	Logger.Warnln(args...)
}

func Warningln(args ...interface{}) {
	Logger.Warningln(args...)
}

func Errorln(args ...interface{}) {
	Logger.Errorln(args...)
}

func Fatalln(args ...interface{}) {
	Logger.Fatalln(args...)
}

func Panicln(args ...interface{}) {
	Logger.Panicln(args...)
}

//use for defer recover
func PrintPanicStack() {
	if x := recover(); x != nil {
		Logger.Errorf("Recovered %v,Stack:%s", x, DumpStack(1))
	}
}

var MiscRegexp = regexp.MustCompile(`zxfonline/misc(@.*)?/log/.*.go`)
var LogrusRegexp = regexp.MustCompile(`sirupsen/logrus(@.*)?/.*.go`)

var SuffixesToIgnoreArray = []*regexp.Regexp{
	MiscRegexp,
	LogrusRegexp,
}

func DumpStack(callDepth int) string {
	var buff bytes.Buffer
	for i := callDepth + 1; ; i++ {
		/*funcName*/ _, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		for _, s := range SuffixesToIgnoreArray {
			if s.MatchString(file) {
				break
			}
		}
		//buff.WriteString(fmt.Sprintf(" %d:%s[%s:%d]", i, runtime.FuncForPC(funcName).Name(), filepath.Base(file), line))
		buff.WriteString(fmt.Sprintf(" %d:[%s:%d]", i, filepath.Base(file), line))
	}
	return buff.String()
}

func LogStack(level log.Level) {
	if !Logger.IsLevelEnabled(level) {
		return
	}
	stack := DumpStack(1)
	switch level {
	case log.PanicLevel:
		Logger.Panicf("Stack:%s", stack)
	case log.FatalLevel:
		Logger.Fatalf("Stack:%s", stack)
	case log.ErrorLevel:
		Logger.Errorf("Stack:%s", stack)
	case log.WarnLevel:
		Logger.Warnf("Stack:%s", stack)
	case log.InfoLevel:
		Logger.Infof("Stack:%s", stack)
	case log.DebugLevel:
		Logger.Debugf("Stack:%s", stack)
	case log.TraceLevel:
		Logger.Tracef("Stack:%s", stack)
	}
}
