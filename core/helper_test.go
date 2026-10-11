package core

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestDefaultValueIfNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    any
		typeName string
		expected any
	}{
		{"nil string", nil, "string", "no-value"},
		{"nil int", nil, "int", 0},
		{"nil float64", nil, "float64", 0.0},
		{"nil bool", nil, "bool", false},
		{"nil map", nil, "map", map[string]any{}},
		{"nil slice", nil, "slice", []any{}},
		{"nil any", nil, "any", "(unset)"},
		{"empty string", "", "string", "no-value"},
		{"zero int", 0, "int", 0},
		{"zero float64", 0.0, "float64", 0.0},
		{"non-empty string", "hello", "string", "hello"},
		{"non-zero int", 42, "int", 42},
		{"non-empty map", map[string]any{"key": "value"}, "map", map[string]any{"key": "value"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := DefaultValueIfNull(tc.input, tc.typeName)
			if tc.typeName == "map" {
				if _, ok := result.(map[string]any); !ok {
					t.Errorf("expected map[string]any, got %T", result)
				}
			} else if tc.typeName == "slice" {
				if _, ok := result.([]any); !ok {
					t.Errorf("expected []any, got %T", result)
				}
			} else if result != tc.expected {
				t.Errorf("DefaultValueIfNull(%v, %q) = %v, want %v", tc.input, tc.typeName, result, tc.expected)
			}
		})
	}
}

func TestBuildIntervalParts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		interval Interval
		expected []string
	}{
		{
			"single value",
			Interval{Years: 1},
			[]string{"1 year"},
		},
		{
			"multiple values",
			Interval{Years: 2, Months: 3, Days: 5},
			[]string{"2 years", "3 months", "5 days"},
		},
		{
			"empty interval",
			Interval{},
			[]string{},
		},
		{
			"all values",
			Interval{Years: 1, Months: 2, Weeks: 1, Days: 1, Hours: 3, Minutes: 30, Seconds: 45},
			[]string{"1 year", "2 months", "1 week", "1 day", "3 hours", "30 minutes", "45 seconds"},
		},
		{
			"plural singular mix",
			Interval{Years: 1, Months: 2},
			[]string{"1 year", "2 months"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := buildIntervalParts(tc.interval)
			if len(result) != len(tc.expected) {
				t.Errorf("got %d parts, want %d", len(result), len(tc.expected))
				return
			}
			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("part %d: got %q, want %q", i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestFormatParts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{"single part", []string{"daily"}, "daily"},
		{"two parts", []string{"1 day", "2 hours"}, "1 day and 2 hours"},
		{"three parts", []string{"1 day", "2 hours", "30 minutes"}, "1 day, 2 hours, and 30 minutes"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatParts(tc.parts)
			if result != tc.expected {
				t.Errorf("formatParts(%v) = %q, want %q", tc.parts, result, tc.expected)
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		value    int
		unit     string
		expected string
	}{
		{"singular", 1, "day", "1 day"},
		{"plural", 5, "day", "5 days"},
		{"plural month", 2, "month", "2 months"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := pluralize(tc.value, tc.unit)
			if result != tc.expected {
				t.Errorf("pluralize(%d, %q) = %q, want %q", tc.value, tc.unit, result, tc.expected)
			}
		})
	}
}

func TestDescribeInterval(t *testing.T) {
	start := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		interval      Interval
		intervalType  string
		shouldContain string
	}{
		{"delay description", Interval{Hours: 2}, "delay", "delay of 2 hours"},
		{"interval description", Interval{Days: 1}, "interval", "every 1 day"},
		{"empty delay", Interval{}, "delay", "no delay"},
		{"empty interval", Interval{}, "interval", "immediately"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			desc, _ := DescribeInterval(tc.interval, start, tc.intervalType)
			if desc != tc.shouldContain {
				t.Errorf("got description %q, want %q", desc, tc.shouldContain)
			}
		})
	}
}

func TestGetFirstAndNextRun(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	delay := Interval{Hours: 1}
	interval := Interval{Days: 1}

	first, next := GetFirstAndNextRun(start, time.Time{}, delay, interval)

	expected := time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC)
	if !first.Equal(expected) {
		t.Errorf("first run = %v, want %v", first, expected)
	}

	expectedNext := time.Date(2024, 1, 2, 11, 0, 0, 0, time.UTC)
	if !next.Equal(expectedNext) {
		t.Errorf("next run = %v, want %v", next, expectedNext)
	}
}

