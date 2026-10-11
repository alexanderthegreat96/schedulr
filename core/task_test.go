package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConvertToPascalCase(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"my-task":       "MyTask",
		"my_task":       "MyTask",
		"my--task":      "MyTask",
		"alreadyPascal": "Alreadypascal",
		"test":          "Test",
		"":              "",
	}

	for input, expected := range cases {
		result := convertToPascalCase(input)
		if result != expected {
			t.Errorf("convertToPascalCase(%q) = %q, want %q", input, result, expected)
		}
	}
}

func TestNormalizeTaskParams(t *testing.T) {
	t.Parallel()
	tests := []struct {
		taskType string
		taskName string
		expType  string
		expName  string
	}{
		{"HTTP", "my-task", "http", "MyTask"},
		{"shell", "test_name", "shell", "TestName"},
		{"SHELL", "PascalCase", "shell", "Pascalcase"},
	}

	for _, tc := range tests {
		tp, tn := NormalizeTaskParams(tc.taskType, tc.taskName)
		if tp != tc.expType || tn != tc.expName {
			t.Errorf("NormalizeTaskParams(%q, %q) = (%q, %q), want (%q, %q)",
				tc.taskType, tc.taskName, tp, tn, tc.expType, tc.expName)
		}
	}
}

func TestEnforceMethods(t *testing.T) {
	t.Parallel()
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "get", "post", "put"}
	for _, method := range validMethods {
		if !enforceMethods(method) {
			t.Errorf("enforceMethods(%q) = false, want true", method)
		}
	}

	invalidMethods := []string{"NOTAMETHOD", "INVALID"}
	for _, method := range invalidMethods {
		if enforceMethods(method) {
			t.Errorf("enforceMethods(%q) = true, want false", method)
		}
	}
}

func setupTestEnv(t *testing.T) string {
	temp := t.TempDir()
	old := TaskLocation
	TaskLocation = filepath.Join(temp, "tasks")
	t.Cleanup(func() {
		TaskLocation = old
	})
	return temp
}

func TestCreateTask(t *testing.T) {
	setupTestEnv(t)

	tests := []struct {
		name      string
		taskName  string
		taskType  string
		shouldErr bool
	}{
		{"shell task", "MyShell", SHELL_TASK, false},
		{"http task", "MyHttp", HTTP_TASK, false},
		{"invalid type", "BadTask", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := CreateTask(tt.taskName, tt.taskType)
			if (err != nil) != tt.shouldErr {
				t.Errorf("CreateTask(%q, %q) error = %v, shouldErr %v", tt.taskName, tt.taskType, err, tt.shouldErr)
			}
			if !tt.shouldErr && msg == "" {
				t.Error("expected non-empty success message")
			}
		})
	}
}

func TestTaskExists(t *testing.T) {
	setupTestEnv(t)

	CreateTask("TestTask", SHELL_TASK)

	if !TaskExists(SHELL_TASK, "TestTask") {
		t.Error("TaskExists should return true for created task")
	}

	if TaskExists(SHELL_TASK, "NonExistent") {
		t.Error("TaskExists should return false for non-existent task")
	}

	if TaskExists("invalid", "TestTask") {
		t.Error("TaskExists should return false for invalid directory")
	}
}

func TestGetTask(t *testing.T) {
	setupTestEnv(t)

	CreateTask("ShellTest", SHELL_TASK)
	task, err := GetTask(SHELL_TASK, "ShellTest")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if task.GetName() != "ShellTest" {
		t.Errorf("task name = %q, want ShellTest", task.GetName())
	}

	_, err = GetTask(SHELL_TASK, "NonExistent")
	if err == nil {
		t.Error("GetTask should error for non-existent task")
	}
}

func TestDeleteTask(t *testing.T) {
	setupTestEnv(t)

	CreateTask("DeleteMe", SHELL_TASK)
	if err := DeleteTask(SHELL_TASK, "DeleteMe"); err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	if TaskExists(SHELL_TASK, "DeleteMe") {
		t.Error("task should be deleted")
	}

	if err := DeleteTask(SHELL_TASK, "NonExistent"); err == nil {
		t.Error("DeleteTask should error for non-existent task")
	}
}

func TestGetTasks(t *testing.T) {
	setupTestEnv(t)

	CreateTask("Task1", SHELL_TASK)
	CreateTask("Task2", SHELL_TASK)

	tasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	if len(tasks) < 2 {
		t.Errorf("GetTasks returned %d tasks, expected at least 2", len(tasks))
	}

	_, err = GetTasks("invalid")
	if err == nil {
		t.Error("GetTasks should error for invalid task type")
	}
}

