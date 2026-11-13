package posterhandler

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
	GetPosterAll() ([]model.Poster, error)
	GetPosterByID(id uuid.UUID) (model.Poster, error)
	CreatePoster(poster model.Poster) error
	DeletePoster(id uuid.UUID) error
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
	mux.HandleFunc("GET /posters", h.GetPosterAll)
	mux.HandleFunc("GET /posters/{id}", h.GetPosterByID)
	mux.HandleFunc("POST /posters", h.CreatePoster)
	mux.HandleFunc("DELETE /posters/{id}", h.DeletePoster)
}

func (h *Handler) GetPosterAll(w http.ResponseWriter, r *http.Request) {
	posters, error := h.svc.GetPosterAll()
	if error != nil {
		slog.Error("get poster err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, posters, http.StatusOK)
}

func (h *Handler) GetPosterByID(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("get poster by id error parse poster", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	poster, error := h.svc.GetPosterByID(uuid)
	if error != nil {
		slog.Error("get poster by id err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, poster, http.StatusOK)
}

func (h *Handler) CreatePoster(w http.ResponseWriter, r *http.Request) {
	var body model.Poster
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		slog.Error("create poster error parse", "msg", err, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error := h.svc.CreatePoster(body)
	if error != nil {
		slog.Error("create poster error", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusOK)
}

func (h *Handler) DeletePoster(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	uuid, error := uuid.Parse(id)
	if error != nil {
		slog.Error("delete poster err parse id", "msg", error, "requestID", utils.GetRequestID(r))
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}

	error = h.svc.DeletePoster(uuid)
	if error != nil {
		slog.Error("delete poster err", "msg", error, "requestID", utils.GetRequestID(r))
		if errors.Is(error, model.ErrInternal) {
			utils.RespondErrorJSON(w, "internal server error", http.StatusInternalServerError)
			return
		}
		utils.RespondErrorJSON(w, "bad request", http.StatusBadRequest)
		return
	}
	utils.RespondJSON(w, nil, http.StatusNoContent)
}
