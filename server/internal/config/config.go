package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPath      string
	ListenAddr  string
	FrontendDir string
	BasePath    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/vrchat.db"
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		listenAddr = ":8080"
	}

	frontendDir := os.Getenv("FRONTEND_DIR")
	if frontendDir == "" {
		frontendDir = "static"
	}

	return &Config{
		DBPath:      dbPath,
		ListenAddr:  listenAddr,
		FrontendDir: frontendDir,
		BasePath:    normalizeBasePath(os.Getenv("BASE_PATH")),
	}, nil
}

func normalizeBasePath(raw string) string {
	p := strings.Trim(strings.TrimSpace(raw), "/")
	if p == "" {
		return ""
	}
	return "/" + p
}
