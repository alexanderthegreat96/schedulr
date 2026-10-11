//go:build windows
// +build windows

package core

import (
	"syscall"
)

func GetProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS,
		HideWindow:    true,
	}
}
