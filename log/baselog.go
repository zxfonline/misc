package log

import (
	"runtime"

	log "github.com/sirupsen/logrus"
)

var Logger *log.Logger

var BaseLogFields = log.Fields{}

func init() {
	log.SetLevel(log.TraceLevel)
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	//log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	//log.SetLevel(log.WarnLevel)
	//log.SetReportCaller(true)
	//TODO output set...
	Logger = log.StandardLogger()
}

func Trace(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Trace(args...)
}

func Debug(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Debug(args...)
}

func Print(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Print(args...)
}

func Info(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Info(args...)
}

func Warn(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Warn(args...)
}

//func Warning(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Warning(args...)
//}

func Error(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Error(args...)
}

//Not recommended
func Fatal(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Fatal(args...)
}

func Panic(args ...interface{}) {
	Logger.WithFields(BaseLogFields).Panic(args...)
}

func Tracef(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Tracef(format, args...)
}

func Debugf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Infof(format, args...)
}

func Printf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Printf(format, args...)
}

func Warnf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Warnf(format, args...)
}

//func Warningf(format string, args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Warningf(format, args...)
//}

func Errorf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Errorf(format, args...)
}

//Not recommended
func Fatalf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Fatalf(format, args...)
}

func Panicf(format string, args ...interface{}) {
	Logger.WithFields(BaseLogFields).Panicf(format, args...)
}

//func Traceln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Traceln(args...)
//}
//
//func Debugln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Debugln(args...)
//}
//
//func Infoln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Infoln(args...)
//}
//
//func Println(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Println(args...)
//}
//
//func Warnln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Warnln(args...)
//}
//
//func Warningln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Warningln(args...)
//}
//
//func Errorln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Errorln(args...)
//}

//func Fatalln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Fatalln(args...)
//}

//func Panicln(args ...interface{}) {
//	Logger.WithFields(BaseLogFields).Panicln(args...)
//}

//use for defer recover
func PrintPanicStack() {
	if x := recover(); x != nil {
		buf := make([]byte, 4<<20) // 4 KB should be enough
		n := runtime.Stack(buf, false)
		Logger.WithFields(BaseLogFields).Errorf("Recovered %v\nStack:%s", x, buf[:n])
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
		Logger.WithFields(BaseLogFields).Panicf("Stack:%s", buf[:n])
	case log.FatalLevel:
		Logger.WithFields(BaseLogFields).Fatalf("Stack:%s", buf[:n])
	case log.ErrorLevel:
		Logger.WithFields(BaseLogFields).Errorf("Stack:%s", buf[:n])
	case log.WarnLevel:
		Logger.WithFields(BaseLogFields).Warnf("Stack:%s", buf[:n])
	case log.InfoLevel:
		Logger.WithFields(BaseLogFields).Infof("Stack:%s", buf[:n])
	case log.DebugLevel:
		Logger.WithFields(BaseLogFields).Debugf("Stack:%s", buf[:n])
	case log.TraceLevel:
		Logger.WithFields(BaseLogFields).Tracef("Stack:%s", buf[:n])
	}
}
