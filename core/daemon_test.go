package core

import (
	"os"
	"path/filepath"
	"strings"
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

// not parallel: setupDaemonTestEnv swaps package-level paths and restores them
// on cleanup, which races with any other test reading them
func TestRunSchedulerLoopWithNoTasks(t *testing.T) {
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

// not parallel: see TestRunSchedulerLoopWithNoTasks
func TestSchedulerTaskDispatchLogic(t *testing.T) {
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

// not parallel: see TestRunSchedulerLoopWithNoTasks
func TestScheduleTaskLogic(t *testing.T) {
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

func TestPersistLastRanAtKeepsConcurrentEdits(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)

	if _, err := CreateTask("DriftTask", SHELL_TASK); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}
	if err := UpdateTaskStatus(SHELL_TASK, "DriftTask", true); err != nil {
		t.Fatalf("failed to enable task: %v", err)
	}

	// what the daemon would be holding in memory
	stale, err := GetTask(SHELL_TASK, "DriftTask")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if !stale.GetExecution().IsEnabled {
		t.Fatal("task should be enabled at this point")
	}

	// meanwhile the CLI disables it on disk
	if err := UpdateTaskStatus(SHELL_TASK, "DriftTask", false); err != nil {
		t.Fatalf("failed to disable task: %v", err)
	}

	if err := persistLastRanAt(SHELL_TASK, "DriftTask", time.Now()); err != nil {
		t.Fatalf("persistLastRanAt failed: %v", err)
	}

	after, err := GetTask(SHELL_TASK, "DriftTask")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if after.GetExecution().IsEnabled {
		t.Error("persisting last_ran_at reverted the is_enabled change made on disk")
	}
	if after.GetExecution().GetLastRanAtTime() == nil {
		t.Error("last_ran_at was not stamped")
	}
}

func TestLoadScheduleSkipsDisabledTasks(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "On", enabled: true, interval: Interval{Minutes: 5}})
	writeHttpTask(t, rec, taskSpec{name: "Off", enabled: false, interval: Interval{Minutes: 5}})

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	if len(schedule) != 1 || schedule[0].Task.GetName() != "On" {
		t.Fatalf("expected only the enabled task, got %v", scheduleNames(schedule))
	}
}

func TestLoadSchedulePreservesPendingNextRun(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Recurring", enabled: true, interval: Interval{Hours: 1}})

	first, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}
	if len(first) != 1 {
		t.Fatalf("expected 1 scheduled task, got %d", len(first))
	}

	pending := time.Now().Add(42 * time.Minute)
	first[0].NextRun = pending

	second, err := loadSchedule(first)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if len(second) != 1 {
		t.Fatalf("expected 1 scheduled task after reload, got %d", len(second))
	}

	if !second[0].NextRun.Equal(pending) {
		t.Errorf("NextRun was reset by the reload: got %v, want %v", second[0].NextRun, pending)
	}
}

func TestLoadSchedulePicksUpNewlyEnabledTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Existing", enabled: true, interval: Interval{Hours: 1}})

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	writeHttpTask(t, rec, taskSpec{name: "Added", enabled: true, interval: Interval{Hours: 1}})

	schedule, err = loadSchedule(schedule)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if len(schedule) != 2 {
		t.Fatalf("reload should have picked up the new task, got %v", scheduleNames(schedule))
	}
}

func TestLoadScheduleDropsDisabledTaskOnReload(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Toggle", enabled: true, interval: Interval{Hours: 1}})

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}
	if len(schedule) != 1 {
		t.Fatalf("expected 1 scheduled task, got %d", len(schedule))
	}

	if err := UpdateTaskStatus(HTTP_TASK, "Toggle", false); err != nil {
		t.Fatalf("failed to disable task: %v", err)
	}

	schedule, err = loadSchedule(schedule)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if len(schedule) != 0 {
		t.Errorf("disabled task should have dropped out of the schedule, got %v", scheduleNames(schedule))
	}
}

func TestLoadScheduleDoesNotReArmCompletedOneShot(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "OneShot", enabled: true})

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}
	if len(schedule) != 1 {
		t.Fatalf("a one-shot task should be scheduled before it runs, got %d", len(schedule))
	}

	if err := persistLastRanAt(HTTP_TASK, "OneShot", time.Now()); err != nil {
		t.Fatalf("persistLastRanAt failed: %v", err)
	}

	schedule, err = loadSchedule(nil)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if len(schedule) != 0 {
		t.Errorf("a completed one-shot task should not be rescheduled, got %v", scheduleNames(schedule))
	}
}

