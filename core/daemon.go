package core

import (
	"fmt"
	"os"
	"time"
)

func RunDaemon() {
	InitLogger()
	SetupGracefulShutdown()

	wipeLogsAfter := AppConfig().WipeLogDataInterval

	StartLogWiper(AppLogDirPath, time.Duration(wipeLogsAfter)*time.Second)
	StartLogWiper(TasksLogDirPath, time.Duration(wipeLogsAfter)*time.Second)

	LogMessage("Schedulr daemon running...", "info")

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

	shellTasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		err := LogMessageToFile(fmt.Sprintf("Error loading shell tasks: %v", err), "error", "app", nil)
		if err != nil {
			return err
		}
		return err
	}
	httpTasks, err := GetTasks(HTTP_TASK)
	if err != nil {
		err := LogMessageToFile(fmt.Sprintf("Error loading HTTP tasks: %v", err), "error", "app", nil)
		if err != nil {
			return err
		}
		return err
	}

	tasks := append(shellTasks, httpTasks...)

	scheduledTasks := make([]ScheduledTask, 0)
	for _, t := range tasks {
		// skip disabled tasks
		if !t.GetExecution().IsEnabled {
			continue
		}

		if st := scheduleTask(t); st != nil {
			scheduledTasks = append(scheduledTasks, *st)
		}
	}

	taskQueue := make(chan Task)

	for i := 0; i < AppConfig().WorkerCount; i++ {
		go func(id int) {
			for task := range taskQueue {
				err := LogMessageToFile(fmt.Sprintf("Worker %d executing %s", id, task.GetName()), "info", "app", nil)
				if err != nil {
					return
				}
				executeTaskWithDependencies(task)
			}
		}(i + 1)
	}

	for {
		now := time.Now()
		nextIndex := -1
		for i, st := range scheduledTasks {
			if nextIndex == -1 || st.NextRun.Before(scheduledTasks[nextIndex].NextRun) {
				nextIndex = i
			}
		}
		if nextIndex == -1 {
			time.Sleep(time.Second)
			continue
		}

		sleepDuration := scheduledTasks[nextIndex].NextRun.Sub(now)
		if sleepDuration > 0 {
			time.Sleep(sleepDuration)
		}

		taskQueue <- scheduledTasks[nextIndex].Task
		err := LogMessageToFile(fmt.Sprintf("Dispatched task: %s", scheduledTasks[nextIndex].Task.GetName()), "info", "app", nil)
		if err != nil {
			return err
		}
		if !isZeroInterval(scheduledTasks[nextIndex].Interval) {
			scheduledTasks[nextIndex].NextRun = calculateNextRun(time.Now(), scheduledTasks[nextIndex].Interval)
		} else {
			scheduledTasks = append(scheduledTasks[:nextIndex], scheduledTasks[nextIndex+1:]...)
		}
	}
}
