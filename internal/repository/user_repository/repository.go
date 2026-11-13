package userrepository

import (
	"looqa/model"
	"sync"

	"github.com/google/uuid"
)

type InMemoryRepository struct {
	data  map[uuid.UUID]model.User
	mutex sync.RWMutex
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		data: make(map[uuid.UUID]model.User),
	}
}
