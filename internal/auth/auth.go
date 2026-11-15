// Package auth выпускает и проверяет JWT access-токены looqa.
//
// Токены подписываются ключом Ed25519 (алгоритм EdDSA) и живут недолго
// (AUTH_ACCESS_TTL). Проверять их можно двумя способами:
//   - Issuer — в самом auth-сервисе: у него есть приватный ключ;
//   - JWKSVerifier — в остальных сервисах: они забирают публичные ключи
//     с открытого эндпоинта GET /jwks и проверяют токены локально,
//     без запросов к auth-сервису и mongo.
package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// IssuerName — значение claim iss во всех access-токенах looqa.
const IssuerName = "looqa-auth"

// Claims — проверенные данные access-токена, нужные сервисам.
type Claims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

// ErrInvalidToken — токен не передан, битый, просрочен или подписан не нами.
var ErrInvalidToken = errors.New("invalid token")

// ErrJWKSUnavailable — не удалось забрать ключи с /jwks: auth-сервис
// недоступен. Отдавать клиенту стоит 503, а не 401.
var ErrJWKSUnavailable = errors.New("jwks unavailable")

// tokenClaims — JSON-представление полезной нагрузки токена:
// sub — пользователь, sid — сессия (семейство refresh-токенов).
type tokenClaims struct {
	SID string `json:"sid"`
	jwt.RegisteredClaims
}

// Issuer выпускает access-токены и проверяет их своим приватным ключом.
// Ключ генерируется на старте процесса: перезапуск auth-сервиса делает
// действующие access-токены невалидными, но refresh-токены хранятся
// в mongo и не зависят от ключа — клиенты просто получают новый access
// через POST /refresh.
type Issuer struct {
	key ed25519.PrivateKey
	kid string // идентификатор ключа в JWKS
	ttl time.Duration
}

// NewIssuer генерирует пару ключей Ed25519 и возвращает выпускальщик
// токенов со временем жизни ttl.
func NewIssuer(ttl time.Duration) (*Issuer, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ed25519 key: %w", err)
	}
	sum := sha256.Sum256(pub)
	return &Issuer{
		key: priv,
		kid: hex.EncodeToString(sum[:]),
		ttl: ttl,
	}, nil
}

// TTL возвращает время жизни выпускаемых токенов.
func (i *Issuer) TTL() time.Duration { return i.ttl }

// Issue подписывает access-токен для пользователя userID сессии sessionID
// и возвращает его вместе с моментом истечения.
func (i *Issuer) Issue(userID, sessionID uuid.UUID) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(i.ttl)
	claims := tokenClaims{
		SID: sessionID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    IssuerName,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = i.kid

	signed, err := token.SignedString(i.key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// Verify проверяет подпись и стандартные claim'ы токена (alg, exp, iss)
// и достаёт из него пользователя и сессию.
func (i *Issuer) Verify(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, func(kid string) (any, error) {
		if kid != i.kid {
			return nil, fmt.Errorf("%w: unknown kid %q", ErrInvalidToken, kid)
		}
		return i.key.Public(), nil
	})
}

// JWKS — документ с публичными ключами для эндпоинта GET /jwks.
func (i *Issuer) JWKS() jwks {
	pub, _ := i.key.Public().(ed25519.PublicKey)
	return jwks{Keys: []jwk{{
		Kty: "OKP",
		Crv: "Ed25519",
		X:   base64.RawURLEncoding.EncodeToString(pub),
		Use: "sig",
		Alg: jwt.SigningMethodEdDSA.Alg(),
		Kid: i.kid,
	}}}
}

// jwks/jwk — JSON-структуры по RFC 7517 (набор ключей для Ed25519).
type jwks struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

// parseToken — общая проверка для Issuer и JWKSVerifier: keyfunc
// возвращает публичный ключ по kid из заголовка токена.
func parseToken(tokenStr string, keyfunc func(kid string) (any, error)) (*Claims, error) {
	claims := &tokenClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return keyfunc(kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithIssuer(IssuerName),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		// недоступность auth-сервиса (ошибка keyfunc) — не «битый токен»:
		// её отличает middleware, отвечая 503 вместо 401
		if errors.Is(err, ErrJWKSUnavailable) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("%w: parse sub %q: %v", ErrInvalidToken, claims.Subject, err)
	}
	sessionID, err := uuid.Parse(claims.SID)
	if err != nil {
		return nil, fmt.Errorf("%w: parse sid %q: %v", ErrInvalidToken, claims.SID, err)
	}
	return &Claims{UserID: userID, SessionID: sessionID}, nil
}
