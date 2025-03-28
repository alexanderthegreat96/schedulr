package cmd

import (
	"fmt"
	"syscall"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the scheduler daemon",
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Daemon shutdown comenced, please wait...", "info")

		pid, err := core.ReadPidFile()
		if err != nil {
			core.LogMessage(err.Error(), "error")
			return
		}

		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to stop schedulr daemon. Error: %s", err.Error()), "error")
			return
		}

		if err := core.DeletePidFile(); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to delete pid file. Error: %s", err.Error()), "error")
			return
		}

		core.LogMessage("Schedulr daemon stopped.", "success")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
