package main

import (
	"context"
	"log/slog"
	"looqa/cmd/server/auth"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := auth.StartAuthService(ctx); err != nil {
		slog.Error("auth service stopped with error", "err", err)
		os.Exit(1)
	}
	slog.Info("auth service stopped")
}
