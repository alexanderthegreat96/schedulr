//go:build !windows
// +build !windows

package core

import "os"

func FindProcess(pid int) (process *os.Process, err error) {
	return nil, nil
}
