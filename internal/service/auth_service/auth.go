package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"looqa/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// minPasswordLen — минимальная длина пароля при регистрации.
const minPasswordLen = 8

// Register создаёт пользователя с учётными данными email+пароль.
// Пароль хранится только в виде bcrypt-хеша.
func (s *Service) Register(ctx context.Context, user model.User, email, password string) (model.User, error) {
	email = normalizeEmail(email)
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return model.User{}, fmt.Errorf("email %q: %w", email, model.ErrInvalid)
	}
	if len(password) < minPasswordLen {
		return model.User{}, fmt.Errorf("пароль короче %d символов: %w", minPasswordLen, model.ErrInvalid)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return model.User{}, fmt.Errorf("%w: hash password: %v", model.ErrInternal, err)
	}

	user.Id = uuid.New()
	cred := model.Credential{UserID: user.Id, Email: email, PasswordHash: string(hash)}
	if err := s.users.CreateUser(ctx, user, cred); err != nil {
		return model.User{}, err
	}
	return user, nil
}

// Login проверяет учётные данные и выдаёт пару токенов: короткоживущий
// JWT access и одноразовый refresh (в хранилище — только его хеш).
func (s *Service) Login(ctx context.Context, email, password string) (model.TokenPair, error) {
	cred, err := s.users.GetCredentialByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			// одна и та же ошибка для неизвестного email и неверного пароля,
			// чтобы по ответу нельзя было перебирать адреса
			return model.TokenPair{}, fmt.Errorf("login: %w", model.ErrUnauthorized)
		}
		return model.TokenPair{}, err
	}

	if bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(password)) != nil {
		return model.TokenPair{}, fmt.Errorf("login: %w", model.ErrUnauthorized)
	}

	// Сессия = семейство refresh-токенов: все ротации одного входа
	// связаны одним family_id, по нему же происходит отзыв.
	return s.issuePair(ctx, cred.UserID, uuid.New())
}

// Refresh обменивает refresh-токен на новую пару. Токен одноразовый:
// при обмене он ротируется (старый помечается использованным). Предъявление
// уже ротированного токена считается кражей — вся сессия отзывается.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (model.TokenPair, error) {
	hash := hashToken(refreshToken)

	rotated, err := s.refresh.RotateRefresh(ctx, hash)
	switch {
	case errors.Is(err, model.ErrNotFound):
		return model.TokenPair{}, fmt.Errorf("refresh: %w", model.ErrUnauthorized)
	case errors.Is(err, model.ErrConflict):
		// переиспользование живого ротированного токена: отзываем семейство
		if revErr := s.refresh.RevokeFamily(ctx, rotated.FamilyID); revErr != nil {
			return model.TokenPair{}, fmt.Errorf("refresh reuse, revoke family: %w", revErr)
		}
		return model.TokenPair{}, fmt.Errorf("refresh token reuse, family revoked: %w", model.ErrUnauthorized)
	case err != nil:
		return model.TokenPair{}, err
	}

	// Новая пара — в том же семействе (family_id сохраняется).
	return s.issuePair(ctx, rotated.UserID, rotated.FamilyID)
}

// Logout отзывает сессию: все refresh-токены её семейства удаляются.
// Access-токен при этом не трогается — он истечёт сам (AUTH_ACCESS_TTL);
// мгновенный отзыв stateless-токенов потребовал бы чёрный список.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	err := s.refresh.RevokeFamily(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("logout: %w", err)
	}
	return nil
}

// issuePair выпускает access-токен и сохраняет новый refresh-токен
// семейства sessionID пользователя userID.
func (s *Service) issuePair(ctx context.Context, userID, sessionID uuid.UUID) (model.TokenPair, error) {
	access, expiresAt, err := s.issuer.Issue(userID, sessionID)
	if err != nil {
		return model.TokenPair{}, fmt.Errorf("%w: issue access token: %v", model.ErrInternal, err)
	}

	// refresh — 32 случайных байта; в mongo уходит только sha256-хеш
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return model.TokenPair{}, fmt.Errorf("%w: gen refresh token: %v", model.ErrInternal, err)
	}
	refreshToken := hex.EncodeToString(raw)
	if err := s.refresh.AddRefresh(ctx, model.RefreshToken{
		Hash:      hashToken(refreshToken),
		FamilyID:  sessionID,
		UserID:    userID,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}); err != nil {
		return model.TokenPair{}, err
	}

	return model.TokenPair{
		AccessToken:  access,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Until(expiresAt).Round(time.Second).Seconds()),
	}, nil
}

// hashToken — sha256-хеш refresh-токена: он и только он хранится в mongo.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
