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
	appLogFile    *os.File
	appLogger     *log.Logger
	taskLogFile   *os.File
	taskLogger    *log.Logger
	logMutex      sync.Mutex
)

func InitLogger() {
	consoleLogger = log.New(os.Stdout, "", 0)
}

func LogMessage(message, level string) {
	logMutex.Lock()
	defer logMutex.Unlock()

	color := getColorForLevel(level)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("%s[Schedulr][%s][%s]%s - ", color, levelUpper(level), timestamp, ColorReset)

	consoleLogger.SetPrefix(header)
	consoleLogger.Println(message)
}

func LogMessageToFile(message, level, logType string) error {
	if !AppConfig().LogData {
		return nil
	}

	logMutex.Lock()
	defer logMutex.Unlock()

	var (
		logFilePath string
		logger      *log.Logger
		fileHandle  **os.File // pointer to the appropriate file variable
	)

	switch strings.ToLower(logType) {
	case "task":
		logFilePath = TasksLogFilePath
		fileHandle = &taskLogFile
		logger = taskLogger
	case "app":
		logFilePath = AppLogFilePath
		fileHandle = &appLogFile
		logger = appLogger
	default:
		return fmt.Errorf("unknown log type: %s", logType)
	}

	// Create directory if necessary.
	dir := filepath.Dir(logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Initialize the file and logger if not set.
	if *fileHandle == nil {
		f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		*fileHandle = f
		// Set the logger for this type.
		if strings.ToLower(logType) == "task" {
			taskLogger = log.New(f, "", 0)
			logger = taskLogger
		} else {
			appLogger = log.New(f, "", 0)
			logger = appLogger
		}
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	header := fmt.Sprintf("[Schedulr][%s][%s] - ", levelUpper(level), timestamp)

	logger.SetPrefix(header)
	logger.Println(message)
	return nil
}

func CloseLoggers() {
	logMutex.Lock()
	defer logMutex.Unlock()
	if appLogFile != nil {
		appLogFile.Close()
		appLogFile = nil
	}
	if taskLogFile != nil {
		taskLogFile.Close()
		taskLogFile = nil
	}
}

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
		return level
	}
}
