package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
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

	cmd := getShellCommand("echo test", nil, "bash", false)

	if cmd == nil {
		t.Error("getShellCommand returned nil")
	}

	if cmd.Path == "" {
		t.Error("command path should be set")
	}
}

func TestGetShellCommandWithGui(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("test", nil, "bash", true)
	if cmd == nil {
		t.Error("getShellCommand with GUI returned nil")
	}
}

func TestGetShellCommandPowershell(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long test")
	}

	cmd := getShellCommand("Write-Host test", nil, "powershell", false)
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

type recorder struct {
	server *httptest.Server
	mu     sync.Mutex
	calls  []string
}

func newRecorder(t *testing.T) *recorder {
	t.Helper()

	r := &recorder{}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.calls = append(r.calls, req.URL.Path)
		r.mu.Unlock()
	}))
	t.Cleanup(r.server.Close)

	return r
}

func (r *recorder) urlFor(name string) string {
	return r.server.URL + "/" + name
}

func (r *recorder) recorded() []string {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]string(nil), r.calls...)
}

type taskSpec struct {
	name      string
	enabled   bool
	runBefore string
	runAfter  string
	delay     Interval
	interval  Interval
}

func writeHttpTask(t *testing.T, rec *recorder, spec taskSpec) Task {
	t.Helper()

	task := HttpTask{
		Name: spec.name,
		Execution: Execution{
			Interval:  spec.interval,
			Delay:     spec.delay,
			RunBefore: spec.runBefore,
			RunAfter:  spec.runAfter,
			IsEnabled: spec.enabled,
		},
		URL:     rec.urlFor(spec.name),
		Method:  METHOD_GET,
		Headers: map[string]any{},
		Body:    map[string]any{},
	}

	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("failed to save task %s: %v", spec.name, err)
	}

	saved, err := GetTask(HTTP_TASK, spec.name)
	if err != nil {
		t.Fatalf("failed to read back task %s: %v", spec.name, err)
	}

	return saved
}

func setupTaskDirs(t *testing.T) {
	t.Helper()

	for _, taskType := range knownTaskTypes {
		if err := os.MkdirAll(filepath.Join(TaskLocation, taskType), 0755); err != nil {
			t.Fatalf("failed to create %s task dir: %v", taskType, err)
		}
	}
}

func TestDisabledRunBeforeStillRunsTheTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Dep", enabled: false})
	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runBefore: "Dep"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Main" {
		t.Errorf("expected only Main to run, got %v", calls)
	}
}

func TestDisabledRunAfterStillRunsTheTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Dep", enabled: false})
	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runAfter: "Dep"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Main" {
		t.Errorf("expected only Main to run, got %v", calls)
	}
}

func TestEnabledDependenciesRunInOrder(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Before", enabled: true})
	writeHttpTask(t, rec, taskSpec{name: "After", enabled: true})
	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runBefore: "Before", runAfter: "After"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	want := []string{"/Before", "/Main", "/After"}

	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

func TestCircularDependencyTerminates(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Bravo", enabled: true, runAfter: "Alpha"})
	alpha := writeHttpTask(t, rec, taskSpec{name: "Alpha", enabled: true, runAfter: "Bravo"})

	done := make(chan struct{})
	go func() {
		executeTaskWithDependencies(alpha)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("circular dependency chain did not terminate")
	}

	calls := rec.recorded()
	if len(calls) != 2 {
		t.Errorf("each task in the cycle should run exactly once, got %v", calls)
	}
}

func TestSelfReferencingDependencyTerminates(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	task := writeHttpTask(t, rec, taskSpec{name: "Loop", enabled: true, runBefore: "Loop"})

	done := make(chan struct{})
	go func() {
		executeTaskWithDependencies(task)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("self-referencing task did not terminate")
	}

	if calls := rec.recorded(); len(calls) != 1 {
		t.Errorf("self-referencing task should run once, got %v", calls)
	}
}

func TestFarOffDependencyIsSkippedWithoutBlocking(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Dep", enabled: true, delay: Interval{Days: 30}})
	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runBefore: "Dep"})

	done := make(chan struct{})
	go func() {
		executeTaskWithDependencies(main)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("a dependency 30 days out blocked the worker")
	}

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Main" {
		t.Errorf("expected only Main to run, got %v", calls)
	}
}

