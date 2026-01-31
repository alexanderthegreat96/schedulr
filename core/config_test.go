package core

import (
	"testing"
)

func TestAppConfig(t *testing.T) {
	t.Parallel()
	cfg := AppConfig()

	if cfg == nil {
		t.Fatal("AppConfig returned nil")
	}

	if cfg.env == nil {
		t.Error("env should not be nil")
	}
}

func TestConfigGetBool(t *testing.T) {
	t.Parallel()
	cfg := AppConfig()

	result := cfg.getBool("NONEXISTENT_BOOL_KEY", true)
	if result != true {
		t.Errorf("getBool with default true returned %v", result)
	}

	result = cfg.getBool("NONEXISTENT_BOOL_KEY", false)
	if result != false {
		t.Errorf("getBool with default false returned %v", result)
	}
}

func TestConfigGetInt(t *testing.T) {
	t.Parallel()
	cfg := AppConfig()

	result := cfg.getInt("NONEXISTENT_INT_KEY", 42)
	if result != 42 {
		t.Errorf("getInt returned %d, want 42", result)
	}

	result = cfg.getInt("NONEXISTENT_INT_KEY", 0)
	if result != 0 {
		t.Errorf("getInt returned %d, want 0", result)
	}
}

func TestConfigGetStr(t *testing.T) {
	t.Parallel()
	cfg := AppConfig()

	result := cfg.getStr("NONEXISTENT_STR_KEY", "default")
	if result != "default" {
		t.Errorf("getStr returned %q, want %q", result, "default")
	}

	result = cfg.getStr("NONEXISTENT_STR_KEY", "")
	if result != "" {
		t.Errorf("getStr returned %q, want empty string", result)
	}
}

func TestConfigDefaults(t *testing.T) {
	t.Parallel()
	cfg := AppConfig()

	if cfg.WipeLogDataInterval < 0 {
		t.Error("WipeLogDataInterval should be non-negative")
	}

	if cfg.WorkerCount <= 0 {
		t.Error("WorkerCount should be positive")
	}

	if cfg.ServiceName == "" {
		t.Error("ServiceName should not be empty")
	}

	if cfg.SystemDCommand == "" {
		t.Error("SystemDCommand should not be empty")
	}

	if cfg.LaunchDCommand == "" {
		t.Error("LaunchDCommand should not be empty")
	}
}
