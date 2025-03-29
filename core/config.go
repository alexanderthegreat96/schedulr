package core

import (
	"strings"

	"github.com/alexanderthegreat96/envparser"
)

func AppConfig() *Config {
	envParser := envparser.NewEnvParser("schedulr.config")
	cfg := &Config{
		env: envParser,
	}

	cfg.DevMode = cfg.getBool("SCHEDULR_DEV", false)
	cfg.LogData = cfg.getBool("LOG_DATA", true)
	cfg.WipeLogDataInterval = cfg.getInt("LOG_WIPE_INTERVAL_SECONDS", 30)
	cfg.WorkerCount = cfg.getInt("WORKER_COUNT", 4)
	cfg.SystemDCommand = cfg.getStr("SYSTEMD_COMMAND", SYSTEMD_COMMAND)
	cfg.LaunchDCommand = cfg.getStr("LAUNCHD_COMMAND", LAUNCHD_COMMAND)
	cfg.ServiceName = cfg.getStr("SERVICE_NAME", SERVICE_NAME)

	return cfg
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
