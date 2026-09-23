package config

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/joho/godotenv"
)

const defaultEnvFile = ".env"
const defaultConfigFile = "config.yaml"
const defaultServerAddr = ":9000"
const defaultLogLevel = "info"
const defaultReadTimeout = "10s"
const defaultWriteTimeout = "10s"
const defaultShutdownTimeout = "15s"
const defaultRequestTimeout = "15s"
const defaultAccessTTL = "15m"

type Config struct {
	ConfigFile  string
	DatabaseURL string
	JWTSecret   string
	Server      Server `yaml:"server"`
	JWT         JWT    `yaml:"jwt"`
}

type Server struct {
	Addr            string `yaml:"addr"`
	LogLevel        string `yaml:"log_level"`
	ReadTimeout     string `yaml:"read_timeout"`
	WriteTimeout    string `yaml:"write_timeout"`
	ShutdownTimeout string `yaml:"shutdown_timeout"`
	RequestTimeout  string `yaml:"request_timeout"`
}

type JWT struct {
	Access_ttl string `yaml:"access_ttl"`
}

// Load tries to load all the env vars.
// Also reads config file for Server settings.
// Sets standard values for non set server variables
// Returns filled config, error
func Load() (*Config, error) {
	godotenv.Load(defaultEnvFile)
	c := Config{}
	var ok bool

	c.ConfigFile, ok = os.LookupEnv("CONFIG_FILE")
	if !ok {
		c.ConfigFile = defaultConfigFile
	}

	c.DatabaseURL, ok = os.LookupEnv("DATABASE_URL")
	if !ok {
		return nil, fmt.Errorf("error reading `POSTGRES_URL`")
	}

	c.JWTSecret, ok = os.LookupEnv("JWT_SECRET")
	if !ok {
		return nil, fmt.Errorf("error reading `JWT_SECRET`")
	}

	// reading server section of config.yaml file
	data, err := os.ReadFile(c.ConfigFile)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	errUn := yaml.Unmarshal(data, &c)
	if errUn != nil {
		return nil, fmt.Errorf("parse config: %w", errUn)
	}

	if c.Server.Addr == "" {
		c.Server.Addr = defaultServerAddr
	}
	if c.Server.LogLevel == "" {
		c.Server.LogLevel = defaultLogLevel
	}
	if c.Server.ReadTimeout == "" {
		c.Server.ReadTimeout = defaultReadTimeout
	}
	if c.Server.WriteTimeout == "" {
		c.Server.WriteTimeout = defaultWriteTimeout
	}
	if c.Server.ShutdownTimeout == "" {
		c.Server.ShutdownTimeout = defaultShutdownTimeout
	}
	if c.Server.RequestTimeout == "" {
		c.Server.RequestTimeout = defaultRequestTimeout
	}

	if c.JWT.Access_ttl == "" {
		c.JWT.Access_ttl = defaultAccessTTL
	}

	return &c, nil
}
