package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIssueVerifyRoundtrip(t *testing.T) {
	issuer, err := NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}

	userID, sessionID := uuid.New(), uuid.New()
	token, expiresAt, err := issuer.Issue(userID, sessionID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	if d := time.Until(expiresAt); d <= 0 || d > time.Minute {
		t.Errorf("срок жизни токена %v вне интервала (0, ttl]", d)
	}

	claims, err := issuer.Verify(token)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if claims.UserID != userID || claims.SessionID != sessionID {
		t.Errorf("claims %+v, ожидали пользователя %s и сессию %s", claims, userID, sessionID)
	}
}

func TestVerifyRejects(t *testing.T) {
	issuer, err := NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	userID, sessionID := uuid.New(), uuid.New()

	tests := []struct {
		name  string
		token string
	}{
		{"garbage", "not-a-jwt"},
		{"empty", ""},
		{"tampered payload", func() string {
			token, _, err := issuer.Issue(userID, sessionID)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			// переворачиваем один символ полезной нагрузки
			if token[20] == 'a' {
				return token[:20] + "b" + token[21:]
			}
			return token[:20] + "a" + token[21:]
		}()},
		{"signed by another key", func() string {
			other, err := NewIssuer(time.Minute)
			if err != nil {
				t.Fatalf("NewIssuer: %v", err)
			}
			token, _, err := other.Issue(userID, sessionID)
			if err != nil {
				t.Fatalf("Issue: %v", err)
			}
			return token
		}()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := issuer.Verify(tt.token); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("ожидали ErrInvalidToken, получили: %v", err)
			}
		})
	}
}

func TestVerifyExpired(t *testing.T) {
	issuer, err := NewIssuer(-2 * time.Minute) // токен рождается просроченным
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}

	token, _, err := issuer.Issue(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	if _, err := issuer.Verify(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("ожидали ErrInvalidToken для просроченного токена, получили: %v", err)
	}
}

func TestJWKSVerifier(t *testing.T) {
	issuer, err := NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}

	// сервер отдаёт текущий набор ключей; позволяет подменить его в тесте
	var current atomic.Value
	current.Store(issuer.JWKS())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, err := json.Marshal(current.Load().(jwks))
		if err != nil {
			t.Errorf("marshal jwks: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(body); err != nil {
			t.Errorf("write jwks: %v", err)
		}
	}))
	t.Cleanup(srv.Close)

	verifier := NewJWKSVerifier(srv.URL)
	userID, sessionID := uuid.New(), uuid.New()

	t.Run("verify by fetched keys", func(t *testing.T) {
		token, _, err := issuer.Issue(userID, sessionID)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}
		claims, err := verifier.Verify(token)
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("claims %+v, ожидали пользователя %s", claims, userID)
		}
	})

	t.Run("refetch on unknown kid", func(t *testing.T) {
		// auth-сервис перезапустился: новый ключ, старый kid неизвестен
		restarted, err := NewIssuer(time.Minute)
		if err != nil {
			t.Fatalf("NewIssuer: %v", err)
		}
		token, _, err := restarted.Issue(userID, sessionID)
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}

		current.Store(restarted.JWKS())
		if _, err := verifier.Verify(token); err != nil {
			t.Fatalf("после перезапуска auth токен должен проверяться: %v", err)
		}
	})

	t.Run("unavailable jwks", func(t *testing.T) {
		dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(dead.Close)

		// синтаксически валидный токен: парсер дойдёт до keyfunc и попробует
		// загрузить ключи — а сервер недоступен
		someone, err := NewIssuer(time.Minute)
		if err != nil {
			t.Fatalf("NewIssuer: %v", err)
		}
		wellFormed, _, err := someone.Issue(uuid.New(), uuid.New())
		if err != nil {
			t.Fatalf("Issue: %v", err)
		}

		broken := NewJWKSVerifier(dead.URL)
		if _, err := broken.Verify(wellFormed); !errors.Is(err, ErrJWKSUnavailable) {
			t.Fatalf("ожидали ErrJWKSUnavailable, получили: %v", err)
		}
	})
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
		err  bool
	}{
		{name: "ok", auth: "Bearer jwt-value", want: "jwt-value"},
		{name: "case insensitive scheme", auth: "bearer jwt-value", want: "jwt-value"},
		{name: "trims spaces", auth: "Bearer   jwt-value  ", want: "jwt-value"},
		{name: "missing header", auth: "", err: true},
		{name: "wrong scheme", auth: "Basic jwt-value", err: true},
		{name: "empty token", auth: "Bearer ", err: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BearerToken(tt.auth)
			if tt.err {
				if !errors.Is(err, ErrInvalidToken) {
					t.Fatalf("ожидали ErrInvalidToken, получили: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("не ожидали ошибку: %v", err)
			}
			if got != tt.want {
				t.Errorf("получили %q, want %q", got, tt.want)
			}
		})
	}
}
