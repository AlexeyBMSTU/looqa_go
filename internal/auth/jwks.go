package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// JWKSVerifier проверяет access-токены публичными ключами, которые он
// забирает с GET /jwks auth-сервиса. Ключи кэшируются в памяти; при
// встрече с неизвестным kid (auth-сервис перезапустился с новым ключом)
// набор один раз перезагружается.
type JWKSVerifier struct {
	url    string
	client *http.Client

	mu   sync.RWMutex
	keys map[string]ed25519.PublicKey
}

// NewJWKSVerifier возвращает проверяльщик, читающий ключи с jwksURL.
// Сами ключи не качает — первый запрос к защищённому эндпоинту лениво
// их загрузит, поэтому запуск сервиса возможен раньше auth-сервиса.
func NewJWKSVerifier(jwksURL string) *JWKSVerifier {
	return &JWKSVerifier{
		url:    jwksURL,
		client: &http.Client{Timeout: 3 * time.Second},
		keys:   make(map[string]ed25519.PublicKey),
	}
}

// Verify проверяет подпись и claim'ы токена (см. parseToken). Если ключи
// ещё не загружены или kid неизвестен — пробует перечитать /jwks.
// Недоступность auth-сервиса возвращает ErrJWKSUnavailable.
func (v *JWKSVerifier) Verify(tokenStr string) (*Claims, error) {
	return parseToken(tokenStr, func(kid string) (any, error) {
		if key := v.key(kid); key != nil {
			return key, nil
		}
		if err := v.refetch(); err != nil {
			return nil, err
		}
		if key := v.key(kid); key != nil {
			return key, nil
		}
		return nil, fmt.Errorf("%w: unknown kid %q", ErrInvalidToken, kid)
	})
}

// key возвращает закэшированный ключ или nil.
func (v *JWKSVerifier) key(kid string) ed25519.PublicKey {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.keys[kid]
}

// refetch перечитывает /jwks и заменяет кэш ключей.
func (v *JWKSVerifier) refetch() error {
	resp, err := v.client.Get(v.url)
	if err != nil {
		return fmt.Errorf("%w: get %s: %v", ErrJWKSUnavailable, v.url, err)
	}
	defer func() {
		// ошибка закрытия тела ответа не влияет на уже прочитанные ключи
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: get %s: status %d", ErrJWKSUnavailable, v.url, resp.StatusCode)
	}

	var doc jwks
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("%w: decode jwks: %v", ErrJWKSUnavailable, err)
	}

	keys := make(map[string]ed25519.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return fmt.Errorf("%w: decode key %q: %v", ErrJWKSUnavailable, k.Kid, err)
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	if len(keys) == 0 {
		return fmt.Errorf("%w: no usable keys at %s", ErrJWKSUnavailable, v.url)
	}

	v.mu.Lock()
	v.keys = keys
	v.mu.Unlock()
	return nil
}

// BearerToken достаёт access-токен из заголовка
// Authorization: Bearer <token>. Пустой префикс или отсутствие заголовка —
// ошибка ErrInvalidToken.
func BearerToken(authorization string) (string, error) {
	const prefix = "bearer "
	if len(authorization) <= len(prefix) || !strings.EqualFold(authorization[:len(prefix)], prefix) {
		return "", fmt.Errorf("%w: missing bearer token", ErrInvalidToken)
	}
	return strings.TrimSpace(authorization[len(prefix):]), nil
}
