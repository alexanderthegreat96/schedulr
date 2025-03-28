package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/alexanderthegreat96/envparser"
)

// global access to config variables
// useing schedulr.config for this
// with env syntax using my custom env parser lib

var envData *envparser.EnvData = envparser.NewEnvParser("schedulr.config")

func DefaultValueIfNull(value any, typeName string) any {
	if value == nil {
		return defaultForType(typeName)
	}

	v := reflect.ValueOf(value)

	switch typeName {
	case "string":
		if str, ok := value.(string); ok && str == "" {
			return "no-value"
		}
	case "int":
		if i, ok := value.(int); ok && i == 0 {
			return 0
		}
	case "float64":
		if f, ok := value.(float64); ok && f == 0 {
			return 0.0
		}
	case "bool":
		if b, ok := value.(bool); ok && !b {
			return false
		}
	case "map":
		if v.Kind() == reflect.Map && v.Len() == 0 {
			return map[string]any{}
		}
	case "slice":
		if v.Kind() == reflect.Slice && v.Len() == 0 {
			return []any{}
		}
	case "any":
		if v.IsZero() {
			return defaultForType("any")
		}
	}

	return value
}

func defaultForType(typeName string) any {
	switch typeName {
	case "string":
		return "no-value"
	case "int":
		return 0
	case "float64":
		return 0.0
	case "bool":
		return false
	case "map":
		return map[string]any{}
	case "slice":
		return []any{}
	case "any":
		return "(unset)"
	default:
		return nil
	}
}

func IsDevMode() bool {
	isDev, err := envData.GetValue("SCHEDULR_DEV", "bool", false)
	if err != nil {
		return false
	}

	boolVal, ok := isDev.(bool)
	if !ok {
		return false
	}

	return boolVal
}

func ShouldLogData() bool {
	shouldLog, err := envData.GetValue("LOG_DATA", "bool", true)
	if err != nil {
		return true
	}

	boolVal, ok := shouldLog.(bool)
	if !ok {
		return true
	}

	return boolVal
}

func ClearLogsAfterSeconds() int {
	seconds, err := envData.GetValue("LOG_WIPE_INTERVAL_SECONDS", "int", 30)
	if err != nil {
		return 30
	}

	intVal, ok := seconds.(int)
	if !ok {
		return 30
	}

	return intVal
}