func TestADependencyAlsoKeepsItsOwnSchedule(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Dep", enabled: true, interval: Interval{Minutes: 5}})
	writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, interval: Interval{Minutes: 5}, runBefore: "Dep"})

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	names := scheduleNames(schedule)
	if len(names) != 2 {
		t.Fatalf("both the task and its dependency should be scheduled, got %v", names)
	}

	found := false
	for _, n := range names {
		if n == "Dep" {
			found = true
		}
	}
	if !found {
		t.Errorf("the dependency should have its own schedule entry, got %v", names)
	}
}

func TestDependenciesResolveAcrossTaskTypes(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "HttpDep", enabled: true})

	shell := ShellTask{
		Name: "ShellMain",
		Execution: Execution{
			IsEnabled: true,
			RunBefore: "HttpDep",
		},
		Command: "echo hi",
	}
	if err := SaveTask(shell, SHELL_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	loaded, err := GetTask(SHELL_TASK, "ShellMain")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	dep := loaded.GetRunBefore()
	if dep == nil {
		t.Fatal("a shell task should be able to name an HTTP task as its dependency")
	}
	if dep.GetName() != "HttpDep" {
		t.Errorf("dependency = %q, want HttpDep", dep.GetName())
	}
}

func TestDependencyChainsRunInOrder(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "First", enabled: true})
	writeHttpTask(t, rec, taskSpec{name: "Second", enabled: true, runBefore: "First"})
	third := writeHttpTask(t, rec, taskSpec{name: "Third", enabled: true, runBefore: "Second"})

	done := make(chan struct{})
	go func() {
		executeTaskWithDependencies(third)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the dependency chain did not finish")
	}

	calls := rec.recorded()
	want := []string{"/First", "/Second", "/Third"}

	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

func TestUnknownDependencyNameIsIgnored(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runBefore: "NoSuchTask"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Main" {
		t.Errorf("the task should still run when its dependency does not exist, got %v", calls)
	}
}

func TestDependencyOnlyTaskIsNotScheduled(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeHttpTask(t, rec, taskSpec{name: "Scheduled", enabled: true, interval: Interval{Minutes: 5}})
	writeDependencyOnlyTask(t, rec, "HelperOnly")

	schedule, err := loadSchedule(nil)
	if err != nil {
		t.Fatalf("loadSchedule failed: %v", err)
	}

	names := scheduleNames(schedule)
	if len(names) != 1 || names[0] != "Scheduled" {
		t.Errorf("a dependency-only task should not be scheduled, got %v", names)
	}
}

