package middleware

import (
	"context"
	"errors"
	"log/slog"
	"looqa/internal/auth"
	"looqa/model"
	"looqa/utils"
	"net/http"
)

// MiddlewareAuth требует валидный JWT access-токен в заголовке
// Authorization: Bearer <token>, проверяет его публичным ключом из JWKS
// auth-сервиса и кладёт claims (looqa/internal/auth.Claims) в контекст
// запроса под model.AuthClaimsKey.
//
// Ответы: 401 — токен не передан/битый/просрочен; 503 — auth-сервис
// недоступен (не удалось забрать ключи), стоит повторить позже.
func MiddlewareAuth(verifier *auth.JWKSVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := auth.BearerToken(r.Header.Get("Authorization"))
			if err != nil {
				respondUnauthorized(w)
				return
			}

			claims, err := verifier.Verify(tokenStr)
			if err != nil {
				if errors.Is(err, auth.ErrJWKSUnavailable) {
					slog.Error("auth service with error", "err", err)
					utils.RespondErrorJSON(w, "auth service unavailable", http.StatusServiceUnavailable)
					return
				}
				respondUnauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), model.AuthClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// respondUnauthorized отвечает 401 с вызовом WWW-Authenticate (RFC 9110).
func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="looqa"`)
	utils.RespondErrorJSON(w, "unauthorized", http.StatusUnauthorized)
}
