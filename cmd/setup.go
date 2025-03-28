package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Creates necessary files and folders for schedulr.",
	Long: `
This command will create all files and folders necessary for the schedulr to run.
This includes:
 - app-logs
 - task-logs
 - tasks/shell
 - tasks/http
 - schedulr.config 
	`,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		err := core.AutoSetup()
		if err != nil {
			core.LogMessage(fmt.Sprintf("Setup error: %s", err), "error")
			return
		}

		core.LogMessage("Setup complete.", "success")
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
