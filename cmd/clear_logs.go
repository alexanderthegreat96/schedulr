package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var clearCmd = &cobra.Command{
	Use:   "clear-logs",
	Short: "Will wipe logs",
	Long: `
Wipes all generated log files.
	`,
	Run: func(cmd *cobra.Command, args []string) {

		core.InitLogger()

		err := core.DeleteLogFiles(core.APP_LOGS_DIR)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Issue deleting logs from %s. Error: %s", core.APP_LOGS_DIR, err.Error()), "error")
			return
		}

		err = core.DeleteLogFiles(core.TASK_LOGS_DIR)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Issue deleting logs from %s. Error: %s", core.TASK_LOGS_DIR, err.Error()), "error")
			return
		}

		core.LogMessage(fmt.Sprintf("Wiped logs from both: %s and %s", core.APP_LOGS_DIR, core.TASK_LOGS_DIR), "success")
	},
}

func init() {
	rootCmd.AddCommand(clearCmd)
}
