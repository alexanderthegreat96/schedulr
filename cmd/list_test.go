package cmd

import (
	"testing"

	"github.com/alexanderthegreat96/schedulr/core"
)

// This used to go through core.DefaultValueIfNull(dep, "string"), which never
// matches its "string" case for a Task and so returned the value untouched -
// printing the whole marshalled struct where the name belonged.
func TestDependencyName(t *testing.T) {
	t.Parallel()

	if got := dependencyName(nil); got != "no-value" {
		t.Errorf("dependencyName(nil) = %q, want %q", got, "no-value")
	}

	shell := core.ShellTask{Name: "BackupDatabase"}
	if got := dependencyName(shell); got != "BackupDatabase" {
		t.Errorf("dependencyName(shell task) = %q, want %q", got, "BackupDatabase")
	}

	http := core.HttpTask{Name: "PingApi"}
	if got := dependencyName(http); got != "PingApi" {
		t.Errorf("dependencyName(http task) = %q, want %q", got, "PingApi")
	}
}
