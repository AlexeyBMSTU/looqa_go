package bridgeservice

import (
	"context"
	"looqa/model"

	bridge "github.com/AlexeyBMSTU/ai-bridge"
)

type Service struct {
	repo Repository
}

type Repository interface {
	InitBridge(ctx context.Context, cred model.CredentialAIData) *bridge.BridgeStruct
	ClearMessages(ctx context.Context, cred model.CredentialAIData) error
	GetMessages(ctx context.Context, cred model.CredentialAIData) ([]bridge.RoleSchema, error)
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}