func GetRootDir() (string, error) {
	if IsDevMode() {
		return os.Getwd()
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func PidFileExists() bool {
	info, err := os.Stat(pidFilePath)
	return err == nil && !info.IsDir()
}

func CreatePidFile(pid int) error {
	if PidFileExists() {
		return fmt.Errorf("%s already exists", pidFile)
	}

	os.WriteFile(pidFilePath, []byte(strconv.Itoa(pid)), 0644)
	return nil
}

func DeletePidFile() error {
	if !PidFileExists() {
		return fmt.Errorf("%s does not exist", pidFile)
	}

	return os.Remove(pidFilePath)
}

func ReadPidFile() (int, error) {
	if !PidFileExists() {
		return 0, fmt.Errorf("%s does not exist", pidFile)
	}

	data, err := os.ReadFile(pidFilePath)
	if err != nil {
		return 0, fmt.Errorf("unable to read pid file. error: %e", err)
	}

	pid, err := strconv.Atoi(string(data))

	if err != nil {
		return 0, fmt.Errorf("invalid pid format. err: %e", err)
	}

	return pid, nil
}

func GetProcAttr() *syscall.SysProcAttr {
	if runtime.GOOS == "windows" {
		// no clue if this works on windows
		// will leave it here for building reasons anyways
		return &syscall.SysProcAttr{}
	}

	// linux / unix terminal detach
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-c
		LogMessageToFile("Received signal: "+sig.String(), "info", "app")
		LogMessageToFile("Shutting down daemon...", "info", "app")
		CloseLoggers()
		os.Exit(0)
	}()
}

func DescribeInterval(interval Interval, start time.Time, intervalType string) (string, time.Time) {
	parts := buildIntervalParts(interval)
	intervalType = strings.ToLower(intervalType)
	var description string

	switch intervalType {
	case DELAY:
		if len(parts) == 0 {
			description = "no delay"
		} else {
			description = "delay of " + formatParts(parts)
		}
	case INTERVAL:
		if len(parts) == 0 {
			description = "immediately"
		} else {
			description = "every " + formatParts(parts)
		}
	default:
		if len(parts) == 0 {
			description = "no schedule defined"
		} else {
			description = formatParts(parts)
		}
	}

	nextRun := calculateNextRun(start, interval)
	return description, nextRun
}

func DescribeSchedule(exec Execution, start time.Time) (string, time.Time, time.Time) {
	firstRun := calculateNextRun(start, exec.Delay)
	nextRun := calculateNextRun(firstRun, exec.Interval)

	delayDesc := "with no delay"
	intervalDesc := "does not repeat"

	if len(buildIntervalParts(exec.Delay)) > 0 {
		delayDesc = "after " + formatParts(buildIntervalParts(exec.Delay))
	}
	if len(buildIntervalParts(exec.Interval)) > 0 {
		intervalDesc = "every " + formatParts(buildIntervalParts(exec.Interval))
	}

	description := fmt.Sprintf("This task runs %s, %s.", delayDesc, intervalDesc)
	return description, firstRun, nextRun
}

func GetFirstAndNextRun(start time.Time, delay Interval, interval Interval) (time.Time, time.Time) {
	firstRun := calculateNextRun(start, delay)
	nextRun := calculateNextRun(firstRun, interval)
	return firstRun, nextRun
}

func buildIntervalParts(i Interval) []string {
	var parts []string
	if i.Years > 0 {
		parts = append(parts, pluralize(i.Years, "year"))
	}
	if i.Months > 0 {
		parts = append(parts, pluralize(i.Months, "month"))
	}
	if i.Weeks > 0 {
		parts = append(parts, pluralize(i.Weeks, "week"))
	}
	if i.Days > 0 {
		parts = append(parts, pluralize(i.Days, "day"))
	}
	if i.Hours > 0 {
		parts = append(parts, pluralize(i.Hours, "hour"))
	}
	if i.Minutes > 0 {
		parts = append(parts, pluralize(i.Minutes, "minute"))
	}
	if i.Seconds > 0 {
		parts = append(parts, pluralize(i.Seconds, "second"))
	}
	return parts
}

func formatParts(parts []string) string {
	if len(parts) == 1 {
		return parts[0]
	}
	if len(parts) == 2 {
		return parts[0] + " and " + parts[1]
	}
	return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}

func pluralize(value int, unit string) string {
	if value == 1 {
		return fmt.Sprintf("%d %s", value, unit)
	}
	return fmt.Sprintf("%d %ss", value, unit)
}

func StartLogWiper(logFilePath string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			<-ticker.C
			err := os.Truncate(logFilePath, 0)
			if err != nil {
				fmt.Printf("Error wiping log file (%s): %v\n", logFilePath, err)
			} else {
				fmt.Printf("Log file %s wiped at %s\n", logFilePath, time.Now().Format(time.RFC3339))
			}
		}
	}()
}

func DeleteLogFiles(folder string) error {
	return filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(info.Name()) == ".log" {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete %s: %w", path, err)
			}
		}

		return nil
	})
}

func GetLatestLogFile(logDir, pattern string) (string, error) {
	files, err := filepath.Glob(filepath.Join(logDir, pattern))
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("no log files found in %s matching pattern %s", logDir, pattern)
	}

	sort.Slice(files, func(i, j int) bool {
		fi, err := os.Stat(files[i])
		if err != nil {
			return false
		}
		fj, err := os.Stat(files[j])
		if err != nil {
			return false
		}
		return fi.ModTime().After(fj.ModTime())
	})
	return files[0], nil
}

func TailLogFile(logFilePath string) error {
	f, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	_, err = f.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("failed to seek to end of file: %w", err)
	}

	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			return fmt.Errorf("error reading log file: %w", err)
		}
		fmt.Print(line)
	}
}

// ensures that all files and folders
// required for shcedulr are found and are working

func AutoSetup() error {
	dirs := []string{
		filepath.Join(taskLocation, "shell"),
		filepath.Join(taskLocation, "http"),
		filepath.Join(rootPath, APP_LOGS_DIR),
		filepath.Join(rootPath, TASK_LOGS_DIR),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	configFilePath := filepath.Join(rootPath, "schedulr.config")
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		configFileContents := `SCHEDULR_DEV=false
LOG_DATA=true
LOG_WIPE_INTERVAL_SECONDS=20
		`
		if err := os.WriteFile(configFilePath, []byte(configFileContents), 0644); err != nil {
			return fmt.Errorf("failed to create schedulr.config file in: %s: %w", configFilePath, err)
		}

	}

	return nil
}
