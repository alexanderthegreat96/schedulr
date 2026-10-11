package core

import (
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

func TestGetProcAttrReturnsDistinctInstances(t *testing.T) {
	t.Parallel()

	attr1 := GetProcAttr()
	attr2 := GetProcAttr()

	if attr1 == nil || attr2 == nil {
		t.Fatal("GetProcAttr returned nil")
	}

	// each caller must get its own struct, otherwise two concurrently
	// started processes would share (and race on) the same attributes
	if attr1 == attr2 {
		t.Error("GetProcAttr returned the same pointer twice")
	}
}
