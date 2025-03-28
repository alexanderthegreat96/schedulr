package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Runs the actual scheduler process in background",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.SetupGracefulShutdown()

		core.StartLogWiper(core.AppLogFilePath, 60*time.Second)
		core.StartLogWiper(core.TasksLogFilePath, 60*time.Second)

		core.LogMessage("Schedulr daemon running...", "info")

		if err := core.RunSchedulerLoop(); err != nil {
			core.LogMessageToFile(fmt.Sprintf("Daemon crashed: %v", err), "error", "app")
			core.CloseLoggers()
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
