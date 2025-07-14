//go:build !windows
// +build !windows

package core

import "os/exec"

func PreventWindowHide(cmd *exec.Cmd) {
	return
}
