package cmd

import (
	"fmt"
	"runtime"

	"github.com/alexanderthegreat96/schedulr/core"

	"github.com/common-nighthawk/go-figure"
	"github.com/spf13/cobra"
)

var (
	shortDesc = fmt.Sprintf("A modern task scheduler for %s/%s without the complexity of crontab syntax.\n\n", runtime.GOOS, runtime.GOARCH)
	longDesc  = fmt.Sprintf(`Schedulr is a lightweight, flexible task runner that executes commands based on JSON configurations.
It works like crontab but without the need for complex syntax, allowing you to define and manage tasks with ease.

Running on: %s/%s.`, runtime.GOOS, runtime.GOARCH)
)

var rootCmd = &cobra.Command{
	Use:   "scheduler",
	Short: shortDesc,
	Long:  longDesc,
}

func Execute() {
	err := core.AutoSetup()
	if err != nil {
		fmt.Printf("There was an error trying to auto-setup all files and folders. Error: %s\n", err.Error())
	}

	core.InitLogger()
	schedulr := figure.NewColorFigure("Schedulr", "doom", "blue", false)
	schedulr.Print()
	fmt.Println()

	if core.AppConfig().DevMode == true {
		core.LogMessage("App is running in DEV mode.", "info")
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
	}
}
