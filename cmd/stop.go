package cmd

import (
	"fmt"
	"os"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var forceFlag bool

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the scheduler daemon",
	Long: `Stops the schedulr daemon.

A PID file whose process is no longer running is detected and removed
automatically, so --force is rarely needed. It remains as an escape hatch: it
wipes the PID file without looking at the process at all.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Daemon shutdown comenced, please wait...", "info")

		if forceFlag {
			err := core.DeletePidFile()
			if err != nil {
				core.LogMessage(fmt.Sprintf("Unable to wipe the PID file: %s", err.Error()), "error")
			} else {
				core.LogMessage("PID file wiped forcefully.", "success")
			}
			return
		}

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

		pid, running := core.DaemonRunning()
		if !running {
			if stale, removed := core.ClearStalePidFile(); removed {
				core.LogMessage(fmt.Sprintf("Schedulr was not running; cleared the stale PID file for PID %d.", stale), "success")
			} else {
				core.LogMessage("Schedulr is not running.", "warn")
			}
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			core.LogMessage(fmt.Sprintf("Failed to find process %d: %s", pid, err.Error()), "error")
			return
		}

		if err := process.Kill(); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to kill process %d: %s", pid, err.Error()), "error")
			return
		}

		core.LogMessage(fmt.Sprintf("Killed process with PID %d", pid), "success")

		if err := core.DeletePidFile(); err != nil {
			core.LogMessage(fmt.Sprintf("Killed the daemon but could not remove the PID file: %s", err.Error()), "warn")
		}

		core.LogMessage("Schedulr daemon stopped.", "success")
	},
}

func init() {
	stopCmd.Flags().BoolVar(&forceFlag, "force", false, "Forcefully remove the PID file even if the process doesn't exist.")
	rootCmd.AddCommand(stopCmd)
}
