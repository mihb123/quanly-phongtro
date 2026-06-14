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
	client                ZaloClient
	userRepo              model.UserRepository
	roomRepo              model.RoomRepository
	tenantRepo            model.TenantRepository
	houseRepo             model.HouseRepository
	invoiceRepo           model.InvoiceRepository
	imageService          ImageService
	invoiceCommandService ZaloInvoiceCommandService
	encryptionKey         []byte
	publicBaseURL         string
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

func NewZaloService(client ZaloClient, userRepo model.UserRepository, roomRepo model.RoomRepository, tenantRepo model.TenantRepository, houseRepo model.HouseRepository, invoiceRepo model.InvoiceRepository, imageService ImageService, hexKey string, publicBaseURL ...string) (ZaloService, error) {
	keyBytes, err := DecodeEncryptionKey(hexKey)
	if err != nil {
		return nil, err
	}

	baseURL := "http://localhost:8080"
	if len(publicBaseURL) > 0 && publicBaseURL[0] != "" {
		baseURL = strings.TrimRight(publicBaseURL[0], "/")
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
		publicBaseURL: baseURL,
	}, nil
}

// SetInvoiceCommandService attaches the optional invoice chat command handler.
func (s *zaloServiceImpl) SetInvoiceCommandService(commandService ZaloInvoiceCommandService) {
	s.invoiceCommandService = commandService
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

	// 3. Encrypt token and webhook secret
	encToken, err := security.Encrypt(botToken, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt token: %v", err)
	}
	encSecret, err := security.Encrypt(secretToken, s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt webhook secret: %v", err)
	}

	// 4. Save to DB and mark bot as active
	active := true
	input := model.UpdateUserInput{
		ZaloBotToken:      &encToken,
		ZaloWebhookSecret: &encSecret,
		IsZaloBotActive:   &active,
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
	return deliverInvoiceToZalo(ctx, zaloInvoiceDeliveryDeps{
		client:            s.client,
		userRepo:          s.userRepo,
		roomRepo:          s.roomRepo,
		tenantRepo:        s.tenantRepo,
		invoiceRepo:       s.invoiceRepo,
		imageService:      s.imageService,
		encryptionKey:     s.encryptionKey,
		publicBaseURL:     s.publicBaseURL,
		markTokenInactive: s.markTokenInactive,
	}, managerID, invoiceID)
}

type webhookMessageContext struct {
	eventName   string
	message     map[string]interface{}
	senderID    string
	chatID      string
	replyChatID string
	groupName   string
	isGroupChat bool
	text        string
	photoURL    string
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

	if ctx.photoURL == "" && message != nil {
		if attachments, ok := message["attachments"].([]interface{}); ok {
			for _, att := range attachments {
				attMap, ok := att.(map[string]interface{})
				if !ok || attMap["type"] != "image" {
					continue
				}
				payloadMap := mapFromMap(attMap, "payload")
				if payloadMap == nil {
					continue
				}
				if urlStr, ok := payloadMap["url"].(string); ok && urlStr != "" {
					ctx.photoURL = urlStr
					break
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

func (s *zaloServiceImpl) HandleWebhook(ctx context.Context, managerID string, body []byte, secretTokenHeader string) error {
	fmt.Printf("Webhook received for manager %s. Payload: %s\n", managerID, string(body))

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		fmt.Printf("Webhook JSON Unmarshal error: %v\n", err)
		return err
	}

	payload = normalizeWebhookPayload(payload)
	webhookCtx := webhookContextFromPayload(payload)

	if secretTokenHeader == "" {
		return errors.New("missing webhook secret token")
	}

	user, err := s.userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return err
	}
	if err := s.verifyWebhookSecret(user, secretTokenHeader); err != nil {
		return err
	}

	if webhookCtx.eventName == "group.bot.add" || webhookCtx.isGroupChat {
		if webhookCtx.groupName != "" && webhookCtx.chatID != "" {
			_ = s.autoLinkRoom(ctx, managerID, webhookCtx.chatID, webhookCtx.groupName)
		}
	}

	if user.ZaloUserID == nil || *user.ZaloUserID == "" {
		if !webhookCtx.isGroupChat && webhookCtx.senderID != "" && webhookCtx.text != "" {
			hasher := security.NewBcryptHasher()
			if err := hasher.Compare(user.PasswordHash, webhookCtx.text); err == nil {
				_, err := s.userRepo.UpdateUser(ctx, managerID, model.UpdateUserInput{ZaloUserID: &webhookCtx.senderID})
				if err == nil {
					_ = s.SendTextMessage(ctx, managerID, webhookCtx.replyChatID, "✅ Cấu hình Zalo Bot hoàn tất!\n\nTài khoản quản lý của bạn đã được liên kết thành công. Giờ đây hệ thống sẽ tự động gửi thông báo đến bạn.")
				} else {
					_ = s.SendTextMessage(ctx, managerID, webhookCtx.replyChatID, "❌ Có lỗi xảy ra khi liên kết tài khoản. Vui lòng thử lại sau.")
				}
			} else {
				_ = s.SendTextMessage(ctx, managerID, webhookCtx.replyChatID, "Manager cần nhập mật khẩu để kích hoạt tài khoản")
			}
		}
		return nil
	}

	cleanText := strings.ToLower(strings.ReplaceAll(webhookCtx.text, " ", ""))

	if s.invoiceCommandService != nil && webhookCtx.text != "" {
		chatID := commandChatID(webhookCtx)
		if isInvoiceCommandText(webhookCtx.text) || s.invoiceCommandService.HasPendingState(ctx, managerID, chatID) {
			if err := s.invoiceCommandService.HandleInvoiceCommand(ctx, managerID, webhookCtx); err != nil {
				fmt.Printf("Invoice command error: %v\n", err)
			}
			return nil
		}
	}

	if !webhookCtx.isGroupChat && webhookCtx.senderID != "" && webhookCtx.text != "" {
		botToken, err := s.getDecryptedToken(ctx, managerID)
		if err != nil {
			fmt.Printf("Webhook error: failed to get decrypted token for manager %s: %v\n", managerID, err)
		} else {
			linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
			if err != nil {
				cleanPhone := strings.ReplaceAll(webhookCtx.text, " ", "")
				isPhoneFormat := (strings.HasPrefix(cleanPhone, "0") && len(cleanPhone) == 10) || (strings.HasPrefix(cleanPhone, "+84") && len(cleanPhone) == 12)

				if cleanText == "botoi" || cleanText == "botơi" {
					if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Xin chào! Vui lòng nhập số điện thoại của bạn để liên kết tài khoản nhận thông báo."); err != nil {
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
							if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Số điện thoại này đã được liên kết với một tài khoản Zalo. Nếu có sai sót, vui lòng liên hệ quản lý."); err != nil {
								fmt.Printf("Webhook SendMessage error: %v\n", err)
							}
						} else if user.Role == model.RoleManager {
							// 2. Only allow the manager who owns the bot to link their manager account
							if user.ID != managerID {
								if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Bạn không thể liên kết tài khoản quản lý của người khác vào bot này."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							} else {
								_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &webhookCtx.senderID})
								if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Liên kết tài khoản quản lý thành công! Từ giờ hệ thống sẽ gửi các thông báo quan trọng qua đây."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							}
						} else {
							// 3. For Tenants, enforce that the Manager must be linked first
							managerUser, errMgr := s.userRepo.GetByUserID(ctx, managerID)
							if errMgr == nil && managerUser != nil && (managerUser.ZaloUserID == nil || *managerUser.ZaloUserID == "") {
								if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Hệ thống đang trong quá trình thiết lập. Quản lý cần liên kết tài khoản trước khi khách thuê có thể sử dụng."); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							} else {
								// Tenant linked successfully
								_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &webhookCtx.senderID})
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

								if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, msg); err != nil {
									fmt.Printf("Webhook SendMessage error: %v\n", err)
								}
							}
						}
					} else {
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Số điện thoại chưa được đăng ký trong hệ thống. Vui lòng kiểm tra lại."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					}
				} else {
					if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Để bắt đầu kết nối, vui lòng gõ 'bot ơi'."); err != nil {
						fmt.Printf("Webhook SendMessage error: %v\n", err)
					}
				}
			} else {
				// User is already linked
				if cleanText == "botoi" || cleanText == "botơi" {
					if linkedUser.Role == model.RoleManager {
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản quản lý của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					} else {
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
							fmt.Printf("Webhook SendMessage error: %v\n", err)
						}
					}
				} else {
					if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản của bạn đã được liên kết. Nếu cần hỗ trợ, vui lòng liên hệ quản lý."); err != nil {
						fmt.Printf("Webhook SendMessage error: %v\n", err)
					}
				}
			}
		}
	}

	// Handle image attachment for transaction proof in group chat
	if webhookCtx.isGroupChat && webhookCtx.chatID != "" {
		if webhookCtx.photoURL != "" {
			err := s.processTransactionImage(ctx, managerID, webhookCtx.chatID, webhookCtx.photoURL)
			if err != nil {
				// We can just log it, don't fail the webhook
				fmt.Printf("Error processing transaction image: %v\n", err)
			}
		}

		if cleanText == "botoi" || cleanText == "botơi" {
			botToken, err := s.getDecryptedToken(ctx, managerID)
			if err == nil {
				_ = s.client.SendMessage(ctx, botToken, webhookCtx.chatID, "Bot đã kết nối thành công")
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
