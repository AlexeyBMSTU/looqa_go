package authservice

import (
	"context"
	"time"

	"looqa/internal/auth"
	"looqa/model"

	"github.com/google/uuid"
)

// RefreshRepository — хранилище refresh-токенов (MongoDB).
type RefreshRepository interface {
	AddRefresh(ctx context.Context, token model.RefreshToken) error
	RotateRefresh(ctx context.Context, hash string) (model.RefreshToken, error)
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
}

// UserRepository — пользователи и их учётные данные (postgres).
type UserRepository interface {
	CreateUser(ctx context.Context, user model.User, cred model.Credential) error
	GetCredentialByEmail(ctx context.Context, email string) (model.Credential, error)
}

type Service struct {
	refresh    RefreshRepository
	users      UserRepository
	issuer     *auth.Issuer
	refreshTTL time.Duration
}

// NewService собирает сервис аутентификации: хранилище refresh-токенов
// refresh, источник данных о пользователях users, выпускальщик access-
// токенов issuer и время жизни refresh-токена refreshTTL.
func NewService(refresh RefreshRepository, users UserRepository, issuer *auth.Issuer, refreshTTL time.Duration) *Service {
	return &Service{refresh: refresh, users: users, issuer: issuer, refreshTTL: refreshTTL}
}
