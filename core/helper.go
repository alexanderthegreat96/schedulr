package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

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

func GetRootDir() (string, error) {
	if AppConfig().DevMode {
		return os.Getwd()
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(exePath), nil
}

func PidFileExists() bool {
	info, err := os.Stat(PidFilePath)
	return err == nil && !info.IsDir()
}

func CreatePidFile(pid int) error {
	if PidFileExists() {
		return fmt.Errorf("%s already exists", pidFile)
	}

	contents := strconv.Itoa(pid)
	if exe, err := os.Executable(); err == nil {
		contents += "\n" + exe
	}

	if err := os.WriteFile(PidFilePath, []byte(contents+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", pidFile, err)
	}

	return nil
}

func DeletePidFile() error {
	if !PidFileExists() {
		return fmt.Errorf("%s does not exist", pidFile)
	}

	return os.Remove(PidFilePath)
}

func ReadPidFile() (int, error) {
	pid, _, err := readPidFile()
	return pid, err
}

func readPidFile() (int, string, error) {
	if !PidFileExists() {
		return 0, "", fmt.Errorf("%s does not exist", pidFile)
	}

	data, err := os.ReadFile(PidFilePath)
	if err != nil {
		return 0, "", fmt.Errorf("unable to read %s: %w", pidFile, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	pid, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return 0, "", fmt.Errorf("invalid pid format in %s: %w", pidFile, err)
	}

	executable := ""
	if len(lines) > 1 {
		executable = strings.TrimSpace(lines[1])
	}

	return pid, executable, nil
}

func DaemonRunning() (int, bool) {
	pid, recorded, err := readPidFile()
	if err != nil {
		return 0, false
	}

	if !ProcessAlive(pid) {
		return pid, false
	}

	if recorded != "" {
		if actual, err := ProcessExecutable(pid); err == nil && actual != "" {
			if !samePath(actual, recorded) {
				return pid, false
			}
		}
	}

	return pid, true
}

func WaitForProcessToExit(pid int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for {
		if !ProcessAlive(pid) {
			return true
		}

		if time.Now().After(deadline) {
			return false
		}

		time.Sleep(50 * time.Millisecond)
	}
}

func ClearStalePidFile() (int, bool) {
	if !PidFileExists() {
		return 0, false
	}

	pid, running := DaemonRunning()
	if running {
		return pid, false
	}

	if err := DeletePidFile(); err != nil {
		return pid, false
	}

	return pid, true
}

func samePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)

	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}

	return a == b
}

func SetupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-c
		LogMessageToFile("Received signal: "+sig.String(), "info", "app", nil)
		LogMessageToFile("Shutting down daemon...", "info", "app", nil)
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

func NextRunFromNow(exec Execution) time.Duration {
	firstRun, _ := GetFirstAndNextRun(time.Now(), *exec.GetLastRanAtTime(), exec.Delay, exec.Interval)
	return time.Until(firstRun)
}

func DescribeSchedule(exec Execution, start time.Time) (string, time.Time) {
	if exec.HasCalendarSchedule() {
		return describeCalendarSchedule(exec, start)
	}

	firstRun := calculateNextRun(start, exec.Delay)
	if t := exec.GetLastRanAtTime(); t != nil {
		firstRun = calculateNextRun(*exec.GetLastRanAtTime(), exec.Interval)
	}

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

	return description, nextRun
}

func GetFirstAndNextRun(start time.Time, lastRanAt time.Time, delay Interval, interval Interval) (time.Time, time.Time) {
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

func StartLogWiper(logDir string, interval time.Duration) {
	if interval <= 0 {
		LogMessage(fmt.Sprintf("Log wiping disabled for %s (interval is %s)", logDir, interval), "info")
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			if err := DeleteStaleLogFiles(logDir); err != nil {
				LogMessage(fmt.Sprintf("Error wiping logs in %s: %v", logDir, err), "error")
			}
		}
	}()
}

func DeleteStaleLogFiles(folder string) error {
	return deleteLogFiles(folder, ActiveLogPaths())
}

func DeleteLogFiles(folder string) error {
	_, inUse, err := ClearLogFiles(folder)
	if err != nil {
		return err
	}

	if len(inUse) > 0 {
		return fmt.Errorf("could not delete %d log file(s) still in use: %s", len(inUse), strings.Join(inUse, ", "))
	}

	return nil
}

