package main

import (
	"context"
	"log/slog"
	"looqa/cmd/server/comment"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := comment.StartCommentService(ctx); err != nil {
		slog.Error("сервис комментариев остановлен с ошибкой", "err", err)
		os.Exit(1)
	}
	slog.Info("сервис комментариев остановлен")
}
