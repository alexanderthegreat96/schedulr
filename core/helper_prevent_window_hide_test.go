package core

import "testing"

func TestPreventWindowHideDocumentation(t *testing.T) {
	t.Parallel()

	t.Logf("PreventWindowHide is designed to modify cmd.SysProcAttr")
	t.Logf("On Unix: It's a no-op (return immediately)")
	t.Logf("On Windows: It sets HideWindow to false to show the window")
}
