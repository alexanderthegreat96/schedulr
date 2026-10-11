//go:build windows
// +build windows

package core

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetShellCommandHidesWindowForNonGuiTask(t *testing.T) {
	t.Parallel()

	for _, shellType := range []string{"cmd", "powershell", ""} {
		cmd := getShellCommand("echo hi", nil, shellType, false)

		if cmd.SysProcAttr == nil {
			t.Fatalf("shell_type %q: SysProcAttr should be set so no window appears", shellType)
		}
		if !cmd.SysProcAttr.HideWindow {
			t.Errorf("shell_type %q: HideWindow should be true for a non-GUI task", shellType)
		}
		if cmd.SysProcAttr.CreationFlags&CREATE_NO_WINDOW == 0 {
			t.Errorf("shell_type %q: CREATE_NO_WINDOW should be set for a non-GUI task", shellType)
		}
	}
}

func TestGetShellCommandShowsWindowForGuiTask(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("notepad", nil, "cmd", true)

	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr should be set for a GUI task")
	}
	if cmd.SysProcAttr.HideWindow {
		t.Error("HideWindow should be false for a GUI task so its window shows")
	}
}

func TestHideTaskWindowOverridesAnExistingAttr(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand("echo hi", nil, "cmd", true)
	if cmd.SysProcAttr.HideWindow {
		t.Fatal("precondition: a GUI task should start visible")
	}

	HideTaskWindow(cmd)

	if !cmd.SysProcAttr.HideWindow {
		t.Error("HideTaskWindow should hide a command that was previously visible")
	}
}

func TestCmdShellCommandLineIsVerbatim(t *testing.T) {
	t.Parallel()

	cmd := getShellCommand(`"C:\Program Files\app.exe" --flag "a b"`, nil, "cmd", false)

	if cmd.SysProcAttr == nil || cmd.SysProcAttr.CmdLine == "" {
		t.Fatal("the cmd shell path should set a verbatim CmdLine")
	}

	line := cmd.SysProcAttr.CmdLine

	if !strings.Contains(line, "/S /C") {
		t.Errorf("CmdLine should use /S /C, got %q", line)
	}
	if !strings.Contains(line, `"C:\Program Files\app.exe" --flag "a b"`) {
		t.Errorf("the command should appear unescaped, got %q", line)
	}
	if strings.Contains(line, `\"`) {
		t.Errorf("CmdLine must not contain MSVC-style escaped quotes, got %q", line)
	}
	if !strings.HasSuffix(line, `"`) {
		t.Errorf("the command should be wrapped in an outer quote pair, got %q", line)
	}
}

// Setting CmdLine and then applying the window mode must not clobber either.
func TestCmdShellKeepsBothCmdLineAndWindowFlags(t *testing.T) {
	t.Parallel()

	hidden := getShellCommand("echo hi", nil, "cmd", false)
	if hidden.SysProcAttr.CmdLine == "" {
		t.Error("CmdLine was lost when the window mode was applied")
	}
	if !hidden.SysProcAttr.HideWindow {
		t.Error("a non-GUI task should still be hidden")
	}
	if hidden.SysProcAttr.CreationFlags&CREATE_NO_WINDOW == 0 {
		t.Error("CREATE_NO_WINDOW should still be set")
	}

	shown := getShellCommand("echo hi", nil, "cmd", true)
	if shown.SysProcAttr.CmdLine == "" {
		t.Error("CmdLine was lost for a GUI task")
	}
	if shown.SysProcAttr.HideWindow {
		t.Error("a GUI task should not be hidden")
	}
	if shown.SysProcAttr.CreationFlags&CREATE_NO_WINDOW != 0 {
		t.Error("CREATE_NO_WINDOW should be cleared for a GUI task")
	}
}

// The real thing: quote a program whose path contains spaces and check every
// argument arrives intact.
func TestCmdShellRunsAQuotedPathWithSpaces(t *testing.T) {
	exe := copyTestBinaryToSpacedPath(t)

	// in shell mode the caller does the quoting, so each argument carrying a
	// space gets its own pair. An unquoted & would be a cmd separator.
	want := []string{"two words", "also spaced"}

	parts := []string{`"` + exe + `"`, "-test.run=TestArgsHelperProcess", "--"}
	for _, arg := range want {
		parts = append(parts, `"`+arg+`"`)
	}

	cmd := getShellCommand(strings.Join(parts, " "), nil, "cmd", false)
	cmd.Env = append(os.Environ(), argsHelperEnv+"=1")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\noutput:\n%s", err, out)
	}

	text := string(out)
	for _, arg := range want {
		if !strings.Contains(text, "ARG["+arg+"]") {
			t.Errorf("argument %q did not arrive intact, output:\n%s", arg, text)
		}
	}
}

// Shell features are the whole reason this path exists, so they must survive
// the switch to a verbatim command line.
func TestCmdShellStillSupportsShellFeatures(t *testing.T) {
	tests := map[string]struct {
		command string
		expect  string
	}{
		"chaining":    {"echo first && echo second", "second"},
		"pipe":        {"echo haystack | findstr haystack", "haystack"},
		"env var":     {"echo %SCHEDULR_PROBE%", "probe-value"},
		"quoted echo": {`echo "quoted text"`, "quoted text"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := getShellCommand(tc.command, nil, "cmd", false)
			cmd.Env = append(os.Environ(), "SCHEDULR_PROBE=probe-value")

			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%q failed: %v\noutput:\n%s", tc.command, err, out)
			}

			if !strings.Contains(string(out), tc.expect) {
				t.Errorf("%q output %q, want it to contain %q", tc.command, out, tc.expect)
			}
		})
	}
}

func TestCmdShellRedirectsToAFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "out.txt")

	cmd := getShellCommand(`echo redirected > "`+target+`"`, nil, "cmd", false)

	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("redirection failed: %v\noutput:\n%s", err, out)
	}

	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("the redirect target was not written: %v", err)
	}
	if !strings.Contains(string(written), "redirected") {
		t.Errorf("target contains %q", written)
	}
}

// copyTestBinaryToSpacedPath puts this test binary behind a path with spaces in
// both a directory and the file name, giving a real executable to quote.
func copyTestBinaryToSpacedPath(t *testing.T) string {
	t.Helper()

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("could not locate the test binary: %v", err)
	}

	dir := filepath.Join(t.TempDir(), "My Program Dir")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("could not create the spaced directory: %v", err)
	}

	target := filepath.Join(dir, "arg print.exe")

	src, err := os.Open(self)
	if err != nil {
		t.Fatalf("could not open the test binary: %v", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		t.Fatalf("could not create the copy: %v", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		t.Fatalf("could not copy the test binary: %v", err)
	}
	if err := dst.Close(); err != nil {
		t.Fatalf("could not finish writing the copy: %v", err)
	}

	return target
}
