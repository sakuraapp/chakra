package config

import (
	"errors"
	"os"
	"strconv"
)

type ConfigSource interface {
	GetString(key string) string
	GetInt(key string) (int, error)
}

type Config struct {
	ConfigSource
}

func New() *Config {
	return &Config{}
}

func (c *Config) GetString(key string) string {
	return os.Getenv(key)
}

func (c *Config) GetInt(key string) (int, error) {
	str := c.GetString(key)

	if len(str) == 0 {
		return 0, errors.New("environment variable is empty")
	}

	return strconv.Atoi(str)
}