func TestIsZeroInterval(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		interval Interval
		expected bool
	}{
		{"all zero", Interval{}, true},
		{"one value", Interval{Days: 1}, false},
		{"multiple values", Interval{Hours: 1, Minutes: 30}, false},
		{"seconds only", Interval{Seconds: 30}, false},
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

func TestDescribeSchedule(t *testing.T) {
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	exec := Execution{
		Delay:    Interval{Hours: 1},
		Interval: Interval{Days: 1},
	}

	desc, _ := DescribeSchedule(exec, start)

	if desc == "" {
		t.Error("expected non-empty description")
	}

	if !contains(desc, "runs") {
		t.Errorf("description should mention 'runs', got: %s", desc)
	}
}

func TestCalculateNextRun(t *testing.T) {
	t.Parallel()
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		interval Interval
		expected time.Time
	}{
		{
			"add hours",
			Interval{Hours: 2},
			time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			"add days",
			Interval{Days: 1},
			time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC),
		},
		{
			"add months",
			Interval{Months: 1},
			time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			"zero interval",
			Interval{},
			start,
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

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
func TestClearLogFilesReportsWhatItCouldNotRemove(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	if err := LogMessageToFile("active", "info", "app", nil); err != nil {
		t.Fatalf("LogMessageToFile failed: %v", err)
	}

	for _, name := range []string{"old-one.log", "old-two.log", "old-three.log"} {
		if err := os.WriteFile(filepath.Join(AppLogDirPath, name), []byte("x"), 0600); err != nil {
			t.Fatalf("failed to seed %s: %v", name, err)
		}
	}

	removed, inUse, err := ClearLogFiles(AppLogDirPath)
	if err != nil {
		t.Fatalf("ClearLogFiles failed: %v", err)
	}

	if removed < 3 {
		t.Errorf("removed %d files, want at least the 3 closed ones (in use: %v)", removed, inUse)
	}

	for _, name := range []string{"old-one.log", "old-two.log", "old-three.log"} {
		if _, err := os.Stat(filepath.Join(AppLogDirPath, name)); !os.IsNotExist(err) {
			t.Errorf("%s should have been removed", name)
		}
	}
}

func TestClearLogFilesOnAnEmptyDirectory(t *testing.T) {
	setupLogTestEnv(t)

	if err := os.MkdirAll(AppLogDirPath, 0755); err != nil {
		t.Fatalf("failed to create the log dir: %v", err)
	}

	removed, inUse, err := ClearLogFiles(AppLogDirPath)
	if err != nil {
		t.Fatalf("ClearLogFiles failed: %v", err)
	}
	if removed != 0 || len(inUse) != 0 {
		t.Errorf("expected nothing to do, got removed=%d inUse=%v", removed, inUse)
	}
}

func setupPidTestEnv(t *testing.T) string {
	temp := t.TempDir()
	old := RootPath
	RootPath = temp
	PidFilePath = filepath.Join(temp, "schedulr.pid")
	t.Cleanup(func() {
		RootPath = old
		PidFilePath = filepath.Join(old, "schedulr.pid")
	})
	return temp
}

func TestPidFileExists(t *testing.T) {
	setupPidTestEnv(t)

	if PidFileExists() {
		t.Error("PidFileExists should return false for non-existent file")
	}

	CreatePidFile(1234)
	if !PidFileExists() {
		t.Error("PidFileExists should return true after creating file")
	}
}

func TestCreatePidFile(t *testing.T) {
	setupPidTestEnv(t)

	pid := 9999
	err := CreatePidFile(pid)
	if err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	if !PidFileExists() {
		t.Error("pid file should exist after creation")
	}

	data, err := os.ReadFile(PidFilePath)
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}

	// the pid is on the first line; the executable follows on the second so a
	// stale file can be told apart from a recycled pid
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")

	if strings.TrimSpace(lines[0]) != strconv.Itoa(pid) {
		t.Errorf("first line = %q, want %q", lines[0], strconv.Itoa(pid))
	}

	if len(lines) < 2 || strings.TrimSpace(lines[1]) == "" {
		t.Errorf("pid file should record the executable on its second line, got %q", string(data))
	}
}

func TestCreatePidFileAlreadyExists(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(1111)
	err := CreatePidFile(2222)
	if err == nil {
		t.Error("CreatePidFile should error when file already exists")
	}
}

func TestDeletePidFile(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(5555)
	err := DeletePidFile()
	if err != nil {
		t.Fatalf("DeletePidFile failed: %v", err)
	}

	if PidFileExists() {
		t.Error("pid file should not exist after deletion")
	}
}

func TestDeletePidFileNotExists(t *testing.T) {
	setupPidTestEnv(t)

	err := DeletePidFile()
	if err == nil {
		t.Error("DeletePidFile should error when file doesn't exist")
	}
}

func TestReadPidFile(t *testing.T) {
	setupPidTestEnv(t)

	expectedPid := 7777
	CreatePidFile(expectedPid)

	pid, err := ReadPidFile()
	if err != nil {
		t.Fatalf("ReadPidFile failed: %v", err)
	}

	if pid != expectedPid {
		t.Errorf("ReadPidFile returned %d, want %d", pid, expectedPid)
	}
}

func TestReadPidFileNotExists(t *testing.T) {
	setupPidTestEnv(t)

	_, err := ReadPidFile()
	if err == nil {
		t.Error("ReadPidFile should error when file doesn't exist")
	}
}

func TestReadPidFileInvalidFormat(t *testing.T) {
	setupPidTestEnv(t)

	os.WriteFile(PidFilePath, []byte("not-a-number"), 0644)

	_, err := ReadPidFile()
	if err == nil {
		t.Error("ReadPidFile should error with invalid format")
	}
}

func TestPidFileRoundTrip(t *testing.T) {
	setupPidTestEnv(t)

	originalPid := 12345

	err := CreatePidFile(originalPid)
	if err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	pid, err := ReadPidFile()
	if err != nil {
		t.Fatalf("ReadPidFile failed: %v", err)
	}

	if pid != originalPid {
		t.Errorf("round trip failed: got %d, want %d", pid, originalPid)
	}

	err = DeletePidFile()
	if err != nil {
		t.Fatalf("DeletePidFile failed: %v", err)
	}

	if PidFileExists() {
		t.Error("file should not exist after deletion")
	}
}

func TestMultiplePidOperations(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(1111)
	pid1, _ := ReadPidFile()

	DeletePidFile()

	CreatePidFile(2222)
	pid2, _ := ReadPidFile()

	if pid1 == pid2 {
		t.Error("PIDs should be different")
	}

	if pid1 != 1111 || pid2 != 2222 {
		t.Errorf("PIDs not as expected: %d, %d", pid1, pid2)
	}
}

func TestCreateProjectRootMarker(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	if err := createProjectRootMarker(dir); err != nil {
		t.Fatalf("createProjectRootMarker failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, projectRootMarker))
	if err != nil {
		t.Fatalf("%s should have been created: %v", projectRootMarker, err)
	}
	if info.IsDir() {
		t.Errorf("%s should be a file", projectRootMarker)
	}
	if info.Size() == 0 {
		t.Errorf("%s should explain why it exists", projectRootMarker)
	}
}

func TestCreateProjectRootMarkerLeavesAnExistingOneAlone(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, projectRootMarker)
	existing := "# hand written, do not touch\n"

	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatalf("failed to seed the marker: %v", err)
	}

	if err := createProjectRootMarker(dir); err != nil {
		t.Fatalf("createProjectRootMarker failed: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the marker back: %v", err)
	}

	if string(after) != existing {
		t.Errorf("an existing marker should be left untouched, got:\n%s", after)
	}
}

func TestCreateProjectRootMarkerIsIdempotent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	for i := 0; i < 3; i++ {
		if err := createProjectRootMarker(dir); err != nil {
			t.Fatalf("call %d failed: %v", i+1, err)
		}
	}
}