func TestUpdateTaskStatus(t *testing.T) {
	setupTestEnv(t)

	CreateTask("StatusTest", SHELL_TASK)

	if err := UpdateTaskStatus(SHELL_TASK, "StatusTest", true); err != nil {
		t.Fatalf("UpdateTaskStatus enable failed: %v", err)
	}

	task, _ := GetTask(SHELL_TASK, "StatusTest")
	if !task.GetExecution().IsEnabled {
		t.Error("task should be enabled")
	}

	if err := UpdateTaskStatus(SHELL_TASK, "StatusTest", true); err == nil {
		t.Error("UpdateTaskStatus should error when enabling already enabled task")
	}

	if err := UpdateTaskStatus(SHELL_TASK, "StatusTest", false); err != nil {
		t.Fatalf("UpdateTaskStatus disable failed: %v", err)
	}

	task, _ = GetTask(SHELL_TASK, "StatusTest")
	if task.GetExecution().IsEnabled {
		t.Error("task should be disabled")
	}
}

func TestUpdateRanAt(t *testing.T) {
	setupTestEnv(t)

	CreateTask("RanAtTest", SHELL_TASK)

	if err := UpdateRanAt(SHELL_TASK, "RanAtTest"); err != nil {
		t.Fatalf("UpdateRanAt failed: %v", err)
	}

	task, _ := GetTask(SHELL_TASK, "RanAtTest")
	if task.GetExecution().GetLastRanAtTime() == nil {
		t.Error("LastRanAt should be set")
	}

	if _, err := time.Parse(time.RFC3339, task.GetExecution().LastRanAt); err != nil {
		t.Errorf("LastRanAt has invalid format: %v", err)
	}
}

func TestFetchTaskByName(t *testing.T) {
	setupTestEnv(t)

	CreateTask("FetchMe", SHELL_TASK)
	CreateTask("AlsoFetch", HTTP_TASK)

	shell, err := FetchTaskByName("FetchMe")
	if err != nil {
		t.Fatalf("FetchTaskByName failed: %v", err)
	}
	if shell.GetName() != "FetchMe" {
		t.Errorf("fetched task name = %q, want FetchMe", shell.GetName())
	}

	http, err := FetchTaskByName("AlsoFetch")
	if err != nil {
		t.Fatalf("FetchTaskByName for http task failed: %v", err)
	}
	if http.GetName() != "AlsoFetch" {
		t.Errorf("fetched http task name = %q, want AlsoFetch", http.GetName())
	}

	_, err = FetchTaskByName("DoesNotExist")
	if err == nil {
		t.Error("FetchTaskByName should error for non-existent task")
	}
}

func TestShellTaskGetters(t *testing.T) {
	task := ShellTask{
		Name: "TestShell",
		Execution: Execution{
			IsEnabled: true,
		},
		Command:   "echo 'hello'",
		ShellType: "bash",
		IsGui:     false,
	}

	if task.GetName() != "TestShell" {
		t.Errorf("GetName = %q, want TestShell", task.GetName())
	}
	if task.GetCommand() != "echo 'hello'" {
		t.Errorf("GetCommand = %q, want echo 'hello'", task.GetCommand())
	}
	if task.GetShellType() != "bash" {
		t.Errorf("GetShellType = %q, want bash", task.GetShellType())
	}
	if task.GetIsGui() != false {
		t.Errorf("GetIsGui = %v, want false", task.GetIsGui())
	}
	if task.GetURL() == "" {
		t.Error("GetURL should return 'not-available' for shell task")
	}
	if !task.GetExecution().IsEnabled {
		t.Error("IsEnabled should return true")
	}
}

func TestHttpTaskGetters(t *testing.T) {
	task := HttpTask{
		Name: "TestHttp",
		Execution: Execution{
			IsEnabled: true,
		},
		URL:    "https://example.com",
		Method: "GET",
		Headers: map[string]any{
			"Content-Type": "application/json",
		},
		Body: map[string]any{
			"key": "value",
		},
	}

	if task.GetName() != "TestHttp" {
		t.Errorf("GetName = %q, want TestHttp", task.GetName())
	}
	if task.GetURL() != "https://example.com" {
		t.Errorf("GetURL = %q, want https://example.com", task.GetURL())
	}
	if task.GetMethod() != "GET" {
		t.Errorf("GetMethod = %q, want GET", task.GetMethod())
	}
	if len(task.GetHeaders()) != 1 {
		t.Errorf("GetHeaders len = %d, want 1", len(task.GetHeaders()))
	}
	if len(task.GetBody()) != 1 {
		t.Errorf("GetBody len = %d, want 1", len(task.GetBody()))
	}
	if task.GetCommand() == "" {
		t.Error("GetCommand should return 'not-available' for http task")
	}
}

