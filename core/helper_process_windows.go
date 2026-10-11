//go:build windows
// +build windows

package core

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	processQueryLimitedInformation = 0x1000
	processSynchronize             = 0x00100000
	waitTimeout                    = 0x00000102
	maxLongPath                    = 32768
)

var (
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
)

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, err := syscall.OpenProcess(processSynchronize|processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return errors.Is(err, syscall.ERROR_ACCESS_DENIED)
	}
	defer syscall.CloseHandle(handle)

	event, err := syscall.WaitForSingleObject(handle, 0)
	if err != nil {
		return true
	}

	return event == waitTimeout
}

func ProcessExecutable(pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("invalid pid")
	}

	handle, err := syscall.OpenProcess(processQueryLimitedInformation, false, uint32(pid))
	if err != nil {
		return "", err
	}
	defer syscall.CloseHandle(handle)

	buf := make([]uint16, maxLongPath)
	size := uint32(len(buf))

	ret, _, callErr := procQueryFullProcessImageName.Call(
		uintptr(handle),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if ret == 0 {
		return "", callErr
	}

	return syscall.UTF16ToString(buf[:size]), nil
}
