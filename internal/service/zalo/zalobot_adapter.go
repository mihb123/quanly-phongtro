package zalo

import (
	"context"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/zalobot"
)

type ZaloClient interface {
	GetMe(ctx context.Context, botToken string) (*zalobot.BotInfo, error)
	SendMessage(ctx context.Context, botToken, chatID, text string) error
	SendPhoto(ctx context.Context, botToken, chatID, photoURL, caption string) error
	SetWebhook(ctx context.Context, botToken, webhookURL, secretToken string) error
}

var transactionImageDownloader = &zalobot.ImageDownloader{}

// verifyWebhookSecret validates configured Zalo webhook secrets and permits legacy configs without a saved secret.
func (s *zaloServiceImpl) verifyWebhookSecret(user *model.User, secretTokenHeader string) error {
	if secretTokenHeader == "" {
		return zalobot.ErrMissingSecretToken
	}
	if user.ZaloWebhookSecret == nil || *user.ZaloWebhookSecret == "" {
		return nil
	}
	plainSecret, err := security.Decrypt(*user.ZaloWebhookSecret, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt webhook secret failed: %w", err)
	}
	return zalobot.VerifySecretToken(secretTokenHeader, plainSecret)
}
