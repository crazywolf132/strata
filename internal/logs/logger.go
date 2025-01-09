package logs

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
)

var (
	loggerDebug *log.Logger
	loggerInfo  *log.Logger
	loggerWarn  *log.Logger
	loggerError *log.Logger

	logLevel = "INFO"
)

// InitLogger sets up log prefixes and levels. In a production environment,
// we might want to log to a file, or to a structured logging system (JSON).
func InitLogger() {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds
	loggerDebug = log.New(os.Stdout, "[DEBUG] ", flags)
	loggerInfo = log.New(os.Stdout, "[INFO ] ", flags)
	loggerWarn = log.New(os.Stdout, "[WARN ] ", flags)
	loggerError = log.New(os.Stderr, "[ERROR] ", flags)

	// Optionally read a STRATA_LOG_LEVEL env var
	if lvl := os.Getenv("STRATA_LOG_LEVEL"); lvl != "" {
		logLevel = strings.ToUpper(lvl)
	}
	Info("Logger initialized. Level=%s", logLevel)
}

func Debug(format string, v ...interface{}) {
	if logLevel == "DEBUG" {
		loggerDebug.Output(2, fmt.Sprintf(callerInfo()+format, v...))
	}
}

func Info(format string, v ...interface{}) {
	if logLevel == "DEBUG" || logLevel == "INFO" {
		loggerInfo.Output(2, fmt.Sprintf(callerInfo()+format, v...))
	}
}

func Warn(format string, v ...interface{}) {
	if logLevel != "ERROR" { // show WARN for INFO/DEBUG
		loggerWarn.Output(2, fmt.Sprintf(callerInfo()+format, v...))
	}
}

func Error(format string, v ...interface{}) {
	loggerError.Output(2, fmt.Sprintf(callerInfo()+format, v...))
}

func callerInfo() string {
	// Show caller function name.
	// skip=3 to get correct frame in stack (due to usage in wrappers).
	pc, _, line, ok := runtime.Caller(3)
	if !ok {
		return ""
	}
	fn := runtime.FuncForPC(pc)
	var name string
	if fn != nil {
		name = fn.Name()
	}
	return fmt.Sprintf("[%s:%d] ", name, line)
}

// For structured logs or file logs, we can expand this in a real enterprise scenario.
