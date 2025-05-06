package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable [task_type] [task_name]",
	Short: "Enables a scheduled task",
	Long: `
Will enable a scheduled task.
Supported task types: shell, http

 - shell is a basic command
 - http is a task which sends a http request

Examples:
 - schedulr enable shell backup-database -> enables: tasks/shell/BackupDatabase.json
 - schedulr enable http ping-api -> enables: tasks/http/PingAPI.json

Notice: we are using pascal-case.
	`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		core.InitLogger()

		taskType := args[0]
		taskName := args[1]

		taskType, taskName = core.NormalizeTaskParams(taskType, taskName)

		err := core.UpdateTaskStatus(taskType, taskName, true)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Error disabling task: %s", err), "warning")
			return
		}

		core.LogMessage(fmt.Sprintf("Task: %s of type %s was enabled.", taskType, taskName), "success")
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
}
