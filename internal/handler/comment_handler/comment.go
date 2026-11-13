package commenthandler

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
	GetCommentAll() ([]model.Comment, error)
	GetCommentByID(id uuid.UUID) (model.Comment, error)
	CreateComment(comment model.Comment) error
	DeleteComment(id uuid.UUID) error
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
	mux.HandleFunc("GET /comments", h.GetCommentAll)
	mux.HandleFunc("GET /comments/{id}", h.GetCommentByID)
	mux.HandleFunc("POST /comments", h.CreateComment)
	mux.HandleFunc("DELETE /comments/{id}", h.DeleteComment)
}

func (h *Handler) GetCommentAll(w http.ResponseWriter, r *http.Request) {
	comments, error := h.svc.GetCommentAll()
	if error != nil {
		slog.Error("get comment err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, comments, http.StatusOK)
}

func (h *Handler) GetCommentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("get comment by id error parse comment", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	comment, error := h.svc.GetCommentByID(uuid)
	if error != nil {
		slog.Error("get comment by id err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, comment, http.StatusOK)
}

func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	var body model.Comment
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("create comment error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error := h.svc.CreateComment(body)
	if error != nil {
		slog.Error("create comment error", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusOK)
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("delete comment err parse id", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error = h.svc.DeleteComment(uuid)
	if error != nil {
		slog.Error("delete comment err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusNoContent)
}
