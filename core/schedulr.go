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
	"unicode/utf8"
)

func ExecuteTask(task Task) error {
	now := time.Now()

	switch t := task.(type) {
	case ShellTask:
		LogMessageToFile(fmt.Sprintf("Executing shell task: %s", t.GetName()), "info", "app", nil)

		if err := validateShellTask(t); err != nil {
			LogMessageToFile(fmt.Sprintf("Skipping shell task %s: %s", t.GetName(), err.Error()), "error", "task", task)
			return err
		}

		cmd := getShellCommand(t.GetCommand(), t.GetArgs(), t.GetShellType(), t.GetIsGui()) // os aware

		output, err := cmd.CombinedOutput()
		if err != nil {
			failure := strings.TrimSpace(string(output))
			if failure == "" {
				failure = "no output"
			}

			LogMessageToFile(fmt.Sprintf("Error executing shell task %s: %v, output: %s", t.GetName(), err, failure), "error", "task", task)

			return fmt.Errorf("shell task %s failed: %w (output: %s)", t.GetName(), err, failure)
		}

		taskOutput := strings.TrimSpace(string(output))
		if taskOutput == "" {
			taskOutput = "No content"
		}

		LogMessageToFile(fmt.Sprintf("Shell task %s output: %s\n", t.GetName(), taskOutput), "info", "task", task)

		if err := persistLastRanAt(SHELL_TASK, t.GetName(), now); err != nil {
			LogMessageToFile(fmt.Sprintf("failed to save updated shell task: %s", err.Error()), "error", "task", task)
		}

	case HttpTask:
		LogMessageToFile(fmt.Sprintf("Executing HTTP task: %s", t.GetName()), "info", "app", nil)

		method := t.GetMethod()
		if method == METHOD_UNAVAILABLE {
			reason := fmt.Sprintf("unsupported method %q", t.Method)
			if strings.TrimSpace(t.Method) == "" {
				reason = "no method is set in its config"
			}

			LogMessageToFile(fmt.Sprintf("Skipping HTTP task %s: %s", t.GetName(), reason), "error", "task", task)

			return fmt.Errorf("http task %s: %s", t.GetName(), reason)
		}

		client := &http.Client{Timeout: 10 * time.Second}

		var bodyReader io.Reader
		if len(t.GetBody()) > 0 && method != METHOD_GET && method != METHOD_HEAD {
			jsonData, err := json.Marshal(t.GetBody())
			if err != nil {
				LogMessageToFile(fmt.Sprintf("Error marshaling HTTP task body for task %s: %v", t.GetName(), err), "error", "task", task)
				return fmt.Errorf("http task %s: could not marshal its body: %w", t.GetName(), err)
			}
			bodyReader = bytes.NewReader(jsonData)
		}

		req, err := http.NewRequest(method, t.GetURL(), bodyReader)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error creating HTTP request for task %s: %v", t.GetName(), err), "error", "task", task)
			return fmt.Errorf("http task %s: could not build its request: %w", t.GetName(), err)
		}

		for key, value := range t.GetHeaders() {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}

		if bodyReader != nil && req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/json")
		}

		LogMessageToFile(fmt.Sprintf("HTTP Request for task %s: method=%s, url=%s", t.GetName(), method, t.GetURL()), "debug", "task", task)

		resp, err := client.Do(req)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error executing HTTP task %s: %v", t.GetName(), err), "error", "task", task)
			return fmt.Errorf("http task %s: request failed: %w", t.GetName(), err)
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Error reading HTTP response for task %s: %v", t.GetName(), err), "error", "task", task)
			return fmt.Errorf("http task %s: could not read the response: %w", t.GetName(), err)
		}

		LogMessageToFile(fmt.Sprintf("HTTP task %s response status: %s, body: %s",
			t.GetName(), resp.Status, describeResponseBody(bodyBytes, AppConfig().HttpLogBodyLimit)), "info", "task", task)

		if err := persistLastRanAt(HTTP_TASK, t.GetName(), now); err != nil {
			LogMessageToFile(fmt.Sprintf("failed to save updated http task: %s", err.Error()), "error", "task", task)
		}

	default:
		LogMessageToFile(fmt.Sprintf("Unknown task type for task: %s", task.GetName()), "error", "app", nil)

		return fmt.Errorf("unknown task type for task %s", task.GetName())
	}

	return nil
}

func executeTaskWithDependencies(task Task) {
	executeTaskTree(task, map[string]bool{})
}

func executeTaskTree(task Task, visited map[string]bool) {
	name := task.GetName()

	if visited[name] {
		LogMessageToFile(fmt.Sprintf("Skipping task %s: it already ran in this dependency chain (circular run_before/run_after)", name), "warn", "app", nil)
		return
	}
	visited[name] = true

	if before := task.GetRunBefore(); before != nil {
		runDependency(before, "run_before", visited)
	}

	LogMessageToFile(fmt.Sprintf("Executing task: %s", name), "info", "app", nil)

	// one task failing must not stop the rest of the chain
	_ = ExecuteTask(task)

	if after := task.GetRunAfter(); after != nil {
		runDependency(after, "run_after", visited)
	}
}

