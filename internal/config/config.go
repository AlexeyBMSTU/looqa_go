package config

import (
	"errors"
	"log/slog"
	"os"
	"strconv"
	"time"

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
	DBUri     string
	MongoUri  string

	// Auth-сервис: TTL токенов и лимит запросов с одного IP
	// на /reg, /login и /refresh.
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	RateLimit       int

	// JWKSUrl — адрес GET /jwks auth-сервиса: его читают ресурсные
	// сервисы, чтобы проверять access-токены локально.
	JWKSUrl string
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
	dbUri := getEnv("DB_URI", "")

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
		DBUri:     dbUri,
		JWKSUrl:   getEnv("AUTH_JWKS_URL", DEFAULT_AUTH_JWKS_URL),
	}
}

func LoadConfigAuthService() Config {
	port := getEnv("PORT_AUTH_SERVICE", DEFAULT_PORT_AUTH_SERVICE)
	levelStr := getEnv("LEVEL_INFO_AUTH_SERVICE", DEFAULT_LEVEL_INFO_AUTH_SERVICE)
	dbUri := getEnv("DB_URI", "")
	mongoUri := getEnv("MONGO_URI", DEFAULT_MONGO_URI)

	accessTTL, err := time.ParseDuration(getEnv("AUTH_ACCESS_TTL", DEFAULT_AUTH_ACCESS_TTL))
	if err != nil {
		slog.Warn("неверное значение AUTH_ACCESS_TTL, использую значение по умолчанию",
			"value", DEFAULT_AUTH_ACCESS_TTL, "err", err)
		accessTTL, _ = time.ParseDuration(DEFAULT_AUTH_ACCESS_TTL)
	}
	refreshTTL, err := time.ParseDuration(getEnv("AUTH_REFRESH_TTL", DEFAULT_AUTH_REFRESH_TTL))
	if err != nil {
		slog.Warn("неверное значение AUTH_REFRESH_TTL, использую значение по умолчанию",
			"value", DEFAULT_AUTH_REFRESH_TTL, "err", err)
		refreshTTL, _ = time.ParseDuration(DEFAULT_AUTH_REFRESH_TTL)
	}
	rateLimit, err := strconv.Atoi(getEnv("AUTH_RATE_LIMIT", DEFAULT_AUTH_RATE_LIMIT))
	if err != nil || rateLimit < 0 {
		slog.Warn("неверное значение AUTH_RATE_LIMIT, использую значение по умолчанию",
			"value", DEFAULT_AUTH_RATE_LIMIT, "err", err)
		rateLimit, _ = strconv.Atoi(DEFAULT_AUTH_RATE_LIMIT)
	}

	return Config{
		Port:            port,
		LevelInfo:       parseLevel(levelStr),
		DBUri:           dbUri,
		MongoUri:        mongoUri,
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		RateLimit:       rateLimit,
	}
}

func LoadConfigBridgeService() Config {
	port := getEnv("PORT_BRIDGE_SERVICE", DEFAULT_PORT_BRIDGE_SERVICE)
	levelStr := getEnv("LEVEL_INFO_BRIDGE_SERVICE", DEFAULT_LEVEL_INFO_BRIDGE_SERVICE)

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
		JWKSUrl:   getEnv("AUTH_JWKS_URL", DEFAULT_AUTH_JWKS_URL),
	}
}

func LoadConfigPosterService() Config {
	port := getEnv("PORT_POSTER_SERVICE", DEFAULT_PORT_POSTER_SERVICE)
	levelStr := getEnv("LEVEL_INFO_POSTER_SERVICE", DEFAULT_LEVEL_INFO_POSTER_SERVICE)
	dbUri := getEnv("DB_URI", "")

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
		DBUri:     dbUri,
		JWKSUrl:   getEnv("AUTH_JWKS_URL", DEFAULT_AUTH_JWKS_URL),
	}
}

func LoadConfigUserService() Config {
	port := getEnv("PORT_USER_SERVICE", DEFAULT_PORT_USER_SERVICE)
	levelStr := getEnv("LEVEL_INFO_USER_SERVICE", DEFAULT_LEVEL_INFO_USER_SERVICE)
	dbUri := getEnv("DB_URI", "")

	return Config{
		Port:      port,
		LevelInfo: parseLevel(levelStr),
		DBUri:     dbUri,
		JWKSUrl:   getEnv("AUTH_JWKS_URL", DEFAULT_AUTH_JWKS_URL),
	}
}