func TestDispatchDoesNotBlockWhenQueueIsFull(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	first := writeHttpTask(t, rec, taskSpec{name: "First", enabled: true})
	second := writeHttpTask(t, rec, taskSpec{name: "Second", enabled: true})

	// capacity 1 and nothing draining it, so the second send has nowhere to go
	queue := make(chan Task, 1)

	t.Cleanup(func() {
		releaseInFlight("First")
		releaseInFlight("Second")
	})

	done := make(chan struct{})
	go func() {
		dispatch(queue, first)
		dispatch(queue, second)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("dispatch blocked the scheduling loop with a full queue")
	}
}

func TestDispatchSkipsTaskAlreadyRunning(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	task := writeHttpTask(t, rec, taskSpec{name: "Slow", enabled: true})

	queue := make(chan Task, 4)
	t.Cleanup(func() { releaseInFlight("Slow") })

	dispatch(queue, task)
	dispatch(queue, task)

	if len(queue) != 1 {
		t.Errorf("expected 1 queued run, got %d", len(queue))
	}

	releaseInFlight("Slow")

	dispatch(queue, task)
	if len(queue) != 2 {
		t.Errorf("expected the task to be dispatchable again once released, got %d queued", len(queue))
	}
}

func TestDeleteStaleLogFilesKeepsOpenLogs(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	if err := LogMessageToFile("active log line", "info", "app", nil); err != nil {
		t.Fatalf("LogMessageToFile failed: %v", err)
	}

	stale := filepath.Join(AppLogDirPath, "stale.log")
	if err := os.WriteFile(stale, []byte("old\n"), 0644); err != nil {
		t.Fatalf("failed to write stale log: %v", err)
	}

	if err := DeleteStaleLogFiles(AppLogDirPath); err != nil {
		t.Fatalf("DeleteStaleLogFiles failed: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("the stale log file should have been removed")
	}

	if _, err := os.Stat(AppLogFilePath); err != nil {
		t.Errorf("the log file still being written to should have been kept: %v", err)
	}
}

func TestStartLogWiperWithNonPositiveIntervalDoesNotPanic(t *testing.T) {
	setupLogTestEnv(t)
	InitLogger()

	StartLogWiper(AppLogDirPath, 0)
	StartLogWiper(AppLogDirPath, -5*time.Second)
}

func scheduleNames(schedule []ScheduledTask) []string {
	names := make([]string, 0, len(schedule))
	for _, st := range schedule {
		names = append(names, st.Task.GetName())
	}

	return names
}

// The next run used to be computed from time.Now() at dispatch, so however late
// a dispatch was got baked into the series permanently and never corrected.
func TestAdvanceNextRunStaysOnTheGrid(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC)
	interval := Interval{Hours: 24}

	// dispatched 90 seconds late
	late := scheduled.Add(90 * time.Second)

	next := advanceIntervalRun(scheduled, interval, late)
	want := scheduled.AddDate(0, 0, 1)

	if !next.Equal(want) {
		t.Errorf("next = %v, want %v (the lateness must not carry over)", next, want)
	}
}

func TestAdvanceNextRunDoesNotAccumulateDrift(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	interval := Interval{Seconds: 30}

	at := start
	for i := 1; i <= 200; i++ {
		// every dispatch runs a quarter of an interval late
		at = advanceIntervalRun(at, interval, at.Add(7*time.Second))

		want := start.Add(time.Duration(i) * 30 * time.Second)
		if !at.Equal(want) {
			t.Fatalf("after %d late dispatches: %v, want %v", i, at, want)
		}
	}
}

