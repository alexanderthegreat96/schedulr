//go:build windows
// +build windows

package core

func PreventWindowHide(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: false,
	}
}
