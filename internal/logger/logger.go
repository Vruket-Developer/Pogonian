package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level defines log severity levels.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// LogEntry represents a single structured log line.
type LogEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Level     string         `json:"level"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
}

// Logger provides thread-safe structured logging in JSON or colorized console format.
type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	level  Level
	isJSON bool
}

var globalLogger = NewLogger(os.Stdout, LevelInfo, false)

// NewLogger creates a new structured Logger.
func NewLogger(out io.Writer, level Level, isJSON bool) *Logger {
	return &Logger{
		out:    out,
		level:  level,
		isJSON: isJSON,
	}
}

// Default returns the global default logger.
func Default() *Logger {
	return globalLogger
}

// SetLevel sets the active log level.
func SetLevel(level Level) {
	globalLogger.level = level
}

// SetJSONMode toggles JSON vs console output.
func SetJSONMode(isJSON bool) {
	globalLogger.isJSON = isJSON
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()

	if l.isJSON {
		entry := LogEntry{
			Timestamp: now,
			Level:     level.String(),
			Message:   msg,
			Fields:    fields,
		}
		b, _ := json.Marshal(entry)
		fmt.Fprintln(l.out, string(b))
	} else {
		levelColor := "\x1b[1;36m" // info cyan
		switch level {
		case LevelDebug:
			levelColor = "\x1b[1;30m" // dark gray
		case LevelWarn:
			levelColor = "\x1b[1;33m" // yellow
		case LevelError:
			levelColor = "\x1b[1;31m" // red
		}

		timeStr := now.Format("15:04:05.000")
		fmt.Fprintf(l.out, "\x1b[1;30m%s\x1b[0m %s[%s]\x1b[0m %s", timeStr, levelColor, level.String(), msg)

		if len(fields) > 0 {
			fmt.Fprint(l.out, " \x1b[1;30m{")
			first := true
			for k, v := range fields {
				if !first {
					fmt.Fprint(l.out, ", ")
				}
				fmt.Fprintf(l.out, "%s: %v", k, v)
				first = false
			}
			fmt.Fprint(l.out, "}\x1b[0m")
		}
		fmt.Fprintln(l.out)
	}
}

func (l *Logger) Debug(msg string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelDebug, msg, f)
}

func (l *Logger) Info(msg string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelInfo, msg, f)
}

func (l *Logger) Warn(msg string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelWarn, msg, f)
}

func (l *Logger) Error(msg string, fields ...map[string]any) {
	var f map[string]any
	if len(fields) > 0 {
		f = fields[0]
	}
	l.log(LevelError, msg, f)
}

// Global helper wrappers
func Debug(msg string, fields ...map[string]any) { globalLogger.Debug(msg, fields...) }
func Info(msg string, fields ...map[string]any)  { globalLogger.Info(msg, fields...) }
func Warn(msg string, fields ...map[string]any)  { globalLogger.Warn(msg, fields...) }
func Error(msg string, fields ...map[string]any) { globalLogger.Error(msg, fields...) }
