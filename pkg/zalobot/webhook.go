package zalobot

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	SecretTokenHeader   = "X-Bot-Api-Secret-Token"
	MaxWebhookBodyBytes = 1 << 20

	EventTextReceived        = "message.text.received"
	EventUnsupportedReceived = "message.unsupported.received"
)

type Update struct {
	EventName    string
	SenderID     string
	ChatID       string
	ReplyChatID  string
	GroupName    string
	IsGroup      bool
	Text         string
	PhotoURL     string
	ContactPhone string
}

func ParseUpdate(body []byte) (Update, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Update{}, fmt.Errorf("zalobot: decode webhook: %w", err)
	}
	return updateFromPayload(normalizeWebhookPayload(payload)), nil
}

func VerifySecretToken(headerValue, expected string) error {
	if headerValue == "" {
		return ErrMissingSecretToken
	}
	if expected == "" || subtle.ConstantTimeCompare([]byte(headerValue), []byte(expected)) != 1 {
		return ErrInvalidSecretToken
	}
	return nil
}

// normalizeWebhookPayload unwraps Zalo's current webhook envelope while keeping old test payloads valid.
func normalizeWebhookPayload(payload map[string]interface{}) map[string]interface{} {
	result, ok := payload["result"].(map[string]interface{})
	if !ok {
		return payload
	}
	return result
}

// stringField reads a JSON string or number field as the string ID used by Zalo.
func stringField(value map[string]interface{}, key string) string {
	raw, ok := value[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

// mapField returns a nested JSON object from a generic decoded webhook payload.
func mapField(value map[string]interface{}, key string) map[string]interface{} {
	nested, ok := value[key].(map[string]interface{})
	if !ok {
		return nil
	}
	return nested
}

// updateFromPayload extracts the message fields from official and legacy Zalo payloads.
func updateFromPayload(payload map[string]interface{}) Update {
	message := mapField(payload, "message")
	eventName, _ := payload["event_name"].(string)

	update := Update{EventName: eventName}

	if sender := mapField(payload, "sender"); sender != nil {
		update.SenderID = stringField(sender, "id")
	}
	if message != nil {
		if from := mapField(message, "from"); from != nil {
			update.SenderID = stringField(from, "id")
		}
		if text, ok := message["text"].(string); ok {
			update.Text = strings.TrimSpace(text)
		}
		if photo, ok := message["photo"].(string); ok {
			update.PhotoURL = photo
		}
	}

	if group := mapField(payload, "group"); group != nil {
		update.IsGroup = true
		update.ChatID = stringField(group, "id")
		if name, ok := group["name"].(string); ok {
			update.GroupName = name
		}
	}
	if message != nil {
		if chat := mapField(message, "chat"); chat != nil {
			update.ChatID = stringField(chat, "id")
			if title, ok := chat["title"].(string); ok && title != "" {
				update.GroupName = title
			}
			if name, ok := chat["name"].(string); ok && name != "" {
				update.GroupName = name
			}
			if chatType, ok := chat["chat_type"].(string); ok && strings.EqualFold(chatType, "GROUP") {
				update.IsGroup = true
			}
		}
	}
	if update.GroupName != "" && update.ChatID != "" && update.ChatID != update.SenderID {
		update.IsGroup = true
	}
	if update.ChatID == update.SenderID && !update.IsGroup {
		update.ChatID = ""
	}
	if update.ChatID != "" {
		update.ReplyChatID = update.ChatID
	} else {
		update.ReplyChatID = update.SenderID
	}
	if update.IsGroup {
		update.ReplyChatID = update.ChatID
	}

	if message != nil {
		if attachments, ok := message["attachments"].([]interface{}); ok {
			for _, att := range attachments {
				attMap, ok := att.(map[string]interface{})
				if !ok {
					continue
				}
				attType, _ := attMap["type"].(string)
				payloadMap := mapField(attMap, "payload")

				if attType == "image" && update.PhotoURL == "" {
					if payloadMap != nil {
						if urlStr, ok := payloadMap["url"].(string); ok && urlStr != "" {
							update.PhotoURL = urlStr
						}
					}
				} else if attType != "image" {
					if payloadMap != nil {
						if phone, ok := payloadMap["phone"].(string); ok && phone != "" {
							update.ContactPhone = phone
						} else if phoneNum, ok := payloadMap["phone_number"].(string); ok && phoneNum != "" {
							update.ContactPhone = phoneNum
						} else if sharedPhone, ok := payloadMap["shared_phone"].(string); ok && sharedPhone != "" {
							update.ContactPhone = sharedPhone
						}
					}
				}
			}
		}
	}

	return update
}
