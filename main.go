package main

import (
	"runtime"

	"github.com/alexanderthegreat96/schedulr/cmd"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	cmd.Execute()
}
