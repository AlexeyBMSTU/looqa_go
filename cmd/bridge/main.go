package main

import (
	"context"
	"log/slog"
	bridgeserver "looqa/cmd/server/bridge"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := bridgeserver.StartBridgeService(ctx); err != nil {
		slog.Error("bridge ai service stopped with error", "err", err)
		os.Exit(1)
	}
	slog.Info("bridge ai service stopped")
}
