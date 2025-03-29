package cmd

import (
	"fmt"
	"os"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the scheduler daemon",
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Daemon shutdown comenced, please wait...", "info")

		if core.IsRunningUnderSystemd() {
			core.LogMessage("Detected systemd — stopping via systemctl", "info")
			err := core.KillSystemDService()
			if err != nil {
				core.LogMessage(fmt.Sprintf("Failed to stop systemd service: %s", err.Error()), "error")
			} else {
				core.LogMessage("Schedulr stopped via systemd.", "success")
			}
			return
		}

		if core.IsRunningUnderLaunchd() {
			core.LogMessage("Detected launchd — unloading via launchctl", "info")
			err := core.KillLaunchDService()
			if err != nil {
				core.LogMessage(fmt.Sprintf("Failed to unload launchd service: %s", err.Error()), "error")
			} else {
				core.LogMessage("Schedulr stopped via launchd.", "success")
			}
			return
		}

		if !core.PidFileExists() {
			core.LogMessage("Schedulr is not running (no PID file found).", "warn")
			return
		}

		pid, err := core.ReadPidFile()
		if err != nil {
			core.LogMessage(fmt.Sprintf("Failed to read PID file: %s", err.Error()), "error")
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Could not find process: %s", err.Error()), "error")
			return
		}

		if err := process.Kill(); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to kill process: %s", err.Error()), "error")
		} else {
			core.LogMessage(fmt.Sprintf("Killed process with PID %d", pid), "success")
			core.DeletePidFile()
		}

		core.LogMessage("Schedulr daemon stopped.", "success")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