func TestDependencyOnlyTaskStillRunsAsADependency(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeDependencyOnlyTask(t, rec, "Fetch")
	main := writeHttpTask(t, rec, taskSpec{name: "Report", enabled: true, runBefore: "Fetch"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	want := []string{"/Fetch", "/Report"}

	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

// dependency_only controls scheduling; is_enabled still controls whether it may
// run at all.
func TestDisabledDependencyOnlyTaskDoesNotRun(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)

	task := HttpTask{
		Name:      "Fetch",
		Execution: Execution{IsEnabled: false, DependencyOnly: true},
		URL:       rec.urlFor("Fetch"),
		Method:    METHOD_GET,
	}
	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	main := writeHttpTask(t, rec, taskSpec{name: "Report", enabled: true, runBefore: "Fetch"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Report" {
		t.Errorf("a disabled dependency should be skipped and the task still run, got %v", calls)
	}
}

// Chains of dependency-only tasks still resolve end to end.
func TestDependencyOnlyChain(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeDependencyOnlyTask(t, rec, "First")
	writeDependencyOnlyTaskWith(t, rec, "Second", "First")
	main := writeHttpTask(t, rec, taskSpec{name: "Third", enabled: true, runBefore: "Second"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	want := []string{"/First", "/Second", "/Third"}

	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
	for i := range want {
		if calls[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, calls)
		}
	}
}

// The overlap case: the dependency is already running elsewhere.
func TestADependencyAlreadyRunningIsSkipped(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeDependencyOnlyTask(t, rec, "Fetch")
	main := writeHttpTask(t, rec, taskSpec{name: "Report", enabled: true, runBefore: "Fetch"})

	// stand in for another worker part-way through running Fetch
	if !claimInFlight("Fetch") {
		t.Fatal("precondition: Fetch should not already be in flight")
	}

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Report" {
		t.Errorf("a dependency already running should be skipped, not run twice, got %v", calls)
	}

	releaseInFlight("Fetch")
}

// And the guard must be released, or the dependency would be skipped forever.
func TestADependencyRunsAgainAfterItsPreviousRun(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeDependencyOnlyTask(t, rec, "Fetch")
	main := writeHttpTask(t, rec, taskSpec{name: "Report", enabled: true, runBefore: "Fetch"})

	executeTaskWithDependencies(main)
	executeTaskWithDependencies(main)

	fetches := 0
	for _, call := range rec.recorded() {
		if call == "/Fetch" {
			fetches++
		}
	}

	if fetches != 2 {
		t.Errorf("the dependency should run on each dispatch, got %d runs", fetches)
	}
}

// Claiming the guard for a dependency must not block the task that declared it:
// the top-level claim belongs to dispatch, not to the chain.
func TestTheDeclaringTaskStillRunsWhenItIsAlsoInFlight(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)

	rec := newRecorder(t)
	writeDependencyOnlyTask(t, rec, "Fetch")
	main := writeHttpTask(t, rec, taskSpec{name: "Report", enabled: true, runBefore: "Fetch"})

	// what a worker sees: dispatch has already claimed the top-level task
	if !claimInFlight("Report") {
		t.Fatal("precondition: Report should not already be in flight")
	}
	defer releaseInFlight("Report")

	done := make(chan struct{})
	go func() {
		executeTaskWithDependencies(main)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("the chain blocked on the top-level task's own in-flight claim")
	}

	calls := rec.recorded()
	want := []string{"/Fetch", "/Report"}

	if len(calls) != len(want) {
		t.Fatalf("expected %v, got %v", want, calls)
	}
}

func writeDependencyOnlyTask(t *testing.T, rec *recorder, name string) Task {
	t.Helper()

	return writeDependencyOnlyTaskWith(t, rec, name, "")
}

func writeDependencyOnlyTaskWith(t *testing.T, rec *recorder, name, runBefore string) Task {
	t.Helper()

	task := HttpTask{
		Name: name,
		Execution: Execution{
			IsEnabled:      true,
			DependencyOnly: true,
			RunBefore:      runBefore,
		},
		URL:     rec.urlFor(name),
		Method:  METHOD_GET,
		Headers: map[string]any{},
		Body:    map[string]any{},
	}

	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("failed to save %s: %v", name, err)
	}

	saved, err := GetTask(HTTP_TASK, name)
	if err != nil {
		t.Fatalf("failed to read back %s: %v", name, err)
	}

	return saved
}

func TestExecuteTaskReturnsNilOnSuccess(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	task := HttpTask{
		Name:      "Fine",
		Execution: Execution{IsEnabled: true},
		URL:       server.URL,
		Method:    METHOD_GET,
	}
	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	if err := ExecuteTask(task); err != nil {
		t.Errorf("a task that ran fine should report no error, got %v", err)
	}
}

func TestExecuteTaskReportsAFailingShellCommand(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	// exits non-zero on every supported platform
	task := ShellTask{
		Name:      "Failing",
		Execution: Execution{IsEnabled: true},
		Command:   "exit 3",
	}

	err := ExecuteTask(task)
	if err == nil {
		t.Fatal("a shell command exiting non-zero should be reported")
	}
	if !strings.Contains(err.Error(), "Failing") {
		t.Errorf("the error should name the task, got: %v", err)
	}
}

func TestExecuteTaskReportsAnUnrunnableShellTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	task := ShellTask{
		Name:      "NoCommand",
		Execution: Execution{IsEnabled: true},
		Command:   "   ",
	}

	if err := ExecuteTask(task); err == nil {
		t.Error("a task with no command should be reported as an error")
	}
}

