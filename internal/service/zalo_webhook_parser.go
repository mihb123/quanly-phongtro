package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type webhookMessageContext struct {
	eventName    string
	message      map[string]interface{}
	senderID     string
	chatID       string
	replyChatID  string
	groupName    string
	isGroupChat  bool
	text         string
	photoURL     string
	contactPhone string
}

// normalizeWebhookPayload unwraps Zalo's current webhook envelope while keeping old test payloads valid.
func normalizeWebhookPayload(payload map[string]interface{}) map[string]interface{} {
	result, ok := payload["result"].(map[string]interface{})
	if !ok {
		return payload
	}
	return result
}

// stringFromMap reads a JSON string or number field as the string ID used by Zalo.
func stringFromMap(value map[string]interface{}, key string) string {
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

// mapFromMap returns a nested JSON object from a generic decoded webhook payload.
func mapFromMap(value map[string]interface{}, key string) map[string]interface{} {
	nested, ok := value[key].(map[string]interface{})
	if !ok {
		return nil
	}
	return nested
}

// webhookContextFromPayload extracts the fields this service needs from official and legacy Zalo payloads.
func webhookContextFromPayload(payload map[string]interface{}) webhookMessageContext {
	message := mapFromMap(payload, "message")
	eventName, _ := payload["event_name"].(string)

	ctx := webhookMessageContext{
		eventName: eventName,
		message:   message,
	}

	if sender := mapFromMap(payload, "sender"); sender != nil {
		ctx.senderID = stringFromMap(sender, "id")
	}
	if message != nil {
		if from := mapFromMap(message, "from"); from != nil {
			ctx.senderID = stringFromMap(from, "id")
		}
		if text, ok := message["text"].(string); ok {
			ctx.text = strings.TrimSpace(text)
		}
		if photo, ok := message["photo"].(string); ok {
			ctx.photoURL = photo
		}
	}

	if group := mapFromMap(payload, "group"); group != nil {
		ctx.isGroupChat = true
		ctx.chatID = stringFromMap(group, "id")
		if name, ok := group["name"].(string); ok {
			ctx.groupName = name
		}
	}
	if message != nil {
		if chat := mapFromMap(message, "chat"); chat != nil {
			ctx.chatID = stringFromMap(chat, "id")
			if title, ok := chat["title"].(string); ok && title != "" {
				ctx.groupName = title
			}
			if name, ok := chat["name"].(string); ok && name != "" {
				ctx.groupName = name
			}
			if chatType, ok := chat["chat_type"].(string); ok && strings.EqualFold(chatType, "GROUP") {
				ctx.isGroupChat = true
			}
		}
	}
	if ctx.groupName != "" && ctx.chatID != "" && ctx.chatID != ctx.senderID {
		ctx.isGroupChat = true
	}
	if ctx.chatID == ctx.senderID && !ctx.isGroupChat {
		ctx.chatID = ""
	}
	if ctx.chatID != "" {
		ctx.replyChatID = ctx.chatID
	} else {
		ctx.replyChatID = ctx.senderID
	}
	if ctx.isGroupChat {
		ctx.replyChatID = ctx.chatID
	}

	if message != nil {
		if attachments, ok := message["attachments"].([]interface{}); ok {
			for _, att := range attachments {
				attMap, ok := att.(map[string]interface{})
				if !ok {
					continue
				}
				attType, _ := attMap["type"].(string)
				payloadMap := mapFromMap(attMap, "payload")

				if attType == "image" && ctx.photoURL == "" {
					if payloadMap != nil {
						if urlStr, ok := payloadMap["url"].(string); ok && urlStr != "" {
							ctx.photoURL = urlStr
						}
					}
				} else if attType != "image" {
					if payloadMap != nil {
						if phone, ok := payloadMap["phone"].(string); ok && phone != "" {
							ctx.contactPhone = phone
						} else if phoneNum, ok := payloadMap["phone_number"].(string); ok && phoneNum != "" {
							ctx.contactPhone = phoneNum
						} else if sharedPhone, ok := payloadMap["shared_phone"].(string); ok && sharedPhone != "" {
							ctx.contactPhone = sharedPhone
						}
					}
				}
			}
		}
	}

	return ctx
}

// verifyWebhookSecret validates configured Zalo webhook secrets and permits legacy configs without a saved secret.
func (s *zaloServiceImpl) verifyWebhookSecret(user *model.User, secretTokenHeader string) error {
	if secretTokenHeader == "" {
		return errors.New("missing webhook secret token")
	}
	if user.ZaloWebhookSecret == nil || *user.ZaloWebhookSecret == "" {
		return nil
	}
	plainSecret, err := security.Decrypt(*user.ZaloWebhookSecret, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("decrypt webhook secret failed: %w", err)
	}
	if plainSecret != secretTokenHeader {
		return errors.New("invalid webhook secret token")
	}
	return nil
}
