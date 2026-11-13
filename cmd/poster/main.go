package main

import (
	"context"
	"log/slog"
	"looqa/cmd/server/poster"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := poster.StartPosterService(ctx); err != nil {
		slog.Error("сервис постов остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
	slog.Info("сервис постов остановлен")
}
