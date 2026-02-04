package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	Port         int
	DataDir      string
	CSVDir       string
	DownloadsDir string
	NASPath      string
	LogLevel     string
	Environment  string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnvInt("PORT", 3001),
		DataDir:     getEnv("DATA_DIR", "./data"),
		NASPath:     getEnv("NAS_PATH", "/Volumes/ExtHDD-8TB/Music/doyou-world-radio-tracks"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Environment: getEnv("ENVIRONMENT", "development"),
	}

	cfg.CSVDir = filepath.Join(cfg.DataDir, "csv")
	cfg.DownloadsDir = filepath.Join(cfg.DataDir, "downloads")

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", c.Port)
	}

	if c.DataDir == "" {
		return fmt.Errorf("data directory cannot be empty")
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}

func (c *Config) EnsureDirectories() error {
	dirs := []string{c.DataDir, c.CSVDir, c.DownloadsDir}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