func TestAutoSetupCreatesTheProjectRootMarker(t *testing.T) {
	dir := t.TempDir()

	oldRoot := RootPath
	oldTaskLocation := TaskLocation
	RootPath = dir
	TaskLocation = filepath.Join(dir, TASKS_FOLDER)
	t.Cleanup(func() {
		RootPath = oldRoot
		TaskLocation = oldTaskLocation
	})

	if err := AutoSetup(); err != nil {
		t.Fatalf("AutoSetup failed: %v", err)
	}

	for _, want := range []string{projectRootMarker, configFileName} {
		if _, err := os.Stat(filepath.Join(dir, want)); err != nil {
			t.Errorf("AutoSetup should have created %s: %v", want, err)
		}
	}
}

func TestWaitForProcessToExitSeesAProcessGo(t *testing.T) {
	t.Parallel()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "exit", "0")
	} else {
		cmd = exec.Command("sh", "-c", "exit 0")
	}
	HideTaskWindow(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start the helper: %v", err)
	}

	pid := cmd.Process.Pid

	go func() { _ = cmd.Wait() }()

	if !WaitForProcessToExit(pid, 10*time.Second) {
		t.Error("a process that exits immediately should be seen to exit")
	}
}

func TestWaitForProcessToExitTimesOutOnALiveProcess(t *testing.T) {
	t.Parallel()

	start := time.Now()

	if WaitForProcessToExit(os.Getpid(), 300*time.Millisecond) {
		t.Error("a live process should not be reported as exited")
	}

	if elapsed := time.Since(start); elapsed < 250*time.Millisecond {
		t.Errorf("it returned after %s, so it did not wait out the timeout", elapsed)
	}
}

