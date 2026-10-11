package cmd

import (
	"fmt"
	"os"

	"github.com/alexanderthegreat96/schedulr/core"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [task_type] [task_name]",
	Short: "Runs a specified task",
	Long: `Runs a specified task, immediately and once.

Only the named task runs: its run_before / run_after dependencies are resolved
by the daemon, not here, so this is not a way to test a dependency chain.

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
			core.LogMessage(fmt.Sprintf("Unable to execute task: %s, Error: %s", taskName, err.Error()), "error")
			return
		}

		// success used to be reported unconditionally, so a failing task still
		// said it had worked and exited zero
		if err := core.ExecuteTask(task); err != nil {
			// the error already names the task
			core.LogMessage(fmt.Sprintf("Failed: %s", err.Error()), "error")
			core.LogMessage(fmt.Sprintf("Check its output in %s.", core.TASK_LOGS_DIR), "info")
			os.Exit(1)
		}

		core.LogMessage(fmt.Sprintf("Task executed successfully. Check its output in %s.", core.TASK_LOGS_DIR), "success")
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
