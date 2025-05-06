package core

import (
	"fmt"
	"path/filepath"
	"time"
)

var (
	RootPath, _     = GetRootDir()
	TaskLocation    = filepath.Join(RootPath, TASKS_FOLDER)
	knownTaskTypes  = []string{SHELL_TASK, HTTP_TASK}
	pidFile         = "schedulr.pid"
	logFileName     = fmt.Sprintf("scheduler_%s.log", time.Now().UTC().Format("2006-01-02T15-04-05.000Z"))
	PidFilePath     = filepath.Join(RootPath, pidFile)
	AppLogFilePath  = filepath.Join(RootPath, APP_LOGS_DIR, logFileName)
	TasksLogDirPath = filepath.Join(RootPath, TASK_LOGS_DIR)
	AppLogDirPath   = filepath.Join(RootPath, APP_LOGS_DIR)
)
