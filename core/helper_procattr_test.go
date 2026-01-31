package core

import (
	"runtime"
	"syscall"
	"testing"
)

func TestGetProcAttrNotNil(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	if attr == nil {
		t.Fatal("GetProcAttr returned nil")
	}
}

func TestGetProcAttrTypeCorrect(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	_, ok := interface{}(attr).(*syscall.SysProcAttr)
	if !ok {
		t.Errorf("GetProcAttr returned wrong type: %T", attr)
	}
}

func TestGetProcAttrUnixSettings(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test, skipping on Windows")
	}

	attr := GetProcAttr()

	if attr.Setsid != true {
		t.Errorf("On Unix, Setsid should be true, got %v", attr.Setsid)
	}
}

func TestGetProcAttrWindowsSettings(t *testing.T) {
	t.Parallel()

	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test, skipping on non-Windows")
	}

	attr := GetProcAttr()

	if attr == nil {
		t.Fatal("GetProcAttr returned nil on Windows")
	}

	t.Logf("Windows SysProcAttr: %+v", attr)
}

func TestGetProcAttrConsistency(t *testing.T) {
	t.Parallel()

	attr1 := GetProcAttr()
	attr2 := GetProcAttr()

	if attr1 == nil || attr2 == nil {
		t.Fatal("GetProcAttr returned nil")
	}

	if runtime.GOOS != "windows" {
		if attr1.Setsid != attr2.Setsid {
			t.Errorf("Inconsistent Setsid values: %v vs %v", attr1.Setsid, attr2.Setsid)
		}
	}
}

func TestGetProcAttrCanBeUsed(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	testAttr := *attr

	_ = testAttr.Setsid
	_ = testAttr.Setpgid
	_ = testAttr.Setctty

	t.Logf("Successfully created and accessed SysProcAttr fields")
}
