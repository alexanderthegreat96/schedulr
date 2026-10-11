//go:build windows
// +build windows

package core

import (
	"os/exec"
	"syscall"
	"testing"
)

func TestPreventWindowHideWindows(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("cmd", "/c", "echo test")

	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("On Windows, PreventWindowHide should set SysProcAttr")
	}

	if cmd.SysProcAttr.HideWindow {
		t.Error("PreventWindowHide should leave HideWindow false so the window shows")
	}
}

func TestPreventWindowHideWithExistingAttr(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("cmd", "/c", "echo test")

	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}

	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}

	if cmd.SysProcAttr.HideWindow {
		t.Error("PreventWindowHide should clear a pre-existing HideWindow")
	}
}

func TestPreventWindowHideMultipleCalls(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("cmd", "/c", "echo test")

	PreventWindowHide(cmd)
	PreventWindowHide(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should not be nil")
	}

	t.Logf("After multiple calls: %+v", cmd.SysProcAttr)
}
