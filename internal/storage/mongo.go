package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// NewMongoClient подключается к MongoDB по uri и ждёт готовности
// (с ретраями, по аналогии с NewPool для postgres).
func NewMongoClient(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("create mongo client: %w", err)
	}

	const attempts = 5

	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err = client.Ping(pingCtx, readpref.Primary())
		cancel()
		if err == nil {
			return client, nil
		}
		if attempt == attempts {
			break
		}
		slog.Warn("mongo ещё не готов, пробую снова", "attempt", attempt, "err", err)
		select {
		case <-ctx.Done():
			_ = client.Disconnect(context.Background())
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}

	_ = client.Disconnect(context.Background())
	return nil, fmt.Errorf("mongo недоступен после %d попыток: %w", attempts, err)
}