func TestWaitForProcessToExitReturnsImmediatelyForADeadPid(t *testing.T) {
	t.Parallel()

	start := time.Now()

	if !WaitForProcessToExit(999999999, 5*time.Second) {
		t.Error("an unused pid should report as exited")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("it took %s for an already dead pid", elapsed)
	}
}

func TestProcessAliveForCurrentProcess(t *testing.T) {
	t.Parallel()

	if !ProcessAlive(os.Getpid()) {
		t.Error("the running test process should report as alive")
	}
}

func TestProcessAliveForDeadProcess(t *testing.T) {
	t.Parallel()

	if ProcessAlive(999999999) {
		t.Error("an unused pid should not report as alive")
	}
}

func TestProcessAliveRejectsNonsensePids(t *testing.T) {
	t.Parallel()

	for _, pid := range []int{0, -1, -99999} {
		if ProcessAlive(pid) {
			t.Errorf("pid %d should not report as alive", pid)
		}
	}
}

func TestProcessExecutableForCurrentProcess(t *testing.T) {
	t.Parallel()

	path, err := ProcessExecutable(os.Getpid())
	if err != nil {
		t.Fatalf("ProcessExecutable failed for the current process: %v", err)
	}

	if path == "" {
		t.Skip("this platform does not expose the executable behind a pid")
	}

	own, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}

	if !samePath(path, own) {
		t.Errorf("ProcessExecutable = %q, want %q", path, own)
	}
}

func TestDaemonRunningIsFalseForAStalePidFile(t *testing.T) {
	setupPidTestEnv(t)

	if err := os.WriteFile(PidFilePath, []byte("999999999\n"), 0644); err != nil {
		t.Fatalf("failed to seed the pid file: %v", err)
	}

	pid, running := DaemonRunning()

	if running {
		t.Error("a pid file pointing at a dead process should not report as running")
	}
	if pid != 999999999 {
		t.Errorf("the recorded pid should still be reported, got %d", pid)
	}
}

func TestDaemonRunningIsTrueForALiveProcess(t *testing.T) {
	setupPidTestEnv(t)

	if err := CreatePidFile(os.Getpid()); err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	pid, running := DaemonRunning()

	if !running {
		t.Error("a pid file pointing at this live process should report as running")
	}
	if pid != os.Getpid() {
		t.Errorf("pid = %d, want %d", pid, os.Getpid())
	}
}

func TestDaemonRunningRejectsARecycledPid(t *testing.T) {
	setupPidTestEnv(t)

	if _, err := ProcessExecutable(os.Getpid()); err != nil {
		t.Fatalf("ProcessExecutable failed: %v", err)
	}
	if path, _ := ProcessExecutable(os.Getpid()); path == "" {
		t.Skip("this platform cannot tell which binary a pid belongs to")
	}

	contents := strconv.Itoa(os.Getpid()) + "\n" + "/definitely/not/schedulr\n"
	if err := os.WriteFile(PidFilePath, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to seed the pid file: %v", err)
	}

	if _, running := DaemonRunning(); running {
		t.Error("a live pid belonging to a different binary should not report as running")
	}
}

