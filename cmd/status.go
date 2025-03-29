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

		if core.IsRunningUnderSystemd() {
			status, err := core.CheckSystemdStatus(core.AppConfig().ServiceName)
			if err != nil {
				core.LogMessage(fmt.Sprintf("Systemd check failed: %s", err.Error()), "error")
			}
			if status == "active" {
				core.LogMessage("Schedulr is running under systemd.", "success")
			} else {
				core.LogMessage("Schedulr is NOT running under systemd.", "warn")
			}
			return
		}

		if core.IsRunningUnderLaunchd() {
			status, err := core.CheckLaunchdStatus(core.AppConfig().ServiceName)
			if err != nil {
				core.LogMessage(fmt.Sprintf("Launchd check failed: %s", err.Error()), "error")
			}
			if status == "active" {
				core.LogMessage("Schedulr is running under launchd (macOS).", "success")
			} else {
				core.LogMessage("Schedulr is NOT running under launchd (macOS).", "warn")
			}
			return
		}

		if core.PidFileExists() {
			pid, err := core.ReadPidFile()
			if err != nil {
				core.LogMessage(fmt.Sprintf("Unable to read PID file: %s", err.Error()), "error")
				return
			}
			core.LogMessage(fmt.Sprintf("Schedulr daemon is running with PID: %d", pid), "success")
		} else {
			core.LogMessage("Schedulr IS NOT running (no PID file found).", "warn")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