func TestExecutionSetters(t *testing.T) {
	exec := Execution{}

	now := time.Now()
	exec.SetLastRanAt(now)
	if exec.GetLastRanAtTime() == nil {
		t.Error("GetLastRanAtTime should not be nil after SetLastRanAt")
	}

	exec.SetIsEnabled(true)
	if !exec.IsEnabled {
		t.Error("IsEnabled should be true after SetIsEnabled(true)")
	}

	exec.SetIsEnabled(false)
	if exec.IsEnabled {
		t.Error("IsEnabled should be false after SetIsEnabled(false)")
	}
}

func TestCreateTaskDuplicate(t *testing.T) {
	setupTestEnv(t)

	CreateTask("Duplicate", SHELL_TASK)
	_, err := CreateTask("Duplicate", SHELL_TASK)
	if err == nil {
		t.Error("CreateTask should error when task already exists")
	}
}

func TestUpdateRanAtNonExistent(t *testing.T) {
	setupTestEnv(t)

	err := UpdateRanAt(SHELL_TASK, "NonExistent")
	if err == nil {
		t.Error("UpdateRanAt should error for non-existent task")
	}
}

func TestUpdateTaskStatusNonExistent(t *testing.T) {
	setupTestEnv(t)

	err := UpdateTaskStatus(SHELL_TASK, "NonExistent", true)
	if err == nil {
		t.Error("UpdateTaskStatus should error for non-existent task")
	}
}

func TestSaveTaskHttpAndShell(t *testing.T) {
	setupTestEnv(t)

	shellTask := ShellTask{
		Name:      "SavedShell",
		Command:   "ls",
		ShellType: "bash",
	}
	if err := SaveTask(shellTask, SHELL_TASK); err != nil {
		t.Fatalf("SaveTask for shell failed: %v", err)
	}

	httpTask := HttpTask{
		Name:    "SavedHttp",
		URL:     "https://api.example.com",
		Method:  "POST",
		Headers: map[string]any{},
		Body:    map[string]any{},
	}
	if err := SaveTask(httpTask, HTTP_TASK); err != nil {
		t.Fatalf("SaveTask for http failed: %v", err)
	}

	retrieved, err := GetTask(SHELL_TASK, "SavedShell")
	if err != nil {
		t.Fatalf("GetTask after save failed: %v", err)
	}
	if retrieved.GetName() != "SavedShell" {
		t.Error("retrieved task name mismatch")
	}
}

func TestGetTasksInvalidType(t *testing.T) {
	setupTestEnv(t)

	_, err := GetTasks("unknown")
	if err == nil {
		t.Error("GetTasks should error for invalid type")
	}
}

func TestGetTaskInvalidType(t *testing.T) {
	setupTestEnv(t)

	_, err := GetTask("unknown", "anything")
	if err == nil {
		t.Error("GetTask should error for invalid type")
	}
}

func TestExecutionSetLastRanAt(t *testing.T) {
	t.Parallel()

	exec := Execution{}
	now := time.Now()

	exec.SetLastRanAt(now)

	if exec.LastRanAt == "" {
		t.Fatal("LastRanAt should be set")
	}

	retrieved := exec.GetLastRanAtTime()
	if retrieved == nil {
		t.Fatal("GetLastRanAtTime returned nil")
	}

	diff := now.Sub(*retrieved)
	if diff < 0 {
		diff = -diff
	}
	if diff > 1*time.Second {
		t.Errorf("Time difference too large: %v", diff)
	}
}

func TestExecutionSetIsEnabled(t *testing.T) {
	t.Parallel()

	exec := Execution{}

	exec.SetIsEnabled(true)
	if !exec.GetIsEnabled() {
		t.Error("GetIsEnabled should return true")
	}

	exec.SetIsEnabled(false)
	if exec.GetIsEnabled() {
		t.Error("GetIsEnabled should return false")
	}
}

