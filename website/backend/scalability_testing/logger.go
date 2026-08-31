package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity level of log messages.
type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

// String returns the string representation of the log level.
func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarn:
		return "WARN"
	case LogError:
		return "ERROR"
	case LogFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured logging for the scalability testing system.
type Logger struct {
	logger   *log.Logger
	level    LogLevel
	mu       sync.RWMutex
	fields   map[string]interface{}
}

// NewLogger creates a new logger instance with default configuration.
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0), // We'll format ourselves
		level:  LogInfo,
		fields: make(map[string]interface{}),
	}
}

// SetLevel sets the minimum log level.
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithField returns a new logger with an additional field.
func (l *Logger) WithField(key string, value interface{}) *Logger {
	l.mu.RLock()
	fields := make(map[string]interface{})
	for k, v := range l.fields {
		fields[k] = v
	}
	l.mu.RUnlock()

	fields[key] = value

	return &Logger{
		logger: l.logger,
		level:  l.level,
		fields: fields,
	}
}

// WithFields returns a new logger with additional fields.
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	l.mu.RLock()
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		logger: l.logger,
		level:  l.level,
		fields: newFields,
	}
}

// Debug logs a debug message.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogDebug, format, args...)
}

// Info logs an info message.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogInfo, format, args...)
}

// Warn logs a warning message.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogWarn, format, args...)
}

// Error logs an error message.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogError, format, args...)
}

// Fatal logs a fatal message and exits the program.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogFatal, format, args...)
	os.Exit(1)
}

// log formats and writes a log message.
func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	fields := l.fields
	l.mu.RUnlock()

	// Check if we should log this level
	if level < currentLevel {
		return
	}

	// Format timestamp
	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	// Get caller info for debugging
	var caller string
	if level >= LogError {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			// Extract just the filename from full path
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' || file[i] == '\\' {
					file = file[i+1:]
					break
				}
			}
			caller = fmt.Sprintf(" [%s:%d]", file, line)
		}
	}

	// Format message
	message := fmt.Sprintf(format, args...)

	// Format fields
	var fieldsStr string
	if len(fields) > 0 {
		fieldsStr = " "
		for k, v := range fields {
			fieldsStr += fmt.Sprintf("%s=%v ", k, v)
		}
	}

	// Format final log line
	logLine := fmt.Sprintf("%s [%s] %s%s%s\n", timestamp, level.String(), message, fieldsStr, caller)

	// Write to logger
	l.logger.Print(logLine)
}