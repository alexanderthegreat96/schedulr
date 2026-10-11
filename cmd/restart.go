package cmd

import (
	"fmt"
	"time"

	"github.com/alexanderthegreat96/schedulr/core"
	"github.com/spf13/cobra"
)

const restartGracePeriod = 10 * time.Second

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts the Schedulr daemon",
	Long:  `All it does is to restart the schedulr. Nothing more.`,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Restarting schedulr...", "info")

		// the pid has to be captured before stopping: stop removes the pid
		// file, so afterwards there is nothing left to wait on
		pid, running := core.DaemonRunning()

		stopCmd.Run(stopCmd, args)

		// this used to be a flat one second sleep. A kill only asks the process
		// to die, so on a slow shutdown start would spawn a second daemon while
		// the first was still going.
		if running {
			if !core.WaitForProcessToExit(pid, restartGracePeriod) {
				core.LogMessage(fmt.Sprintf(
					"PID %d is still running after %s; not starting a second daemon.", pid, restartGracePeriod), "error")
				return
			}

			core.LogMessage(fmt.Sprintf("PID %d has exited.", pid), "info")
		}

		startCmd.Run(startCmd, args)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
