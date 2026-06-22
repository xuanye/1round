package logger

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuanye/one-round/apps/server/internal/config"
	"go.uber.org/zap"
)

type logLine struct {
	Level   string `json:"level"`
	Msg     string `json:"msg"`
	Func    string `json:"func"`
	File    string `json:"file"`
	Line    int    `json:"line"`
}

func TestLoggerAdapter(t *testing.T) {
	// Create temporary directory for tests
	tmpDir, err := os.MkdirTemp("", "oneround-logger-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logFilePath := filepath.Join(tmpDir, "test.log")

	// Setup config
	cfg := config.Default()
	cfg.Log.Level = "debug"
	cfg.Log.OutputPath = logFilePath

	// Create logger
	l := NewZapLoggerAdapter(&cfg)
	defer l.Sync()

	// Log some messages
	l.Debug("debug message", zap.String("key", "val"))
	l.Info("info message")
	l.Warn("warn message")
	l.Error("error message")

	l.DebugF("debugf message %d", 123)
	l.InfoF("infof message %s", "hello")
	l.WarnF("warnf message %t", true)
	l.ErrorF("errorf message %f", 3.14)

	// We must Sync the logger to ensure all logs are flushed
	_ = l.Sync()

	// Check if the log file was created (the writer formats it as file-YYYY-MM-DD.log)
	// Let's locate the file
	files, err := filepath.Glob(filepath.Join(tmpDir, "test-*.log"))
	if err != nil {
		t.Fatalf("failed to glob temp log files: %v", err)
	}

	if len(files) == 0 {
		t.Fatalf("expected log file to be created in temp dir, but none found")
	}

	content, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	var lines []logLine
	for scanner.Scan() {
		var line logLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			t.Logf("Failed to unmarshal log line %q: %v", scanner.Text(), err)
			continue
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}

	// We logged 8 messages total (4 debug/info/warn/error + 4 formatted variants)
	// Verify total messages
	expectedCount := 8
	if len(lines) != expectedCount {
		t.Errorf("expected %d log lines, got %d", expectedCount, len(lines))
	}

	// Verify details of some lines
	// Line 0: Debug message (should have caller info since Debug is logged with callerFields)
	if lines[0].Level != "debug" || lines[0].Msg != "debug message" {
		t.Errorf("unexpected debug log: %+v", lines[0])
	}
	if lines[0].Func == "" || !strings.Contains(lines[0].File, "logger_test.go") || lines[0].Line == 0 {
		t.Errorf("expected caller info on debug log, got: func=%q, file=%q, line=%d", lines[0].Func, lines[0].File, lines[0].Line)
	}

	// Line 1: Info message (no caller info added by adapter since Info uses log.logger.Info directly)
	if lines[1].Level != "info" || lines[1].Msg != "info message" {
		t.Errorf("unexpected info log: %+v", lines[1])
	}
	if lines[1].Func != "" || lines[1].File != "" || lines[1].Line != 0 {
		t.Errorf("expected NO adapter caller info on info log, got: func=%q, file=%q, line=%d", lines[1].Func, lines[1].File, lines[1].Line)
	}

	// Line 2: Warn message (should have caller info)
	if lines[2].Level != "warn" || lines[2].Msg != "warn message" {
		t.Errorf("unexpected warn log: %+v", lines[2])
	}
	if lines[2].Func == "" {
		t.Errorf("expected caller info on warn log")
	}

	// Line 3: Error message (should have caller info)
	if lines[3].Level != "error" || lines[3].Msg != "error message" {
		t.Errorf("unexpected error log: %+v", lines[3])
	}
	if lines[3].Func == "" {
		t.Errorf("expected caller info on error log")
	}

	// Line 4: DebugF message
	if lines[4].Level != "debug" || lines[4].Msg != "debugf message 123" {
		t.Errorf("unexpected debugf log: %+v", lines[4])
	}

	// Line 5: InfoF message
	if lines[5].Level != "info" || lines[5].Msg != "infof message hello" {
		t.Errorf("unexpected infof log: %+v", lines[5])
	}
}

func TestNewConsoleAndNop(t *testing.T) {
	// Simple test to ensure NewConsole and NewNop run without crashing
	consoleLogger := NewConsole()
	if consoleLogger == nil {
		t.Error("NewConsole returned nil")
	}
	consoleLogger.Info("test console info message")
	_ = consoleLogger.Sync()

	nopLogger := NewNop()
	if nopLogger == nil {
		t.Error("NewNop returned nil")
	}
	nopLogger.Info("test nop info message")
	_ = nopLogger.Sync()
}
