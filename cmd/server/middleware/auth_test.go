package middleware

import (
	"encoding/json"
	"looqa/internal/auth"
	"looqa/model"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// newAuthMux поднимает тестовый «ресурсный сервис»: GET /data под
// MiddlewareAuth с ключами из jwksURL.
func newAuthMux(t *testing.T, jwksURL string) http.Handler {
	t.Helper()
	return MiddlewareAuth(auth.NewJWKSVerifier(jwksURL))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := r.Context().Value(model.AuthClaimsKey).(*auth.Claims)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(claims); err != nil {
			t.Errorf("encode claims: %v", err)
		}
	}))
}

func TestMiddlewareAuth(t *testing.T) {
	issuer, err := auth.NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}

	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(issuer.JWKS()); err != nil {
			t.Errorf("encode jwks: %v", err)
		}
	}))
	t.Cleanup(jwksSrv.Close)

	userID, sessionID := uuid.New(), uuid.New()
	access, _, err := issuer.Issue(userID, sessionID)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	tests := []struct {
		name       string
		authorizat string
		wantStatus int
	}{
		{"valid token", "Bearer " + access, http.StatusOK},
		{"no header", "", http.StatusUnauthorized},
		{"not a bearer", access, http.StatusUnauthorized},
		{"garbage token", "Bearer garbage", http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/data", nil)
			if tt.authorizat != "" {
				req.Header.Set("Authorization", tt.authorizat)
			}

			newAuthMux(t, jwksSrv.URL).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var claims auth.Claims
				if err := json.NewDecoder(rec.Body).Decode(&claims); err != nil {
					t.Fatalf("decode claims: %v", err)
				}
				if claims.UserID != userID || claims.SessionID != sessionID {
					t.Errorf("claims %+v, ожидали пользователя %s и сессию %s", claims, userID, sessionID)
				}
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if wa := rec.Header().Get("WWW-Authenticate"); wa == "" {
					t.Error("у 401 нет заголовка WWW-Authenticate")
				}
			}
		})
	}
}

func TestMiddlewareAuthJWKSUnavailable(t *testing.T) {
	dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(dead.Close)

	// валидный по форме токен: middleware дойдёт до загрузки ключей
	// и упрётся в недоступный auth-сервис → 503, а не 401
	issuer, err := auth.NewIssuer(time.Minute)
	if err != nil {
		t.Fatalf("NewIssuer: %v", err)
	}
	token, _, err := issuer.Issue(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	newAuthMux(t, dead.URL).ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
