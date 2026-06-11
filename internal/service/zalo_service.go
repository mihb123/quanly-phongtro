package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type ZaloService interface {
	SaveZaloConfig(ctx context.Context, managerID, botToken, webhookUrl, secretToken string) error
	GetZaloConfigStatus(ctx context.Context, managerID string) (ZaloBotStatus, error)
	SendTextMessage(ctx context.Context, managerID, chatID, text string) error
	HandleWebhook(ctx context.Context, managerID string, body []byte, secretTokenHeader string) error
	SendInvoiceToZalo(ctx context.Context, managerID, invoiceID string) error
}

// ZaloBotStatus describes the current connectivity status of a manager's Zalo bot.
type ZaloBotStatus struct {
	HasConfig bool   `json:"has_config"`
	IsActive  bool   `json:"is_zalo_bot_active"`
	IsLinked  bool   `json:"is_linked"`
	BotID     string `json:"bot_id"`
	ManagerID string `json:"manager_id"`
}

type zaloServiceImpl struct {
	client        ZaloClient
	userRepo      model.UserRepository
	roomRepo      model.RoomRepository
	tenantRepo    model.TenantRepository
	houseRepo     model.HouseRepository
	invoiceRepo   model.InvoiceRepository
	imageService  ImageService
	encryptionKey []byte
}

// DecodeEncryptionKey decodes the hex or raw 32-byte encryption key string used for Zalo tokens.
func DecodeEncryptionKey(hexKey string) ([]byte, error) {
	keyBytes, err := hex.DecodeString(hexKey)
	if err != nil {
		if len(hexKey) == 32 {
			return []byte(hexKey), nil
		}
		return nil, fmt.Errorf("invalid encryption key: %v", err)
	}
	if len(keyBytes) != 32 {
		return nil, errors.New("ZALO_BOT_ENCRYPTION_KEY must be exactly 32 bytes")
	}
	return keyBytes, nil
}

func NewZaloService(client ZaloClient, userRepo model.UserRepository, roomRepo model.RoomRepository, tenantRepo model.TenantRepository, houseRepo model.HouseRepository, invoiceRepo model.InvoiceRepository, imageService ImageService, hexKey string) (ZaloService, error) {
	keyBytes, err := DecodeEncryptionKey(hexKey)
	if err != nil {
		return nil, err
	}

	return &zaloServiceImpl{
		client:        client,
		userRepo:      userRepo,
		roomRepo:      roomRepo,
		tenantRepo:    tenantRepo,
		houseRepo:     houseRepo,
		invoiceRepo:   invoiceRepo,
		imageService:  imageService,
		encryptionKey: keyBytes,
	}, nil
}

func (s *zaloServiceImpl) SaveZaloConfig(ctx context.Context, managerID, botToken, webhookUrl, secretToken string) error {
	// 1. Validate bot token by calling GetMe
	if _, err := s.client.GetMe(ctx, botToken); err != nil {
		return fmt.Errorf("invalid bot token: %v", err)
	}

	// 2. Set Webhook
	if err := s.client.SetWebhook(ctx, botToken, webhookUrl, secretToken); err != nil {
		return fmt.Errorf("set webhook failed: %v", err)
	}

	// 3. Encrypt token
	encToken, err := security.Encrypt(botToken, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt token: %v", err)
	}

	// 4. Save to DB and mark bot as active
	active := true
	input := model.UpdateUserInput{
		ZaloBotToken:    &encToken,
		IsZaloBotActive: &active,
	}

	_, err = s.userRepo.UpdateUser(ctx, managerID, input)
	return err
}

// GetZaloConfigStatus returns the current Zalo bot configuration and token health for a manager.
func (s *zaloServiceImpl) GetZaloConfigStatus(ctx context.Context, managerID string) (ZaloBotStatus, error) {
	user, err := s.userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return ZaloBotStatus{}, err
	}

	hasConfig := user.ZaloBotToken != nil && *user.ZaloBotToken != ""
	status := ZaloBotStatus{
		HasConfig: hasConfig,
		IsActive:  hasConfig && user.IsZaloBotActive,
		IsLinked:  user.ZaloUserID != nil && *user.ZaloUserID != "",
		ManagerID: managerID,
	}

	if hasConfig {
		botToken, err := security.Decrypt(*user.ZaloBotToken, s.encryptionKey)
		if err == nil {
			botInfo, err := s.client.GetMe(ctx, botToken)
			if err == nil && botInfo != nil {
				status.BotID = botInfo.AppID
			}
		}
	}

	return status, nil
}