func runDependency(dep Task, kind string, visited map[string]bool) {
	if !dep.GetExecution().IsEnabled {
		LogMessageToFile(fmt.Sprintf("Skipping disabled %s dependency: %s", kind, dep.GetName()), "info", "app", nil)
		return
	}

	// only dispatch used to take this guard, so a dependency that also had a
	// schedule of its own - or that two chains both named - could run twice at
	// once. Skipped rather than waited for: in-flight covers queued tasks, so
	// waiting could leave every worker blocked on something nothing can run.
	if !claimInFlight(dep.GetName()) {
		LogMessageToFile(fmt.Sprintf(
			"Skipping %s dependency %s: it is already running. Mark it dependency_only so it has no schedule of its own.",
			kind, dep.GetName()), "warn", "app", nil)
		return
	}
	defer releaseInFlight(dep.GetName())

	if wait := time.Until(dependencyFirstRun(dep)); wait > 0 {
		maxWait := time.Duration(AppConfig().DependencyMaxWaitSeconds) * time.Second
		if wait > maxWait {
			LogMessageToFile(fmt.Sprintf(
				"Skipping %s dependency %s: not due for %s, which is over the %s dependency wait limit",
				kind, dep.GetName(), wait.Truncate(time.Second), maxWait), "warn", "app", nil)
			return
		}

		LogMessageToFile(fmt.Sprintf("Waiting %s to execute dependency %s", wait.Truncate(time.Second), dep.GetName()), "info", "app", nil)
		time.Sleep(wait)
	}

	executeTaskTree(dep, visited)
}

func dependencyFirstRun(dep Task) time.Time {
	execution := dep.GetExecution()

	var lastRan time.Time
	if t := execution.GetLastRanAtTime(); t != nil {
		lastRan = *t
	}

	firstRun, _ := GetFirstAndNextRun(time.Now(), lastRan, execution.Delay, execution.Interval)
	return firstRun
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

	if execution.HasCalendarSchedule() {
		firstRun, err := nextCalendarRun(now, execution)
		if err != nil {
			// not scheduling it is better than scheduling something that can
			// never fire, which would look identical to a task that works
			LogMessageToFile(fmt.Sprintf("Not scheduling %s: %s", task.GetName(), err.Error()), "error", "app", nil)
			return nil
		}

		if !isZeroInterval(execution.Interval) || !isZeroInterval(execution.Delay) {
			LogMessageToFile(fmt.Sprintf(
				"Task %s sets at alongside interval/delay; those are ignored in favour of the clock times", task.GetName()), "warn", "app", nil)
		}

		return &ScheduledTask{Task: task, NextRun: firstRun, Interval: execution.Interval}
	}

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

func describeResponseBody(body []byte, limit int) string {
	if limit <= 0 {
		return fmt.Sprintf("<%d bytes, body logging disabled>", len(body))
	}

	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) <= limit {
		return trimmed
	}

	// back off to a rune boundary
	cut := limit
	for cut > 0 && !utf8.RuneStart(trimmed[cut]) {
		cut--
	}

	return fmt.Sprintf("%s... <truncated, %d bytes total>", trimmed[:cut], len(trimmed))
}

func validateShellTask(task ShellTask) error {
	command := strings.TrimSpace(task.Command)

	if command == "" {
		return fmt.Errorf("no command is set in its config")
	}

	if len(task.Args) == 0 {
		return nil
	}

	if runtime.GOOS == "darwin" && task.IsGui {
		return nil
	}

	if _, err := exec.LookPath(command); err != nil {
		return fmt.Errorf(
			"args are set, so command must be a single executable rather than a shell line, but %q could not be resolved: %w",
			command, err)
	}

	return nil
}

func getShellCommand(command string, args []string, shellType string, isGui bool) *exec.Cmd {
	if len(args) > 0 {
		return getDirectCommand(command, args, isGui)
	}

	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		if shellType == "powershell" {
			powershellType := "powershell"
			if _, err := exec.LookPath("pwsh"); err == nil {
				powershellType = "pwsh"
			}

			args := []string{"-NoLogo", "-NoProfile"}
			if !isGui {
				args = append(args, "-NonInteractive")
			}
			args = append(args, "-Command", command)

			cmd = exec.Command(powershellType, args...)
		} else {
			cmd = exec.Command(cmdShellName)
			setCmdShellCommandLine(cmd, command)
		}

		applyWindowMode(cmd, isGui)
	} else if runtime.GOOS == "darwin" && isGui {
		cmd = exec.Command("open", "-a", command)
	} else {
		cmd = exec.Command(unixShell(shellType), "-c", command)
	}

	return cmd
}

func unixShell(shellType string) string {
	shellType = strings.TrimSpace(shellType)
	if shellType == "" {
		return "sh"
	}

	if _, err := exec.LookPath(shellType); err != nil {
		return "sh"
	}

	return shellType
}

func getDirectCommand(command string, args []string, isGui bool) *exec.Cmd {
	var cmd *exec.Cmd

	if runtime.GOOS == "darwin" && isGui {
		openArgs := append([]string{"-a", command, "--args"}, args...)
		cmd = exec.Command("open", openArgs...)
	} else {
		cmd = exec.Command(command, args...)
	}

	applyWindowMode(cmd, isGui)

	return cmd
}

func applyWindowMode(cmd *exec.Cmd, isGui bool) {
	if isGui {
		PreventWindowHide(cmd)
	} else {
		HideTaskWindow(cmd)
	}
}
