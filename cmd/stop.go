package cmd

import (
	"fmt"
	"os"
	"runtime"
	"syscall"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var process *os.Process
var err error
var forceFlag bool

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the scheduler daemon",
	Long:  "Stops the schedulr daemon. You may provide --force to wipe the PID file in case you lost it or something happened and that process does not exist anymore.",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Daemon shutdown comenced, please wait...", "info")

		if forceFlag {
			err = core.DeletePidFile()
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

		pid, err := core.ReadPidFile()
		if err != nil {
			core.LogMessage(fmt.Sprintf("Failed to read PID file: %s", err.Error()), "error")
			return
		}

		if runtime.GOOS == "windows" {
			handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
			if err != nil {
				core.LogMessage(fmt.Sprintf("OpenProcess failed: %s", err.Error()), "error")
				return
			}
			syscall.CloseHandle(handle)

			process, err = os.FindProcess(pid)
			if err != nil {
				core.LogMessage(fmt.Sprintf("os.FindProcess failed: %s", err.Error()), "error")
				return
			}
		} else {
			process, err = os.FindProcess(pid)
			if err != nil || process.Signal(syscall.Signal(0)) != nil {
				core.LogMessage(fmt.Sprintf("Process check failed: %v", err), "error")
				return
			}
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
	stopCmd.Flags().BoolVar(&forceFlag, "force", false, "Forcefully remove the PID file even if the process doesn't exist.")
	rootCmd.AddCommand(stopCmd)
}
