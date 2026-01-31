package core

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func setupPidTestEnv(t *testing.T) string {
	temp := t.TempDir()
	old := RootPath
	RootPath = temp
	PidFilePath = filepath.Join(temp, "schedulr.pid")
	t.Cleanup(func() {
		RootPath = old
		PidFilePath = filepath.Join(old, "schedulr.pid")
	})
	return temp
}

func TestPidFileExists(t *testing.T) {
	setupPidTestEnv(t)

	if PidFileExists() {
		t.Error("PidFileExists should return false for non-existent file")
	}

	CreatePidFile(1234)
	if !PidFileExists() {
		t.Error("PidFileExists should return true after creating file")
	}
}

func TestCreatePidFile(t *testing.T) {
	setupPidTestEnv(t)

	pid := 9999
	err := CreatePidFile(pid)
	if err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	if !PidFileExists() {
		t.Error("pid file should exist after creation")
	}

	data, err := os.ReadFile(PidFilePath)
	if err != nil {
		t.Fatalf("failed to read pid file: %v", err)
	}

	if string(data) != strconv.Itoa(pid) {
		t.Errorf("pid file content = %q, want %q", string(data), strconv.Itoa(pid))
	}
}

func TestCreatePidFileAlreadyExists(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(1111)
	err := CreatePidFile(2222)
	if err == nil {
		t.Error("CreatePidFile should error when file already exists")
	}
}

func TestDeletePidFile(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(5555)
	err := DeletePidFile()
	if err != nil {
		t.Fatalf("DeletePidFile failed: %v", err)
	}

	if PidFileExists() {
		t.Error("pid file should not exist after deletion")
	}
}

func TestDeletePidFileNotExists(t *testing.T) {
	setupPidTestEnv(t)

	err := DeletePidFile()
	if err == nil {
		t.Error("DeletePidFile should error when file doesn't exist")
	}
}

func TestReadPidFile(t *testing.T) {
	setupPidTestEnv(t)

	expectedPid := 7777
	CreatePidFile(expectedPid)

	pid, err := ReadPidFile()
	if err != nil {
		t.Fatalf("ReadPidFile failed: %v", err)
	}

	if pid != expectedPid {
		t.Errorf("ReadPidFile returned %d, want %d", pid, expectedPid)
	}
}

func TestReadPidFileNotExists(t *testing.T) {
	setupPidTestEnv(t)

	_, err := ReadPidFile()
	if err == nil {
		t.Error("ReadPidFile should error when file doesn't exist")
	}
}

func TestReadPidFileInvalidFormat(t *testing.T) {
	setupPidTestEnv(t)

	os.WriteFile(PidFilePath, []byte("not-a-number"), 0644)

	_, err := ReadPidFile()
	if err == nil {
		t.Error("ReadPidFile should error with invalid format")
	}
}

func TestPidFileRoundTrip(t *testing.T) {
	setupPidTestEnv(t)

	originalPid := 12345

	err := CreatePidFile(originalPid)
	if err != nil {
		t.Fatalf("CreatePidFile failed: %v", err)
	}

	pid, err := ReadPidFile()
	if err != nil {
		t.Fatalf("ReadPidFile failed: %v", err)
	}

	if pid != originalPid {
		t.Errorf("round trip failed: got %d, want %d", pid, originalPid)
	}

	err = DeletePidFile()
	if err != nil {
		t.Fatalf("DeletePidFile failed: %v", err)
	}

	if PidFileExists() {
		t.Error("file should not exist after deletion")
	}
}

func TestMultiplePidOperations(t *testing.T) {
	setupPidTestEnv(t)

	CreatePidFile(1111)
	pid1, _ := ReadPidFile()

	DeletePidFile()

	CreatePidFile(2222)
	pid2, _ := ReadPidFile()

	if pid1 == pid2 {
		t.Error("PIDs should be different")
	}

	if pid1 != 1111 || pid2 != 2222 {
		t.Errorf("PIDs not as expected: %d, %d", pid1, pid2)
	}
}