func TestTaskDependencies(t *testing.T) {
	t.Parallel()

	task := ShellTask{
		Name:      "DependentTask",
		Command:   "echo 'test'",
		ShellType: "sh",
		Execution: Execution{
			IsEnabled: true,
			RunBefore: "Dependency",
		},
	}

	if task.Execution.RunBefore != "Dependency" {
		t.Errorf("Expected RunBefore to be 'Dependency', got %s", task.Execution.RunBefore)
	}

	if task.GetExecution().RunBefore == "" {
		t.Error("Execution.RunBefore should not be empty")
	}
}

func TestTaskDependenciesWithRunAfter(t *testing.T) {
	t.Parallel()

	task := ShellTask{
		Name:      "TaskWithAfter",
		Command:   "echo 'test'",
		ShellType: "sh",
		Execution: Execution{
			IsEnabled: true,
			RunAfter:  "Dependency",
		},
	}

	if task.Execution.RunAfter != "Dependency" {
		t.Errorf("Expected RunAfter to be 'Dependency', got %s", task.Execution.RunAfter)
	}

	if task.GetExecution().RunAfter == "" {
		t.Error("Execution.RunAfter should not be empty")
	}
}

func TestHttpTaskMethods(t *testing.T) {
	tempDir := setupTestEnv(t)

	oldTaskLocation := TaskLocation
	TaskLocation = filepath.Join(tempDir, TASKS_FOLDER)
	defer func() {
		TaskLocation = oldTaskLocation
	}()

	httpTask := HttpTask{
		Name:   "APITask",
		URL:    "https://api.example.com/endpoint",
		Method: "POST",
		Headers: map[string]any{
			"Authorization": "Bearer token123",
			"Content-Type":  "application/json",
		},
		Body: map[string]any{
			"key": "value",
		},
		Execution: Execution{IsEnabled: true},
	}

	err := SaveTask(httpTask, HTTP_TASK)
	if err != nil {
		t.Fatalf("Failed to save HTTP task: %v", err)
	}

	retrieved, err := GetTask(HTTP_TASK, "APITask")
	if err != nil {
		t.Fatalf("Failed to retrieve HTTP task: %v", err)
	}

	if retrieved.GetURL() != "https://api.example.com/endpoint" {
		t.Errorf("URL mismatch: got %s", retrieved.GetURL())
	}

	if retrieved.GetMethod() != "POST" {
		t.Errorf("Method mismatch: got %s", retrieved.GetMethod())
	}

	headers := retrieved.GetHeaders()
	if len(headers) != 2 {
		t.Errorf("Expected 2 headers, got %d", len(headers))
	}

	body := retrieved.GetBody()
	if len(body) != 1 {
		t.Errorf("Expected 1 body item, got %d", len(body))
	}
}

func TestShellTaskGettersExtended(t *testing.T) {
	tempDir := setupTestEnv(t)

	oldTaskLocation := TaskLocation
	TaskLocation = filepath.Join(tempDir, TASKS_FOLDER)
	defer func() {
		TaskLocation = oldTaskLocation
	}()

	task := ShellTask{
		Name:      "ShellGetterTest",
		Command:   "ls -la",
		ShellType: "bash",
		IsGui:     true,
		Execution: Execution{IsEnabled: true},
	}

	if task.GetCommand() != "ls -la" {
		t.Errorf("Command mismatch: got %s", task.GetCommand())
	}

	if task.GetShellType() != "bash" {
		t.Errorf("ShellType mismatch: got %s", task.GetShellType())
	}

	if !task.GetIsGui() {
		t.Error("IsGui should be true")
	}
}

func TestTaskNotDependentWhenEmpty(t *testing.T) {
	t.Parallel()

	task := ShellTask{
		Name:      "IndependentTask",
		Command:   "echo 'independent'",
		ShellType: "sh",
		Execution: Execution{IsEnabled: true},
	}

	if task.GetRunBefore() != nil {
		t.Error("GetRunBefore should return nil when not set")
	}

	if task.GetRunAfter() != nil {
		t.Error("GetRunAfter should return nil when not set")
	}
}

func writeRawTaskFile(t *testing.T, taskType, fileName, contents string) string {
	t.Helper()

	dir := filepath.Join(TaskLocation, taskType)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create task dir: %v", err)
	}

	path := filepath.Join(dir, fileName)
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", fileName, err)
	}

	return path
}

const validShellTaskJSON = `{
  "name": "Good",
  "execution": { "interval": { "seconds": 5 }, "is_enabled": true },
  "command": "echo hi",
  "shell_type": "cmd"
}`