func ClearLogFiles(folder string) (removed int, inUse []string, err error) {
	walkErr := filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(info.Name()) != ".log" {
			return nil
		}

		if err := os.Remove(path); err != nil {
			inUse = append(inUse, filepath.Base(path))
			return nil
		}

		removed++

		return nil
	})

	return removed, inUse, walkErr
}

func deleteLogFiles(folder string, keep map[string]bool) error {
	return filepath.Walk(folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(info.Name()) != ".log" {
			return nil
		}

		if keep[filepath.Clean(path)] {
			return nil
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to delete %s: %w", path, err)
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
	const numLines = 10

	file, err := os.Open(logFilePath)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file stat: %w", err)
	}

	var lines []string
	var pos int64 = stat.Size()
	buf := make([]byte, 1)
	lineBuf := ""
	for pos > 0 && len(lines) < numLines {
		pos--
		_, err := file.ReadAt(buf, pos)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		if buf[0] == '\n' {
			if lineBuf != "" {
				lines = append([]string{lineBuf}, lines...)
				lineBuf = ""
			}
		} else {
			lineBuf = string(buf) + lineBuf
		}
	}
	if lineBuf != "" {
		lines = append([]string{lineBuf}, lines...)
	}

	for _, line := range lines {
		fmt.Println(line)
	}

	// Now tail the file
	file.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(file)
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
		filepath.Join(TaskLocation, "shell"),
		filepath.Join(TaskLocation, "http"),
		filepath.Join(RootPath, APP_LOGS_DIR),
		filepath.Join(RootPath, TASK_LOGS_DIR),
	}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dir, err)
			}
		}
	}

	err := createConfigFile(RootPath)
	if err != nil {
		return err
	}

	err = createProjectRootMarker(RootPath)
	if err != nil {
		return err
	}

	err = createAutoStartScript(RootPath)
	if err != nil {
		return err
	}

	return nil
}

func createProjectRootMarker(RootPath string) error {
	markerPath := filepath.Join(RootPath, projectRootMarker)

	if _, err := os.Stat(markerPath); err == nil {
		return nil
	}

	contents := "# Marks this directory as the schedulr root.\n" +
		"# Required so schedulr.config beside the binary is picked up.\n" +
		"# Safe to commit, and recreated automatically if deleted.\n"

	if err := os.WriteFile(markerPath, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to create %s in %s: %w", projectRootMarker, RootPath, err)
	}

	return nil
}

func IsRunningUnderSystemd() bool {
	cmd := exec.Command(AppConfig().SystemDCommand, "is-active", AppConfig().ServiceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}

	status := strings.TrimSpace(string(output))
	return status == "active" || status == "activating"
}

func IsRunningUnderLaunchd() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	ppid := os.Getppid()
	cmd := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == AppConfig().LaunchDCommand
}

func IsManagedByInitSystem() bool {
	return IsRunningUnderSystemd() || IsRunningUnderLaunchd()
}

func CheckSystemdStatus(serviceName string) (string, error) {
	output, err := exec.Command(AppConfig().SystemDCommand, "is-active", serviceName).Output()
	return strings.TrimSpace(string(output)), err
}

func CheckLaunchdStatus(label string) (string, error) {
	output, err := exec.Command(AppConfig().LaunchDCommand, "list").Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, label) {
			return "active", nil
		}
	}
	return "inactive", nil
}

func KillSystemDService() error {
	return exec.Command(AppConfig().SystemDCommand, "stop", AppConfig().ServiceName).Run()
}

func KillLaunchDService() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	plistPath := fmt.Sprintf("%s/Library/LaunchAgents/%s.plist", homeDir, AppConfig().ServiceName)
	return exec.Command(AppConfig().LaunchDCommand, "unload", plistPath).Run()
}

