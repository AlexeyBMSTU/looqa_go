package userservice

import (
	"errors"
	"fmt"
	"looqa/model"

	"github.com/google/uuid"
)

func (s *Service) GetUserAll() ([]model.User, error) {
	response, error := s.db.GetUserAll()
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return nil, fmt.Errorf("users not found: %w", error)
		}
		return nil, error
	}
	return response, nil
}

func (s *Service) GetUserByID(id uuid.UUID) (model.User, error) {
	response, error := s.db.GetUserByID(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return model.User{}, fmt.Errorf("user by id %s not found: %w", id, error)
		}
		return model.User{}, error
	}
	return response, nil
}

func (s *Service) CreateUser(user model.User) error {
	error := s.db.CreateUser(user)
	if error != nil {
		if errors.Is(error, model.ErrConflict) {
			return fmt.Errorf("user by id %s already created: %w", user.Id, error)
		}
		return error
	}
	return nil
}

func (s *Service) DeleteUser(id uuid.UUID) error {
	error := s.db.DeleteUser(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return fmt.Errorf("user by id %s is not found: %w", id, error)
		}
		return error
	}
	return nil
}
