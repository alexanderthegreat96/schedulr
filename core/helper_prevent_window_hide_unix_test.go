//go:build !windows
// +build !windows

package core

import (
	"os/exec"
	"testing"
)

func TestPreventWindowHideUnix(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("echo", "test")

	PreventWindowHide(cmd)

	if cmd.SysProcAttr != nil {
		t.Errorf("On Unix, PreventWindowHide should not modify SysProcAttr, got %v", cmd.SysProcAttr)
	}
}
