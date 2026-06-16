package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server        ServerConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Auth          AuthConfig
	Observability ObservabilityConfig
	Log           LogConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

type RedisConfig struct {
	URL      string
	Password string
	DB       int
}

type AuthConfig struct {
	APIKey string
}

type ObservabilityConfig struct {
	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string
}

type LogConfig struct {
	Level  string
	Format string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getInt("SERVER_PORT", 8080),
			ReadTimeout:  getDuration("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDuration("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getStr("DB_URL", "postgres://fms:fmspass@localhost:5432/featureflags?sslmode=disable"),
			MaxConns:        int32(getInt("DB_MAX_CONNS", 20)),
			MinConns:        int32(getInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime: getDuration("DB_MAX_CONN_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			URL:      getStr("REDIS_URL", "redis://localhost:6379"),
			Password: getStr("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			APIKey: getStr("AUTH_API_KEY", ""),
		},
		Observability: ObservabilityConfig{
			OTLPEndpoint:   getStr("OTLP_ENDPOINT", "localhost:4317"),
			ServiceName:    getStr("SERVICE_NAME", "feature-management-service"),
			ServiceVersion: getStr("SERVICE_VERSION", "1.0.0"),
		},
		Log: LogConfig{
			Level:  getStr("LOG_LEVEL", "info"),
			Format: getStr("LOG_FORMAT", "json"),
		},
	}
	return cfg, nil
}

func getStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
