package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [log_type]",
	Short: "Tail the latest log file in real time",
	Long: `Tail the most recently modified log file in real time.

Supported log types:
  - app: Application logs
  - task: Task logs

If no log type is specified, "app" is used by default.
Example:
  schedulr logs app`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()

		logType := "app"
		if len(args) > 0 {
			logType = strings.ToLower(args[0])
		}

		var logFilePath string
		switch logType {
		case "app":
			logDir := filepath.Dir(core.AppLogFilePath)
			path, err := core.GetLatestLogFile(logDir, "*.log")
			if err != nil {
				core.LogMessage(fmt.Sprintf("Error finding latest app log file: %v\n", err), "error")
				os.Exit(1)
			}
			logFilePath = path
		case "task":
			logDir := filepath.Dir(core.TasksLogFilePath)
			path, err := core.GetLatestLogFile(logDir, "*.log")
			if err != nil {
				core.LogMessage(fmt.Sprintf("Error finding latest task log file: %v\n", err), "error")
				os.Exit(1)
			}
			logFilePath = path
		default:
			core.LogMessage(fmt.Sprintf("Unknown log type: %s\n", logType), "warn")
			os.Exit(1)
		}

		if err := core.TailLogFile(logFilePath); err != nil {
			core.LogMessage(fmt.Sprintf("Error tailing logs: %v\n", err), "error")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
