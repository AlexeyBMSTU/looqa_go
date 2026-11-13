package commentservice

import (
	"looqa/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetCommentAll() ([]model.Comment, error)
	GetCommentByID(id uuid.UUID) (model.Comment, error)
	CreateComment(comment model.Comment) error
	DeleteComment(id uuid.UUID) error
}

type Service struct {
	db Repository
}

func NewService(db Repository) *Service {
	return &Service{db: db}
}
