package cmd

import (
	"time"

	"github.com/alexanderthegreat96/schedulr/core"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restarts the Schedulr daemon",
	Long:  `All it does is to restart the schedulr. Nothing more.`,
	Run: func(cmd *cobra.Command, args []string) {
		core.InitLogger()
		core.LogMessage("Restarting schedulr...", "info")

		stopCmd.Run(stopCmd, args)
		time.Sleep(1 * time.Second)
		startCmd.Run(startCmd, args)
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
