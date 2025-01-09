package logs

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	loggerDebug *log.Logger
	loggerInfo  *log.Logger
	loggerWarn  *log.Logger
	loggerError *log.Logger

	logLevel = "INFO"
	verbose  = false
	logFile  *os.File
)

// SetVerbose enables or disables verbose logging
func SetVerbose(v bool) {
	verbose = v
}

// InitLogger sets up log prefixes and levels. Logs are written to a file in the config directory,
// and only shown on stdout/stderr if verbose mode is enabled.
func InitLogger() error {
	flags := log.Ldate | log.Ltime | log.Lmicroseconds

	// Create log file in config directory
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %v", err)
		}
		xdg = filepath.Join(home, ".config")
	}
	logDir := filepath.Join(xdg, "strata", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	var err error
	logFile, err = os.OpenFile(filepath.Join(logDir, "strata.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// If verbose mode is enabled, write to both file and stdout/stderr
	// Otherwise, write only to file
	debugWriter := io.MultiWriter(logFile)
	infoWriter := io.MultiWriter(logFile)
	warnWriter := io.MultiWriter(logFile)
	errorWriter := io.MultiWriter(logFile)

	if verbose {
		debugWriter = io.MultiWriter(logFile, os.Stdout)
		infoWriter = io.MultiWriter(logFile, os.Stdout)
		warnWriter = io.MultiWriter(logFile, os.Stdout)
		errorWriter = io.MultiWriter(logFile, os.Stderr)
	}

	loggerDebug = log.New(debugWriter, "[DEBUG] ", flags)
	loggerInfo = log.New(infoWriter, "[INFO ] ", flags)
	loggerWarn = log.New(warnWriter, "[WARN ] ", flags)
	loggerError = log.New(errorWriter, "[ERROR] ", flags)

	// Optionally read a STRATA_LOG_LEVEL env var
	if lvl := os.Getenv("STRATA_LOG_LEVEL"); lvl != "" {
		logLevel = strings.ToUpper(lvl)
	}
	Info("Logger initialized. Level=%s, Verbose=%v", logLevel, verbose)
	return nil
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

// Close closes the log file
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// For structured logs or file logs, we can expand this in a real enterprise scenario.
