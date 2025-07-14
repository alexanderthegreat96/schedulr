//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
)

func PreventWindowHide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: false,
	}
}
