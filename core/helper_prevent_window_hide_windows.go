//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
)

func PreventWindowHide(cmd *exec.Cmd) {
	attr := procAttr(cmd)
	attr.HideWindow = false
	attr.CreationFlags &^= CREATE_NO_WINDOW
}

func HideTaskWindow(cmd *exec.Cmd) {
	attr := procAttr(cmd)
	attr.HideWindow = true
	attr.CreationFlags |= CREATE_NO_WINDOW
}

func procAttr(cmd *exec.Cmd) *syscall.SysProcAttr {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	return cmd.SysProcAttr
}
