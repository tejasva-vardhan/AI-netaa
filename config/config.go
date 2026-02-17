package config

import (
	"os"
	"strconv"
)

// Config holds application configuration
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	Pilot    PilotConfig
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Host string
}

// PilotConfig holds pilot-specific configuration
type PilotConfig struct {
	DryRun                        bool // PILOT_DRY_RUN: Enable dry-run/testing mode
	DryRunSLAOverrideMinutes      int  // PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES: Override SLA hours with minutes (0 = disabled)
	TestEscalationOverrideMinutes int  // TEST_ESCALATION_OVERRIDE_MINUTES: Safe test-only SLA override in minutes (0 = disabled)
	EscalationWorkerIntervalSeconds int // ESCALATION_WORKER_INTERVAL_SECONDS: Worker run interval in seconds (0 = use default: 1h or pilot 30s)
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "3306"),
			User:     getEnv("DB_USER", "root"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "finalneta"),
		},
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Pilot: PilotConfig{
			DryRun:                         getEnvBool("PILOT_DRY_RUN", false),
			DryRunSLAOverrideMinutes:       getEnvInt("PILOT_DRY_RUN_SLA_OVERRIDE_MINUTES", 0),
			TestEscalationOverrideMinutes:  getEnvInt("TEST_ESCALATION_OVERRIDE_MINUTES", 0),
			EscalationWorkerIntervalSeconds: getEnvInt("ESCALATION_WORKER_INTERVAL_SECONDS", 0),
		},
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool gets a boolean environment variable or returns a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// getEnvInt gets an integer environment variable or returns a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
