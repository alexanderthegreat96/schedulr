package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func ExecuteTask(task Task) {
	now := time.Now().UTC()

	switch t := task.(type) {
	case ShellTask:
		LogMessageToFile(fmt.Sprintf("Executing shell task: %s", t.GetName()), "info", "app", nil)
		cmd := getShellCommand(t.GetCommand(), t.ShellType) // os aware

		output, err := cmd.CombinedOutput()
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error executing shell task %s: %v", t.GetName(), err), "error", "task", task)
		} else {
			LogMessageToFile(fmt.Sprintf("Shell task %s output: %s\n", t.GetName(), strings.TrimSpace(string(output))), "info", "task", task)
			(&t.Execution).SetLastRanAt(now)

			if err := SaveTask(t, SHELL_TASK); err != nil {
				LogMessageToFile(fmt.Sprintf("failed to save updated shell task: %s", err.Error()), "error", "task", task)
			}
		}

	case HttpTask:
		LogMessageToFile(fmt.Sprintf("Executing HTTP task: %s", t.GetName()), "info", "app", nil)
		client := &http.Client{Timeout: 10 * time.Second}

		var bodyReader io.Reader
		if len(t.GetBody()) > 0 && !strings.EqualFold(t.GetMethod(), "GET") && !strings.EqualFold(t.GetMethod(), "HEAD") {
			jsonData, err := json.Marshal(t.GetBody())
			if err != nil {
				LogMessageToFile(fmt.Sprintf("Error marshaling HTTP task body for task %s: %v", t.GetName(), err), "error", "task", task)
				return
			}
			bodyReader = bytes.NewReader(jsonData)
		}

		req, err := http.NewRequest(t.GetMethod(), t.GetURL(), bodyReader)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error creating HTTP request for task %s: %v", t.GetName(), err), "error", "task", task)
			return
		}

		for key, value := range t.GetHeaders() {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}

		if bodyReader != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		LogMessageToFile(fmt.Sprintf("HTTP Request for task %s: method=%s, url=%s", t.GetName(), t.GetMethod(), t.GetURL()), "debug", "task", task)

		resp, err := client.Do(req)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error executing HTTP task %s: %v", t.GetName(), err), "error", "task", task)
			return
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error reading HTTP response for task %s: %v", t.GetName(), err), "error", "task", task)
		} else {
			LogMessageToFile(fmt.Sprintf("HTTP task %s response status: %s, body: %s", t.GetName(), resp.Status, strings.TrimSpace(string(bodyBytes))), "info", "task", task)

			(&t.Execution).SetLastRanAt(now)

			if err := SaveTask(t, HTTP_TASK); err != nil {
				LogMessageToFile(fmt.Sprintf("failed to save updated http task: %s", err.Error()), "error", "task", task)
			}
		}

	default:
		LogMessageToFile(fmt.Sprintf("Unknown task type for task: %s", task.GetName()), "error", "app", nil)
	}
}

func executeTaskWithDependencies(task Task) {
	if before := task.GetRunBefore(); before != nil {
		if !before.GetExecution().IsEnabled {
			return
		}
		var depLastRan time.Time
		if t := before.GetExecution().GetLastRanAtTime(); t != nil {
			depLastRan = *t
		}
		depFirstRun, _ := GetFirstAndNextRun(time.Now(), depLastRan, before.GetExecution().Delay, before.GetExecution().Interval)
		now := time.Now()
		if now.Before(depFirstRun) {
			waitDuration := depFirstRun.Sub(now)
			LogMessageToFile(fmt.Sprintf("Waiting %s to execute dependency %s", waitDuration, before.GetName()), "info", "app", nil)
			time.Sleep(waitDuration)
		}
		executeTaskWithDependencies(before)
	}

	LogMessageToFile(fmt.Sprintf("Executing task: %s", task.GetName()), "info", "app", nil)
	ExecuteTask(task)

	if after := task.GetRunAfter(); after != nil {
		if !after.GetExecution().IsEnabled {
			return
		}

		var depLastRan time.Time
		if t := after.GetExecution().GetLastRanAtTime(); t != nil {
			depLastRan = *t
		}
		depFirstRun, _ := GetFirstAndNextRun(time.Now(), depLastRan, after.GetExecution().Delay, after.GetExecution().Interval)
		now := time.Now()
		if now.Before(depFirstRun) {
			waitDuration := depFirstRun.Sub(now)
			LogMessageToFile(fmt.Sprintf("Waiting %s to execute dependency %s", waitDuration, after.GetName()), "info", "app", nil)
			time.Sleep(waitDuration)
		}
		executeTaskWithDependencies(after)
	}
}

func calculateNextRun(start time.Time, i Interval) time.Time {
	return start.
		AddDate(i.Years, i.Months, i.Weeks*7+i.Days).
		Add(time.Hour * time.Duration(i.Hours)).
		Add(time.Minute * time.Duration(i.Minutes)).
		Add(time.Second * time.Duration(i.Seconds))
}

func scheduleTask(task Task) *ScheduledTask {
	now := time.Now()

	execution := task.GetExecution()
	lastRan := execution.GetLastRanAtTime()

	var firstRun time.Time
	if lastRan != nil && !lastRan.IsZero() {
		firstRun = calculateNextRun(*lastRan, execution.Interval)
		if now.Before(firstRun) {
			LogMessageToFile(fmt.Sprintf("Skipping task '%s' – next run at %s", task.GetName(), firstRun.Format(time.RFC3339)), "info", "app", nil)
			return &ScheduledTask{
				Task:     task,
				NextRun:  firstRun,
				Interval: execution.Interval,
			}
		} else {
			firstRun = now
		}
	} else {
		firstRun = calculateNextRun(now, execution.Delay)
	}

	return &ScheduledTask{
		Task:     task,
		NextRun:  firstRun,
		Interval: execution.Interval,
	}
}

func isZeroInterval(interval Interval) bool {
	return interval.Years == 0 && interval.Months == 0 && interval.Weeks == 0 &&
		interval.Days == 0 && interval.Hours == 0 && interval.Minutes == 0 && interval.Seconds == 0
}

func getShellCommand(command, shellType string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		if shellType == "powershell" {
			powershellType := "powershell"
			if _, err := exec.LookPath("pwsh"); err == nil {
				powershellType = "pwsh"
			}

			return exec.Command(
				powershellType,
				"-NoLogo", "-NoProfile", "-NonInteractive",
				"-Command", command,
			)
		} else {
			return exec.Command("cmd", "/C", command)
		}
	}

	return exec.Command("sh", "-c", command)
}
