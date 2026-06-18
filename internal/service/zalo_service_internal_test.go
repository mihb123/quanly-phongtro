package service

import (
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// TestNewZaloService_PublicBaseURL covers optional public base URL normalization.
func TestNewZaloService_PublicBaseURL(t *testing.T) {
	svc, err := NewZaloService(nil, nil, nil, nil, nil, nil, nil, nil, "z123456789abcdef0123456789abcdef", "https://example.com/")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	impl := svc.(*zaloServiceImpl)
	if impl.publicBaseURL != "https://example.com" {
		t.Errorf("expected trimmed base URL, got %s", impl.publicBaseURL)
	}
}

// TestStringFromMap covers missing, numeric, and unsupported webhook ID values.
func TestStringFromMap(t *testing.T) {
	values := map[string]interface{}{
		"number": float64(123),
		"bad":    true,
	}

	if got := stringFromMap(values, "missing"); got != "" {
		t.Errorf("expected empty missing value, got %s", got)
	}
	if got := stringFromMap(values, "number"); got != "123" {
		t.Errorf("expected numeric string, got %s", got)
	}
	if got := stringFromMap(values, "bad"); got != "" {
		t.Errorf("expected empty unsupported value, got %s", got)
	}
}

// TestWebhookContextFromPayload_AdditionalShapes covers sender, chat, and attachment variants.
func TestWebhookContextFromPayload_AdditionalShapes(t *testing.T) {
	ctx := webhookContextFromPayload(map[string]interface{}{
		"event_name": "message.photo.received",
		"sender":     map[string]interface{}{"id": float64(123)},
		"message": map[string]interface{}{
			"photo": "https://example.com/photo.jpg",
			"chat": map[string]interface{}{
				"id":        "group-1",
				"name":      "Room House",
				"chat_type": "GROUP",
			},
		},
	})

	if ctx.senderID != "123" || ctx.photoURL == "" || !ctx.isGroupChat || ctx.replyChatID != "group-1" {
		t.Errorf("unexpected webhook context: %+v", ctx)
	}

	ctx = webhookContextFromPayload(map[string]interface{}{
		"message": map[string]interface{}{
			"from": map[string]interface{}{"id": "u1"},
			"chat": map[string]interface{}{"id": "u1"},
			"attachments": []interface{}{
				"bad attachment",
				map[string]interface{}{"type": "video"},
				map[string]interface{}{"type": "image"},
				map[string]interface{}{"type": "image", "payload": map[string]interface{}{"url": "https://example.com/image.jpg"}},
			},
		},
	})

	if ctx.chatID != "" || ctx.replyChatID != "u1" || ctx.photoURL != "https://example.com/image.jpg" {
		t.Errorf("unexpected private context: %+v", ctx)
	}
}

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
