package userhandler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"looqa/model"
	"looqa/utils"
	"net/http"
	"os"

	"github.com/google/uuid"
)


type Service interface {
	GetUserAll() ([]model.User, error)
	GetUserByID(id uuid.UUID) (model.User, error)
	CreateUser(user model.User) error
	DeleteUser(id uuid.UUID) error
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
	mux.HandleFunc("GET /users", h.GetUserAll)
	mux.HandleFunc("GET /users/{id}", h.GetUserByID)
	mux.HandleFunc("POST /users", h.CreateUser)
	mux.HandleFunc("DELETE /users/{id}", h.DeleteUser)
}

func (h *Handler) GetUserAll(w http.ResponseWriter, r *http.Request) {
	users, error := h.svc.GetUserAll()
	if error != nil {
		slog.Error("get user err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, users, http.StatusOK)
}

func (h *Handler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("get user by id error parse user", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	user, error := h.svc.GetUserByID(uuid)
	if error != nil {
		slog.Error("get user by id err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, user, http.StatusOK)
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var body model.User
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("create user error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error := h.svc.CreateUser(body)
	if error != nil {
		slog.Error("create user error", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusOK)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("delete user err parse id", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error = h.svc.DeleteUser(uuid)
	if error != nil {
		slog.Error("delete user err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusNoContent)
}