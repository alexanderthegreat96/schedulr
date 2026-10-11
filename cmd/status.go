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

		pid, running := core.DaemonRunning()

		switch {
		case running:
			core.LogMessage(fmt.Sprintf("Schedulr daemon is running with PID: %d", pid), "success")
		case core.PidFileExists():
			// the file outliving the process is the normal aftermath of a crash
			// or a reboot, so say that rather than claiming it is running
			core.LogMessage(fmt.Sprintf(
				"Schedulr IS NOT running. A stale PID file points at PID %d; it will be cleared on the next start.", pid), "warn")
		default:
			core.LogMessage("Schedulr IS NOT running (no PID file found).", "warn")
		}
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
