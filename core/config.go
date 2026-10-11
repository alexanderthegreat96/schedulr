package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexanderthegreat96/envparser"
)

func ConfigFilePath() string {
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), configFileName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), configFileName)
	}

	return configFileName
}

func AppConfig() *Config {
	envParser := envparser.NewEnvParser(ConfigFilePath(), false)
	cfg := &Config{
		env: envParser,
	}

	cfg.DevMode = cfg.getBool("SCHEDULR_DEV", false)
	cfg.LogData = cfg.getBool("LOG_DATA", true)
	cfg.WipeLogDataInterval = cfg.getInt("LOG_WIPE_INTERVAL_SECONDS", 100)
	cfg.WorkerCount = cfg.getInt("WORKER_COUNT", 4)
	cfg.ReloadIntervalSeconds = cfg.getInt("RELOAD_INTERVAL_SECONDS", 30)
	cfg.DependencyMaxWaitSeconds = cfg.getInt("DEPENDENCY_MAX_WAIT_SECONDS", 60)
	cfg.HttpLogBodyLimit = cfg.getInt("HTTP_LOG_BODY_LIMIT", 2048)
	cfg.EnableSchedulrService = cfg.getBool("ENABLE_SCHEDULR_SERVICE", true)
	cfg.SystemDCommand = cfg.getStr("SYSTEMD_COMMAND", SYSTEMD_COMMAND)
	cfg.LaunchDCommand = cfg.getStr("LAUNCHD_COMMAND", LAUNCHD_COMMAND)
	cfg.ServiceName = cfg.getStr("SERVICE_NAME", SERVICE_NAME)

	return cfg
}

func (c *Config) Status() string {
	if c.env == nil {
		return "not loaded"
	}

	if c.env.EnvError != nil {
		return fmt.Sprintf("NOT loaded, using defaults: %s", c.env.EnvError.Error())
	}

	return "loaded"
}

func (c *Config) getBool(key string, defaultVal bool) bool {
	val, err := c.env.GetValue(key, "bool", defaultVal)
	if err != nil {
		return defaultVal
	}

	boolVal, ok := val.(bool)
	if !ok {
		return defaultVal
	}

	return boolVal
}

func (c *Config) getInt(key string, defaultVal int) int {
	val, err := c.env.GetValue(key, "int", defaultVal)

	if err != nil {
		return defaultVal
	}

	intVal, ok := val.(int)
	if !ok {
		return defaultVal
	}

	return intVal
}

func (c *Config) getStr(key, defaultVal string) string {
	val, err := c.env.GetValue(key, "string", defaultVal)
	if err != nil {
		return defaultVal
	}

	strVal, ok := val.(string)
	if !ok {
		return defaultVal
	}

	return strings.Trim(strVal, `"`)
}
