package core

import (
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