// Falling a long way behind must not leave a backlog of past-due runs.
func TestAdvanceNextRunSkipsPastDueRuns(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	interval := Interval{Minutes: 10}

	// the machine was asleep for three hours
	now := scheduled.Add(3 * time.Hour)

	next := advanceIntervalRun(scheduled, interval, now)

	if !next.After(now) {
		t.Errorf("next = %v, should be in the future relative to %v", next, now)
	}

	// still on the original grid, not now + interval
	if next.Sub(scheduled)%(10*time.Minute) != 0 {
		t.Errorf("next = %v is off the 10 minute grid anchored at %v", next, scheduled)
	}
	if want := scheduled.Add(3*time.Hour + 10*time.Minute); !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestAdvanceNextRunHandlesCalendarIntervals(t *testing.T) {
	t.Parallel()

	// January has 31 days, so a month is not a fixed duration
	scheduled := time.Date(2026, 1, 31, 12, 0, 0, 0, time.UTC)

	next := advanceIntervalRun(scheduled, Interval{Months: 1}, scheduled)
	want := scheduled.AddDate(0, 1, 0)

	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

func TestAdvanceNextRunWithAZeroIntervalTerminates(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// a zero interval would otherwise loop forever; the caller drops one-shot
	// tasks instead of advancing them, but the guard must hold regardless
	if got := advanceIntervalRun(scheduled, Interval{}, scheduled.Add(time.Hour)); !got.Equal(scheduled) {
		t.Errorf("got %v, want the scheduled time back unchanged", got)
	}
}

func TestAdvanceNextRunUsesTheCalendarSchedule(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 3, 4, 3, 0, 0, 0, time.UTC)

	st := ScheduledTask{
		Task: HttpTask{
			Name:      "Nightly",
			Execution: Execution{IsEnabled: true, At: []string{"03:00"}},
		},
		NextRun: scheduled,
	}

	next, repeats := advanceNextRun(st, scheduled)
	if !repeats {
		t.Fatal("a calendar task repeats; it must not be dropped")
	}

	want := time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

// A clock time missed while the daemon was busy or down is simply missed.
func TestAdvanceNextRunSkipsAMissedClockTime(t *testing.T) {
	t.Parallel()

	scheduled := time.Date(2026, 3, 4, 3, 0, 0, 0, time.UTC)

	st := ScheduledTask{
		Task: HttpTask{
			Name:      "Nightly",
			Execution: Execution{IsEnabled: true, At: []string{"03:00"}},
		},
		NextRun: scheduled,
	}

	// dispatched nine hours late
	next, repeats := advanceNextRun(st, scheduled.Add(9*time.Hour))
	if !repeats {
		t.Fatal("should still repeat")
	}

	// tomorrow's 03:00, with no attempt to make up the one it missed
	want := time.Date(2026, 3, 5, 3, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Errorf("next = %v, want %v", next, want)
	}
}

// A calendar task has no interval, so the one-shot rule must not claim it.
func TestACalendarTaskIsNotTreatedAsOneShot(t *testing.T) {
	t.Parallel()

	st := ScheduledTask{
		Task: HttpTask{
			Name:      "Nightly",
			Execution: Execution{IsEnabled: true, At: []string{"03:00"}},
		},
		NextRun:  time.Date(2026, 3, 4, 3, 0, 0, 0, time.UTC),
		Interval: Interval{},
	}

	if _, repeats := advanceNextRun(st, st.NextRun); !repeats {
		t.Error("a calendar task must not be dropped as a one-shot")
	}
}

func TestLoadScheduleKeepsACalendarTaskThatHasAlreadyRun(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	// zero interval plus a last_ran_at stamp is the one-shot shape; the at
	// field has to override that
	task := HttpTask{
		Name: "Nightly",
		Execution: Execution{
			IsEnabled: true,
			At:        []string{"03:00"},
			LastRanAt: time.Now().Add(-5 * time.Hour).Format(time.RFC3339),
		},
		URL:    "http://127.0.0.1:1",
		Method: METHOD_GET,
	}
	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	if len(schedule) != 1 {
		t.Fatalf("expected the calendar task to stay scheduled, got %v", scheduleNames(schedule))
	}
	if schedule[0].NextRun.Hour() != 3 || schedule[0].NextRun.Minute() != 0 {
		t.Errorf("NextRun = %v, want an 03:00 slot", schedule[0].NextRun)
	}
}

// An unusable calendar schedule must keep the task out of the schedule rather
// than parking something that can never fire.
func TestLoadScheduleDropsATaskWithAnInvalidCalendarSchedule(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	task := HttpTask{
		Name:      "Broken",
		Execution: Execution{IsEnabled: true, At: []string{"half past three"}},
		URL:       "http://127.0.0.1:1",
		Method:    METHOD_GET,
	}
	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	if len(schedule) != 0 {
		t.Errorf("a task with an unparsable at time should not be scheduled, got %v", scheduleNames(schedule))
	}

	logged, err := os.ReadFile(AppLogFilePath)
	if err != nil {
		t.Fatalf("failed to read the app log: %v", err)
	}
	if !strings.Contains(string(logged), "Broken") {
		t.Errorf("the problem should be logged, got:\n%s", logged)
	}
}

func TestScheduleTaskPrefersTheCalendarOverInterval(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	task := HttpTask{
		Name: "Both",
		Execution: Execution{
			IsEnabled: true,
			At:        []string{"03:00"},
			Interval:  Interval{Minutes: 5},
		},
		URL:    "http://127.0.0.1:1",
		Method: METHOD_GET,
	}

	st := scheduleTask(task)
	if st == nil {
		t.Fatal("the task should still be scheduled")
	}

	if st.NextRun.Hour() != 3 || st.NextRun.Minute() != 0 {
		t.Errorf("NextRun = %v, want an 03:00 slot rather than now+5m", st.NextRun)
	}

	logged, err := os.ReadFile(AppLogFilePath)
	if err != nil {
		t.Fatalf("failed to read the app log: %v", err)
	}
	if !strings.Contains(string(logged), "those are ignored") {
		t.Errorf("the ignored interval should be logged, got:\n%s", logged)
	}
}
