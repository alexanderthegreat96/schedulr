package cmd

import (
	"fmt"
	"os"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var debugPathsCmd = &cobra.Command{
	Use:   "debug-paths",
	Short: "Will list all the paths currently used by Schedulr",
	Long: `
Lists all the paths used by Schedulr:
 - logs
 - tasks
 - root path
	`,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		execPath, _ := os.Executable()

		core.LogMessage("Listing available paths", "info")
		core.LogMessage(fmt.Sprintf("Root Path: %s", core.RootPath), "info")
		core.LogMessage(fmt.Sprintf("Executable Path: %s", execPath), "info")
		core.LogMessage(fmt.Sprintf("Pid File Path: %s", core.PidFilePath), "info")
		core.LogMessage(fmt.Sprintf("App Log File Path: %s", core.AppLogFilePath), "info")
		core.LogMessage(fmt.Sprintf("App Log Dir Path: %s", core.AppLogDirPath), "info")
		core.LogMessage(fmt.Sprintf("Task Log Dir Path: %s", core.TasksLogDirPath), "info")
		core.LogMessage(fmt.Sprintf("Task Dir Path: %s", core.TaskLocation), "info")
	},
}

func init() {
	rootCmd.AddCommand(debugPathsCmd)
}