func (s *zaloServiceImpl) getDecryptedToken(ctx context.Context, managerID string) (string, error) {
	user, err := s.userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return "", err
	}

	if user.ZaloBotToken == nil || *user.ZaloBotToken == "" {
		return "", errors.New("zalo config not found for manager")
	}

	botToken, err := security.Decrypt(*user.ZaloBotToken, s.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt bot token failed: %v", err)
	}

	return botToken, nil
}

// markTokenInactive marks the manager's Zalo bot token as inactive in the DB.
// Called when an API call indicates the token is no longer valid.
func (s *zaloServiceImpl) markTokenInactive(ctx context.Context, managerID string) {
	inactive := false
	_, _ = s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{IsZaloBotActive: &inactive})
}

func (s *zaloServiceImpl) SendTextMessage(ctx context.Context, managerID, chatID, text string) error {
	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		return err
	}

	return s.client.SendMessage(ctx, botToken, chatID, text)
}

func (s *zaloServiceImpl) SendInvoiceToZalo(ctx context.Context, managerID, invoiceID string) error {
	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		return err
	}

	invoice, err := s.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return err
	}

	room, err := s.roomRepo.GetRoomByID(ctx, managerID, invoice.RoomID)
	if err != nil {
		return err
	}

	imageBytes, err := s.imageService.GenerateInvoiceImage(ctx, invoice)
	if err != nil {
		return fmt.Errorf("generate image: %w", err)
	}

	caption := fmt.Sprintf("Hóa đơn tiền nhà tháng %s cho phòng %s.\nTổng tiền: %s", invoice.Period, invoice.RoomName, formatCurrencyToVND(invoice.TotalAmount))

	var sendErrors []string

	// Send to Group Chat
	if room.GroupChatID != nil && *room.GroupChatID != "" {
		err := s.client.SendPhoto(ctx, botToken, *room.GroupChatID, imageBytes, caption)
		if err != nil {
			if isZaloAuthError(err) {
				s.markTokenInactive(ctx, managerID)
				return fmt.Errorf("zalo bot token is invalid or expired")
			}
			sendErrors = append(sendErrors, fmt.Sprintf("group chat error: %v", err))
		}
	}

	// Send to active tenants' Private Chats
	tenants, err := s.tenantRepo.ListTenantByRoomID(ctx, managerID, invoice.RoomID)
	if err == nil {
		for _, t := range tenants {
			if t.ZaloUserID != "" {
				err := s.client.SendPhoto(ctx, botToken, t.ZaloUserID, imageBytes, caption)
				if err != nil {
					sendErrors = append(sendErrors, fmt.Sprintf("tenant %s error: %v", t.FullName, err))
				}
			}
		}
	}

	// If neither group chat nor any private chat exists, return error
	if (room.GroupChatID == nil || *room.GroupChatID == "") && len(sendErrors) == 0 {
		hasLinkedTenant := false
		for _, t := range tenants {
			if t.ZaloUserID != "" {
				hasLinkedTenant = true
				break
			}
		}
		if !hasLinkedTenant {
			return errors.New("room does not have a linked zalo group chat or any linked tenant")
		}
	}

	if len(sendErrors) > 0 {
		return fmt.Errorf("some messages failed: %s", strings.Join(sendErrors, ", "))
	}

	return nil
}

