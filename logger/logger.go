// logger/logger.go
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	debugLogger        *log.Logger
	infoLogger         *log.Logger
	warnLogger         *log.Logger
	errorLogger        *log.Logger
	debugLoggerNoColor *log.Logger
	infoLoggerNoColor  *log.Logger
	warnLoggerNoColor  *log.Logger
	errorLoggerNoColor *log.Logger
	file               *os.File
	consoleOutput      io.Writer
	fileOutput         io.Writer
	minLevel           LogLevel
}

var (
	defaultLogger *Logger
	once          sync.Once
	mu            sync.Mutex
)

// ensureInitialized creates a default logger if one doesn't exist
func ensureInitialized() {
	once.Do(func() {
		defaultLogger = &Logger{
			consoleOutput: os.Stdout,
			minLevel:      DEBUG,
		}
		defaultLogger.setupLoggers()
	})
}

// Init initializes the logger with optional file and console output
// If filename is empty, logs only to console
// If console is false, logs only to file
func Init(filename string, console bool) error {
	mu.Lock()
	defer mu.Unlock()

	// Close existing file if any
	if defaultLogger != nil && defaultLogger.file != nil {
		defaultLogger.file.Close()
	}

	defaultLogger = &Logger{
		minLevel: DEBUG,
	}

	// Add file output if filename provided
	if filename != "" {
		file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		defaultLogger.file = file
		defaultLogger.fileOutput = file
	}

	// Add console output if enabled
	if console {
		defaultLogger.consoleOutput = os.Stdout
	}

	if defaultLogger.fileOutput == nil && defaultLogger.consoleOutput == nil {
		return fmt.Errorf("no output destination specified")
	}

	defaultLogger.setupLoggers()
	return nil
}

// SetLevel sets the minimum log level (DEBUG, INFO, WARN, ERROR)
// Messages below this level will not be logged
func SetLevel(level LogLevel) {
	ensureInitialized()
	mu.Lock()
	defer mu.Unlock()
	defaultLogger.minLevel = level
}

func (l *Logger) setupLoggers() {
	flags := log.Ldate | log.Ltime | log.Lshortfile

	// Setup colored loggers for console
	if l.consoleOutput != nil {
		l.debugLogger = log.New(l.consoleOutput, colorGray+"[DEBUG] "+colorReset, flags)
		l.infoLogger = log.New(l.consoleOutput, colorReset+"[INFO]  "+colorReset, flags)
		l.warnLogger = log.New(l.consoleOutput, colorYellow+"[WARN]  "+colorReset, flags)
		l.errorLogger = log.New(l.consoleOutput, colorRed+"[ERROR] "+colorReset, flags)
	}

	// Setup non-colored loggers for file
	if l.fileOutput != nil {
		l.debugLoggerNoColor = log.New(l.fileOutput, "[DEBUG] ", flags)
		l.infoLoggerNoColor = log.New(l.fileOutput, "[INFO]  ", flags)
		l.warnLoggerNoColor = log.New(l.fileOutput, "[WARN]  ", flags)
		l.errorLoggerNoColor = log.New(l.fileOutput, "[ERROR] ", flags)
	}
}

// Close closes the log file if one is open
func Close() {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger != nil && defaultLogger.file != nil {
		defaultLogger.file.Close()
		defaultLogger.file = nil
		defaultLogger.fileOutput = nil
	}
}

func (l *Logger) shouldLog(level LogLevel) bool {
	return level >= l.minLevel
}

func (l *Logger) output(level LogLevel, colorLogger, noColorLogger *log.Logger, msg string) {
	if !l.shouldLog(level) {
		return
	}

	// Log to console with colors
	if l.consoleOutput != nil && colorLogger != nil {
		colorLogger.Output(3, msg)
	}

	// Log to file without colors
	if l.fileOutput != nil && noColorLogger != nil {
		noColorLogger.Output(3, msg)
	}
}

// Debug logs a debug message
func Debug(v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprint(v...)
	defaultLogger.output(DEBUG, defaultLogger.debugLogger, defaultLogger.debugLoggerNoColor, msg)
}

// Debugf logs a formatted debug message
func Debugf(format string, v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprintf(format, v...)
	defaultLogger.output(DEBUG, defaultLogger.debugLogger, defaultLogger.debugLoggerNoColor, msg)
}

// Info logs an info message
func Info(v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprint(v...)
	defaultLogger.output(INFO, defaultLogger.infoLogger, defaultLogger.infoLoggerNoColor, msg)
}

// Infof logs a formatted info message
func Infof(format string, v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprintf(format, v...)
	defaultLogger.output(INFO, defaultLogger.infoLogger, defaultLogger.infoLoggerNoColor, msg)
}

// Warn logs a warning message
func Warn(v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprint(v...)
	defaultLogger.output(WARN, defaultLogger.warnLogger, defaultLogger.warnLoggerNoColor, msg)
}

// Warnf logs a formatted warning message
func Warnf(format string, v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprintf(format, v...)
	defaultLogger.output(WARN, defaultLogger.warnLogger, defaultLogger.warnLoggerNoColor, msg)
}

// Error logs an error message
func Error(v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprint(v...)
	defaultLogger.output(ERROR, defaultLogger.errorLogger, defaultLogger.errorLoggerNoColor, msg)
}

// Errorf logs a formatted error message
func Errorf(format string, v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprintf(format, v...)
	defaultLogger.output(ERROR, defaultLogger.errorLogger, defaultLogger.errorLoggerNoColor, msg)
}

// Fatal logs an error message and exits the program
func Fatal(v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprint(v...)
	defaultLogger.output(ERROR, defaultLogger.errorLogger, defaultLogger.errorLoggerNoColor, msg)
	os.Exit(1)
}

// Fatalf logs a formatted error message and exits the program
func Fatalf(format string, v ...interface{}) {
	ensureInitialized()
	msg := fmt.Sprintf(format, v...)
	defaultLogger.output(ERROR, defaultLogger.errorLogger, defaultLogger.errorLoggerNoColor, msg)
	os.Exit(1)
}
