package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func NormalizeTaskParams(taskType, taskName string) (string, string) {
	taskType = strings.ToLower(taskType)
	taskName = convertToPascalCase(taskName)
	return taskType, taskName
}

func taskFilePath(taskType, taskName string) string {
	return filepath.Join(TaskLocation, taskType, fmt.Sprintf("%s.json", taskName))
}

func taskFileExists(taskType, taskName string) bool {
	if taskType != HTTP_TASK && taskType != SHELL_TASK {
		return false
	}

	info, err := os.Stat(taskFilePath(taskType, taskName))

	return err == nil && !info.IsDir()
}

func TaskExists(taskType, taskName string) bool {
	if !taskFileExists(taskType, taskName) {
		return false
	}

	_, err := fromTaskJson(taskFilePath(taskType, taskName), taskType)

	return err == nil
}

func GetTasks(taskType string) ([]Task, error) {
	taskType, _ = NormalizeTaskParams(taskType, "")
	taskDir := filepath.Join(TaskLocation, taskType)

	if taskType != HTTP_TASK && taskType != SHELL_TASK {
		return []Task{}, fmt.Errorf("valid task types include: %s | %s", HTTP_TASK, SHELL_TASK)
	}

	files, err := os.ReadDir(taskDir)
	if err != nil {
		return []Task{}, err
	}

	foundTasks := []Task{}
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(taskDir, file.Name())
			data, err := fromTaskJson(path, taskType)
			if err != nil {
				reportSkippedTaskFile(path, err)
				continue
			}

			if base := strings.TrimSuffix(file.Name(), ".json"); data.GetName() != base {
				reportTaskNameMismatch(path, data.GetName(), base)
			}

			foundTasks = append(foundTasks, data)
		}
	}

	return foundTasks, nil
}

func reportSkippedTaskFile(path string, cause error) {
	message := fmt.Sprintf("Skipping task file %s: %s", path, cause.Error())

	fmt.Fprintln(os.Stderr, message)
	LogMessageToFile(message, "error", "app", nil)
}

func reportTaskNameMismatch(path, nameField, fileName string) {
	message := fmt.Sprintf(
		"Task file %s has name %q but is addressed as %q; rename the file or the name field so they match",
		path, nameField, fileName)

	fmt.Fprintln(os.Stderr, message)
	LogMessageToFile(message, "warn", "app", nil)
}

func GetTask(taskType, taskName string) (Task, error) {
	if taskType != HTTP_TASK && taskType != SHELL_TASK {
		return nil, fmt.Errorf("valid task types include: %s | %s", HTTP_TASK, SHELL_TASK)
	}

	taskPath := taskFilePath(taskType, taskName)

	if !taskFileExists(taskType, taskName) {
		return nil, fmt.Errorf("a %s task with the name %s does not exist", taskType, taskName)
	}

	data, err := fromTaskJson(taskPath, taskType)
	if err != nil {
		return nil, fmt.Errorf("the %s task file %s could not be read: %w", taskType, taskPath, err)
	}

	return data, nil
}

func TaskPath(taskType, taskName string) (string, error) {
	if taskType != HTTP_TASK && taskType != SHELL_TASK {
		return "", fmt.Errorf("valid task types include: %s | %s", HTTP_TASK, SHELL_TASK)
	}

	if !taskFileExists(taskType, taskName) {
		return "", fmt.Errorf("a %s task with the name %s does not exist", taskType, taskName)
	}

	return taskFilePath(taskType, taskName), nil
}

func DeleteTask(taskType, taskName string) error {
	if !taskFileExists(taskType, taskName) {
		return fmt.Errorf("a %s task with the name %s does not exist", taskType, taskName)
	}

	return os.Remove(taskFilePath(taskType, taskName))
}

