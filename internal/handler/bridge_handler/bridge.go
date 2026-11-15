package bridgehandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"looqa/model"
	"looqa/utils"
	"net/http"
	"os"

	bridge "github.com/AlexeyBMSTU/ai-bridge"
)

type Service interface {
	PostMessage(context context.Context, message model.MessageRequest, cred model.CredentialAIData) (string, error)
	ClearMessages(context context.Context, cred model.CredentialAIData) error
	GetMessages(ctx context.Context, cred model.CredentialAIData) ([]bridge.RoleSchema, error)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service, logLevel slog.Leveler) *Handler {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /message", h.PostMessage)
	mux.HandleFunc("POST /clear", h.ClearMessages)
	mux.HandleFunc("GET /message", h.GetMessages)
}

func getCred(w http.ResponseWriter, r *http.Request) (model.CredentialAIData, error) {
	key := r.Header.Get("X-API-Key")
	modelName := r.Header.Get("X-Model")
	url := r.Header.Get("X-Provider-URL")

	if key == "" || modelName == "" || url == "" {
		slog.Error("clear message error parse credentials", "msg", "key, model, url is required", "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return model.CredentialAIData{}, fmt.Errorf("key or model or url is nil")
	}
	return model.CredentialAIData{Key: key, Model: modelName, Url: url}, nil
}

func (h *Handler) GetMessages(w http.ResponseWriter, r *http.Request) {
	var body model.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("clear message error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	cred, error := getCred(w, r)
	if error != nil {
		utils.RespondErrorJSON(w, "bad request in credentials", http.StatusBadRequest)
		return
	}

	dialog, err := h.svc.GetMessages(r.Context(), cred)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			utils.RespondJSON(w, []any{}, http.StatusOK)
			return
		}

		slog.Error("clear message error", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
		return
	}

	utils.RespondJSON(w, dialog, http.StatusOK)
}

func (h *Handler) ClearMessages(w http.ResponseWriter, r *http.Request) {
	var body model.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("clear message error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	cred, error := getCred(w, r)
	if error != nil {
		utils.RespondErrorJSON(w, "bad request in credentials", http.StatusBadRequest)
		return
	}

	if err := h.svc.ClearMessages(r.Context(), cred); err != nil {
		slog.Error("clear message error", "msg", err, "requestID", utils.GetRequestID(r))
		if errors.Is(err, model.ErrNotFound) {
			utils.RespondErrorJSON(w, "messages doesn't exists", http.StatusBadRequest)
			return
		}
		utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
		return
	}

	utils.RespondJSON(w, nil, http.StatusNoContent)
}

func (h *Handler) PostMessage(w http.ResponseWriter, r *http.Request) {
	var body model.MessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("post message error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	cred, error := getCred(w, r)
	if error != nil {
		utils.RespondErrorJSON(w, "bad request in credentials", http.StatusBadRequest)
		return
	}

	response, err := h.svc.PostMessage(r.Context(), body, cred)
	if err != nil {
		slog.Error("post message error", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
		return
	}
	utils.RespondJSON(w, response, http.StatusOK)
}
