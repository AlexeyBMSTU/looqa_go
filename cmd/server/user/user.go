package user

import (
	"context"
	"fmt"
	"log/slog"
	"looqa/cmd/server/middleware"
	"looqa/internal/config"
	userhandler "looqa/internal/handler/user_handler"
	userrepository "looqa/internal/repository/user_repository"
	userservice "looqa/internal/service/user_service"
	"looqa/internal/storage"
	"net/http"
	"time"
)

func StartUserService(ctx context.Context) error {
	cfg := config.LoadConfigUserService()

	pool, err := storage.NewPool(ctx, cfg.DBUri)
	if err != nil {
		return fmt.Errorf("user service: %w", err)
	}
	defer pool.Close()

	repo := userrepository.NewPostgresRepository(pool)
	svc := userservice.NewService(repo)
	h := userhandler.NewHandler(svc, cfg.LevelInfo)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	muxer := middleware.MiddlewareSetRequestID(middleware.MiddlewareLogger(middleware.MiddlewarePanicRecover(mux)))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: muxer,
	}

	errCh := make(chan error)

	go func() {
		slog.Info("Сервис юзеров запущен со следующими хар-ми", slog.Any("config", cfg))
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic user service detected", "msg", r)
				errCh <- fmt.Errorf("panic user service detected: %v", r)
			}
		}()

		if error := srv.ListenAndServe(); error != nil {
			errCh <- error
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("user service: %w", err)
	case <-ctx.Done():
		slog.Info("Сервис юзеров останавливается...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)
	}
}
