package auth

import (
	"context"
	"fmt"
	"log/slog"
	"looqa/cmd/server/middleware"
	"looqa/internal/auth"
	"looqa/internal/config"
	authhandler "looqa/internal/handler/auth_handler"
	authrepository "looqa/internal/repository/auth_repository"
	authservice "looqa/internal/service/auth_service"
	"looqa/internal/storage"
	"net/http"
	"time"
)

func StartAuthService(ctx context.Context) error {
	cfg := config.LoadConfigAuthService()

	// postgres — таблица "User": проверка существования пользователя
	// перед выдачей токена.
	pool, err := storage.NewPool(ctx, cfg.DBUri)
	if err != nil {
		return fmt.Errorf("auth service: %w", err)
	}
	defer pool.Close()

	// mongo — хранилище refresh-токенов.
	mongoClient, err := storage.NewMongoClient(ctx, cfg.MongoUri)
	if err != nil {
		return fmt.Errorf("auth service: %w", err)
	}
	defer func() {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoClient.Disconnect(disconnectCtx); err != nil {
			slog.Warn("mongo disconnect error", "msg", err)
		}
	}()

	refresh := mongoClient.Database(mongoDatabase).Collection(mongoCollection)
	refreshRepo, err := authrepository.NewMongoRepository(ctx, refresh)
	if err != nil {
		return fmt.Errorf("auth service: %w", err)
	}

	// ключи Ed25519 для подписи access-токенов: генерируются на старте,
	// публичный доступен ресурсным сервисам через GET /jwks.
	issuer, err := auth.NewIssuer(cfg.AccessTokenTTL)
	if err != nil {
		return fmt.Errorf("auth service: %w", err)
	}

	userRepo := authrepository.NewPostgresUserRepository(pool)
	svc := authservice.NewService(refreshRepo, userRepo, issuer, cfg.RefreshTokenTTL)
	h := authhandler.NewHandler(svc, issuer, cfg.LevelInfo, cfg.RateLimit)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	muxer := middleware.MiddlewareSetRequestID(middleware.MiddlewareLogger(middleware.MiddlewarePanicRecover(mux)))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: muxer,
	}

	errCh := make(chan error)

	go func() {
		slog.Info("Сервис аутентификации запущен со следующими хар-ми", slog.Any("config", cfg))
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic auth service detected", "msg", r)
				errCh <- fmt.Errorf("panic auth service detected: %v", r)
			}
		}()

		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("auth service: %w", err)
	case <-ctx.Done():
		slog.Info("Сервис аутентификации останавливается...")
		// context.Background: сам ctx уже отменён, а Shutdown должен
		// дождаться активных запросов.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)
	}
}

const (
	mongoDatabase   = "looqa"
	mongoCollection = "refresh_tokens"
)
