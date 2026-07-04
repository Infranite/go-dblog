package dblog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

type captureLogger struct {
	level    LogLevel
	messages []string
}

func (l *captureLogger) Logf(level LogLevel, format string, args ...any) {
	if !l.Enabled(level) {
		return
	}
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

func (l *captureLogger) Enabled(level LogLevel) bool {
	return l.level != LevelOff && level >= l.level
}

func (l *captureLogger) SetLevel(level LogLevel) { l.level = level }

func (l *captureLogger) Level() LogLevel { return l.level }

func TestStdLoggerHonorsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := NewStdLogger("test", LevelWarn)
	logger.StandardLogger().SetOutput(&buf)
	logger.StandardLogger().SetFlags(0)

	logger.Logf(LevelInfo, "hidden")
	logger.Logf(LevelError, "visible %s", "message")

	output := buf.String()
	if strings.Contains(output, "hidden") {
		t.Fatalf("info log was written at warn level: %q", output)
	}
	if !strings.Contains(output, "ERROR visible message") {
		t.Fatalf("error log missing from output: %q", output)
	}
}

func TestLoggerSlotCanReplaceAndReset(t *testing.T) {
	slot := NewLoggerSlot("test")
	custom := &captureLogger{level: LevelInfo}
	slot.SetLogger(custom)

	slot.SetLevel(LevelDebug)
	if !slot.Enabled(LevelDebug) {
		t.Fatal("debug level is disabled")
	}
	slot.Logf(LevelDebug, "debug %d", 1)

	if custom.Level() != LevelDebug {
		t.Fatalf("custom level = %s, want %s", custom.Level(), LevelDebug)
	}
	if len(custom.messages) != 1 || custom.messages[0] != "debug 1" {
		t.Fatalf("messages = %#v", custom.messages)
	}

	slot.SetLogger(nil)
	if slot.GetLogger() == custom {
		t.Fatal("nil logger did not restore default logger")
	}
}
