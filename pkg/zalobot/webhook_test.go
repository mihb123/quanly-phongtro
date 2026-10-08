package zalobot

import (
	"errors"
	"testing"
)

func TestParseUpdate(t *testing.T) {
	tests := []struct {
		name string
		body string
		want Update
	}{
		{
			name: "private text",
			body: `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"u1","chat_type":"PRIVATE"},"text":" botoi "}}}`,
			want: Update{EventName: EventTextReceived, SenderID: "u1", ReplyChatID: "u1", Text: "botoi"},
		},
		{
			name: "group text",
			body: `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"group-1","chat_type":"GROUP"},"text":"bot ơi"}}}`,
			want: Update{EventName: EventTextReceived, SenderID: "u1", ChatID: "group-1", ReplyChatID: "group-1", IsGroup: true, Text: "bot ơi"},
		},
		{
			name: "legacy group photo with numeric sender",
			body: `{"event_name":"message.photo.received","sender":{"id":123},"message":{"photo":"https://example.com/photo.jpg","chat":{"id":"group-1","name":"Room House","chat_type":"GROUP"}}}`,
			want: Update{EventName: "message.photo.received", SenderID: "123", ChatID: "group-1", ReplyChatID: "group-1", GroupName: "Room House", IsGroup: true, PhotoURL: "https://example.com/photo.jpg"},
		},
		{
			name: "attachments",
			body: `{"message":{"from":{"id":"u1"},"chat":{"id":"u1"},"attachments":["bad",{"type":"video"},{"type":"image"},{"type":"image","payload":{"url":"https://example.com/image.jpg"}},{"type":"contact","payload":{"phone_number":"0912345678"}}]}}`,
			want: Update{SenderID: "u1", ReplyChatID: "u1", PhotoURL: "https://example.com/image.jpg", ContactPhone: "0912345678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUpdate([]byte(tt.body))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}

	if _, err := ParseUpdate([]byte("not json")); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}

func TestStringField(t *testing.T) {
	values := map[string]interface{}{"number": float64(123), "bad": true}

	if got := stringField(values, "missing"); got != "" {
		t.Errorf("expected empty missing value, got %s", got)
	}
	if got := stringField(values, "number"); got != "123" {
		t.Errorf("expected numeric string, got %s", got)
	}
	if got := stringField(values, "bad"); got != "" {
		t.Errorf("expected empty unsupported value, got %s", got)
	}
}

func TestVerifySecretToken(t *testing.T) {
	if err := VerifySecretToken("", "secret"); !errors.Is(err, ErrMissingSecretToken) {
		t.Errorf("expected missing token error, got %v", err)
	}
	if err := VerifySecretToken("wrong", "secret"); !errors.Is(err, ErrInvalidSecretToken) {
		t.Errorf("expected invalid token error, got %v", err)
	}
	if err := VerifySecretToken("secret", ""); !errors.Is(err, ErrInvalidSecretToken) {
		t.Errorf("expected empty expected secret to be rejected, got %v", err)
	}
	if err := VerifySecretToken("secret", "secret"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestIsAuthError(t *testing.T) {
	if IsAuthError(nil) || IsAuthError(errors.New("random error")) {
		t.Error("expected non-auth errors")
	}
	for _, err := range []error{
		errors.New("some error -216"),
		errors.New("INVALID ACCESS TOKEN"),
		errors.New("unauthorized request"),
		&APIError{Method: "sendMessage", StatusCode: 200, Code: -216},
		&APIError{Method: "getMe", StatusCode: 401},
	} {
		if !IsAuthError(err) {
			t.Errorf("expected auth error for %v", err)
		}
	}
}
