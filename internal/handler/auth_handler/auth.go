package authhandler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"looqa/internal/auth"
	"looqa/model"
	"looqa/utils"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Service interface {
	Register(ctx context.Context, user model.User, email, password string) (model.User, error)
	Login(ctx context.Context, email, password string) (model.TokenPair, error)
	Refresh(ctx context.Context, refreshToken string) (model.TokenPair, error)
	Logout(ctx context.Context, sessionID uuid.UUID) error
}

type Handler struct {
	svc    Service
	issuer *auth.Issuer
	limit  *rateLimiter
}

// NewHandler создаёт обработчик HTTP-запросов сервиса аутентификации
// и настраивает логгер с уровнем logLevel. ratePerMinute — лимит запросов
// с одного IP на /reg, /login и /refresh (0 — без лимита).
func NewHandler(svc Service, issuer *auth.Issuer, logLevel slog.Leveler, ratePerMinute int) *Handler {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return &Handler{svc: svc, issuer: issuer, limit: newRateLimiter(ratePerMinute, time.Minute)}
}

// RegisterRoutes регистрирует ручки сервиса на mux:
// POST /reg, /login, /refresh, /logout и публичный GET /jwks.
// Регистрация, вход и обновление токена дополнительно ограничены по частоте.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST /reg", h.limit.wrap(h.Register))
	mux.Handle("POST /login", h.limit.wrap(h.Login))
	mux.Handle("POST /refresh", h.limit.wrap(h.Refresh))
	mux.HandleFunc("POST /logout", h.Logout)
	mux.HandleFunc("GET /jwks", h.JWKS)
}

// registerRequest — тело POST /reg.
type registerRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var body registerRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("register error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	user, err := h.svc.Register(r.Context(), model.User{
		FirstName: body.FirstName,
		LastName:  body.LastName,
	}, body.Email, body.Password)
	if err != nil {
		slog.Error("register error", "msg", err, "requestID", utils.GetRequestID(r))
		respondAuthError(w, err)
		return
	}
	utils.RespondJSON(w, user, http.StatusCreated)
}

// loginRequest — тело POST /login.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("login error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	pair, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		slog.Error("login error", "msg", err, "requestID", utils.GetRequestID(r))
		respondAuthError(w, err)
		return
	}
	utils.RespondJSON(w, pair, http.StatusOK)
}

// refreshRequest — тело POST /refresh.
type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("refresh error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	pair, err := h.svc.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		slog.Error("refresh error", "msg", err, "requestID", utils.GetRequestID(r))
		respondAuthError(w, err)
		return
	}
	utils.RespondJSON(w, pair, http.StatusOK)
}

// Logout отзывает сессию, указанную в access-токене (claim sid):
// Authorization: Bearer <access>. Просроченный access не позволит выйти —
// refresh и так истечёт, а его отзыв требует самого refresh-токена.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	tokenStr, err := auth.BearerToken(r.Header.Get("Authorization"))
	if err != nil {
		slog.Error("logout err parse bearer", "msg", err, "requestID", utils.GetRequestID(r))
		respondUnauthorized(w)
		return
	}

	claims, err := h.issuer.Verify(tokenStr)
	if err != nil {
		slog.Error("logout err verify token", "msg", err, "requestID", utils.GetRequestID(r))
		respondUnauthorized(w)
		return
	}

	if err := h.svc.Logout(r.Context(), claims.SessionID); err != nil {
		slog.Error("logout err", "msg", err, "requestID", utils.GetRequestID(r))
		respondAuthError(w, err)
		return
	}
	utils.RespondJSON(w, nil, http.StatusNoContent)
}

// JWKS отдаёт публичные ключи для проверки access-токенов. Эндпоинт
// публичный: ресурсные сервисы забирают ключи сами, без аутентификации.
func (h *Handler) JWKS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	utils.RespondJSON(w, h.issuer.JWKS(), http.StatusOK)
}

// respondUnauthorized отвечает 401 с вызовом WWW-Authenticate (RFC 9110).
func respondUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="looqa"`)
	utils.RespondErrorJSON(w, "unauthorized", http.StatusUnauthorized)
}

// respondAuthError переводит ошибки сервиса в HTTP-ответы: неверные данные
// при регистрации — 400, неверный логин и неизвестный/истёкший токен — 401,
// занятый email — 409.
func respondAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrInternal):
		utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
	case errors.Is(err, model.ErrInvalid):
		utils.RespondErrorJSON(w, "invalid registration data", http.StatusBadRequest)
	case errors.Is(err, model.ErrUnauthorized), errors.Is(err, model.ErrNotFound):
		respondUnauthorized(w)
	case errors.Is(err, model.ErrConflict):
		utils.RespondErrorJSON(w, "email already registered", http.StatusConflict)
	default:
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
	}
}

// rateLimiter — простое ограничение частоты: не более limit запросов
// с одного IP за фиксированное окно window. Для защиты от брутфорса
// паролей этого достаточно, распределённые лимиты здесь не нужны.
type rateLimiter struct {
	limit  int
	window time.Duration

	mu       sync.Mutex
	counters map[string]*windowCounter
}

type windowCounter struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{limit: limit, window: window, counters: make(map[string]*windowCounter)}
}

// wrap возвращает http.HandlerFunc, отбрасывающий запросы сверх лимита
// с ответом 429. Лимит <= 0 отключает ограничение.
func (l *rateLimiter) wrap(next http.HandlerFunc) http.HandlerFunc {
	if l == nil || l.limit <= 0 {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		l.mu.Lock()
		now := time.Now()
		c := l.counters[ip]
		switch {
		case c == nil || now.Sub(c.start) >= l.window:
			l.counters[ip] = &windowCounter{start: now, count: 1}
			l.evictLocked(now)
			l.mu.Unlock()
		default:
			c.count++
			over := c.count > l.limit
			l.mu.Unlock()
			if over {
				utils.RespondErrorJSON(w, "too many requests", http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	}
}

// evictLocked выкидывает просроченные счётчики, чтобы карта не росла
// бесконечно. Вызывается с удержанным l.mu, только когда создаётся новый IP.
func (l *rateLimiter) evictLocked(now time.Time) {
	if len(l.counters) < 1000 {
		return
	}
	for ip, c := range l.counters {
		if now.Sub(c.start) >= l.window {
			delete(l.counters, ip)
		}
	}
}
