package core

// various constants used across the application
// from status codes, to interval types and task types
const (
	APP_LOGS_DIR   string = "app-logs"
	TASK_LOGS_DIR  string = "task-logs"
	TASKS_FOLDER   string = "tasks"
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

// colors for the logger
const (
	ColorReset     = "\033[0m"
	ColorRed       = "\033[31m"
	ColorYellow    = "\033[33m"
	ColorGreen     = "\033[32m"
	ColorBlue      = "\033[34m"
	ColorOpenGreen = "\033[38;5;48m"
)

// sys calls for windows
// unsure if they work
const (
	CREATE_NEW_PROCESS_GROUP = 0x00000200
	DETACHED_PROCESS         = 0x00000008
)
