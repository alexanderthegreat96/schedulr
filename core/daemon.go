package core

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var (
	inFlightMu sync.Mutex
	inFlight   = make(map[string]bool)
)

func RunDaemon() {
	InitLogger()
	SetupGracefulShutdown()

	wipeLogsAfter := AppConfig().WipeLogDataInterval

	StartLogWiper(AppLogDirPath, time.Duration(wipeLogsAfter)*time.Second)
	StartLogWiper(TasksLogDirPath, time.Duration(wipeLogsAfter)*time.Second)

	LogMessage("Schedulr daemon running...", "info")

	cfg := AppConfig()
	cwd, _ := os.Getwd()
	LogMessageToFile(fmt.Sprintf(
		"Effective config [%s]: root=%s cwd=%s config=%s workers=%d reload=%ds dependency_max_wait=%ds log_wipe=%ds",
		cfg.Status(), RootPath, cwd, ConfigFilePath(),
		cfg.WorkerCount, cfg.ReloadIntervalSeconds, cfg.DependencyMaxWaitSeconds, cfg.WipeLogDataInterval),
		"info", "app", nil)

	if err := runSchedulerLoop(); err != nil {
		err := LogMessageToFile(fmt.Sprintf("Daemon crashed: %v", err), "error", "app", nil)
		if err != nil {
			return
		}
		CloseLoggers()
		os.Exit(1)
	}
}

func runSchedulerLoop() error {
	InitLogger()

	workerCount := AppConfig().WorkerCount
	if workerCount < 1 {
		LogMessageToFile(fmt.Sprintf("WORKER_COUNT is %d, falling back to 1 worker", workerCount), "warn", "app", nil)
		workerCount = 1
	}

	taskQueue := make(chan Task, workerCount)
	startWorkers(taskQueue, workerCount)

	scheduledTasks, err := loadSchedule(nil)
	if err != nil {
		return err
	}

	reloadEvery := time.Duration(AppConfig().ReloadIntervalSeconds) * time.Second
	if reloadEvery <= 0 {
		reloadEvery = time.Second
	}
	nextReload := time.Now().Add(reloadEvery)

	for {
		if !time.Now().Before(nextReload) {
			refreshed, err := loadSchedule(scheduledTasks)
			if err != nil {
				LogMessageToFile(fmt.Sprintf("Error reloading tasks, keeping the current schedule: %v", err), "error", "app", nil)
			} else {
				scheduledTasks = refreshed
			}
			nextReload = time.Now().Add(reloadEvery)
			continue
		}

		nextIndex := -1
		for i := range scheduledTasks {
			if nextIndex == -1 || scheduledTasks[i].NextRun.Before(scheduledTasks[nextIndex].NextRun) {
				nextIndex = i
			}
		}

		if nextIndex == -1 {
			sleepUntil(nextReload)
			continue
		}

		if scheduledTasks[nextIndex].NextRun.After(nextReload) {
			sleepUntil(nextReload)
			continue
		}

		sleepUntil(scheduledTasks[nextIndex].NextRun)

		dispatch(taskQueue, scheduledTasks[nextIndex].Task)

		if next, repeats := advanceNextRun(scheduledTasks[nextIndex], time.Now()); repeats {
			scheduledTasks[nextIndex].NextRun = next
		} else {
			scheduledTasks = append(scheduledTasks[:nextIndex], scheduledTasks[nextIndex+1:]...)
		}
	}
}

