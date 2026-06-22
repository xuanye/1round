package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Log.Level != "info" {
		t.Errorf("expected default Log.Level to be 'info', got '%s'", cfg.Log.Level)
	}
	if cfg.Log.OutputPath != "./logs/oneround.log" {
		t.Errorf("expected default Log.OutputPath to be './logs/oneround.log', got '%s'", cfg.Log.OutputPath)
	}
}

func TestApplyEnv(t *testing.T) {
	os.Setenv("ONEROUND_LOG_LEVEL", "debug")
	os.Setenv("ONEROUND_LOG_OUTPUT_PATH", "/tmp/test.log")
	defer func() {
		os.Unsetenv("ONEROUND_LOG_LEVEL")
		os.Unsetenv("ONEROUND_LOG_OUTPUT_PATH")
	}()

	cfg := Default()
	applyEnv(&cfg)

	if cfg.Log.Level != "debug" {
		t.Errorf("expected Log.Level override to be 'debug', got '%s'", cfg.Log.Level)
	}
	if cfg.Log.OutputPath != "/tmp/test.log" {
		t.Errorf("expected Log.OutputPath override to be '/tmp/test.log', got '%s'", cfg.Log.OutputPath)
	}
}
