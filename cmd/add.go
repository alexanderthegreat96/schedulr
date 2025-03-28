package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [task_type] [task_name]",
	Short: "Create a new task configuration",
	Long: `
Create a new scheduled task configuration in JSON format.
Supported task types: shell, http

 - shell is a basic command
 - http is a task which sends a http request

Examples:
 - schedulr add shell "Backup Database"
 - schedulr add http "Ping API"  
	`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		core.InitLogger()

		taskType := args[0]
		taskName := args[1]

		result, err := core.CreateTask(taskName, taskType)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Error creating task: %s", err), "warning")
			return
		}

		core.LogMessage(result, "success")
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
