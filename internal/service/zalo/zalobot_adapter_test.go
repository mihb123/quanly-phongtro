package zalo

import (
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/pkg/zalobot"
)

var _ ZaloClient = (*zalobot.Client)(nil)

// TestVerifyWebhookSecret_Errors covers missing and undecryptable configured secrets.
func TestVerifyWebhookSecret_Errors(t *testing.T) {
	svc := &zaloServiceImpl{encryptionKey: []byte("z123456789abcdef0123456789abcdef")}

	if err := svc.verifyWebhookSecret(&model.User{}, ""); err == nil {
		t.Errorf("expected missing secret error")
	}

	encrypted, err := security.Encrypt("secret", svc.encryptionKey)
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	badSvc := &zaloServiceImpl{encryptionKey: []byte("another-32-byte-key-for-test-12345")}
	if err := badSvc.verifyWebhookSecret(&model.User{ZaloWebhookSecret: &encrypted}, "secret"); err == nil {
		t.Errorf("expected decrypt error")
	}
}

func TestVerifyWebhookSecret_ComparesSavedSecret(t *testing.T) {
	svc := &zaloServiceImpl{encryptionKey: []byte("z123456789abcdef0123456789abcdef")}
	encrypted, err := security.Encrypt("secret", svc.encryptionKey)
	if err != nil {
		t.Fatalf("encrypt secret: %v", err)
	}
	user := &model.User{ZaloWebhookSecret: &encrypted}

	if err := svc.verifyWebhookSecret(user, "secret"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := svc.verifyWebhookSecret(user, "wrong"); !errors.Is(err, zalobot.ErrInvalidSecretToken) {
		t.Errorf("expected invalid secret error, got %v", err)
	}
	if err := svc.verifyWebhookSecret(&model.User{}, "any"); err != nil {
		t.Errorf("expected legacy config without secret to pass, got %v", err)
	}
}
