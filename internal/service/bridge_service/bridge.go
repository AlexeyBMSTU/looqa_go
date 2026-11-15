package bridgeservice

import (
	"context"
	"fmt"
	"looqa/model"

	bridge "github.com/AlexeyBMSTU/ai-bridge"
)

func (s *Service) PostMessage(ctx context.Context, message model.MessageRequest, cred model.CredentialAIData) (string, error) {
	bridge := s.repo.InitBridge(ctx, cred)
	res := <-bridge.Send(ctx, message.Message)
	if res.Err != nil {
		return "", fmt.Errorf("error in send to ai %w", res.Err)
	}
	return res.Text, nil
}

func (s *Service) ClearMessages(ctx context.Context, cred model.CredentialAIData) error {
	return s.repo.ClearMessages(ctx, cred)
}

func (s *Service) GetMessages(ctx context.Context, cred model.CredentialAIData) ([]bridge.RoleSchema, error) {
	return s.repo.GetMessages(ctx, cred)
}
