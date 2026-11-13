package userrepository

import (
	"looqa/model"

	"github.com/google/uuid"
)

func (t *InMemoryRepository) GetUserAll() ([]model.User, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	res := make([]model.User, len(t.data))
	for _, v := range t.data {
		res = append(res, v)
	}
	return res, nil
}

func (t *InMemoryRepository) GetUserByID(id uuid.UUID) (model.User, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	if v, ok := t.data[id]; ok {
		return v, nil
	}
	return model.User{}, model.ErrNotFound
}

func (t *InMemoryRepository) CreateUser(user model.User) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[user.Id]; ok {
		return model.ErrConflict
	}
	t.data[user.Id] = user
	return nil
}

func (t *InMemoryRepository) DeleteUser(id uuid.UUID) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if _, ok := t.data[id]; !ok {
		return model.ErrNotFound
	}
	delete(t.data, id)

	return nil
}
