package model

import (
	"time"

	"github.com/google/uuid"
)

type Poster struct {
	Id            uuid.UUID `json:"id"`
	Label         string    `json:"name"`
	Text          string    `json:"text"`
	AuthorID      uuid.UUID `json:"author_id"`
	CommentsCount int64     `json:"comments_count"`
	LikesCount    int64     `json:"likes_count"`
	PublicationAt time.Time `json:"publication_at"`
}

type User struct {
	Id        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	AvatarURL string    `json:"avatar_url"`
}

type Comment struct {
	Id         uuid.UUID `json:"id"`
	AuthorID   uuid.UUID `json:"author_id"`
	LikesCount int64     `json:"likes_count"`
}

// TokenPair — ответ POST /login и POST /refresh: короткоживущий JWT
// access-токен и одноразовый refresh-токен для получения новой пары.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // секунды до истечения access-токена
}

// RefreshToken — refresh-токен на стороне хранилища: сам токен клиенту
// не хранится, только его SHA-256-хеш; FamilyID связывает цепочку
// ротированных токенов одной сессии (для отзыва и детекта переиспользования).
type RefreshToken struct {
	Hash      string
	FamilyID  uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
}

type Credential struct {
	UserID       uuid.UUID `json:"-"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
}

type CredentialAIData struct {
	Key   string `json:"key"`
	Url   string `json:"url"`
	Model string `json:"model"`
}

type MessageRequest struct {
	Message string `json:"message"`
}
