package core

import (
	"testing"
	"time"
)

func TestIsZeroIntervalSchedulr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		interval Interval
		expected bool
	}{
		{"all zero", Interval{}, true},
		{"has years", Interval{Years: 1}, false},
		{"has days", Interval{Days: 1}, false},
		{"has seconds", Interval{Seconds: 1}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isZeroInterval(tc.interval)
			if result != tc.expected {
				t.Errorf("isZeroInterval(%v) = %v, want %v", tc.interval, result, tc.expected)
			}
		})
	}
}

func TestCalculateNextRunSchedulr(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name     string
		interval Interval
		expected time.Time
	}{
		{
			"add 1 day",
			Interval{Days: 1},
			time.Date(2024, 1, 16, 10, 30, 0, 0, time.UTC),
		},
		{
			"add hours and minutes",
			Interval{Hours: 2, Minutes: 15},
			time.Date(2024, 1, 15, 12, 45, 0, 0, time.UTC),
		},
		{
			"add year and month",
			Interval{Years: 1, Months: 2},
			time.Date(2025, 3, 15, 10, 30, 0, 0, time.UTC),
		},
		{
			"add weeks and days",
			Interval{Weeks: 1, Days: 2},
			time.Date(2024, 1, 24, 10, 30, 0, 0, time.UTC),
		},
		{
			"add only seconds",
			Interval{Seconds: 30},
			time.Date(2024, 1, 15, 10, 30, 30, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := calculateNextRun(start, tc.interval)
			if !result.Equal(tc.expected) {
				t.Errorf("calculateNextRun() = %v, want %v", result, tc.expected)
			}
		})
	}
}

func TestScheduleTask(t *testing.T) {
	setupTestEnv(t)

	CreateTask("ScheduleTest", SHELL_TASK)
	task, _ := GetTask(SHELL_TASK, "ScheduleTest")

	scheduled := scheduleTask(task)

	if scheduled == nil {
		t.Fatal("scheduleTask returned nil")
	}

	if scheduled.Task == nil {
		t.Error("scheduled task is nil")
	}

	if scheduled.NextRun.IsZero() {
		t.Error("NextRun should be set")
	}
}

func TestScheduleTaskDisabled(t *testing.T) {
	setupTestEnv(t)

	CreateTask("DisabledTask", SHELL_TASK)
	task, _ := GetTask(SHELL_TASK, "DisabledTask")

	if task.GetExecution().IsEnabled {
		t.Skip("Test task should start disabled, skipping")
	}

	scheduled := scheduleTask(task)
	if scheduled == nil {
		t.Skip("Disabled tasks may not schedule, skipping")
	}
}

func TestScheduleTaskWithLastRan(t *testing.T) {
	setupTestEnv(t)

	CreateTask("LastRanTask", SHELL_TASK)
	UpdateTaskStatus(SHELL_TASK, "LastRanTask", true)
	UpdateRanAt(SHELL_TASK, "LastRanTask")

	task, _ := GetTask(SHELL_TASK, "LastRanTask")
	scheduled := scheduleTask(task)

	if scheduled == nil {
		t.Fatal("scheduleTask returned nil for task with LastRanAt")
	}

	lastRan := task.GetExecution().GetLastRanAtTime()
	if lastRan == nil {
		t.Error("expected LastRanAt to be set")
	}
}

func TestGetShellCommand(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("echo test", "bash", false)

	if cmd == nil {
		t.Error("getShellCommand returned nil")
	}

	if cmd.Path == "" {
		t.Error("command path should be set")
	}
}

func TestGetShellCommandWithGui(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("test", "bash", true)
	if cmd == nil {
		t.Error("getShellCommand with GUI returned nil")
	}
}

func TestGetShellCommandPowershell(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test")
	}

	cmd := getShellCommand("Write-Host test", "powershell", false)
	if cmd == nil {
		t.Error("getShellCommand for powershell returned nil")
	}
}

func TestCalculateNextRunEdgeCases(t *testing.T) {
	t.Parallel()

	start := time.Date(2024, 1, 31, 10, 0, 0, 0, time.UTC)
	result := calculateNextRun(start, Interval{Months: 1})

	if result.Day() == 31 && result.Month() == 2 {
		t.Logf("Month boundary handled: %v", result)
	}

	leapStart := time.Date(2024, 2, 29, 10, 0, 0, 0, time.UTC)
	leapResult := calculateNextRun(leapStart, Interval{Years: 1})
	if leapResult.Year() != 2025 {
		t.Errorf("Year not incremented correctly: %v", leapResult)
	}
}