func TestDaemonRunningIsFalseWithNoPidFile(t *testing.T) {
	setupPidTestEnv(t)

	if pid, running := DaemonRunning(); running || pid != 0 {
		t.Errorf("expected (0, false) with no pid file, got (%d, %v)", pid, running)
	}
}

func TestReadPidFileToleratesWhitespaceAndLineEndings(t *testing.T) {
	tests := map[string]string{
		"trailing newline":   "4242\n",
		"trailing spaces":    "4242   ",
		"windows line ends":  "4242\r\n",
		"leading whitespace": "  4242\n",
		"with executable":    "4242\n/usr/local/bin/schedulr\n",
		"legacy single line": "4242",
	}

	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			setupPidTestEnv(t)

			if err := os.WriteFile(PidFilePath, []byte(contents), 0644); err != nil {
				t.Fatalf("failed to write the pid file: %v", err)
			}

			pid, err := ReadPidFile()
			if err != nil {
				t.Fatalf("ReadPidFile(%q) failed: %v", contents, err)
			}
			if pid != 4242 {
				t.Errorf("pid = %d, want 4242", pid)
			}
		})
	}
}

func TestReadPidFileErrorMentionsTheFile(t *testing.T) {
	setupPidTestEnv(t)

	if err := os.WriteFile(PidFilePath, []byte("not-a-pid\n"), 0644); err != nil {
		t.Fatalf("failed to write the pid file: %v", err)
	}

	_, err := ReadPidFile()
	if err == nil {
		t.Fatal("expected an error for an unparsable pid")
	}

	if strings.Contains(err.Error(), "%!e") {
		t.Errorf("error should not contain a botched format verb: %v", err)
	}
	if !strings.Contains(err.Error(), pidFile) {
		t.Errorf("error should name the pid file, got: %v", err)
	}
}

func TestClearStalePidFileRemovesAStaleFile(t *testing.T) {
	setupPidTestEnv(t)

	if err := os.WriteFile(PidFilePath, []byte("999999999\n"), 0644); err != nil {
		t.Fatalf("failed to seed the pid file: %v", err)
	}

	pid, removed := ClearStalePidFile()

	if !removed {
		t.Error("a stale pid file should have been removed")
	}
	if pid != 999999999 {
		t.Errorf("pid = %d, want 999999999", pid)
	}
	if PidFileExists() {
		t.Error("the pid file should be gone")
	}
}

func TestClearStalePidFileKeepsALiveFile(t *testing.T) {
	setupPidTestEnv(t)

	if err := CreatePidFile(os.Getpid()); err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	if _, removed := ClearStalePidFile(); removed {
		t.Error("a pid file for a live process must not be removed")
	}
	if !PidFileExists() {
		t.Error("the pid file should still be there")
	}
}

func TestClearStalePidFileRemovesACorruptFile(t *testing.T) {
	setupPidTestEnv(t)

	if err := os.WriteFile(PidFilePath, []byte("garbage\n"), 0644); err != nil {
		t.Fatalf("failed to seed the pid file: %v", err)
	}

	if _, removed := ClearStalePidFile(); !removed {
		t.Error("a corrupt pid file should be treated as stale and removed")
	}
	if PidFileExists() {
		t.Error("the pid file should be gone")
	}
}

func TestClearStalePidFileWithNoFile(t *testing.T) {
	setupPidTestEnv(t)

	if _, removed := ClearStalePidFile(); removed {
		t.Error("nothing to remove should report false")
	}
}

func TestCreatePidFileReportsWriteFailure(t *testing.T) {
	setupPidTestEnv(t)

	blocker := PidFilePath + ".blocker"
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create the blocker: %v", err)
	}

	old := PidFilePath
	PidFilePath = blocker + string(os.PathSeparator) + "schedulr.pid"
	t.Cleanup(func() { PidFilePath = old })

	if err := CreatePidFile(1234); err == nil {
		t.Error("CreatePidFile should report a write failure instead of returning nil")
	}
}