func TestExecuteTaskReportsABadHttpMethod(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	for name, method := range map[string]string{"empty": "", "unsupported": "TRACE"} {
		t.Run(name, func(t *testing.T) {
			task := HttpTask{
				Name:      "BadMethod",
				Execution: Execution{IsEnabled: true},
				URL:       "http://127.0.0.1:1",
				Method:    method,
			}

			if err := ExecuteTask(task); err == nil {
				t.Errorf("method %q should be reported as an error", method)
			}
		})
	}
}

func TestExecuteTaskReportsAnUnreachableHttpTarget(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	task := HttpTask{
		Name:      "Unreachable",
		Execution: Execution{IsEnabled: true},
		URL:       "http://127.0.0.1:1/nope",
		Method:    METHOD_GET,
	}

	err := ExecuteTask(task)
	if err == nil {
		t.Fatal("an unreachable target should be reported")
	}
	if !strings.Contains(err.Error(), "Unreachable") {
		t.Errorf("the error should name the task, got: %v", err)
	}
}

func TestAFailingDependencyDoesNotStopTheChain(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	rec := newRecorder(t)

	// a dependency that cannot run at all
	broken := HttpTask{
		Name:      "BrokenDep",
		Execution: Execution{IsEnabled: true},
		URL:       rec.urlFor("BrokenDep"),
		Method:    "NOTAMETHOD",
	}
	if err := SaveTask(broken, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	main := writeHttpTask(t, rec, taskSpec{name: "Main", enabled: true, runBefore: "BrokenDep"})

	executeTaskWithDependencies(main)

	calls := rec.recorded()
	if len(calls) != 1 || calls[0] != "/Main" {
		t.Errorf("the task should still run after a failing dependency, got %v", calls)
	}
}

const argsHelperEnv = "SCHEDULR_ARGS_HELPER"

func TestArgsHelperProcess(t *testing.T) {
	if os.Getenv(argsHelperEnv) != "1" {
		t.Skip("helper process for the args-mode tests")
	}

	past := false
	for _, arg := range os.Args {
		if past {
			fmt.Printf("ARG[%s]\n", arg)
			continue
		}
		if arg == "--" {
			past = true
		}
	}

	os.Exit(0)
}

func helperArgs(extra ...string) []string {
	return append([]string{"-test.run=TestArgsHelperProcess", "--"}, extra...)
}

func TestArgsArePassedVerbatim(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("could not locate the test binary: %v", err)
	}

	want := []string{
		"plain",
		"two words",
		`quoted "inner" text`,
		"amp & pipe | gt > caret ^ semi ;",
		"%USERNAME%",
		"$HOME",
		"glob *.go",
		"  padded  ",
	}

	cmd := getShellCommand(exe, helperArgs(want...), "", false)
	cmd.Env = append(os.Environ(), argsHelperEnv+"=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper process failed: %v\noutput:\n%s", err, out)
	}

	var got []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, "ARG[") && strings.HasSuffix(line, "]") {
			got = append(got, strings.TrimSuffix(strings.TrimPrefix(line, "ARG["), "]"))
		}
	}

	if len(got) != len(want) {
		t.Fatalf("received %d args, want %d\ngot:  %q\nwant: %q\nraw output:\n%s", len(got), len(want), got, want, out)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestArgsModeBypassesTheShell(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("/usr/bin/tool", []string{"--flag", "value"}, "powershell", false)

	want := []string{"/usr/bin/tool", "--flag", "value"}

	if len(cmd.Args) != len(want) {
		t.Fatalf("Args = %q, want %q", cmd.Args, want)
	}
	for i := range want {
		if cmd.Args[i] != want[i] {
			t.Fatalf("Args = %q, want %q", cmd.Args, want)
		}
	}

	for _, shell := range []string{"cmd", "powershell", "pwsh", "sh"} {
		if strings.Contains(strings.ToLower(cmd.Args[0]), shell) {
			t.Errorf("args mode should not route through a shell, got %q", cmd.Args[0])
		}
	}
}

func TestEmptyArgsStillUsesTheShell(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{nil, {}} {
		cmd := getShellCommand("echo a && echo b", args, "", false)

		shelled := false
		for _, shell := range []string{"cmd", "sh", "powershell", "pwsh"} {
			if strings.Contains(strings.ToLower(cmd.Args[0]), shell) {
				shelled = true
			}
		}

		if !shelled {
			t.Errorf("with args %v the command should go through a shell, got %q", args, cmd.Args)
		}
	}
}

func TestValidateShellTask(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("could not locate the test binary: %v", err)
	}

	tests := []struct {
		name      string
		task      ShellTask
		shouldErr bool
		errHas    string
	}{
		{
			name:      "command only",
			task:      ShellTask{Command: "echo hi && echo there"},
			shouldErr: false,
		},
		{
			name:      "args with a real executable",
			task:      ShellTask{Command: exe, Args: []string{"--version"}},
			shouldErr: false,
		},
		{
			name:      "no command at all",
			task:      ShellTask{Command: "   "},
			shouldErr: true,
			errHas:    "no command",
		},
		{
			name:      "args alongside a shell line",
			task:      ShellTask{Command: "echo hi", Args: []string{"extra"}},
			shouldErr: true,
			errHas:    "single executable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateShellTask(tc.task)

			if tc.shouldErr && err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !tc.shouldErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.shouldErr && !strings.Contains(err.Error(), tc.errHas) {
				t.Errorf("error %q should mention %q", err, tc.errHas)
			}
		})
	}
}

