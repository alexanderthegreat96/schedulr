//go:build windows
// +build windows

package core

import (
	"os/exec"
)

const (
	cmdShellFlags = "/S /C"
)

func setCmdShellCommandLine(cmd *exec.Cmd, command string) {
	attr := procAttr(cmd)

	// CreateProcess takes the first token as argv[0], so the shell itself has
	// to lead the line
	shell := cmdShellName
	if path, err := exec.LookPath(cmdShellName); err == nil {
		shell = path
	}

	attr.CmdLine = `"` + shell + `" ` + cmdShellFlags + ` "` + command + `"`
}
