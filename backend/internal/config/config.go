package config

import (
	"os"
)

type Config struct {
	Listen      string
	Database    string
	Key         string
	FrontendDir string
}

func Load() *Config {
	return &Config{
		Listen:      getEnv("LISTEN", "0.0.0.0:18888"),
		Database:    getEnv("DATABASE", "sqlite3://data.db"),
		Key:         getEnv("KEY", ""),
		FrontendDir: getEnv("FRONTEND_DIR", "./public"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
