package posterservice

import (
	"errors"
	"fmt"
	"looqa/model"

	"github.com/google/uuid"
)

func (s *Service) GetPosterAll() ([]model.Poster, error) {
	response, error := s.db.GetPosterAll()
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return nil, fmt.Errorf("posters not found: %w", error)
		}
		return nil, error
	}
	return response, nil
}

func (s *Service) GetPosterByID(id uuid.UUID) (model.Poster, error) {
	response, error := s.db.GetPosterByID(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return model.Poster{}, fmt.Errorf("poster by id %s not found: %w", id, error)
		}
		return model.Poster{}, error
	}
	return response, nil
}

func (s *Service) CreatePoster(poster model.Poster) error {
	error := s.db.CreatePoster(poster)
	if error != nil {
		if errors.Is(error, model.ErrConflict) {
			return fmt.Errorf("poster by id %s already created: %w", poster.Id, error)
		}
		return error
	}
	return nil
}

func (s *Service) DeletePoster(id uuid.UUID) error {
	error := s.db.DeletePoster(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return fmt.Errorf("poster by id %s is not found: %w", id, error)
		}
		return error
	}
	return nil
}
