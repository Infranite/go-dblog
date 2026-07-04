package dblog

import (
	"log"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
)

// LogLevel controls which log records are written.
type LogLevel int32

const (
	// LevelDebug writes all log records.
	LevelDebug LogLevel = iota
	// LevelInfo writes informational, warning, and error records.
	LevelInfo
	// LevelWarn writes warning and error records.
	LevelWarn
	// LevelError writes only error records.
	LevelError
	// LevelOff disables logging.
	LevelOff
)

// String returns the stable label used in log output.
func (level LogLevel) String() string {
	switch level {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelOff:
		return "OFF"
	default:
		return "LEVEL(" + strconv.Itoa(int(level)) + ")"
	}
}

// Logger is the logging contract used by go-dblog packages.
type Logger interface {
	Enabled(level LogLevel) bool
	Logf(level LogLevel, format string, args ...any)
	SetLevel(level LogLevel)
	Level() LogLevel
}

// StdLogger is the default Logger implementation backed by the standard log
// package.
type StdLogger struct {
	logger *log.Logger
	level  atomic.Int32
}

// NewStdLogger returns a standard-library logger with a package prefix.
func NewStdLogger(name string, level LogLevel) *StdLogger {
	logger := &StdLogger{
		logger: log.New(os.Stderr, logPrefix(name), log.LstdFlags),
	}
	logger.SetLevel(level)
	return logger
}

// StandardLogger returns the wrapped standard-library logger for output,
// prefix, and flag customization.
func (logger *StdLogger) StandardLogger() *log.Logger {
	return logger.logger
}

// Logf writes a formatted log record when level is enabled.
func (logger *StdLogger) Logf(level LogLevel, format string, args ...any) {
	if logger == nil || !logger.Enabled(level) {
		return
	}
	logger.logger.Printf(level.String()+" "+format, args...)
}

// Enabled reports whether level is enabled.
func (logger *StdLogger) Enabled(level LogLevel) bool {
	if logger == nil {
		return false
	}
	current := LogLevel(logger.level.Load())
	recordLevel := normalizeLogLevel(level)
	return current != LevelOff && recordLevel != LevelOff && recordLevel >= current
}

// SetLevel changes the minimum enabled log level.
func (logger *StdLogger) SetLevel(level LogLevel) {
	if logger == nil {
		return
	}
	logger.level.Store(int32(normalizeLogLevel(level)))
}

// Level returns the minimum enabled log level.
func (logger *StdLogger) Level() LogLevel {
	if logger == nil {
		return LevelOff
	}
	return LogLevel(logger.level.Load())
}

// LoggerSlot stores one replaceable package-global logger.
type LoggerSlot struct {
	mu       sync.RWMutex
	logger   Logger
	fallback Logger
}

// NewLoggerSlot creates a package-global logger slot with a default StdLogger.
func NewLoggerSlot(name string) *LoggerSlot {
	logger := NewStdLogger(name, LevelInfo)
	return &LoggerSlot{
		logger:   logger,
		fallback: logger,
	}
}

// GetLogger returns the current logger.
func (slot *LoggerSlot) GetLogger() Logger {
	if slot == nil {
		return nil
	}
	slot.mu.RLock()
	logger := slot.logger
	if logger == nil {
		logger = slot.fallback
	}
	slot.mu.RUnlock()
	return logger
}

// SetLogger replaces the current logger. Passing nil restores the default
// standard-library logger for slots created with NewLoggerSlot.
func (slot *LoggerSlot) SetLogger(logger Logger) {
	if slot == nil {
		return
	}
	slot.mu.Lock()
	if logger == nil {
		logger = slot.fallback
	}
	slot.logger = logger
	slot.mu.Unlock()
}

// Logf writes through the current logger.
func (slot *LoggerSlot) Logf(level LogLevel, format string, args ...any) {
	logger := slot.GetLogger()
	if logger == nil {
		return
	}
	logger.Logf(level, format, args...)
}

// Enabled reports whether level is enabled on the current logger.
func (slot *LoggerSlot) Enabled(level LogLevel) bool {
	logger := slot.GetLogger()
	return logger != nil && logger.Enabled(level)
}

// SetLevel changes the level on the current logger.
func (slot *LoggerSlot) SetLevel(level LogLevel) {
	logger := slot.GetLogger()
	if logger == nil {
		return
	}
	logger.SetLevel(level)
}

// Level returns the current logger level.
func (slot *LoggerSlot) Level() LogLevel {
	logger := slot.GetLogger()
	if logger == nil {
		return LevelOff
	}
	return logger.Level()
}

// Log is the package-global logger for the root dblog package.
var Log = NewLoggerSlot("go-dblog")

func logPrefix(name string) string {
	if name == "" {
		return ""
	}
	return name + ": "
}

func normalizeLogLevel(level LogLevel) LogLevel {
	if level < LevelDebug {
		return LevelDebug
	}
	if level > LevelOff {
		return LevelOff
	}
	return level
}
