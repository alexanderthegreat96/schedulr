package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check if scheduler daemon is running",
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()

		if !core.PidFileExists() {
			core.LogMessage("Schdulr IS NOT running.", "warn")
			return
		}

		pid, err := core.ReadPidFile()

		if err != nil {
			core.LogMessage(fmt.Sprintf("Unable to read PID file: %s", err.Error()), "error")
			return
		}

		core.LogMessage(fmt.Sprintf("Schedulr daemon is running with process ID: %d", pid), "success")
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