func createConfigFile(RootPath string) error {
	configFilePath := filepath.Join(RootPath, configFileName)
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		configFileContents := `# ====================
# Development Settings
# ====================

# Enable developer mode (adds extra debug info, logs, etc.)
# Set to true only during development.
SCHEDULR_DEV=false

# ===================
# Logging Configuration
# ===================

# Enable or disable log writing to files
LOG_DATA=true

# Interval (in seconds) for wiping old logs
# Useful for cleanup in long-running daemons
# Log files the daemon still has open are never wiped. Set to 0 to disable.
LOG_WIPE_INTERVAL_SECONDS=100

# ===================
# Worker Configuration
# ===================

# Number of parallel worker threads/tasks to run
WORKER_COUNT=4

# ===================
# Scheduler Configuration
# ===================

# How often (in seconds) the daemon re-reads the task files from disk.
# This is what makes add / enable / disable / remove take effect while the
# daemon is running, without having to restart it.
RELOAD_INTERVAL_SECONDS=30

# Longest time (in seconds) a worker will wait for a run_before / run_after
# dependency to become due. A dependency that is further out than this is
# skipped for that run, and the task that declared it still executes.
# Waiting happens on a worker, so an unbounded wait would starve the pool.
DEPENDENCY_MAX_WAIT_SECONDS=60

# ============================
# Service Execution Mode
# ============================

# Determines whether Schedulr should run as a system service (e.g., via systemd/launchd)
# true  = run in background as a system service (default)
# false = run manually or as a foreground process
ENABLE_SCHEDULR_SERVICE=true

# System command to manage the service if ENABLE_SCHEDULR_SERVICE is true
# Set to 'systemctl' for Linux or 'launchctl' for macOS
SYSTEMD_COMMAND=systemctl
LAUNCHD_COMMAND=launchctl

# The name of the service (used for systemd or launchctl)
SERVICE_NAME=schedulr

# ===================
# Logging Privacy
# ===================

# Largest HTTP response body, in bytes, written to a task log.
# Responses can carry tokens and personal data, and logs outlive the request.
# Set to 0 to keep response bodies out of the logs entirely.
HTTP_LOG_BODY_LIMIT=2048

		`
		if err := os.WriteFile(configFilePath, []byte(configFileContents), 0644); err != nil {
			return fmt.Errorf("failed to create schedulr.config file in: %s: %w", configFilePath, err)
		}
	}

	return nil
}
func createAutoStartScript(RootPath string) error {
	var fileName string
	var scriptContents string

	if runtime.GOOS == "windows" {
		fileName = "auto-start.ps1"
		scriptContents = `# 1. Get the script's actual directory
$DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $DIR

# 2. Define Log File
$LOG_FILE = "schedulr_autostart.log"

# 3. Start Logging (Overwrites the file and shows output in console)
Start-Transcript -Path $LOG_FILE -Force

Write-Host "---------------------------------------"
Write-Host "LOG START: $(Get-Date)"
Write-Host "Working Directory: $DIR"

# 4. Remove PID if it exists
if (Test-Path "schedulr.pid") {
    Write-Host "Found old schedulr.pid, removing..."
    Remove-Item "schedulr.pid" -Force
} else {
    Write-Host "No old PID file found. Clean start."
}

# 5. Run the daemon
Write-Host "Attempting to start schedulr..."
Start-Process -FilePath ".\schedulr.exe" -ArgumentList "start" -WindowStyle Hidden

# 6. Check if it actually started
Start-Sleep -Seconds 2
if (Get-Process -Name "schedulr" -ErrorAction SilentlyContinue) {
    Write-Host "SUCCESS: Schedulr is running!"
} else {
    Write-Host "FAILED: Schedulr did not start."
}

Write-Host "---------------------------------------"
Stop-Transcript`

	} else {
		// macos / linux
		fileName = "auto-start.sh"
		scriptContents = `#!/bin/bash
# 1. Get the script's actual directory
DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
cd "$DIR" || { echo "Error: Could not change directory"; exit 1; }

# 2. Define Log File
LOG_FILE="schedulr_autostart.log"

# 3. THE MAGIC LINE: Redirect all output (stdout and stderr)
# to both the terminal and the log file.
exec > >(tee "$LOG_FILE") 2>&1

echo "---------------------------------------"
echo "Timestamp: $(date)"
echo "Schedulr Auto-Start Script Initialized"
echo "Working Directory: $DIR"

# Remove PID if it exists
if [ -f "schedulr.pid" ]; then
    echo "Found old schedulr.pid, removing..."
    rm -f "schedulr.pid"
else
    echo "No old PID file found. Clean start."
fi

# Run the daemon
echo "Attempting to start schedulr..."
./schedulr start

# Check if it actually started
sleep 1
if pgrep -f "schedulr" > /dev/null; then
    echo "SUCCESS: Schedulr is running!"
else
    echo "FAILED: Schedulr did not start. Check permissions of the binary."
fi
echo "---------------------------------------"
		`
	}

	filePath := filepath.Join(RootPath, fileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		err := os.WriteFile(filePath, []byte(scriptContents), 0755)
		if err != nil {
			return fmt.Errorf("failed to create auto-start script in %s: %w", filePath, err)
		}
	}

	return nil
}
