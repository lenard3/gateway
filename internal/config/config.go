package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

type Config struct {
	ConfigFile string
	Server     Server `yaml:"server"`
}

type Server struct {
	Addr            string `yaml:"addr"`
	LogLevel        string `yaml:"log_level"`
	ReadTimeout     string `yaml:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout"`
	ShutdownTimeout string `yaml:"shutdown_timeout"`
}

// Load tries to load all the env vars.
// Also reads config file for Server settings.
// Returns filled config, error
func Load() (*Config, error) {
	godotenv.Load("./.env")
	c := Config{}
	var ok bool

	c.ConfigFile, ok = os.LookupEnv("CONFIG_FILE")
	if !ok {
		c.ConfigFile = "config.yaml"
	}

	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	errUn := yaml.Unmarshal(data, &c)
	if errUn != nil {
		return nil, fmt.Errorf("parse config: %w", errUn)
	}

	if c.Server.Addr == "" {
		c.Server.Addr = ":9000"
	}
	if c.Server.LogLevel == "" {
		c.Server.LogLevel = "info"
	}
	if c.Server.ReadTimeout == "" {
		c.Server.ReadTimeout = "10s"
	}
	if c.Server.WriteTimeout == "" {
		c.Server.WriteTimeout = "10s"
	}
	if c.Server.ShutdownTimeout == "" {
		c.Server.ShutdownTimeout = "15s"
	}
	return &c, nil
}
