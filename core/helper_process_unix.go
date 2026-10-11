//go:build !windows
// +build !windows

package core

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func ProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}

	return errors.Is(err, syscall.EPERM)
}

func ProcessExecutable(pid int) (string, error) {
	if pid <= 0 {
		return "", errors.New("invalid pid")
	}

	path, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return "", nil
	}

	return path, nil
}
