package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr            string
	LogLevel        string
	ConfigFile      string
	MusicBackendURL string
}

// Load tries to load all the environment variables.
// If variable is not set, default is added in some cases
// Current defaults are: GATEWAY_ADDR, LOG_LEVEL, MUSIC_BACKEND_URL
// Returns filled config, error
func Load() (*Config, error) {
	godotenv.Load("./.env")

	gatewayAddr, gwFilled := os.LookupEnv("GATEWAY_ADDR")
	if !gwFilled {
		gatewayAddr = ":9999"
	}

	logLevel, llFilled := os.LookupEnv("LOG_LEVEL")
	if !llFilled {
		logLevel = "info"
	}

	configFile, cfFilled := os.LookupEnv("CONFIG_FILE")
	if !cfFilled {
		configFile = "config.yaml"
	}

	musicBackendURL, mbFilled := os.LookupEnv("MUSIC_BACKEND_URL")
	if !mbFilled {
		musicBackendURL = "http://localhost:8080"
	}

	return &Config{
		Addr:            gatewayAddr,
		LogLevel:        logLevel,
		ConfigFile:      configFile,
		MusicBackendURL: musicBackendURL,
	}, nil
}
