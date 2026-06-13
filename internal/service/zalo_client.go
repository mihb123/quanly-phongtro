package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ZaloAppInfo struct {
	AppID       string `json:"app_id"`
	DisplayName string `json:"display_name"`
}

type ZaloClient interface {
	GetMe(ctx context.Context, botToken string) (*ZaloAppInfo, error)
	SendMessage(ctx context.Context, botToken, chatID, text string) error
	SendPhoto(ctx context.Context, botToken, chatID, photoURL, caption string) error
	SetWebhook(ctx context.Context, botToken, webhookUrl, secretToken string) error
}

type zaloClientImpl struct {
	client *http.Client
}

func NewZaloClient() ZaloClient {
	return &zaloClientImpl{
		client: &http.Client{},
	}
}

func (c *zaloClientImpl) GetMe(ctx context.Context, botToken string) (*ZaloAppInfo, error) {
	url := fmt.Sprintf("https://bot-api.zaloplatforms.com/bot%s/getMe", botToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("zalo api error: status %d", resp.StatusCode)
	}

	// Actually let's just decode into a map and log or parse what we can
	var generic map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&generic); err != nil {
		return nil, err
	}

	// Check for Zalo structure
	if errCode, ok := generic["error"].(float64); ok && errCode != 0 {
		return nil, fmt.Errorf("zalo api returned error: %v", generic["message"])
	}

	appInfo := &ZaloAppInfo{}
	if result, ok := generic["result"].(map[string]interface{}); ok {
		if id, ok := result["id"].(string); ok {
			appInfo.AppID = id
		}
		if name, ok := result["account_name"].(string); ok {
			appInfo.DisplayName = name
		}
	}

	return appInfo, nil
}

func (c *zaloClientImpl) SendMessage(ctx context.Context, botToken, chatID, text string) error {
	url := fmt.Sprintf("https://bot-api.zaloplatforms.com/bot%s/sendMessage", botToken)

	body := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zalo api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if val, ok := res["ok"].(bool); ok && !val {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}
	if errCode, ok := res["error"].(float64); ok && errCode != 0 {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}

	return nil
}

func (c *zaloClientImpl) SendPhoto(ctx context.Context, botToken, chatID, photoURL, caption string) error {
	url := fmt.Sprintf("https://bot-api.zaloplatforms.com/bot%s/sendPhoto", botToken)

	body := map[string]interface{}{
		"chat_id": chatID,
		"photo":   photoURL,
	}
	if caption != "" {
		body["caption"] = caption
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zalo api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	if val, ok := res["ok"].(bool); ok && !val {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}
	if errCode, ok := res["error"].(float64); ok && errCode != 0 {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}

	return nil
}

func (c *zaloClientImpl) SetWebhook(ctx context.Context, botToken, webhookUrl, secretToken string) error {
	url := fmt.Sprintf("https://bot-api.zaloplatforms.com/bot%s/setWebhook", botToken)

	body := map[string]interface{}{
		"url":          webhookUrl,
		"secret_token": secretToken,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("zalo api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var res map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	if val, ok := res["ok"].(bool); ok && !val {
		return fmt.Errorf("zalo api returned error: %+v", res)
	}

	return nil
}
