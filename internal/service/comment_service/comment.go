package commentservice

import (
	"errors"
	"fmt"
	"looqa/model"

	"github.com/google/uuid"
)

func (s *Service) GetCommentAll() ([]model.Comment, error) {
	response, error := s.db.GetCommentAll()
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return nil, fmt.Errorf("comments not found: %w", error)
		}
		return nil, error
	}
	return response, nil
}

func (s *Service) GetCommentByID(id uuid.UUID) (model.Comment, error) {
	response, error := s.db.GetCommentByID(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return model.Comment{}, fmt.Errorf("comment by id %s not found: %w", id, error)
		}
		return model.Comment{}, error
	}
	return response, nil
}

func (s *Service) CreateComment(comment model.Comment) error {
	error := s.db.CreateComment(comment)
	if error != nil {
		if errors.Is(error, model.ErrConflict) {
			return fmt.Errorf("comment by id %s already created: %w", comment.Id, error)
		}
		return error
	}
	return nil
}

func (s *Service) DeleteComment(id uuid.UUID) error {
	error := s.db.DeleteComment(id)
	if error != nil {
		if errors.Is(error, model.ErrNotFound) {
			return fmt.Errorf("comment by id %s is not found: %w", id, error)
		}
		return error
	}
	return nil
}
