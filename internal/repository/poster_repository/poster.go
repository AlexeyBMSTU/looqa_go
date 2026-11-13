package posterrepository

import (
	"looqa/model"

	"github.com/google/uuid"
)


func (t *InMemoryRepository) GetPosterAll() ([]model.Poster, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	res := make([]model.Poster, len(t.data))
	for _, v := range t.data {
		res = append(res, v)
	}
	return res, nil
}

func (t *InMemoryRepository) GetPosterByID(id uuid.UUID) (model.Poster, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if v, ok := t.data[id]; ok {
		return v, nil
	}
	return model.Poster{}, model.ErrNotFound
}

func (t *InMemoryRepository) CreatePoster(poster model.Poster) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[poster.Id]; ok {
		return model.ErrConflict
	}
	t.data[poster.Id] = poster
	return nil
}

func (t *InMemoryRepository) DeletePoster(id uuid.UUID) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[id]; !ok {
		return model.ErrNotFound
	}
	delete(t.data, id)

	return nil
}