func TestGetTasksReportsMalformedFile(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Good.json", validShellTaskJSON)
	writeRawTaskFile(t, SHELL_TASK, "Broken.json", "{ this is not json")

	tasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	// the good task still loads - one bad file must not break the listing
	if len(tasks) != 1 || tasks[0].GetName() != "Good" {
		t.Fatalf("expected just the parsable task, got %v", taskNames(tasks))
	}

	logged, err := os.ReadFile(AppLogFilePath)
	if err != nil {
		t.Fatalf("failed to read the app log: %v", err)
	}

	if !strings.Contains(string(logged), "Broken.json") {
		t.Errorf("the skipped file should be named in the app log, got:\n%s", logged)
	}
}

func TestFromTaskJsonToleratesBOM(t *testing.T) {
	setupTestEnv(t)

	path := writeRawTaskFile(t, SHELL_TASK, "Bommed.json", utf8BOM+validShellTaskJSON)

	task, err := fromTaskJson(path, SHELL_TASK)
	if err != nil {
		t.Fatalf("a task file with a leading BOM should still parse, got: %v", err)
	}

	if task.GetName() != "Good" {
		t.Errorf("GetName = %q, want %q", task.GetName(), "Good")
	}
}

func TestGetTasksLoadsBOMPrefixedFile(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Good.json", utf8BOM+validShellTaskJSON)

	tasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("a BOM-prefixed task file should load, got %v", taskNames(tasks))
	}
}

