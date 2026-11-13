package posterservice

import (
	"looqa/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetPosterAll() ([]model.Poster, error)
	GetPosterByID(id uuid.UUID) (model.Poster, error)
	CreatePoster(poster model.Poster) error
	DeletePoster(id uuid.UUID) error
}

type Service struct {
	db Repository
}

func NewService(db Repository) *Service {
	return &Service{db: db}
}
