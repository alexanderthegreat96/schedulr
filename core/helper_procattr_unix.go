//go:build !windows
// +build !windows

package core

import (
	"syscall"
)

func GetProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setsid: true,
	}
}
