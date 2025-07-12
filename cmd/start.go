package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the scheduler daemon",
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Starting daemon, please wait...", "info")

		if core.PidFileExists() {
			core.LogMessage("Daemon already running. Canceling.", "warn")
			return
		}

		// support for systemd / launchd only if schedulr service usage is enabled
		if core.AppConfig().EnableSchedulrService && core.IsManagedByInitSystem() {
			core.LogMessage("Detected system-managed environment — running in foreground", "info")
			core.RunDaemon()
			return
		}

		// fallback to schedulr service and start the daemon directly
		execFile, err := os.Executable()
		if err != nil {
			core.LogMessage(fmt.Sprintf("Could not get executable path: %v", err), "error")
			return
		}

		process := exec.Command(execFile, "daemon")

		process.Stdout = nil
		process.Stderr = nil
		process.Stdin = nil
		process.SysProcAttr = core.GetProcAttr()

		if err := process.Start(); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to start daemon: %v", err), "error")
			return
		}

		if err := core.CreatePidFile(process.Process.Pid); err != nil {
			core.LogMessage(fmt.Sprintf("Failed to write PID file: %v", err), "error")
			return
		}

		core.LogMessage(fmt.Sprintf("Schedulr daemon started with PID %d", process.Process.Pid), "success")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
