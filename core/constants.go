package core

// app related
const (
	configFileName    string = "schedulr.config"
	projectRootMarker string = ".project-root"

	APP_LOGS_DIR    string = "app-logs"
	TASK_LOGS_DIR   string = "task-logs"
	TASKS_FOLDER    string = "tasks"
	SYSTEMD_COMMAND string = "systemctl"
	LAUNCHD_COMMAND string = "launchctl"
	SERVICE_NAME    string = "schedulr"
)

var utf8BOM = string([]byte{0xEF, 0xBB, 0xBF})

const (
	METHOD_GET     string = "GET"
	METHOD_POST    string = "POST"
	METHOD_PUT     string = "PUT"
	METHOD_PATCH   string = "PATCH"
	METHOD_DELETE  string = "DELETE"
	METHOD_HEAD    string = "HEAD"
	METHOD_OPTIONS string = "OPTIONS"
	DELAY          string = "delay"
	INTERVAL       string = "interval"
	SHELL_TASK     string = "shell"
	HTTP_TASK      string = "http"
)

// the Windows default shell; referenced from shared code, built verbatim in
// helper_cmdline_windows.go
const cmdShellName string = "cmd"

const METHOD_UNAVAILABLE string = "not-available"

const (
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorYellow    = "\033[33m"
	ColorGreen     = "\033[32m"
	ColorBlue      = "\033[34m"
	ColorOpenGreen = "\033[38;5;48m"
)

const (
	CREATE_NEW_PROCESS_GROUP = 0x00000200
	DETACHED_PROCESS         = 0x00000008
	CREATE_NO_WINDOW         = 0x08000000
)
