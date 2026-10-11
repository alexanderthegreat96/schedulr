//go:build !windows
// +build !windows

package core

import "os/exec"

func setCmdShellCommandLine(cmd *exec.Cmd, command string) {
}
