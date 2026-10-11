//go:build windows
// +build windows

package core

import "testing"

func TestGetProcAttrWindowsSettings(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	if attr == nil {
		t.Fatal("GetProcAttr returned nil on Windows")
	}

	t.Logf("Windows SysProcAttr: %+v", attr)
}

func TestGetProcAttrWindowsDetachesDaemon(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	if attr.CreationFlags&DETACHED_PROCESS == 0 {
		t.Error("DETACHED_PROCESS should be set so the daemon gets no console")
	}

	if attr.CreationFlags&CREATE_NEW_PROCESS_GROUP == 0 {
		t.Error("CREATE_NEW_PROCESS_GROUP should be set so console control events do not reach the daemon")
	}

	if !attr.HideWindow {
		t.Error("HideWindow should be set so starting the daemon flashes no window")
	}
}

func TestGetProcAttrWindowsFieldsAccessible(t *testing.T) {
	t.Parallel()

	testAttr := *GetProcAttr()

	_ = testAttr.HideWindow
	_ = testAttr.CreationFlags

	t.Logf("Successfully created and accessed Windows SysProcAttr fields")
}
