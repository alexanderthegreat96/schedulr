package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	consoleLogger *log.Logger

	appLogFile *os.File
	appLogger  *log.Logger

	logMutex     sync.Mutex
	taskLoggers  = make(map[string]*log.Logger)
	taskLogFiles = make(map[string]*os.File)
	taskLogPaths = make(map[string]string)
)

// console logger initialization
func InitLogger() {
	consoleLogger = log.New(os.Stdout, "", 0)
}

// print message to console
func LogMessage(message, level string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	color := getColorForLevel(level)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("%s[Schedulr][%s][%s]%s - ", color, levelUpper(level), timestamp, ColorReset)

	consoleLogger.SetPrefix(header)
	consoleLogger.Println(message)
}

// save the logs to file
func LogMessageToFile(message, level, logType string, task Task) error {
	if !AppConfig().LogData {
		return nil
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	var logger *log.Logger

	switch strings.ToLower(logType) {
	case "task":
		key := task.GetName()
		logFilePath := getOrCreateLogFilePathForTask(task)

		fileMissing := false
		if f, ok := taskLogFiles[key]; !ok || f == nil {
			fileMissing = true
		} else if _, err := os.Stat(logFilePath); os.IsNotExist(err) {
			_ = f.Close()
			fileMissing = true
		}

		if _, exists := taskLoggers[key]; !exists || fileMissing {
			dir := filepath.Dir(logFilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create task log directory: %w", err)
			}

			f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open task log file: %w", err)
			}

			taskLogFiles[key] = f
			taskLoggers[key] = log.New(f, "", 0)
		}

		logger = taskLoggers[key]

	case "app":
		fileMissing := false
		if appLogFile == nil {
			fileMissing = true
		} else if _, err := os.Stat(AppLogFilePath); os.IsNotExist(err) {
			_ = appLogFile.Close()
			appLogFile = nil
			fileMissing = true
		}

		if appLogger == nil || fileMissing {
			dir := filepath.Dir(AppLogFilePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create app log directory: %w", err)
			}

			f, err := os.OpenFile(AppLogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open app log file: %w", err)
			}

			appLogFile = f
			appLogger = log.New(f, "", 0)
		}

		logger = appLogger

	default:
		return fmt.Errorf("unknown log type: %s", logType)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("[Schedulr][%s][%s] - ", levelUpper(level), timestamp)

	logger.SetPrefix(header)
	logger.Println(message)
	return nil
}

// clear all loggers
func CloseLoggers() {
	logMutex.Lock()
	defer logMutex.Unlock()

	if appLogFile != nil {
		_ = appLogFile.Close()
		appLogFile = nil
		appLogger = nil
	}

	for k, f := range taskLogFiles {
		_ = f.Close()
		delete(taskLogFiles, k)
		delete(taskLoggers, k)
		delete(taskLogPaths, k)
	}
}

// doing this because if not
// then then when the system checks if a
// log file for tasks which includes a timestamp
// will always be different and re-created
func getOrCreateLogFilePathForTask(task Task) string {
	key := task.GetName()

	if path, exists := taskLogPaths[key]; exists {
		return path
	}

	timestamp := time.Now().Format("2006-01-02T15-04-05.000Z")
	RootPath, _ := GetRootDir()
	name := sanitizeFileName(task.GetName())
	path := filepath.Join(RootPath, TASK_LOGS_DIR, fmt.Sprintf("%s_%s.log", name, timestamp))

	taskLogPaths[key] = path
	return path
}

// some sanitization
func sanitizeFileName(name string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(`\/:*?"<>| `, r) {
			return '_'
		}
		return r
	}, name)
}

// self explanatory
func getColorForLevel(level string) string {
	switch level {
	case "info":
		return ColorGreen
	case "success":
		return ColorOpenGreen
	case "warn", "warning":
		return ColorYellow
	case "error":
		return ColorRed
	case "debug":
		return ColorBlue
	default:
		return ColorReset
	}
}

func levelUpper(level string) string {
	switch level {
	case "info":
		return "INFO"
	case "success":
		return "SUCCESS"
	case "warn":
		return "WARN"
	case "warning":
		return "WARNING"
	case "error":
		return "ERROR"
	case "debug":
		return "DEBUG"
	default:
		return strings.ToUpper(level)
	}
}
