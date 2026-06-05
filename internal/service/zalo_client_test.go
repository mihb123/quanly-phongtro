package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/service"
)

// mockRoundTripper intercepts HTTP requests and returns mock responses
type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req), nil
}

func TestZaloClient_GetMe(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	tests := []struct {
		name          string
		roundTripFunc func(req *http.Request) *http.Response
		expectError   bool
		expectAppID   string
	}{
		{
			name: "Happy path",
			roundTripFunc: func(req *http.Request) *http.Response {
				respBody := map[string]interface{}{
					"error": 0,
					"result": map[string]interface{}{
						"id":           "app-1",
						"account_name": "Test App",
					},
				}
				jsonBytes, _ := json.Marshal(respBody)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
				}
			},
			expectError: false,
			expectAppID: "app-1",
		},
		{
			name: "API returns error",
			roundTripFunc: func(req *http.Request) *http.Response {
				respBody := map[string]interface{}{
					"error":   -216,
					"message": "Invalid token",
				}
				jsonBytes, _ := json.Marshal(respBody)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
				}
			},
			expectError: true,
		},
		{
			name: "Invalid JSON response",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte("invalid json"))),
				}
			},
			expectError: true,
		},
		{
			name: "HTTP status non-200",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
				}
			},
			expectError: true,
		},
	}

	client := service.NewZaloClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = &mockRoundTripper{roundTripFunc: tt.roundTripFunc}
			info, err := client.GetMe(context.Background(), "token")
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if info.AppID != tt.expectAppID {
					t.Errorf("expected AppID %s, got %s", tt.expectAppID, info.AppID)
				}
			}
		})
	}
}

func TestZaloClient_SendMessage(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	tests := []struct {
		name          string
		roundTripFunc func(req *http.Request) *http.Response
		expectError   bool
	}{
		{
			name: "Happy path",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": 0, "message": "Success"}`))),
				}
			},
			expectError: false,
		},
		{
			name: "API Error",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": -216, "message": "Error"}`))),
				}
			},
			expectError: true,
		},
	}

	client := service.NewZaloClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = &mockRoundTripper{roundTripFunc: tt.roundTripFunc}
			err := client.SendMessage(context.Background(), "token", "chat-1", "hello")
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestZaloClient_SendPhoto(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	tests := []struct {
		name          string
		roundTripFunc func(req *http.Request) *http.Response
		expectError   bool
	}{
		{
			name: "Happy path",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": 0, "message": "Success"}`))),
				}
			},
			expectError: false,
		},
		{
			name: "API Error",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": -216, "message": "Error"}`))),
				}
			},
			expectError: true,
		},
	}

	client := service.NewZaloClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = &mockRoundTripper{roundTripFunc: tt.roundTripFunc}
			err := client.SendPhoto(context.Background(), "token", "chat-1", []byte("img"), "caption")
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("Caption empty", func(t *testing.T) {
		http.DefaultTransport = &mockRoundTripper{roundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": 0, "message": "Success"}`))),
			}
		}}
		err := client.SendPhoto(context.Background(), "token", "chat-1", []byte("img"), "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("HTTP non-200 status", func(t *testing.T) {
		http.DefaultTransport = &mockRoundTripper{roundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{}`))),
			}
		}}
		err := client.SendPhoto(context.Background(), "token", "chat-1", []byte("img"), "caption")
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})
}

func TestZaloClient_SetWebhook(t *testing.T) {
	originalTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = originalTransport }()

	tests := []struct {
		name          string
		roundTripFunc func(req *http.Request) *http.Response
		expectError   bool
	}{
		{
			name: "Happy path",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": 0, "message": "Success"}`))),
				}
			},
			expectError: false,
		},
		{
			name: "API Error",
			roundTripFunc: func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusBadRequest,
					Body:       io.NopCloser(bytes.NewReader([]byte(`{"error": -216, "message": "Error"}`))),
				}
			},
			expectError: true,
		},
	}

	client := service.NewZaloClient()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			http.DefaultTransport = &mockRoundTripper{roundTripFunc: tt.roundTripFunc}
			err := client.SetWebhook(context.Background(), "token", "url", "secret")
			if tt.expectError && err == nil {
				t.Errorf("expected error but got nil")
			} else if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	t.Run("OK response but decode error", func(t *testing.T) {
		http.DefaultTransport = &mockRoundTripper{roundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`invalid json`))),
			}
		}}
		err := client.SetWebhook(context.Background(), "token", "url", "secret")
		if err == nil { t.Errorf("expected error, got nil") }
	})

	t.Run("API returns ok=false", func(t *testing.T) {
		http.DefaultTransport = &mockRoundTripper{roundTripFunc: func(req *http.Request) *http.Response {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ok": false}`))),
			}
		}}
		err := client.SetWebhook(context.Background(), "token", "url", "secret")
		if err == nil { t.Errorf("expected error, got nil") }
	})
}
