//go:build !windows
// +build !windows

package core

import "testing"

func TestGetProcAttrUnixSettings(t *testing.T) {
	t.Parallel()

	attr := GetProcAttr()

	if attr.Setsid != true {
		t.Errorf("On Unix, Setsid should be true, got %v", attr.Setsid)
	}
}

func TestGetProcAttrUnixConsistency(t *testing.T) {
	t.Parallel()

	attr1 := GetProcAttr()
	attr2 := GetProcAttr()

	if attr1 == nil || attr2 == nil {
		t.Fatal("GetProcAttr returned nil")
	}

	if attr1.Setsid != attr2.Setsid {
		t.Errorf("Inconsistent Setsid values: %v vs %v", attr1.Setsid, attr2.Setsid)
	}
}

func TestGetProcAttrUnixFieldsAccessible(t *testing.T) {
	t.Parallel()

	testAttr := *GetProcAttr()

	_ = testAttr.Setsid
	_ = testAttr.Setpgid
	_ = testAttr.Setctty

	t.Logf("Successfully created and accessed SysProcAttr fields")
}
