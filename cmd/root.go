package cmd

import (
	"fmt"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/common-nighthawk/go-figure"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: "A modern task scheduler without the complexity of crontab syntax.",
	Long: `Scheduler is a lightweight, flexible task runner that executes commands based on JSON configurations.
It works like crontab but without the need for complex syntax, allowing you to define and manage tasks with ease.`,
}

func Execute() {
	err := core.AutoSetup()
	if err != nil {
		fmt.Printf("There was an error trying to auto-setup all files and folders. Error: %s\n", err.Error())
	}

	schedulr := figure.NewColorFigure("Schedulr", "doom", "blue", false)
	schedulr.Print()
	fmt.Println()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
