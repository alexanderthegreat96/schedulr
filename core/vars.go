package core

import (
	"fmt"
	"path/filepath"
	"strconv"
	"time"
)

var (
	rootPath, _      = GetRootDir()
	taskLocation     = filepath.Join(rootPath, TASKS_FOLDER)
	knownTaskTypes   = []string{SHELL_TASK, HTTP_TASK}
	pidFile          = "schedulr.pid"
	logFileName      = fmt.Sprintf("scheduler_%s.log", strconv.FormatInt(time.Now().UTC().UnixNano(), 10))
	pidFilePath      = filepath.Join(rootPath, pidFile)
	AppLogFilePath   = filepath.Join(rootPath, APP_LOGS_DIR, logFileName)
	TasksLogFilePath = filepath.Join(rootPath, TASK_LOGS_DIR, logFileName)
)
