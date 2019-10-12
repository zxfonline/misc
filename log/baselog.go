package log

import (
	"os"
	"runtime"

	log "github.com/sirupsen/logrus"
)

var Logger *log.Logger

var BaseLogFields = log.Fields{}

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
		buf := make([]byte, 4<<20) // 4 KB should be enough
		n := runtime.Stack(buf, false)
		Logger.Errorf("Recovered %v\nStack:%s", x, buf[:n])
	}
}

//print dump stack
func LogStack(level log.Level) {
	if !Logger.IsLevelEnabled(level) {
		return
	}
	buf := make([]byte, 4<<20) // 4 KB should be enough
	n := runtime.Stack(buf, false)
	switch level {
	case log.PanicLevel:
		Logger.Panicf("Stack:%s", buf[:n])
	case log.FatalLevel:
		Logger.Fatalf("Stack:%s", buf[:n])
	case log.ErrorLevel:
		Logger.Errorf("Stack:%s", buf[:n])
	case log.WarnLevel:
		Logger.Warnf("Stack:%s", buf[:n])
	case log.InfoLevel:
		Logger.Infof("Stack:%s", buf[:n])
	case log.DebugLevel:
		Logger.Debugf("Stack:%s", buf[:n])
	case log.TraceLevel:
		Logger.Tracef("Stack:%s", buf[:n])
	}
}
