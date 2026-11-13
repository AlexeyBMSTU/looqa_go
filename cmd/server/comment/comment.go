package comment

import (
	"context"
	"fmt"
	"log/slog"
	"looqa/cmd/server/middleware"
	"looqa/internal/config"
	commenthandler "looqa/internal/handler/comment_handler"
	commentrepository "looqa/internal/repository/comment_repository"
	commentservice "looqa/internal/service/comment_service"
	"looqa/internal/storage"
	"net/http"
	"time"
)

func StartCommentService(ctx context.Context) error {
	cfg := config.LoadConfigCommentService()

	pool, err := storage.NewPool(ctx, cfg.DBUri)
	if err != nil {
		return fmt.Errorf("comment service: %w", err)
	}
	defer pool.Close()

	repo := commentrepository.NewPostgresRepository(pool)
	svc := commentservice.NewService(repo)
	h := commenthandler.NewHandler(svc, cfg.LevelInfo)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	muxer := middleware.MiddlewareSetRequestID(middleware.MiddlewareLogger(middleware.MiddlewarePanicRecover(mux)))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: muxer,
	}

	errCh := make(chan error)

	go func() {
		slog.Info("Сервис комментариев запущен со следующими хар-ми", slog.Any("config", cfg))
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic comment service detected", "msg", r)
				errCh <- fmt.Errorf("panic comment service detected: %v", r)
			}
		}()

		if error := srv.ListenAndServe(); error != nil {
			errCh <- error
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("comment service: %w", err)
	case <-ctx.Done():
		slog.Info("Сервис комментариев останавливается...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		return srv.Shutdown(shutdownCtx)
	}
}