func TestGetTaskDistinguishesMalformedFromMissing(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Broken.json", "{ nope")

	_, err := GetTask(SHELL_TASK, "Broken")
	if err == nil {
		t.Fatal("GetTask should fail for a malformed task file")
	}

	if strings.Contains(err.Error(), "does not exist") {
		t.Errorf("a malformed file should not be reported as missing, got: %v", err)
	}
	if !strings.Contains(err.Error(), "could not be read") {
		t.Errorf("the error should say the file could not be read, got: %v", err)
	}

	// a genuinely absent task still reports as absent
	_, err = GetTask(SHELL_TASK, "NeverExisted")
	if err == nil {
		t.Fatal("GetTask should fail for an absent task")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("an absent task should report as missing, got: %v", err)
	}
}

func taskNames(tasks []Task) []string {
	names := make([]string, 0, len(tasks))
	for _, t := range tasks {
		names = append(names, t.GetName())
	}

	return names
}

func TestADivergedTaskIsReachableByItsFileName(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Alpha.json", `{
  "name": "Beta",
  "execution": { "interval": { "seconds": 10 }, "is_enabled": true },
  "command": "echo hi"
}`)

	if !TaskExists(SHELL_TASK, "Alpha") {
		t.Error("the task should be reachable by its file name")
	}

	task, err := GetTask(SHELL_TASK, "Alpha")
	if err != nil {
		t.Fatalf("GetTask by file name failed: %v", err)
	}
	if task.GetCommand() != "echo hi" {
		t.Errorf("unexpected task loaded: %+v", task)
	}

	if TaskExists(SHELL_TASK, "Beta") {
		t.Error("the name field should not be an address")
	}
	if _, err := GetTask(SHELL_TASK, "Beta"); err == nil {
		t.Error("GetTask should fail for a name with no file")
	}
}

func TestUpdatingADivergedTaskWritesBackToTheSameFile(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Alpha.json", `{
  "name": "Beta",
  "execution": { "interval": { "seconds": 10 }, "is_enabled": false },
  "command": "echo hi"
}`)

	if err := UpdateTaskStatus(SHELL_TASK, "Alpha", true); err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	task, err := GetTask(SHELL_TASK, "Alpha")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}
	if !task.GetExecution().IsEnabled {
		t.Error("the enable should have been written to Alpha.json")
	}

	if taskFileExists(SHELL_TASK, "Beta") {
		t.Error("updating must not create a second file under the name field")
	}
}

func TestGetTasksWarnsAboutADivergedTask(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Alpha.json", `{
  "name": "Beta",
  "execution": { "interval": { "seconds": 10 }, "is_enabled": true },
  "command": "echo hi"
}`)

	if _, err := GetTasks(SHELL_TASK); err != nil {
		t.Fatalf("GetTasks failed: %v", err)
	}

	logged, err := os.ReadFile(AppLogFilePath)
	if err != nil {
		t.Fatalf("failed to read the app log: %v", err)
	}

	if !strings.Contains(string(logged), "Alpha.json") {
		t.Errorf("the mismatch should be reported, got:\n%s", logged)
	}
}

func TestABrokenTaskCanStillBeLocatedAndDeleted(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Broken.json", "{ not json at all")

	if TaskExists(SHELL_TASK, "Broken") {
		t.Error("an unparsable file is not a usable task")
	}

	path, err := TaskPath(SHELL_TASK, "Broken")
	if err != nil {
		t.Fatalf("TaskPath should work for a broken task: %v", err)
	}
	if !strings.HasSuffix(path, "Broken.json") {
		t.Errorf("unexpected path %q", path)
	}

	if err := DeleteTask(SHELL_TASK, "Broken"); err != nil {
		t.Fatalf("DeleteTask should work for a broken task: %v", err)
	}
	if taskFileExists(SHELL_TASK, "Broken") {
		t.Error("the file should be gone")
	}
}

func TestCreateTaskRefusesToOverwriteABrokenFile(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Broken.json", "{ not json at all")

	if _, err := CreateTask("Broken", SHELL_TASK); err == nil {
		t.Error("CreateTask should refuse when a file already occupies the name")
	}

	raw, err := os.ReadFile(taskFilePath(SHELL_TASK, "Broken"))
	if err != nil {
		t.Fatalf("failed to read the file back: %v", err)
	}
	if string(raw) != "{ not json at all" {
		t.Errorf("the existing file should be untouched, got %q", raw)
	}
}

func TestMutatorsDistinguishBrokenFromMissing(t *testing.T) {
	setupLogTestEnv(t)
	setupTestEnv(t)
	setupTaskDirs(t)
	InitLogger()

	writeRawTaskFile(t, SHELL_TASK, "Broken.json", "{ nope")

	broken := UpdateTaskStatus(SHELL_TASK, "Broken", true)
	if broken == nil {
		t.Fatal("expected an error for a broken task")
	}
	if !strings.Contains(broken.Error(), "could not be read") {
		t.Errorf("a broken task should say so, got: %v", broken)
	}

	missing := UpdateTaskStatus(SHELL_TASK, "NeverExisted", true)
	if missing == nil {
		t.Fatal("expected an error for a missing task")
	}
	if !strings.Contains(missing.Error(), "does not exist") {
		t.Errorf("a missing task should say so, got: %v", missing)
	}
}

func TestSaveTaskKeepsFileNameAndNameFieldInStep(t *testing.T) {
	setupTestEnv(t)
	setupTaskDirs(t)

	task := ShellTask{Name: "Tidy", Execution: Execution{IsEnabled: true}, Command: "echo hi"}
	if err := SaveTask(task, SHELL_TASK); err != nil {
		t.Fatalf("SaveTask failed: %v", err)
	}

	raw, err := os.ReadFile(taskFilePath(SHELL_TASK, "Tidy"))
	if err != nil {
		t.Fatalf("the file should be named after the name field: %v", err)
	}

	var decoded ShellTask
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if decoded.Name != "Tidy" {
		t.Errorf("name = %q, want Tidy", decoded.Name)
	}
}

func TestTaskFileExistsRejectsUnknownTypesAndDirectories(t *testing.T) {
	setupTestEnv(t)
	setupTaskDirs(t)

	if taskFileExists("ftp", "Whatever") {
		t.Error("an unsupported task type has no files")
	}

	dir := taskFilePath(SHELL_TASK, "ADirectory")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create the directory: %v", err)
	}
	if taskFileExists(SHELL_TASK, "ADirectory") {
		t.Error("a directory should not count as a task file")
	}
}

func TestTaskPath(t *testing.T) {
	setupTestEnv(t)

	if _, err := CreateTask("PathProbe", SHELL_TASK); err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	path, err := TaskPath(SHELL_TASK, "PathProbe")
	if err != nil {
		t.Fatalf("TaskPath failed: %v", err)
	}

	want := filepath.Join(TaskLocation, SHELL_TASK, "PathProbe.json")
	if path != want {
		t.Errorf("TaskPath = %q, want %q", path, want)
	}
}

func TestTaskPathRejectsUnknownTask(t *testing.T) {
	setupTestEnv(t)
	setupTaskDirs(t)

	if _, err := TaskPath(SHELL_TASK, "NeverCreated"); err == nil {
		t.Error("TaskPath should error for a task that does not exist")
	}
}

func TestTaskPathRejectsUnknownType(t *testing.T) {
	setupTestEnv(t)

	if _, err := TaskPath("ftp", "Whatever"); err == nil {
		t.Error("TaskPath should error for an unsupported task type")
	}
}
