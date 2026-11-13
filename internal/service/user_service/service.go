package userservice

import (
	"looqa/model"

	"github.com/google/uuid"
)

type Repository interface {
	GetUserAll() ([]model.User, error)
	GetUserByID(id uuid.UUID) (model.User, error)
	CreateUser(user model.User) error
	DeleteUser(id uuid.UUID) error
}

type Service struct {
	db Repository
}

func NewService(db Repository) *Service {
	return &Service{db: db}
}