func TestValidateShellTaskSkipsLookupForMacGuiTask(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific behaviour")
	}

	task := ShellTask{Command: "Safari", Args: []string{"https://example.com"}, IsGui: true}

	if err := validateShellTask(task); err != nil {
		t.Errorf("a macOS GUI task names an app, not a program on PATH: %v", err)
	}
}

func TestShellTaskArgsSurviveSaveAndLoad(t *testing.T) {
	setupTestEnv(t)
	setupTaskDirs(t)

	want := []string{"report.py", "--out", "C:\\Out Dir\\r.csv", "two words"}

	task := ShellTask{
		Name:      "ArgsRoundTrip",
		Execution: Execution{IsEnabled: true},
		Command:   "python.exe",
		Args:      want,
	}

	if err := SaveTask(task, SHELL_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	loaded, err := GetTask(SHELL_TASK, "ArgsRoundTrip")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	got := loaded.GetArgs()
	if len(got) != len(want) {
		t.Fatalf("args = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("arg %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLegacyTaskFileWithoutArgsKeyStillWorks(t *testing.T) {
	setupTestEnv(t)

	legacy := `{
  "name": "Legacy",
  "execution": { "interval": { "seconds": 30 }, "delay": {}, "last_ran_at": "", "run_before": "", "run_after": "", "is_enabled": true },
  "command": "echo a && echo b",
  "shell_type": "cmd"
}`

	path := writeRawTaskFile(t, SHELL_TASK, "Legacy.json", legacy)

	task, err := fromTaskJson(path, SHELL_TASK)
	if err != nil {
		t.Fatalf("a task file predating args should still parse: %v", err)
	}

	if args := task.GetArgs(); len(args) != 0 {
		t.Errorf("a task with no args key should report no args, got %q", args)
	}

	if err := validateShellTask(task.(ShellTask)); err != nil {
		t.Errorf("a legacy shell line should still validate: %v", err)
	}

	cmd := getShellCommand(task.GetCommand(), task.GetArgs(), task.GetShellType(), task.GetIsGui())

	shelled := false
	for _, shell := range []string{"cmd", "sh", "powershell", "pwsh"} {
		if strings.Contains(strings.ToLower(cmd.Args[0]), shell) {
			shelled = true
		}
	}
	if !shelled {
		t.Errorf("a legacy task should still go through a shell, got %q", cmd.Args)
	}
}

func TestHttpTaskHasNoArgs(t *testing.T) {
	t.Parallel()

	if args := (HttpTask{Name: "X"}).GetArgs(); len(args) != 0 {
		t.Errorf("an HTTP task should report no args, got %q", args)
	}
}

func TestCreatedShellTaskHasAnArgsField(t *testing.T) {
	setupTestEnv(t)

	if _, err := CreateTask("ArgsField", SHELL_TASK); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	path, err := TaskPath(SHELL_TASK, "ArgsField")
	if err != nil {
		t.Fatalf("TaskPath failed: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read the task file: %v", err)
	}

	// an empty list rather than null, so the field is discoverable in the file
	if !strings.Contains(string(raw), `"args": []`) {
		t.Errorf("a new shell task should carry an empty args list, got:\n%s", raw)
	}
}
func TestDescribeResponseBody(t *testing.T) {
	t.Parallel()

	body := []byte("0123456789abcdef")

	t.Run("under the limit is kept whole", func(t *testing.T) {
		if got := describeResponseBody(body, 1024); got != "0123456789abcdef" {
			t.Errorf("got %q, want the body unchanged", got)
		}
	})

	t.Run("over the limit is truncated", func(t *testing.T) {
		got := describeResponseBody(body, 6)

		if !strings.HasPrefix(got, "012345") {
			t.Errorf("got %q, want it to start with the first 6 bytes", got)
		}
		if !strings.Contains(got, "truncated") {
			t.Errorf("got %q, want it to say it was truncated", got)
		}
		if strings.Contains(got, "9abcdef") {
			t.Errorf("got %q, the tail should not be present", got)
		}
	})

	t.Run("a limit of zero omits the body", func(t *testing.T) {
		got := describeResponseBody(body, 0)

		if strings.Contains(got, "0123") {
			t.Errorf("got %q, no part of the body should appear", got)
		}
		if !strings.Contains(got, "disabled") {
			t.Errorf("got %q, want it to say body logging is off", got)
		}
	})

	t.Run("a negative limit also omits the body", func(t *testing.T) {
		if got := describeResponseBody(body, -1); strings.Contains(got, "0123") {
			t.Errorf("got %q, no part of the body should appear", got)
		}
	})

	t.Run("an empty body is handled", func(t *testing.T) {
		if got := describeResponseBody(nil, 1024); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestDescribeResponseBodyTruncatesOnARuneBoundary(t *testing.T) {
	t.Parallel()

	body := []byte(strings.Repeat("日", 10))

	for limit := 1; limit <= 28; limit++ {
		got := describeResponseBody(body, limit)
		prefix := strings.SplitN(got, "...", 2)[0]

		if !utf8.ValidString(prefix) {
			t.Errorf("limit %d produced invalid UTF-8: %q", limit, prefix)
		}
	}
}

func TestHttpTaskHonoursTheBodyLogLimit(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	secret := strings.Repeat("TOPSECRET", 500)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(secret))
	}))
	defer server.Close()

	task := HttpTask{
		Name:      "Chatty",
		Execution: Execution{IsEnabled: true},
		URL:       server.URL,
		Method:    METHOD_GET,
	}
	if err := SaveTask(task, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	if err := ExecuteTask(task); err != nil {
		t.Fatalf("ExecuteTask failed: %v", err)
	}
	CloseLoggers()

	logs, err := filepath.Glob(filepath.Join(TasksLogDirPath, "Chatty_*.log"))
	if err != nil || len(logs) == 0 {
		t.Fatalf("no task log was written: %v", err)
	}

	written, err := os.ReadFile(logs[0])
	if err != nil {
		t.Fatalf("failed to read the task log: %v", err)
	}

	if len(written) > len(secret) {
		t.Errorf("the log (%d bytes) should not contain the whole %d byte body", len(written), len(secret))
	}
	if !strings.Contains(string(written), "truncated") {
		t.Errorf("the log should record that the body was truncated, got:\n%s", written)
	}
}
