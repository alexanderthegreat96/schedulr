//go:build !windows
// +build !windows

package core

import (
	"os/exec"
	"strings"
	"testing"
)

func TestUnixShellHonoursShellType(t *testing.T) {
	t.Parallel()

	for _, shell := range []string{"bash", "zsh", "dash", "ksh"} {
		if _, err := exec.LookPath(shell); err != nil {
			continue
		}

		if got := unixShell(shell); got != shell {
			t.Errorf("unixShell(%q) = %q, want %q", shell, got, shell)
		}
	}
}

func TestUnixShellFallsBackToSh(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"unset":               "",
		"whitespace":          "   ",
		"not installed":       "definitely-not-a-real-shell",
		"a windows shell":     "powershell",
		"another windows one": "cmd",
	}

	for name, shellType := range tests {
		t.Run(name, func(t *testing.T) {
			if shellType != "" && strings.TrimSpace(shellType) != "" {
				if _, err := exec.LookPath(shellType); err == nil {
					t.Skipf("%q is installed here, so no fallback is expected", shellType)
				}
			}

			if got := unixShell(shellType); got != "sh" {
				t.Errorf("unixShell(%q) = %q, want sh", shellType, got)
			}
		})
	}
}

func TestGetShellCommandUsesTheRequestedShell(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed here")
	}

	cmd := getShellCommand("echo hi", nil, "bash", false)

	if len(cmd.Args) < 3 {
		t.Fatalf("unexpected invocation: %v", cmd.Args)
	}
	if cmd.Args[1] != "-c" || cmd.Args[2] != "echo hi" {
		t.Errorf("expected `-c \"echo hi\"`, got %v", cmd.Args)
	}

	if path, _ := exec.LookPath("bash"); cmd.Path != path {
		t.Errorf("Path = %q, want %q", cmd.Path, path)
	}
}

func TestARequestedShellActuallyInterpretsTheCommand(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not installed here")
	}

	cmd := getShellCommand("echo ${BASH_VERSION:+bash-ran-this}", nil, "bash", false)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v (output %q)", err, out)
	}

	if len(out) == 0 || string(out) == "\n" {
		t.Errorf("expected bash to expand BASH_VERSION, got %q", out)
	}
}

func TestWindowHelpersAreNoOpsOnUnix(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("echo", "test")

	HideTaskWindow(cmd)
	if cmd.SysProcAttr != nil {
		t.Errorf("HideTaskWindow should not touch SysProcAttr on unix, got %v", cmd.SysProcAttr)
	}

	PreventWindowHide(cmd)
	if cmd.SysProcAttr != nil {
		t.Errorf("PreventWindowHide should not touch SysProcAttr on unix, got %v", cmd.SysProcAttr)
	}
}

func TestGetShellCommandUsesShOnUnix(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("echo hi", nil, "bash", false)

	if cmd.SysProcAttr != nil {
		t.Errorf("a non-GUI task needs no SysProcAttr on unix, got %v", cmd.SysProcAttr)
	}
	if len(cmd.Args) < 2 || cmd.Args[1] != "-c" {
		t.Errorf("expected an `sh -c` invocation, got %v", cmd.Args)
	}
}
