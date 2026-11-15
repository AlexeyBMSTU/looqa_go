package middleware

import (
	"context"
	"log/slog"
	"looqa/model"
	"looqa/utils"
	"net/http"
	"time"
	"uuid"
)

func MiddlewarePanicRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic detected",
					"error", rec,
					"path", r.URL.Path,
					"requestID", utils.GetRequestID(r),
				)
				utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func MiddlewareSetRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		w.Header().Set("X-Request-ID", id.String())
		ctx := context.WithValue(r.Context(), model.RequestIDKey, id.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func MiddlewareLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &model.StatusRecorder{ResponseWriter: w, Status: http.StatusOK}

		next.ServeHTTP(rec, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.Status,
			"duration", time.Since(start).String(),
			"requestID", utils.GetRequestID(r),
		}

		if rec.Status >= http.StatusBadRequest {
			slog.Error(rec.Message, attrs...)
		} else {
			slog.Info(rec.Message, attrs...)
		}
	})
}
