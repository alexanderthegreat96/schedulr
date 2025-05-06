package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [task_type] [task_name]",
	Short: "Runs a specified task",
	Long: `Runs a specified task.
Important: We use kebab-case to pascal-case, which means: my-task-name is converted to MyTaskName
Example: schedulr run shell my-task-name
`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()

		taskType, taskName := args[0], args[1]
		taskType, taskName = core.NormalizeTaskParams(taskType, taskName)
		task, err := core.GetTask(taskType, taskName)

		if err != nil {
			core.LogMessage(fmt.Sprintf("Unable to execute task: %s", err.Error()), "error")
			return
		}

		core.ExecuteTask(task)
		core.LogMessage("Task executed succesfuly!. Check it's output in app-logs.", "success")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
