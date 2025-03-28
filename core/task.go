package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func normalizeTaskParams(taskType, taskName string) (string, string) {
	taskType = strings.ToLower(taskType)
	taskName = convertToPascalCase(taskName)
	return taskType, taskName
}

func TaskExists(taskType, taskName string) bool {
	taskType, taskName = normalizeTaskParams(taskType, taskName)
	taskDir := filepath.Join(taskLocation, taskType)

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
	taskType, _ = normalizeTaskParams(taskType, "")
	taskDir := filepath.Join(taskLocation, taskType)

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

func CreateTask(taskName, taskType string) (string, error) {
	taskType, taskName = normalizeTaskParams(taskType, taskName)

	if TaskExists(taskType, taskName) {
		return "", fmt.Errorf("a %s task with the name %s already exists", taskType, taskName)
	}

	taskDir := filepath.Join(taskLocation, taskType)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create task directory: %w", err)
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

	jsonData, err := json.MarshalIndent(taskSource, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to serialize task: %v", err)
	}

	fileName := filepath.Join(taskDir, taskName+".json")
	if err := os.WriteFile(fileName, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write task file: %v", err)
	}

	return fmt.Sprintf("Task %s of type %s created successfully", taskName, taskType), nil
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
