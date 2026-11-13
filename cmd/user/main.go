package main

import (
	"context"
	"log/slog"
	"looqa/cmd/server/user"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := user.StartUserService(ctx); err != nil {
		slog.Error("сервис юзеров остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
	slog.Info("сервис юзеров остановлен")
}
