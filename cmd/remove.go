package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [task_type] [task_name]",
	Short: "Delete a scheduled task",
	Long: `
Will wipe the task configuration file
Supported task types: shell, http

 - shell is a basic command
 - http is a task which sends a http request

Examples:
 - schedulr remove shell backup-database -> wipes: tasks/shell/BackupDatabase.json
 - schedulr remove http ping-api -> wipes: tasks/http/PingAPI.json

Notice: we are using pascal-case.
	`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		core.InitLogger()

		taskType := args[0]
		taskName := args[1]

		taskType, taskName = core.NormalizeTaskParams(taskType, taskName)

		err := core.DeleteTask(taskType, taskName)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Error deleting task: %s", err), "warning")
			return
		}

		core.LogMessage(fmt.Sprintf("Task: %s of type %s was deleted.", taskName, taskType), "success")
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
