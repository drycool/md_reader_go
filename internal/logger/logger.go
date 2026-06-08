// Package logger provides logging and error handling utilities.
// Uses Go 1.21+ log/slog for structured logging.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Level represents log levels.
type Level slog.Level

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

// Logger wraps slog.Logger with module-level context.
type Logger struct {
	*slog.Logger
	module string
}

var globalLogger *Logger

func init() {
	globalLogger = New("md_reader")
}

// New creates a new logger for the given module.
func New(module string) *Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	return &Logger{
		Logger: slog.New(handler),
		module: module,
	}
}

// SetupLogging configures the global logger.
func SetupLogging(level string, logFile string, format string) error {
	if format == "" {
		format = "%(asctime)s | %(name)s | %(levelname)s | %(message)s"
	}

	var slogLevel slog.Level
	switch strings.ToUpper(level) {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "INFO":
		slogLevel = slog.LevelInfo
	case "WARNING", "WARN":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	var writer io.Writer = os.Stderr
	if logFile != "" {
		dir := filepath.Dir(logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating log dir: %w", err)
		}
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("opening log file: %w", err)
		}
		writer = f
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}
	handler := slog.NewTextHandler(writer, opts)

	globalLogger = &Logger{
		Logger: slog.New(handler),
		module: "md_reader",
	}
	return nil
}

// GetLogger returns a named sub-logger.
func GetLogger(name string) *Logger {
	return &Logger{
		Logger: globalLogger.Logger,
		module: fmt.Sprintf("md_reader.%s", name),
	}
}

// WithContext adds context to the logger.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return l
}

// Debug logs a debug message.
func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, append([]any{"module", l.module}, args...)...)
}

// Info logs an info message.
func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, append([]any{"module", l.module}, args...)...)
}

// Warn logs a warning message.
func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, append([]any{"module", l.module}, args...)...)
}

// Error logs an error message.
func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, append([]any{"module", l.module}, args...)...)
}

// Fatal logs a fatal message and exits.
func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, append([]any{"module", l.module}, args...)...)
	os.Exit(1)
}

// With adds key-value pairs to the logger.
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		Logger: l.Logger.With(args...),
		module: l.module,
	}
}

// --- Decorator-style wrappers ---

// FuncLog logs function entry/exit with timing.
// Returns a function to call on exit (defer pattern):
//
//	defer logger.FuncLog(log, "myFunc")()
func FuncLog(l *Logger, name string) func() {
	l.Debug("Starting", "func", name)
	start := time.Now()
	return func() {
		l.Debug("Completed", "func", name, "duration", time.Since(start))
	}
}

// LogError logs an error and returns it wrapped.
func LogError(l *Logger, err error, msg string) error {
	if err != nil {
		l.Error(msg, "error", err)
	}
	return err
}

// Recover logs a panic.
func Recover(l *Logger) {
	if r := recover(); r != nil {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		l.Error("Panic recovered",
			"recover", fmt.Sprintf("%v", r),
			"stack", string(buf[:n]),
		)
	}
}
