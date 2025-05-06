package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [log_type] [task_name]",
	Short: "Tail the latest log file in real time",
	Long: `Tail the most recently modified log file in real time.

Supported log types:
  - app: Application logs
  - task: Task logs -> taskName

Argument usage:
  - using kebab-case to pascal-case
  - ex: my-task -> MyTask

If no log type is specified, "app" is used by default.
Example:
  schedulr logs app
  schedulr logs task my-task
`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()

		logType := "app"
		taskName := "myTask"
		if len(args) > 0 {
			logType = strings.ToLower(args[0])
			if len(args) > 1 {
				taskName = args[1]
			}
		}

		var logFilePath string
		switch logType {
		case "app":
			path, err := core.GetLatestLogFile(core.AppLogDirPath, "*.log")
			if err != nil {
				core.LogMessage(fmt.Sprintf("Error finding latest app log file: %v\n", err), "error")
				os.Exit(1)
			}
			logFilePath = path
		case "task":
			_, nTask := core.NormalizeTaskParams("", taskName)
			task, err := core.FetchTaskByName(nTask)
			if err != nil {
				core.LogMessage(fmt.Sprint("Error fetching task: ", nTask), "error")
				os.Exit(1)
			}

			path, err := core.GetLatestLogFile(core.TasksLogDirPath, fmt.Sprintf("%s_*.log", task.GetName()))
			if err != nil {
				core.LogMessage(fmt.Sprintf("Error finding latest task log file: %v\n", err), "error")
				os.Exit(1)
			}
			logFilePath = path
		default:
			core.LogMessage(fmt.Sprintf("Unknown log type: %s\n", logType), "warn")
			os.Exit(1)
		}

		if err := core.TailLogFile(logFilePath); err != nil {
			core.LogMessage(fmt.Sprintf("Error tailing logs: %v\n", err), "error")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
