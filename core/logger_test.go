package core

import (
	"path/filepath"
	"testing"
)

func setupLogTestEnv(t *testing.T) {
	tempDir := t.TempDir()

	oldRootPath := RootPath
	oldAppLogFilePath := AppLogFilePath
	oldAppLogDirPath := AppLogDirPath
	oldTasksLogDirPath := TasksLogDirPath

	RootPath = tempDir
	AppLogFilePath = filepath.Join(tempDir, APP_LOGS_DIR, "test.log")
	AppLogDirPath = filepath.Join(tempDir, APP_LOGS_DIR)
	TasksLogDirPath = filepath.Join(tempDir, TASK_LOGS_DIR)

	t.Cleanup(func() {
		RootPath = oldRootPath
		AppLogFilePath = oldAppLogFilePath
		AppLogDirPath = oldAppLogDirPath
		TasksLogDirPath = oldTasksLogDirPath
	})
}

func TestInitLogger(t *testing.T) {
	InitLogger()

	if consoleLogger == nil {
		t.Error("consoleLogger should be initialized")
	}
}

func TestLogMessage(t *testing.T) {
	t.Parallel()
	InitLogger()

	LogMessage("test message", "info")
	LogMessage("debug message", "debug")
	LogMessage("error message", "error")
	LogMessage("warning message", "warning")
}

func TestLogMessageToFileApp(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	err := LogMessageToFile("test app log", "info", "app", nil)
	if err != nil {
		t.Fatalf("LogMessageToFile failed: %v", err)
	}
}

func TestLogMessageToFileTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	CreateTask("LogTest", SHELL_TASK)
	task, _ := GetTask(SHELL_TASK, "LogTest")

	err := LogMessageToFile("test task log", "info", "task", task)
	if err != nil {
		t.Fatalf("LogMessageToFile for task failed: %v", err)
	}
}

func TestLogMessageToFileInvalidType(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	err := LogMessageToFile("test", "info", "invalid", nil)
	if err == nil {
		t.Error("expected error for invalid log type")
	}
}

func TestLogMessageToFileLoggingDisabled(t *testing.T) {
	t.Skip("LogData configuration check requires config file setup")
}

func TestCloseLoggers(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	LogMessageToFile("test", "info", "app", nil)

	CloseLoggers()
}

func TestGetOrCreateLogFilePathForTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	CreateTask("PathTest", SHELL_TASK)
	task, _ := GetTask(SHELL_TASK, "PathTest")

	path1 := getOrCreateLogFilePathForTask(task)
	if path1 == "" {
		t.Error("expected non-empty path")
	}

	path2 := getOrCreateLogFilePathForTask(task)
	if path1 != path2 {
		t.Errorf("paths should match: %q vs %q", path1, path2)
	}

	if !filepath.IsAbs(path1) {
		t.Errorf("path should be absolute: %s", path1)
	}
}

func TestLogLevelColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		level string
		check func(string) bool
	}{
		{"info", func(c string) bool { return c != "" }},
		{"debug", func(c string) bool { return c != "" }},
		{"error", func(c string) bool { return c != "" }},
		{"warning", func(c string) bool { return c != "" }},
	}

	for _, tc := range tests {
		t.Run(tc.level, func(t *testing.T) {
			color := getColorForLevel(tc.level)
			if !tc.check(color) {
				t.Errorf("unexpected color for level %s: %q", tc.level, color)
			}
		})
	}
}

func TestLevelUpper(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input    string
		expected string
	}{
		{"info", "INFO"},
		{"debug", "DEBUG"},
		{"error", "ERROR"},
		{"warning", "WARNING"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := levelUpper(tc.input)
			if result != tc.expected {
				t.Errorf("levelUpper(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}

func TestMultipleTaskLoggers(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	CreateTask("Task1", SHELL_TASK)
	CreateTask("Task2", SHELL_TASK)

	task1, _ := GetTask(SHELL_TASK, "Task1")
	task2, _ := GetTask(SHELL_TASK, "Task2")

	err1 := LogMessageToFile("task1 log", "info", "task", task1)
	if err1 != nil {
		t.Fatalf("logging task1 failed: %v", err1)
	}

	err2 := LogMessageToFile("task2 log", "info", "task", task2)
	if err2 != nil {
		t.Fatalf("logging task2 failed: %v", err2)
	}
}

func TestLogMessageVariousLevels(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	levels := []string{"info", "debug", "error", "warning"}
	for _, level := range levels {
		err := LogMessageToFile("test message", level, "app", nil)
		if err != nil {
			t.Errorf("failed to log with level %q: %v", level, err)
		}
	}
}
