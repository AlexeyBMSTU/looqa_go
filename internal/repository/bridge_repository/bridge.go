package bridgerepository

import (
	"context"
	"looqa/model"
	"sync"

	bridge "github.com/AlexeyBMSTU/ai-bridge"
)

type InMemoryRepository struct {
	data map[string]*bridge.BridgeStruct
	mu   sync.RWMutex
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{data: make(map[string]*bridge.BridgeStruct)}
}

func (t *InMemoryRepository) InitBridge(ctx context.Context, cred model.CredentialAIData) *bridge.BridgeStruct {
	t.mu.Lock()
	defer t.mu.Unlock()

	val, ok := t.data[cred.Key]
	if !ok {
		v := bridge.NewAiBridge(cred.Url, cred.Key, cred.Model, 3, 0)
		t.data[cred.Key] = &v
		val = &v
	}
	return val
}

func (t *InMemoryRepository) ClearMessages(ctx context.Context, cred model.CredentialAIData) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.data[cred.Key]; !ok {
		return model.ErrNotFound
	}
	t.data[cred.Key].Clear(ctx)
	return nil
}

func (t *InMemoryRepository) GetMessages(ctx context.Context, cred model.CredentialAIData) ([]bridge.RoleSchema, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if _, ok := t.data[cred.Key]; !ok {
		return nil, model.ErrNotFound
	}
	
	return t.data[cred.Key].GetMessages(ctx), nil
}
