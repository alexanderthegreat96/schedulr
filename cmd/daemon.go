package cmd

import (
	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:    "daemon",
	Short:  "Runs the actual scheduler process in background",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		core.RunDaemon()
	},
}

func init() {
	rootCmd.AddCommand(daemonCmd)
}
