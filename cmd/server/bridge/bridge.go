package bridgeserver

import (
	"context"
	"fmt"
	"log/slog"
	"looqa/cmd/server/middleware"
	"looqa/internal/auth"
	"looqa/internal/config"
	bridgehandler "looqa/internal/handler/bridge_handler"
	bridgerepository "looqa/internal/repository/bridge_repository"
	bridgeservice "looqa/internal/service/bridge_service"
	"net/http"
	"time"
)

func StartBridgeService(ctx context.Context) error {
	cfg := config.LoadConfigBridgeService()

	repo := bridgerepository.NewInMemoryRepository()
	svc := bridgeservice.NewService(repo)
	h := bridgehandler.NewHandler(svc, cfg.LevelInfo)

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
		slog.Info("Сервис bridge ai запущен со следующими хар-ми", slog.Any("config", cfg))
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic bridge ai service detected", "msg", r)
				errCh <- fmt.Errorf("panic bridge ai service detected: %v", r)
			}
		}()

		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("bridge ai service: %w", err)
	case <-ctx.Done():
		slog.Info("Сервис bridge ai останавливается...")
		// context.Background: сам ctx уже отменён, а Shutdown должен
		// дождаться активных запросов.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)
	}
}
