package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

type LogLevel int

const (
	LogDebug LogLevel = iota
	LogInfo
	LogWarn
	LogError
	LogFatal
)

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

type Logger struct {
	logger   *log.Logger
	level    LogLevel
	mu       sync.RWMutex
	fields   map[string]interface{}
}

func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0), 
		level:  LogInfo,
		fields: make(map[string]interface{}),
	}
}

func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

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

func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(LogDebug, format, args...)
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log(LogInfo, format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(LogWarn, format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log(LogError, format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(LogFatal, format, args...)
	os.Exit(1)
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	l.mu.RLock()
	currentLevel := l.level
	fields := l.fields
	l.mu.RUnlock()

	if level < currentLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	var caller string
	if level >= LogError {
		_, file, line, ok := runtime.Caller(2)
		if ok {

			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' || file[i] == '\\' {
					file = file[i+1:]
					break
				}
			}
			caller = fmt.Sprintf(" [%s:%d]", file, line)
		}
	}

	message := fmt.Sprintf(format, args...)

	var fieldsStr string
	if len(fields) > 0 {
		fieldsStr = " "
		for k, v := range fields {
			fieldsStr += fmt.Sprintf("%s=%v ", k, v)
		}
	}

	logLine := fmt.Sprintf("%s [%s] %s%s%s\n", timestamp, level.String(), message, fieldsStr, caller)

	l.logger.Print(logLine)
}