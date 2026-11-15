package poster

import (
	"context"
	"fmt"
	"log/slog"
	"looqa/cmd/server/middleware"
	"looqa/internal/auth"
	"looqa/internal/config"
	posterhandler "looqa/internal/handler/poster_handler"
	posterrepository "looqa/internal/repository/poster_repository"
	posterservice "looqa/internal/service/poster_service"
	"looqa/internal/storage"
	"net/http"
	"time"
)

func StartPosterService(ctx context.Context) error {
	cfg := config.LoadConfigPosterService()

	pool, err := storage.NewPool(ctx, cfg.DBUri)
	if err != nil {
		return fmt.Errorf("poster service: %w", err)
	}
	defer pool.Close()

	repo := posterrepository.NewPostgresRepository(pool)
	svc := posterservice.NewService(repo)
	h := posterhandler.NewHandler(svc, cfg.LevelInfo)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	muxer := middleware.MiddlewareSetRequestID(middleware.MiddlewareLogger(middleware.MiddlewarePanicRecover(
		middleware.MiddlewareAuth(auth.NewJWKSVerifier(cfg.JWKSUrl))(mux),
	)))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: muxer,
	}

	errCh := make(chan error)

	go func() {
		slog.Info("Сервис постов запущен со следующими хар-ми", slog.Any("config", cfg))
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic poster service detected", "msg", r)
				errCh <- fmt.Errorf("panic poster service detected: %v", r)
			}
		}()

		if error := srv.ListenAndServe(); error != nil {
			errCh <- error
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("poster service: %w", err)
	case <-ctx.Done():
		slog.Info("Сервис постов останавливается...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
