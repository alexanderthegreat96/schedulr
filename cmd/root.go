package cmd

import (
	"fmt"
	"os"
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

// commands whose stdout is meant to be consumed by another tool - command
// substitution, a pipe, an editor invocation. The banner and the dev-mode
// notice would corrupt that, so they are suppressed for these.
var quietCommands = map[string]bool{
	"task-path": true,
}

func wantsCleanStdout() bool {
	if len(os.Args) < 2 {
		return false
	}

	return quietCommands[os.Args[1]]
}

func Execute() {
	err := core.AutoSetup()
	if err != nil {
		fmt.Fprintf(os.Stderr, "There was an error trying to auto-setup all files and folders. Error: %s\n", err.Error())
	}

	core.InitLogger()

	if !wantsCleanStdout() {
		schedulr := figure.NewColorFigure("Schedulr", "doom", "blue", false)
		schedulr.Print()
		fmt.Println()

		if core.AppConfig().DevMode {
			core.LogMessage("App is running in DEV mode.", "info")
		}
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