func (s *zaloServiceImpl) generateDeterministicSecret(managerID string) string {
	return managerID[:min(12, len(managerID))]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *zaloServiceImpl) HandleWebhook(ctx context.Context, managerID string, body []byte, secretTokenHeader string) error {
	if secretTokenHeader == "" {
		return errors.New("missing webhook secret token")
	}

	fmt.Printf("Webhook received for manager %s. Payload: %s\n", managerID, string(body))

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Printf("Webhook JSON Unmarshal error: %v\n", err)
		return err
	}

	eventName, _ := payload["event_name"].(string)

	var groupChatID string
	var groupName string
	var senderID string

	if sender, ok := payload["sender"].(map[string]interface{}); ok {
		if id, ok := sender["id"].(string); ok {
			senderID = id
		} else if idFloat, ok := sender["id"].(float64); ok {
			senderID = fmt.Sprintf("%.0f", idFloat)
		}
	} else if msgInfo, ok := payload["message"].(map[string]interface{}); ok {
		if from, ok := msgInfo["from"].(map[string]interface{}); ok {
			if id, ok := from["id"].(string); ok {
				senderID = id
			} else if idFloat, ok := from["id"].(float64); ok {
				senderID = fmt.Sprintf("%.0f", idFloat)
			}
		}
	}

	isGroupChat := false

	// Attempt extraction based on common patterns
	if groupInfo, ok := payload["group"].(map[string]interface{}); ok {
		isGroupChat = true
		if id, ok := groupInfo["id"].(string); ok {
			groupChatID = id
		} else if idFloat, ok := groupInfo["id"].(float64); ok {
			groupChatID = fmt.Sprintf("%.0f", idFloat)
		}
		if name, ok := groupInfo["name"].(string); ok {
			groupName = name
		}
	} else if messageInfo, ok := payload["message"].(map[string]interface{}); ok {
		if chatInfo, ok := messageInfo["chat"].(map[string]interface{}); ok {
			if id, ok := chatInfo["id"].(string); ok {
				groupChatID = id
			} else if idFloat, ok := chatInfo["id"].(float64); ok {
				groupChatID = fmt.Sprintf("%.0f", idFloat)
			}
			if title, ok := chatInfo["title"].(string); ok {
				groupName = title
				if title != "" {
					isGroupChat = true
				}
			}
		}
	}

	// If it has a groupChatID but it's the same as senderID, it's likely a private chat (some APIs behave this way)
	if groupChatID == senderID {
		isGroupChat = false
		groupChatID = "" // clear it to be safe
	}

	var textMsg string
	if msgInfo, ok := payload["message"].(map[string]interface{}); ok {
		if txt, ok := msgInfo["text"].(string); ok {
			textMsg = strings.TrimSpace(txt)
		}
	}

	if eventName == "group.bot.add" || isGroupChat {
		if groupName != "" && groupChatID != "" {
			_ = s.autoLinkRoom(ctx, managerID, groupChatID, groupName)
		}
	}

	if strings.HasPrefix(textMsg, "Kich hoat") {
		managerNameToLink := strings.TrimSpace(strings.TrimPrefix(textMsg, "Kich hoat"))

		user, err := s.userRepo.GetByUserID(ctx, managerID)
		if err == nil && (managerNameToLink == user.FullName || managerNameToLink == managerID) {
			if user.ZaloUserID != nil && *user.ZaloUserID != "" {
				_ = s.SendTextMessage(ctx, managerID, senderID, "⚠️ Tài khoản này đã được liên kết với một thiết bị khác. Không thể liên kết lại.")
				return nil
			}

			_, err := s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{ZaloUserID: &senderID})
			if err == nil {
				_ = s.SendTextMessage(ctx, managerID, senderID, "✅ Cấu hình Zalo Bot hoàn tất!\n\nTài khoản quản lý của bạn đã được liên kết thành công. Giờ đây hệ thống sẽ tự động gửi thông báo đến bạn.")
			} else {
				_ = s.SendTextMessage(ctx, managerID, senderID, "❌ Có lỗi xảy ra khi liên kết tài khoản. Vui lòng thử lại sau.")
			}
		}
		return nil
	}

	cleanText := strings.ToLower(strings.ReplaceAll(textMsg, " ", ""))

	if !isGroupChat && senderID != "" && textMsg != "" {
		botToken, err := s.getDecryptedToken(ctx, managerID)
		if err != nil {
			fmt.Printf("Webhook error: failed to get decrypted token for manager %s: %v\n", managerID, err)
		} else {
			linkedUser, err := s.userRepo.GetByZaloUserID(ctx, senderID)
			if err != nil {
				cleanPhone := strings.ReplaceAll(textMsg, " ", "")
				isPhoneFormat := (strings.HasPrefix(cleanPhone, "0") && len(cleanPhone) == 10) || (strings.HasPrefix(cleanPhone, "+84") && len(cleanPhone) == 12)

				if cleanText == "botoi" || cleanText == "botơi" {
					if err := s.client.SendMessage(ctx, botToken, senderID, "Xin chào! Vui lòng nhập số điện thoại của bạn để liên kết tài khoản nhận thông báo."); err != nil {
						fmt.Printf("Webhook SendMessage error: %v\n", err)
					}
				} else if isPhoneFormat {
					if strings.HasPrefix(cleanPhone, "+84") {
						cleanPhone = "0" + cleanPhone[3:]
					}

					user, err := s.userRepo.GetByPhone(ctx, cleanPhone)
					if err == nil && user != nil {
						// 1. Prevent hijacking an already linked account
						if user.ZaloUserID != nil && *user.ZaloUserID != "" {
							if err := s.client.SendMessage(ctx, botToken, senderID, "Số điện thoại này đã được liên kết với một tài khoản Zalo. Nếu có sai sót, vui lòng liên hệ quản lý."); err != nil {
								fmt.Printf("Webhook SendMessage error: %v\n", err)
							}
						} else if user.Role == model.RoleManager {
							// 2. Only allow the manager who owns the bot to link their manager account
							if user.ID != managerID {
								if err := s.client.SendMessage(ctx, botToken, senderID, "Bạn không thể liên kết tài khoản quản lý của người khác vào bot này."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							} else {
								_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &senderID})
								if err := s.client.SendMessage(ctx, botToken, senderID, "Liên kết tài khoản quản lý thành công! Từ giờ hệ thống sẽ gửi các thông báo quan trọng qua đây."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							}
						} else {
							// 3. For Tenants, enforce that the Manager must be linked first
							managerUser, errMgr := s.userRepo.GetByUserID(ctx, managerID)
							if errMgr == nil && managerUser != nil && (managerUser.ZaloUserID == nil || *managerUser.ZaloUserID == "") {
								if err := s.client.SendMessage(ctx, botToken, senderID, "Hệ thống đang trong quá trình thiết lập. Quản lý cần liên kết tài khoản trước khi khách thuê có thể sử dụng."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							} else {
								// Tenant linked successfully
								_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &senderID})
								msg := "Liên kết tài khoản thành công! Từ giờ bạn sẽ nhận được thông báo qua Zalo."
								fullInfoTenant, err2 := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, user.ID)
								if err2 == nil && fullInfoTenant != nil {
									msg = fmt.Sprintf("Xin chào %s - %s. Zalo của bạn đã được liên kết hệ thống quản lý trọ thành công.", fullInfoTenant.FullName, fullInfoTenant.RoomName)

									// Notify Manager
									if managerUser != nil && managerUser.ZaloUserID != nil && *managerUser.ZaloUserID != "" {
										mgrMsg := fmt.Sprintf("Khách thuê %s ở phòng %s vừa liên kết Zalo nhận thông báo thành công.", fullInfoTenant.FullName, fullInfoTenant.RoomName)
										_ = s.client.SendMessage(ctx, botToken, *managerUser.ZaloUserID, mgrMsg)
									}
								}

								if err := s.client.SendMessage(ctx, botToken, senderID, msg); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							}
						}
					} else {
						if err := s.client.SendMessage(ctx, botToken, senderID, "Số điện thoại chưa được đăng ký trong hệ thống. Vui lòng kiểm tra lại."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					}
				} else {
					if err := s.client.SendMessage(ctx, botToken, senderID, "Để bắt đầu kết nối, vui lòng gõ 'bot ơi'."); err != nil {
						fmt.Printf("Webhook SendMessage error: %v\n", err)
					}
				}
			} else {
				// User is already linked
				if cleanText == "botoi" || cleanText == "botơi" {
					if linkedUser.Role == model.RoleManager {
						if err := s.client.SendMessage(ctx, botToken, senderID, "Tài khoản quản lý của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					} else {
						if err := s.client.SendMessage(ctx, botToken, senderID, "Tài khoản của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					}
				} else {
					if err := s.client.SendMessage(ctx, botToken, senderID, "Tài khoản của bạn đã được liên kết. Nếu cần hỗ trợ, vui lòng liên hệ quản lý."); err != nil {
						fmt.Printf("Webhook SendMessage error: %v\n", err)
					}
				}
			}
		}
	}

	// Handle image attachment for transaction proof in group chat
	if isGroupChat && groupChatID != "" {
		var imgURL string
		if msgInfo, ok := payload["message"].(map[string]interface{}); ok {
			if attachments, ok := msgInfo["attachments"].([]interface{}); ok {
				for _, att := range attachments {
					if attMap, ok := att.(map[string]interface{}); ok {
						if attMap["type"] == "image" {
							if pl, ok := attMap["payload"].(map[string]interface{}); ok {
								if urlStr, ok := pl["url"].(string); ok && urlStr != "" {
									imgURL = urlStr
									break
								}
							}
						}
					}
				}
			}
		}

		if imgURL != "" {
			err := s.processTransactionImage(ctx, managerID, groupChatID, imgURL)
			if err != nil {
				// We can just log it, don't fail the webhook
				fmt.Printf("Error processing transaction image: %v\n", err)
			}
		}

		if cleanText == "botoi" || cleanText == "botơi" {
			botToken, err := s.getDecryptedToken(ctx, managerID)
			if err == nil {
				_ = s.client.SendMessage(ctx, botToken, groupChatID, "Bot đã kết nối thành công")
			}
		}
	}

	return nil
}

func (s *zaloServiceImpl) processTransactionImage(ctx context.Context, managerID, groupChatID, imgURL string) error {
	room, err := s.roomRepo.GetRoomByGroupChatID(ctx, groupChatID)
	if err != nil {
		return fmt.Errorf("room not found for group: %w", err)
	}

	// Double check ownership
	_, err = s.houseRepo.GetByID(ctx, room.HouseID, managerID)
	if err != nil {
		return fmt.Errorf("house not owned by manager: %w", err)
	}

	latestUnpaid, err := s.invoiceRepo.GetLatestUnpaidInvoiceByRoomID(ctx, room.ID)
	if err != nil {
		if errors.Is(err, model.ErrInvoiceNotFound) {
			// No unpaid invoice, ignore image
			return nil
		}
		return fmt.Errorf("get latest unpaid invoice: %w", err)
	}

	// Download image
	resp, err := http.Get(imgURL)
	if err != nil {
		return fmt.Errorf("download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download image status: %d", resp.StatusCode)
	}

	// Ensure dir exists
	dir := "uploads/transactions"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.jpg", room.ID, uuid.New().String())
	filePath := filepath.Join(dir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("save image: %w", err)
	}

	// Update invoice
	latestUnpaid.Status = model.InvoiceStatusPendingVerification
	pathStr := "/" + filePath // e.g. /uploads/transactions/...
	latestUnpaid.TransactionImagePath = &pathStr

	err = s.invoiceRepo.UpdateInvoice(ctx, managerID, latestUnpaid)
	if err != nil {
		return fmt.Errorf("update invoice: %w", err)
	}

	// Send confirmation message to group
	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err == nil {
		msg := fmt.Sprintf("Hệ thống đã nhận được ảnh giao dịch cho Hóa đơn tháng %s.\nQuản lý sẽ kiểm tra và xác nhận sớm nhất.", latestUnpaid.Period)
		_ = s.client.SendMessage(ctx, botToken, groupChatID, msg)
	}

	return nil
}

func (s *zaloServiceImpl) autoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) error {
	// groupName is expected to be "<RoomName> <HouseName>"
	houses, err := s.houseRepo.ListHouseByManagerID(ctx, managerID, 1000, 0, "")
	if err != nil {
		return err
	}

	for _, house := range houses {
		// Case insensitive check
		if strings.HasSuffix(strings.ToLower(groupName), strings.ToLower(house.Name)) {
			// Find rooms in this house
			rooms, err := s.roomRepo.ListAllRoomsByHouseID(ctx, house.ID)
			if err != nil {
				continue
			}

			for _, room := range rooms {
				expectedPrefix := strings.ToLower(room.Name + " ")
				if strings.HasPrefix(strings.ToLower(groupName), expectedPrefix) {
					// Match found
					params := model.UpdateRoomParams{
						Name:                  room.Name,
						Price:                 room.Price,
						MaxTenants:            room.MaxTenants,
						Status:                room.Status,
						ElectricityPrice:      room.ElectricityPrice,
						WaterPrice:            room.WaterPrice,
						WifiPrice:             room.WifiPrice,
						ParkingPrice:          room.ParkingPrice,
						ServicePrice:          room.ServicePrice,
						ExtraPersonThreshold:  room.ExtraPersonThreshold,
						ExtraPersonFee:        room.ExtraPersonFee,
						ExtraVehicleThreshold: room.ExtraVehicleThreshold,
						ExtraVehicleFee:       room.ExtraVehicleFee,
						GroupChatID:           &groupChatID,
					}
					_, err := s.roomRepo.UpdateRoom(ctx, room.ID, house.ID, params)
					return err
				}
			}
		}
	}

	return nil
}

// isZaloAuthError checks if an error indicates that the Zalo bot token is invalid or expired.
// Zalo API typically returns error code -216 (invalid access token) for auth failures.
func isZaloAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "-216") ||
		strings.Contains(strings.ToLower(msg), "invalid access token") ||
		strings.Contains(strings.ToLower(msg), "unauthorized")
}