func CreateTask(taskName, taskType string) (string, error) {
	if taskFileExists(taskType, taskName) {
		return "", fmt.Errorf("a %s task with the name %s already exists", taskType, taskName)
	}

	taskInterval := Interval{
		Years:   0,
		Months:  0,
		Weeks:   0,
		Days:    0,
		Hours:   0,
		Minutes: 0,
		Seconds: 0,
	}

	taskExecution := Execution{
		Interval:   taskInterval,
		Delay:      taskInterval,
		RunBefore:  "",
		RunAfter:   "",
		LastRanAt:  "",
		At:         []string{},
		Days:       []string{},
		DayOfMonth: []int{},
	}

	var taskSource Task

	switch taskType {
	case SHELL_TASK:
		taskSource = ShellTask{
			Name:      taskName,
			Execution: taskExecution,
			Command:   "",
			Args:      []string{},
			ShellType: "bash",
			IsGui:     false,
		}
	case HTTP_TASK:
		taskSource = HttpTask{
			Name:      taskName,
			Execution: taskExecution,
			URL:       "",
			Method:    "",
			Headers:   map[string]any{},
			Body:      map[string]any{},
		}
	default:
		return "", fmt.Errorf("invalid task type: %s", taskType)
	}

	if err := SaveTask(taskSource, taskType); err != nil {
		return "", fmt.Errorf("failed to save task: %w", err)
	}

	return fmt.Sprintf("Task %s of type %s created successfully", taskName, taskType), nil
}

func SaveTask(task Task, taskType string) error {
	return saveTaskAs(task, taskType, task.GetName())
}

func saveTaskAs(task Task, taskType, taskName string) error {
	taskDir := filepath.Join(TaskLocation, taskType)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %w", err)
	}

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}

	return os.WriteFile(taskFilePath(taskType, taskName), data, 0644)
}

func UpdateRanAt(taskType, taskName string) error {
	return persistLastRanAt(taskType, taskName, time.Now())
}

func persistLastRanAt(taskType, taskName string, ranAt time.Time) error {
	task, err := GetTask(taskType, taskName)
	if err != nil {
		return err
	}

	switch t := task.(type) {
	case ShellTask:
		t.Execution.SetLastRanAt(ranAt)
		if err := saveTaskAs(t, taskType, taskName); err != nil {
			return fmt.Errorf("failed to save updated shell task: %w", err)
		}
	case HttpTask:
		t.Execution.SetLastRanAt(ranAt)
		if err := saveTaskAs(t, taskType, taskName); err != nil {
			return fmt.Errorf("failed to save updated http task: %w", err)
		}
	default:
		return fmt.Errorf("unknown task type for update")
	}

	return nil
}

func UpdateTaskStatus(taskType, taskName string, status bool) error {
	task, err := GetTask(taskType, taskName)
	if err != nil {
		return err
	}

	switch t := task.(type) {
	case ShellTask:
		if t.Execution.GetIsEnabled() == status {
			if status == true {
				return fmt.Errorf("shell task %s is already enabled", taskName)
			} else {
				return fmt.Errorf("shell task %s is already disabled", taskName)
			}
		}

		t.Execution.SetIsEnabled(status)
		if err := saveTaskAs(t, taskType, taskName); err != nil {
			return fmt.Errorf("failed to save updated shell task: %w", err)
		}
	case HttpTask:
		if t.Execution.GetIsEnabled() == status {
			if status == true {
				return fmt.Errorf("http task %s is already enabled", taskName)
			} else {
				return fmt.Errorf("http task %s is already disabled", taskName)
			}
		}
		t.Execution.SetIsEnabled(status)
		if err := saveTaskAs(t, taskType, taskName); err != nil {
			return fmt.Errorf("failed to save updated http task: %w", err)
		}
	default:
		return fmt.Errorf("unknown task type for update")
	}

	return nil
}

func fromTaskJson(filePath, taskType string) (Task, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	data = bytes.TrimPrefix(data, []byte(utf8BOM))

	taskType = strings.ToLower(taskType)
	switch taskType {
	case SHELL_TASK:
		var task ShellTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse shell task: %w", err)
		}
		return task, nil
	case HTTP_TASK:
		var task HttpTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse HTTP task: %w", err)
		}
		return task, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

func convertToPascalCase(input string) string {
	re := regexp.MustCompile(`[-_]+`)
	parts := re.Split(input, -1)

	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, "")
}

func FetchTaskByName(name string) (Task, error) {
	for _, taskType := range knownTaskTypes {
		tasks, err := GetTasks(taskType)
		if err != nil {
			continue
		}

		for _, t := range tasks {
			if t.GetName() == name {
				return t, nil
			}
		}
	}
	return nil, fmt.Errorf("task '%s' not found in any type", name)
}
