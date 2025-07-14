//go:build windows
// +build windows

package core

func FindProcess(pid int) (process *os.Process, err error) {
	handle, err := syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil, err
	}
	syscall.CloseHandle(handle)

	process, err = os.FindProcess(pid)
	if err != nil {
		return nil, err
	}

	return process, nil
}
