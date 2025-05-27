package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config содержит всю конфигурацию приложения
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
}

// ServerConfig конфигурация gRPC сервера
type ServerConfig struct {
	Port            string
	ShutdownTimeout time.Duration
}

// DatabaseConfig конфигурация PostgreSQL
type DatabaseConfig struct {
	Host     string
	Port     string
	Database string
	Username string
	Password string
	SSLMode  string
}

// RedisConfig конфигурация Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// AuthConfig конфигурация аутентификации
type AuthConfig struct {
	SessionTTL time.Duration
}

// Load загружает конфигурацию из environment variables
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:            getEnv("GRPC_PORT", ":50051"),
			ShutdownTimeout: getDurationEnv("SHUTDOWN_TIMEOUT", 30*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			Database: getEnv("POSTGRES_DB", "auth_db"),
			Username: getEnv("POSTGRES_USER", "auth_user"),
			Password: getEnv("POSTGRES_PASSWORD", "auth_password"),
			SSLMode:  getEnv("POSTGRES_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getIntEnv("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			SessionTTL: getDurationEnv("SESSION_TTL", 24*time.Hour),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// validate проверяет корректность конфигурации
func (c *Config) validate() error {
	if c.Database.Host == "" {
		return fmt.Errorf("POSTGRES_HOST is required")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("POSTGRES_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST is required")
	}
	return nil
}

// DSN возвращает строку подключения к PostgreSQL
func (c *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