func loadSchedule(previous []ScheduledTask) ([]ScheduledTask, error) {
	shellTasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		LogMessageToFile(fmt.Sprintf("Error loading shell tasks: %v", err), "error", "app", nil)
		return nil, err
	}

	httpTasks, err := GetTasks(HTTP_TASK)
	if err != nil {
		LogMessageToFile(fmt.Sprintf("Error loading HTTP tasks: %v", err), "error", "app", nil)
		return nil, err
	}

	tracked := make(map[string]ScheduledTask, len(previous))
	for _, st := range previous {
		tracked[st.Task.GetName()] = st
	}

	tasks := make([]Task, 0, len(shellTasks)+len(httpTasks))
	tasks = append(tasks, shellTasks...)
	tasks = append(tasks, httpTasks...)

	schedule := make([]ScheduledTask, 0, len(tasks))
	for _, t := range tasks {
		execution := t.GetExecution()

		if !execution.IsEnabled {
			continue
		}

		// a dependency-only task has no schedule of its own; it runs when
		// something names it in run_before or run_after
		if execution.DependencyOnly {
			continue
		}

		// a task with no interval runs once; a calendar task has no interval by
		// design, so the one-shot rule must not claim it
		if !execution.HasCalendarSchedule() && isZeroInterval(execution.Interval) && execution.GetLastRanAtTime() != nil {
			continue
		}

		if existing, ok := tracked[t.GetName()]; ok {
			existing.Task = t
			existing.Interval = execution.Interval
			schedule = append(schedule, existing)
			continue
		}

		if st := scheduleTask(t); st != nil {
			schedule = append(schedule, *st)
		}
	}

	return schedule, nil
}

func startWorkers(taskQueue <-chan Task, workerCount int) {
	for i := 0; i < workerCount; i++ {
		go func(id int) {
			for task := range taskQueue {
				_ = LogMessageToFile(fmt.Sprintf("Worker %d executing %s", id, task.GetName()), "info", "app", nil)

				executeTaskWithDependencies(task)
				releaseInFlight(task.GetName())
			}
		}(i + 1)
	}
}

func dispatch(taskQueue chan<- Task, task Task) {
	name := task.GetName()

	if !claimInFlight(name) {
		LogMessageToFile(fmt.Sprintf("Skipping dispatch of %s: its previous run has not finished", name), "warn", "app", nil)
		return
	}

	select {
	case taskQueue <- task:
		LogMessageToFile(fmt.Sprintf("Dispatched task: %s", name), "info", "app", nil)
	default:
		releaseInFlight(name)
		LogMessageToFile(fmt.Sprintf("Skipping dispatch of %s: every worker is busy", name), "warn", "app", nil)
	}
}

func claimInFlight(name string) bool {
	inFlightMu.Lock()
	defer inFlightMu.Unlock()

	if inFlight[name] {
		return false
	}

	inFlight[name] = true
	return true
}

func releaseInFlight(name string) {
	inFlightMu.Lock()
	defer inFlightMu.Unlock()

	delete(inFlight, name)
}

// advanceNextRun returns the run time following a dispatch, and whether the
// task repeats at all - a task with no interval and no calendar schedule runs
// once and is dropped from the schedule.
func advanceNextRun(st ScheduledTask, now time.Time) (time.Time, bool) {
	execution := st.Task.GetExecution()

	if execution.HasCalendarSchedule() {
		// a missed clock time is simply missed, cron-style: the next match is
		// found from whichever is later, the slot we just ran or now
		from := st.NextRun
		if now.After(from) {
			from = now
		}

		next, err := nextCalendarRun(from, execution)
		if err != nil {
			LogMessageToFile(fmt.Sprintf("Dropping %s from the schedule: %s", st.Task.GetName(), err.Error()), "error", "app", nil)
			return time.Time{}, false
		}

		return next, true
	}

	if isZeroInterval(st.Interval) {
		return time.Time{}, false
	}

	return advanceIntervalRun(st.NextRun, st.Interval, now), true
}

// advanceIntervalRun advances from the time the run was scheduled for rather
// than from the dispatch itself, so a late dispatch does not shift the whole
// series, then steps forward whole intervals until it is in the future so
// falling behind cannot queue a burst of catch-up runs. Stepping rather than
// multiplying matters because month and day intervals are not fixed durations.
func advanceIntervalRun(scheduled time.Time, interval Interval, now time.Time) time.Time {
	if isZeroInterval(interval) {
		return scheduled
	}

	next := calculateNextRun(scheduled, interval)
	for !next.After(now) {
		next = calculateNextRun(next, interval)
	}

	return next
}

func sleepUntil(t time.Time) {
	if d := time.Until(t); d > 0 {
		time.Sleep(d)
	}
}
