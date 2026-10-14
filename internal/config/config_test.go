package config

import (
	"strings"
	"testing"
)

func setRequired(t *testing.T) {
	t.Helper()
	t.Setenv("DB_USER", "taskuser")
	t.Setenv("DB_PASSWORD", "taskpass")
	t.Setenv("DB_NAME", "tasksdb")
}

func TestLoad_Defaults(t *testing.T) {
	setRequired(t)
	for _, key := range []string{"SERVER_PORT", "LOG_FORMAT", "LOG_LEVEL", "DB_HOST", "DB_PORT", "DB_SSLMODE"} {
		t.Setenv(key, "")
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.ServerPort != "8080" || cfg.LogFormat != "text" || cfg.LogLevel != "info" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	want := "postgres://taskuser:taskpass@localhost:5432/tasksdb?sslmode=disable"
	if got := cfg.DB.URL(); got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}

func TestLoad_MissingRequired(t *testing.T) {
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	for _, key := range []string{"DB_USER", "DB_PASSWORD", "DB_NAME"} {
		if !strings.Contains(err.Error(), key) {
			t.Errorf("error %q does not mention %s", err, key)
		}
	}
}

func TestLoad_InvalidLogFormat(t *testing.T) {
	setRequired(t)
	t.Setenv("LOG_FORMAT", "xml")

	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected error for LOG_FORMAT=xml, got nil")
	}
}

func TestDBConfig_URL_EscapesPassword(t *testing.T) {
	c := DBConfig{Host: "db", Port: "5432", User: "app", Password: "p@ss:w/rd#1", Name: "tasks", SSLMode: "disable"}

	want := "postgres://app:p%40ss%3Aw%2Frd%231@db:5432/tasks?sslmode=disable"
	if got := c.URL(); got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
}
