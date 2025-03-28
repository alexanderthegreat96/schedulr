package core

import (
	"fmt"
	"time"
)

func RunSchedulerLoop() error {
	InitLogger()

	shellTasks, err := GetTasks(SHELL_TASK)
	if err != nil {
		LogMessageToFile(fmt.Sprintf("Error loading shell tasks: %v", err), "error", "app")
		return err
	}
	httpTasks, err := GetTasks(HTTP_TASK)
	if err != nil {
		LogMessageToFile(fmt.Sprintf("Error loading HTTP tasks: %v", err), "error", "app")
		return err
	}

	tasks := append(shellTasks, httpTasks...)

	scheduledTasks := make([]ScheduledTask, 0)
	for _, t := range tasks {
		scheduledTasks = append(scheduledTasks, scheduleTask(t))
	}

	taskQueue := make(chan Task)

	for i := 0; i < WORKER_COUNT; i++ {
		go func(id int) {
			for task := range taskQueue {
				LogMessageToFile(fmt.Sprintf("Worker %d executing %s", id, task.GetName()), "info", "app")
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
		LogMessageToFile(fmt.Sprintf("Dispatched task: %s", scheduledTasks[nextIndex].Task.GetName()), "info", "app")

		if !isZeroInterval(scheduledTasks[nextIndex].Interval) {
			scheduledTasks[nextIndex].NextRun = calculateNextRun(time.Now(), scheduledTasks[nextIndex].Interval)
		} else {
			scheduledTasks = append(scheduledTasks[:nextIndex], scheduledTasks[nextIndex+1:]...)
		}
	}
}
