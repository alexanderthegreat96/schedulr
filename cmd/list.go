package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [task_type]",
	Short: "Lists available tasks",
	Long: `Lists all tasks for the given type.
Supported types: shell, http

Example:
  schedulr list shell`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		taskType := strings.ToLower(args[0])
		core.InitLogger()

		tasks, err := core.GetTasks(taskType)
		if err != nil {
			core.LogMessage(err.Error(), "error")
			return
		}

		if len(tasks) == 0 {
			core.LogMessage("No tasks found.", "info")
			return
		}

		fmt.Printf("\nListing tasks for type: %s\n", strings.ToUpper(taskType))
		fmt.Println(strings.Repeat("=", 80))

		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

		for _, task := range tasks {
			name := task.GetName()
			exec := task.GetExecution()
			runBefore := task.GetRunBefore()
			runAfter := task.GetRunAfter()

			desc, nextRun := core.DescribeSchedule(exec, time.Now())

			switch {
			case !exec.IsEnabled:
				fmt.Fprintf(w, "Status:\tDISABLED\n")
			case exec.DependencyOnly:
				fmt.Fprintf(w, "Status:\tENABLED (dependency-only, no schedule of its own)\n")
			default:
				fmt.Fprintf(w, "Status:\tENABLED\n")
			}

			lastRan := exec.GetLastRanAtTime()

			lastRanFormatted := "never"
			if lastRan != nil {
				lastRanFormatted = exec.GetLastRanAtTime().Format("2006-01-02 15:04:05")
			}

			fmt.Fprintf(w, "Name:\t%s\n", name)
			fmt.Fprintf(w, "Config Path:\t%s\n", filepath.Join(core.TaskLocation, taskType, fmt.Sprintf("%s.json", name)))
			fmt.Fprintf(w, "Schedule:\t%s\n", desc)
			fmt.Fprintf(w, "Next Run:\t%s\n", nextRun.Format("2006-01-02 15:04:05"))
			fmt.Fprintf(w, "Last Ran:\t%s\n", lastRanFormatted)
			fmt.Fprintf(w, "Run Before:\t%s\n", dependencyName(runBefore))
			fmt.Fprintf(w, "Run After:\t%s\n", dependencyName(runAfter))

			switch taskType {
			case core.SHELL_TASK:
				command := core.DefaultValueIfNull(task.GetCommand(), "string")
				fmt.Fprintf(w, "Command:\t%s\n", command)

				// shown quoted so each argument is visibly discrete, and only
				// when set - an empty args list means the command goes through
				// a shell as it always has
				if args := task.GetArgs(); len(args) > 0 {
					fmt.Fprintf(w, "Args:\t%q\n", args)
					fmt.Fprintf(w, "Run Via:\tdirect exec (no shell)\n")
				} else {
					fmt.Fprintf(w, "Run Via:\tshell\n")
				}

				shellType := core.DefaultValueIfNull(task.GetShellType(), "string")
				fmt.Fprintf(w, "ShellType:\t%s\n", shellType)

				isGui := core.DefaultValueIfNull(task.GetIsGui(), "bool")
				fmt.Fprintf(w, "IsGUI:\t%t\n", isGui)
			case core.HTTP_TASK:
				method := core.DefaultValueIfNull(task.GetMethod(), "string")
				url := core.DefaultValueIfNull(task.GetURL(), "string")
				fmt.Fprintf(w, "Method:\t%s\n", method)
				fmt.Fprintf(w, "URL:\t%s\n", url)
			default:
				fmt.Fprintln(w, "Details:\tUnknown task type.")
			}

			fmt.Fprintf(w, "%s\n", strings.Repeat("-", 80))
		}

		w.Flush()
	},
}

// dependencyName renders a run_before / run_after entry. Passing the Task
// through core.DefaultValueIfNull(dep, "string") printed the whole marshalled
// struct, since the value is a Task and never matches the "string" case.
func dependencyName(dep core.Task) string {
	if dep == nil {
		return "no-value"
	}

	return dep.GetName()
}

func init() {
	rootCmd.AddCommand(listCmd)
}
