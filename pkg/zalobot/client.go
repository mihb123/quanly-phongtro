package zalobot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL       = "https://bot-api.zaloplatforms.com"
	defaultClientTimeout = 10 * time.Second
	maxResponseBytes     = 1 << 20
)

type BotInfo struct {
	ID          string `json:"id"`
	AccountName string `json:"account_name"`
}

type Client struct {
	HTTPClient *http.Client
	BaseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultClientTimeout}
	}
	return &Client{HTTPClient: httpClient, BaseURL: DefaultBaseURL}
}

func (c *Client) GetMe(ctx context.Context, botToken string) (*BotInfo, error) {
	result, err := c.call(ctx, botToken, "getMe", nil)
	if err != nil {
		return nil, err
	}

	info := &BotInfo{}
	var fields map[string]interface{}
	if err := json.Unmarshal(result, &fields); err == nil {
		info.ID = stringField(fields, "id")
		info.AccountName, _ = fields["account_name"].(string)
	}
	return info, nil
}

func (c *Client) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	_, err := c.call(ctx, botToken, "sendMessage", map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	})
	return err
}

func (c *Client) SendPhoto(ctx context.Context, botToken, chatID, photoURL, caption string) error {
	body := map[string]interface{}{
		"chat_id": chatID,
		"photo":   photoURL,
	}
	if caption != "" {
		body["caption"] = caption
	}
	_, err := c.call(ctx, botToken, "sendPhoto", body)
	return err
}

func (c *Client) SetWebhook(ctx context.Context, botToken, webhookURL, secretToken string) error {
	_, err := c.call(ctx, botToken, "setWebhook", map[string]interface{}{
		"url":          webhookURL,
		"secret_token": secretToken,
	})
	return err
}

func (c *Client) call(ctx context.Context, botToken, method string, payload interface{}) (json.RawMessage, error) {
	httpMethod := http.MethodGet
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("zalobot: %s: encode request: %w", method, err)
		}
		httpMethod = http.MethodPost
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, c.endpoint(botToken, method), body)
	if err != nil {
		return nil, fmt.Errorf("zalobot: %s: build request: %w", method, withoutURL(err))
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalobot: %s: %w", method, withoutURL(err))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("zalobot: %s: read response: %w", method, err)
	}

	var envelope map[string]json.RawMessage
	decodeErr := json.Unmarshal(raw, &envelope)
	if resp.StatusCode != http.StatusOK {
		apiErr := envelopeError(method, resp.StatusCode, envelope, raw)
		if apiErr == nil {
			apiErr = &APIError{Method: method, StatusCode: resp.StatusCode, Body: string(raw)}
		}
		return nil, apiErr
	}
	if decodeErr != nil {
		return nil, fmt.Errorf("zalobot: %s: decode response: %w", method, decodeErr)
	}
	if apiErr := envelopeError(method, resp.StatusCode, envelope, raw); apiErr != nil {
		return nil, apiErr
	}
	return envelope["result"], nil
}

func (c *Client) endpoint(botToken, method string) string {
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return strings.TrimRight(baseURL, "/") + "/bot" + botToken + "/" + method
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func envelopeError(method string, statusCode int, envelope map[string]json.RawMessage, raw []byte) *APIError {
	if envelope == nil {
		return nil
	}

	code := intField(envelope, "error_code")
	if code == 0 {
		code = intField(envelope, "error")
	}
	var ok *bool
	if rawOK, exists := envelope["ok"]; exists {
		var value bool
		if json.Unmarshal(rawOK, &value) == nil {
			ok = &value
		}
	}
	if code == 0 && (ok == nil || *ok) && statusCode == http.StatusOK {
		return nil
	}

	description := textField(envelope, "description")
	if description == "" {
		description = textField(envelope, "message")
	}
	return &APIError{
		Method:      method,
		StatusCode:  statusCode,
		Code:        code,
		Description: description,
		Body:        string(raw),
	}
}

func intField(envelope map[string]json.RawMessage, key string) int {
	var number json.Number
	if err := json.Unmarshal(envelope[key], &number); err != nil {
		return 0
	}
	value, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return 0
	}
	return int(value)
}

func textField(envelope map[string]json.RawMessage, key string) string {
	var value string
	_ = json.Unmarshal(envelope[key], &value)
	return value
}

func withoutURL(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
