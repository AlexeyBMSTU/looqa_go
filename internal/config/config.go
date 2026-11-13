package config

import (
	"errors"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("не удалось прочитать .env", "err", err)
	}
}

type Config struct {
	Port      string
	LevelInfo slog.Leveler
}

func getEnv(key, defaultV string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultV
}

func parseLevel(s string) slog.Level {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return lvl
}

func LoadConfigCommentService() Config {
	port := getEnv("PORT_COMMENT_SERVICE", DEFAULT_PORT_COMMENT_SERVICE)
	levelStr := getEnv("LEVEL_INFO_COMMENT_SERVICE", DEFAULT_LEVEL_INFO_COMMENT_SERVICE)

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
	}
}


func LoadConfigPosterService() Config {
	port := getEnv("PORT_POSTER_SERVICE", DEFAULT_PORT_POSTER_SERVICE)
	levelStr := getEnv("LEVEL_INFO_POSTER_SERVICE", DEFAULT_LEVEL_INFO_POSTER_SERVICE)

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
	}
}


func LoadConfigUserService() Config {
	port := getEnv("PORT_USER_SERVICE", DEFAULT_PORT_USER_SERVICE)
	levelStr := getEnv("LEVEL_INFO_USER_SERVICE", DEFAULT_LEVEL_INFO_USER_SERVICE)

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
	}
}
