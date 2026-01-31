package core

import (
	"os/exec"
	"runtime"
	"syscall"
	"testing"
)

func TestPreventWindowHideUnix(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test, skipping on Windows")
	}

	cmd := exec.Command("echo", "test")

	PreventWindowHide(cmd)

	if cmd.SysProcAttr != nil {
		t.Errorf("On Unix, PreventWindowHide should not modify SysProcAttr, got %v", cmd.SysProcAttr)
	}
}

func TestPreventWindowHideWindows(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test, skipping on non-Windows")
	}

	cmd := exec.Command("cmd", "/c", "echo test")

	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("On Windows, PreventWindowHide should set SysProcAttr")
	}

	t.Logf("On Windows, SysProcAttr was set: %+v", cmd.SysProcAttr)
}

func TestPreventWindowHideWithExistingAttr(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	cmd := exec.Command("cmd", "/c", "echo test")

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}
	t.Logf("After PreventWindowHide: %+v", cmd.SysProcAttr)
}

func TestPreventWindowHideMultipleCalls(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	cmd := exec.Command("cmd", "/c", "echo test")

	PreventWindowHide(cmd)
	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}

	t.Logf("After multiple calls: %+v", cmd.SysProcAttr)
}

func TestPreventWindowHideDocumentation(t *testing.T) {
	t.Parallel()

	t.Logf("PreventWindowHide is designed to modify cmd.SysProcAttr")
	t.Logf("On Unix: It's a no-op (return immediately)")
	t.Logf("On Windows: It sets HideWindow to false to show the window")
}
