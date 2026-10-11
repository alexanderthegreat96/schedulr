package cmd

import (
	"fmt"
	"strings"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear-logs",
	Short: "Will wipe logs",
	Long: `
Wipes all generated log files.

Log files the running daemon still has open cannot be removed while it is up -
on Windows not at all - so those are reported and left in place rather than
aborting the whole wipe. Stop the daemon first to clear everything.
	`,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()

		dirs := map[string]string{
			core.APP_LOGS_DIR:  core.AppLogDirPath,
			core.TASK_LOGS_DIR: core.TasksLogDirPath,
		}

		total := 0
		var stillInUse []string

		for label, path := range dirs {
			// one unremovable file used to abort the whole command, leaving
			// the other directory untouched
			removed, inUse, err := core.ClearLogFiles(path)
			if err != nil {
				core.LogMessage(fmt.Sprintf("Issue reading logs from %s. Error: %s", path, err.Error()), "error")
				continue
			}

			total += removed

			for _, name := range inUse {
				stillInUse = append(stillInUse, fmt.Sprintf("%s/%s", label, name))
			}
		}

		core.LogMessage(fmt.Sprintf("Wiped %d log file(s) from %s and %s.", total, core.APP_LOGS_DIR, core.TASK_LOGS_DIR), "success")

		if len(stillInUse) > 0 {
			core.LogMessage(fmt.Sprintf(
				"Still in use by the running daemon, left in place: %s", strings.Join(stillInUse, ", ")), "warn")
			core.LogMessage("Stop the daemon and run this again to clear those too.", "info")
		}
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
