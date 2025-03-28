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
	switch t := task.(type) {
	case ShellTask:

		LogMessageToFile(fmt.Sprintf("Executing shell task: %s", t.GetName()), "info", "app")
		cmd := getShellCommand(t.GetCommand()) // os aware

		output, err := cmd.CombinedOutput()
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error executing shell task %s: %v", t.GetName(), err), "error", "task")
		} else {
			LogMessageToFile(fmt.Sprintf("Shell task %s output: %s", t.GetName(), strings.TrimSpace(string(output))), "info", "task")
		}

	case HttpTask:
		LogMessageToFile(fmt.Sprintf("Executing HTTP task: %s", t.GetName()), "info", "app")
		client := &http.Client{Timeout: 10 * time.Second}

		var bodyReader io.Reader
		if len(t.GetBody()) > 0 && !strings.EqualFold(t.GetMethod(), "GET") && !strings.EqualFold(t.GetMethod(), "HEAD") {
			jsonData, err := json.Marshal(t.GetBody())
			if err != nil {
				LogMessageToFile(fmt.Sprintf("Error marshaling HTTP task body for task %s: %v", t.GetName(), err), "error", "task")
				return
			}
			bodyReader = bytes.NewReader(jsonData)
		}

		req, err := http.NewRequest(t.GetMethod(), t.GetURL(), bodyReader)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error creating HTTP request for task %s: %v", t.GetName(), err), "error", "task")
			return
		}

		for key, value := range t.GetHeaders() {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}

		if bodyReader != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		LogMessageToFile(fmt.Sprintf("HTTP Request for task %s: method=%s, url=%s", t.GetName(), t.GetMethod(), t.GetURL()), "debug", "task")

		resp, err := client.Do(req)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error executing HTTP task %s: %v", t.GetName(), err), "error", "task")
			return
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error reading HTTP response for task %s: %v", t.GetName(), err), "error", "task")
		} else {
			LogMessageToFile(fmt.Sprintf("HTTP task %s response status: %s, body: %s", t.GetName(), resp.Status, strings.TrimSpace(string(bodyBytes))), "info", "task")
		}

	default:
		LogMessageToFile(fmt.Sprintf("Unknown task type for task: %s", task.GetName()), "error", "app")
	}
}

func executeTaskWithDependencies(task Task) {

	if before := task.GetRunBefore(); before != nil {
		depFirstRun, _ := GetFirstAndNextRun(time.Now(), before.GetExecution().Delay, before.GetExecution().Interval)
		now := time.Now()
		if now.Before(depFirstRun) {
			waitDuration := depFirstRun.Sub(now)
			LogMessageToFile(fmt.Sprintf("Waiting %s to execute dependency %s", waitDuration, before.GetName()), "info", "app")
			time.Sleep(waitDuration)
		}
		executeTaskWithDependencies(before)
	}

	LogMessageToFile(fmt.Sprintf("Executing task: %s", task.GetName()), "info", "app")
	ExecuteTask(task)

	if after := task.GetRunAfter(); after != nil {
		depFirstRun, _ := GetFirstAndNextRun(time.Now(), after.GetExecution().Delay, after.GetExecution().Interval)
		now := time.Now()
		if now.Before(depFirstRun) {
			waitDuration := depFirstRun.Sub(now)
			LogMessageToFile(fmt.Sprintf("Waiting %s to execute dependency %s", waitDuration, after.GetName()), "info", "app")
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

func scheduleTask(task Task) ScheduledTask {
	firstRun, _ := GetFirstAndNextRun(time.Now(), task.GetExecution().Delay, task.GetExecution().Interval)
	return ScheduledTask{
		Task:     task,
		NextRun:  firstRun,
		Interval: task.GetExecution().Interval,
	}
}

func isZeroInterval(interval Interval) bool {
	return interval.Years == 0 && interval.Months == 0 && interval.Weeks == 0 &&
		interval.Days == 0 && interval.Hours == 0 && interval.Minutes == 0 && interval.Seconds == 0
}

func getShellCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}
	return exec.Command("sh", "-c", command)
}
