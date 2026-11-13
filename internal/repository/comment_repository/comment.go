package commentrepository

import (
	"looqa/model"

	"github.com/google/uuid"
)

func (t *InMemoryRepository) GetCommentAll() ([]model.Comment, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	res := make([]model.Comment, len(t.data))
	for _, v := range t.data {
		res = append(res, v)
	}

	return res, nil
}

func (t *InMemoryRepository) GetCommentByID(id uuid.UUID) (model.Comment, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if v, ok := t.data[id]; ok {
		return v, nil
	}

	return model.Comment{}, model.ErrNotFound
}

func (t *InMemoryRepository) CreateComment(comment model.Comment) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[comment.Id]; ok {
		return model.ErrConflict
	}
	t.data[comment.Id] = comment
	return nil
}

func (t *InMemoryRepository) DeleteComment(id uuid.UUID) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[id]; !ok {
		return model.ErrNotFound
	}
	delete(t.data, id)

	return nil
}
