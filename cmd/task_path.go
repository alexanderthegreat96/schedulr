package cmd

import (
	"fmt"
	"os"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var taskPathCmd = &cobra.Command{
	Use:   "task-path [task_type] [task_name]",
	Short: "Prints the full path to a task's JSON config file",
	Long: `
Prints the absolute path of the JSON file backing a task, and nothing else, so
it can be fed straight to an editor or any other tool.

Supported task types: shell, http

Important: we use kebab-case to pascal-case, which means my-task-name resolves
to MyTaskName.json

Examples:
 - schedulr task-path shell my-task-name
 - vim $(schedulr task-path shell backup-database)
 - code $(schedulr task-path http ping-api)

Exits with a non-zero status and writes to stderr if the task does not exist,
so it fails loudly inside command substitution.
	`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		taskType, taskName := core.NormalizeTaskParams(args[0], args[1])

		path, err := core.TaskPath(taskType, taskName)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		fmt.Println(path)
	},
}

func init() {
	rootCmd.AddCommand(taskPathCmd)
}
