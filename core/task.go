package core

import (
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

func TaskExists(taskType, taskName string) bool {
	taskDir := filepath.Join(TaskLocation, taskType)
	files, err := os.ReadDir(taskDir)
	if err != nil {
		return false
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(taskDir, file.Name())
			data, err := fromTaskJson(path, taskType)
			if err != nil {
				continue
			}

			if data.GetName() == taskName {
				return true
			}
		}
	}
	return false
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
				continue
			}

			foundTasks = append(foundTasks, data)
		}
	}

	return foundTasks, nil
}

func GetTask(taskType, taskName string) (Task, error) {
	taskDir := filepath.Join(TaskLocation, taskType)
	if taskType != HTTP_TASK && taskType != SHELL_TASK {
		return nil, fmt.Errorf("valid task types include: %s | %s", HTTP_TASK, SHELL_TASK)
	}

	if !TaskExists(taskType, taskName) {
		return nil, fmt.Errorf("a %s task with the name %s does not exist", taskType, taskName)
	}

	taskPath := filepath.Join(taskDir, fmt.Sprintf("%s.json", taskName))
	data, err := fromTaskJson(taskPath, taskType)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func DeleteTask(taskType, taskName string) error {
	if !TaskExists(taskType, taskName) {
		return fmt.Errorf("a %s task with the name %s does not exist", taskType, taskName)
	}
	return os.Remove(filepath.Join(TaskLocation, taskType, fmt.Sprintf("%s.json", taskName)))
}

func CreateTask(taskName, taskType string) (string, error) {
	if TaskExists(taskType, taskName) {
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
		Interval:  taskInterval,
		Delay:     taskInterval,
		RunBefore: "",
		RunAfter:  "",
		LastRanAt: "",
	}

	var taskSource Task

	switch taskType {
	case SHELL_TASK:
		taskSource = ShellTask{
			Name:      taskName,
			Execution: taskExecution,
			Command:   "",
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
	taskDir := filepath.Join(TaskLocation, taskType)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %w", err)
	}

	taskPath := filepath.Join(taskDir, fmt.Sprintf("%s.json", task.GetName()))

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize task: %w", err)
	}
	return os.WriteFile(taskPath, data, 0644)
}

func UpdateRanAt(taskType, taskName string) error {
	if !TaskExists(taskType, taskName) {
		return fmt.Errorf("no task with the name %s of type %s found", taskName, taskType)
	}

	task, err := GetTask(taskType, taskName)
	if err != nil {
		return fmt.Errorf("unable to get task with name %s of type %s", taskName, taskType)
	}

	now := time.Now()

	switch t := task.(type) {
	case ShellTask:
		(&t.Execution).SetLastRanAt(now)
		if err := SaveTask(t, taskType); err != nil {
			return fmt.Errorf("failed to save updated shell task: %w", err)
		}
	case HttpTask:
		(&t.Execution).SetLastRanAt(now)
		if err := SaveTask(t, taskType); err != nil {
			return fmt.Errorf("failed to save updated http task: %w", err)
		}
	default:
		return fmt.Errorf("unknown task type for update")
	}

	return nil
}

func UpdateTaskStatus(taskType, taskName string, status bool) error {
	if !TaskExists(taskType, taskName) {
		return fmt.Errorf("no task with the name %s of type %s found", taskName, taskType)
	}

	task, err := GetTask(taskType, taskName)
	if err != nil {
		return fmt.Errorf("unable to get task with name %s of type %s", taskName, taskType)
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
		if err := SaveTask(t, taskType); err != nil {
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
		if err := SaveTask(t, taskType); err != nil {
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
