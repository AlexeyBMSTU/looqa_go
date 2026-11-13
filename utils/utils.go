package utils

import (
	"encoding/json"
	"log/slog"
	"looqa/model"
	"net/http"
)

func GetRequestID(r *http.Request) string {
	return r.Context().Value(model.RequestIDKey).(string)
}


func RespondJSON(w http.ResponseWriter, body any, code int) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encode response error", "msg", err)
	}
}

func RespondErrorJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	err := model.ErrorResponse{Code: code, Message: message}
	if encErr := json.NewEncoder(w).Encode(err); encErr != nil {
		slog.Error("encode error response error", "msg", encErr)
	}
}