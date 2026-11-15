package authhandler

import (
	"context"
	"encoding/json"
	"log/slog"
	"looqa/internal/auth"
	"looqa/model"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockService — подмена Service для тестов хендлеров: запоминает переданное
// и возвращает заданные pair/user/err.
type mockService struct {
	pair     model.TokenPair
	user     model.User
	err      error
	gotSID   uuid.UUID // session_id, переданный в Logout
	gotToken string    // refresh-токен, переданный в Refresh
}

func (m *mockService) Register(_ context.Context, _ model.User, _, _ string) (model.User, error) {
	return m.user, m.err
}

func (m *mockService) Login(_ context.Context, _, _ string) (model.TokenPair, error) {
	return m.pair, m.err
}

func (m *mockService) Refresh(_ context.Context, refreshToken string) (model.TokenPair, error) {
	m.gotToken = refreshToken
	return m.pair, m.err
}

func (m *mockService) Logout(_ context.Context, sessionID uuid.UUID) error {
	m.gotSID = sessionID
	return m.err
}

// newTestHandler собирает хендлер с настоящим выпускальщиком токенов
// (issuer нужен для /logout и /jwks) и моком сервиса. Возвращает mux.
func newTestHandler(t *testing.T, svc Service) (*http.ServeMux, *auth.Issuer) {
	t.Helper()
	issuer, err := auth.NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	h := NewHandler(svc, issuer, slog.LevelError, 0) // 0 — без rate limit
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux, issuer
}

func TestLoginAndRefresh(t *testing.T) {
	pair := model.TokenPair{
		AccessToken:  "jwt",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		ExpiresIn:    900,
	}

	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		svcErr     error
		wantStatus int
	}{
		{"login ok", http.MethodPost, "/login", `{"email":"ivan@looqa.dev","password":"secret123"}`, nil, http.StatusOK},
		{"login bad json", http.MethodPost, "/login", `not-json`, nil, http.StatusBadRequest},
		{"login wrong password", http.MethodPost, "/login", `{"email":"ivan@looqa.dev","password":"wrong"}`, model.ErrUnauthorized, http.StatusUnauthorized},
		{"login internal", http.MethodPost, "/login", `{"email":"ivan@looqa.dev","password":"secret123"}`, model.ErrInternal, http.StatusInternalServerError},
		{"refresh ok", http.MethodPost, "/refresh", `{"refresh_token":"abc"}`, nil, http.StatusOK},
		{"refresh bad json", http.MethodPost, "/refresh", ``, nil, http.StatusBadRequest},
		{"refresh unknown token", http.MethodPost, "/refresh", `{"refresh_token":"abc"}`, model.ErrUnauthorized, http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockService{pair: pair, err: tt.svcErr}
			mux, _ := newTestHandler(t, svc)

			req := httptest.NewRequest(tt.method, tt.target, strings.NewReader(tt.body))
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus != http.StatusOK {
				return
			}

			var got model.TokenPair
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got != pair {
				t.Errorf("body = %+v, want %+v", got, pair)
			}
			if tt.target == "/refresh" && svc.gotToken != "abc" {
				t.Errorf("сервис получил refresh %q вместо %q", svc.gotToken, "abc")
			}
		})
	}
}

func TestLogout(t *testing.T) {
	userID, sessionID := uuid.New(), uuid.New()

	t.Run("ok", func(t *testing.T) {
		svc := &mockService{}
		mux, issuer := newTestHandler(t, svc)

		access, _, err := issuer.Issue(userID, sessionID)
		if err != nil {
			t.Fatalf("issue: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.Header.Set("Authorization", "Bearer "+access)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if svc.gotSID != sessionID {
			t.Errorf("отзывают сессию %s вместо %s", svc.gotSID, sessionID)
		}
	})

	tests := []struct {
		name string
		auth string
	}{
		{"no Authorization", ""},
		{"not a bearer", sessionID.String()},
		{"garbage token", "Bearer garbage"},
		{"foreign token", "Bearer eyJhbGciOiJFUzI1NiJ9.eyJpc3MiOiJsb29xYS1hdXRoIn0.Zm9v"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux, _ := newTestHandler(t, &mockService{})

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if wa := rec.Header().Get("WWW-Authenticate"); wa == "" {
				t.Error("у 401 нет заголовка WWW-Authenticate")
			}
		})
	}
}

func TestJWKS(t *testing.T) {
	mux, issuer := newTestHandler(t, &mockService{})

	// /jwks отдаёт ключи, пригодные для проверки токенов
	jwksSrv := httptest.NewServer(mux)
	t.Cleanup(jwksSrv.Close)

	userID, sessionID := uuid.New(), uuid.New()
	access, _, err := issuer.Issue(userID, sessionID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	verifier := auth.NewJWKSVerifier(jwksSrv.URL + "/jwks")
	claims, err := verifier.Verify(access)
	if err != nil {
		t.Fatalf("токен не проверяется ключами из /jwks: %v", err)
	}
	if claims.UserID != userID || claims.SessionID != sessionID {
		t.Errorf("claims %+v, ожидали пользователя %s и сессию %s", claims, userID, sessionID)
	}
}

func TestTokensRoutesRemoved(t *testing.T) {
	mux, _ := newTestHandler(t, &mockService{})

	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		req := httptest.NewRequest(method, "/tokens", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed && rec.Code != http.StatusNotFound {
			t.Errorf("%s /tokens: status = %d, want 405/404", method, rec.Code)
		}
	}
}

func TestRateLimit(t *testing.T) {
	svc := &mockService{user: model.User{Id: uuid.New()}}
	issuer, err := auth.NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	h := NewHandler(svc, issuer, slog.LevelError, 2) // не более 2 запросов в минуту

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	do := func() int {
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{}`))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := do(); code != http.StatusOK {
		t.Fatalf("первый запрос: %d, want 200", code)
	}
	if code := do(); code != http.StatusOK {
		t.Fatalf("второй запрос: %d, want 200", code)
	}
	if code := do(); code != http.StatusTooManyRequests {
		t.Fatalf("третий запрос: %d, want 429", code)
	}

	// остальные ручки лимит не трогают
	req := httptest.NewRequest(http.MethodGet, "/jwks", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /jwks под лимитом: %d, want 200", rec.Code)
	}
}
