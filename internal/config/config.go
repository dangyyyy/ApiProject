package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
)

type Config struct {
	ServerPort string
	LogFormat  string
	LogLevel   string
	DB         DBConfig
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (c DBConfig) URL() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     net.JoinHostPort(c.Host, c.Port),
		Path:     "/" + c.Name,
		RawQuery: url.Values{"sslmode": {c.SSLMode}}.Encode(),
	}
	return u.String()
}

func LoadConfig() (Config, error) {
	var errs []error
	required := func(key string) string {
		value := os.Getenv(key)
		if value == "" {
			errs = append(errs, fmt.Errorf("%s must be set", key))
		}
		return value
	}
	cfg := Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),
		LogFormat:  getEnv("LOG_FORMAT", "text"),
		LogLevel:   getEnv("LOG_LEVEL", "info"),
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     required("DB_USER"),
			Password: required("DB_PASSWORD"),
			Name:     required("DB_NAME"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}
	if cfg.LogFormat != "text" && cfg.LogFormat != "json" {
		errs = append(errs, fmt.Errorf("invalid log format: %s", cfg.LogFormat))
	}
	if err := errors.Join(errs...); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getEnv(key string, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
