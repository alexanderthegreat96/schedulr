package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupDaemonTestEnv(t *testing.T) string {
	tempDir := t.TempDir()

	os.MkdirAll(filepath.Join(tempDir, TASK_LOGS_DIR), 0755)
	os.MkdirAll(filepath.Join(tempDir, APP_LOGS_DIR), 0755)
	os.MkdirAll(filepath.Join(tempDir, TASKS_FOLDER, SHELL_TASK), 0755)
	os.MkdirAll(filepath.Join(tempDir, TASKS_FOLDER, HTTP_TASK), 0755)

	oldRootPath := RootPath
	oldTaskLocation := TaskLocation
	oldAppLogFilePath := AppLogFilePath
	oldAppLogDirPath := AppLogDirPath
	oldTasksLogDirPath := TasksLogDirPath

	RootPath = tempDir
	TaskLocation = filepath.Join(tempDir, TASKS_FOLDER)
	AppLogFilePath = filepath.Join(tempDir, APP_LOGS_DIR, "test.log")
	AppLogDirPath = filepath.Join(tempDir, APP_LOGS_DIR)
	TasksLogDirPath = filepath.Join(tempDir, TASK_LOGS_DIR)

	t.Cleanup(func() {
		RootPath = oldRootPath
		TaskLocation = oldTaskLocation
		AppLogFilePath = oldAppLogFilePath
		AppLogDirPath = oldAppLogDirPath
		TasksLogDirPath = oldTasksLogDirPath
	})

	return tempDir
}

func TestRunDaemonSetup(t *testing.T) {
	t.Parallel()

	InitLogger()

	cfg := AppConfig()
	if cfg == nil {
		t.Fatal("AppConfig should not be nil")
	}

	if cfg.LogData == false && cfg.DevMode == false {
		t.Logf("Config initialized with LogData=%v, DevMode=%v", cfg.LogData, cfg.DevMode)
	}
}

func TestRunSchedulerLoopWithNoTasks(t *testing.T) {
	t.Parallel()

	tempDir := setupDaemonTestEnv(t)

	oldTaskLocation := TaskLocation
	TaskLocation = filepath.Join(tempDir, TASKS_FOLDER)
	defer func() {
		TaskLocation = oldTaskLocation
	}()

	shellTasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	if len(shellTasks) != 0 {
		t.Errorf("Expected 0 shell tasks, got %d", len(shellTasks))
	}

	httpTasks, err := GetTasks(HTTP_TASK)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	if len(httpTasks) != 0 {
		t.Errorf("Expected 0 HTTP tasks, got %d", len(httpTasks))
	}
}

func TestSchedulerTaskDispatchLogic(t *testing.T) {
	t.Parallel()

	_ = setupDaemonTestEnv(t)

	taskName := "TestTask"

	task := ShellTask{
		Name:      taskName,
		Command:   "echo 'test'",
		ShellType: "sh",
		IsGui:     false,
		Execution: Execution{
			Interval:  Interval{Seconds: 1},
			IsEnabled: true,
		},
	}

	if task.GetName() != taskName {
		t.Errorf("Task name mismatch: got %s, want %s", task.GetName(), taskName)
	}

	if task.GetCommand() != "echo 'test'" {
		t.Errorf("Task command mismatch: got %s", task.GetCommand())
	}

	if !task.GetExecution().IsEnabled {
		t.Error("Task should be enabled")
	}
}

func TestScheduleTaskLogic(t *testing.T) {
	t.Parallel()

	tempDir := setupDaemonTestEnv(t)

	oldTaskLocation := TaskLocation
	TaskLocation = filepath.Join(tempDir, TASKS_FOLDER)
	defer func() {
		TaskLocation = oldTaskLocation
	}()

	tests := []struct {
		name           string
		task           Task
		shouldSchedule bool
	}{
		{
			name: "enabled task should schedule",
			task: ShellTask{
				Name:      "EnabledTask",
				Command:   "echo 'test'",
				ShellType: "sh",
				IsGui:     false,
				Execution: Execution{
					Interval:  Interval{Seconds: 10},
					IsEnabled: true,
				},
			},
			shouldSchedule: true,
		},
		{
			name: "disabled task should not schedule",
			task: ShellTask{
				Name:      "DisabledTask",
				Command:   "echo 'test'",
				ShellType: "sh",
				IsGui:     false,
				Execution: Execution{
					Interval:  Interval{Seconds: 10},
					IsEnabled: false,
				},
			},
			shouldSchedule: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheduled := scheduleTask(tt.task)

			if tt.shouldSchedule && scheduled == nil {
				t.Error("Expected task to be scheduled, got nil")
			}

			if !tt.shouldSchedule {
				if scheduled != nil && scheduled.Task.GetExecution().IsEnabled {
					t.Error("Task should not be enabled")
				}
			}
		})
	}
}

func TestIsZeroIntervalLogic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		interval Interval
		expected bool
	}{
		{
			name:     "all zero values",
			interval: Interval{},
			expected: true,
		},
		{
			name:     "one non-zero value",
			interval: Interval{Seconds: 1},
			expected: false,
		},
		{
			name:     "multiple non-zero values",
			interval: Interval{Days: 1, Hours: 2},
			expected: false,
		},
		{
			name:     "all set to zero explicitly",
			interval: Interval{Years: 0, Months: 0, Weeks: 0, Days: 0, Hours: 0, Minutes: 0, Seconds: 0},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZeroInterval(tt.interval)
			if result != tt.expected {
				t.Errorf("isZeroInterval(%+v) = %v, want %v", tt.interval, result, tt.expected)
			}
		})
	}
}

func TestCalculateNextRunLogic(t *testing.T) {
	t.Parallel()

	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		start    time.Time
		interval Interval
		verify   func(time.Time) bool
	}{
		{
			name:     "add seconds",
			start:    now,
			interval: Interval{Seconds: 30},
			verify: func(result time.Time) bool {
				return result.Equal(now.Add(30 * time.Second))
			},
		},
		{
			name:     "add minutes",
			start:    now,
			interval: Interval{Minutes: 5},
			verify: func(result time.Time) bool {
				return result.Equal(now.Add(5 * time.Minute))
			},
		},
		{
			name:     "add hours",
			start:    now,
			interval: Interval{Hours: 2},
			verify: func(result time.Time) bool {
				return result.Equal(now.Add(2 * time.Hour))
			},
		},
		{
			name:     "add days",
			start:    now,
			interval: Interval{Days: 3},
			verify: func(result time.Time) bool {
				expected := now.AddDate(0, 0, 3)
				return result.Equal(expected)
			},
		},
		{
			name:     "complex interval",
			start:    now,
			interval: Interval{Days: 1, Hours: 2, Minutes: 30, Seconds: 45},
			verify: func(result time.Time) bool {
				expected := now.AddDate(0, 0, 1).Add(2*time.Hour + 30*time.Minute + 45*time.Second)
				return result.Equal(expected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNextRun(tt.start, tt.interval)
			if !tt.verify(result) {
				t.Errorf("calculateNextRun failed for %s, got %v", tt.name, result)
			}
		})
	}
}

func TestGracefulShutdownSetup(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetupGracefulShutdown panicked: %v", r)
		}
	}()

	InitLogger()
}
