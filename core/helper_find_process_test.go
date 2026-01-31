package core

import (
	"os"
	"runtime"
	"testing"
)

func TestFindProcessCurrentProcess(t *testing.T) {
	t.Parallel()

	pid := os.Getpid()

	process, err := FindProcess(pid)

	if runtime.GOOS == "windows" {
		if err != nil {
			t.Logf("FindProcess on Windows returned error: %v (may be expected)", err)
		}
		if process != nil && process.Pid != pid {
			t.Errorf("FindProcess returned process with PID %d, expected %d", process.Pid, pid)
		}
	} else {
		if err != nil {
			t.Errorf("FindProcess on Unix returned unexpected error: %v", err)
		}
		if process != nil {
			t.Errorf("FindProcess on Unix should return nil process, got %v", process)
		}
	}
}

func TestFindProcessInvalidPID(t *testing.T) {
	t.Parallel()

	invalidPID := 999999999

	process, err := FindProcess(invalidPID)

	if runtime.GOOS == "windows" {
		if err == nil {
			t.Error("FindProcess on Windows should return error for invalid PID")
		}
	} else {
		if err != nil {
			t.Errorf("FindProcess on Unix returned unexpected error: %v", err)
		}
		if process != nil {
			t.Errorf("FindProcess on Unix should return nil for any PID, got %v", process)
		}
	}
}

func TestFindProcessZeroPID(t *testing.T) {
	t.Parallel()

	process, err := FindProcess(0)

	if runtime.GOOS == "windows" {
		if process != nil && process.Pid == 0 {
			t.Logf("FindProcess on Windows returned PID 0 process")
		}
	} else {
		if err != nil {
			t.Errorf("FindProcess on Unix returned unexpected error: %v", err)
		}
		if process != nil {
			t.Errorf("FindProcess on Unix should return nil, got %v", process)
		}
	}
}

func TestFindProcessNegativePID(t *testing.T) {
	t.Parallel()

	process, err := FindProcess(-1)

	if runtime.GOOS == "windows" {
		if err == nil && process == nil {
			t.Logf("FindProcess on Windows handled negative PID gracefully")
		}
	} else {
		if process != nil {
			t.Errorf("FindProcess on Unix should return nil for negative PID, got %v", process)
		}
	}
}

func TestFindProcessReturnType(t *testing.T) {
	t.Parallel()

	pid := os.Getpid()
	process, err := FindProcess(pid)

	if runtime.GOOS == "windows" {
		if err == nil && process != nil {
			if process.Pid != pid {
				t.Errorf("Returned process PID %d doesn't match requested %d", process.Pid, pid)
			}
		}
	} else {
		if process != nil || err != nil {
			t.Errorf("Unix FindProcess should return (nil, nil), got (%v, %v)", process, err)
		}
	}
}
