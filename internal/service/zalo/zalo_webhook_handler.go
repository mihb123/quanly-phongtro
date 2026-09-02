package zalo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

const zaloTransactionImageDownloadTimeout = 10 * time.Second

func (s *zaloServiceImpl) HandleWebhook(ctx context.Context, managerID string, body []byte, secretTokenHeader string) error {
	logger.Info(nil, 0, fmt.Sprintf("zalo webhook received manager=%s size=%d", managerID, len(body)), nil)

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		logger.Warn(nil, 0, "zalo webhook json unmarshal failed", err)
		return err
	}

	payload = normalizeWebhookPayload(payload)
	webhookCtx := webhookContextFromPayload(payload)
	logger.Info(nil, 0, fmt.Sprintf("zalo webhook metadata manager=%s event=%s group=%t size=%d", managerID, webhookCtx.eventName, webhookCtx.isGroupChat, len(body)), nil)

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

	// The Zalo Bot API only forwards group messages when the bot is @mentioned or replied to,
	// and it never emits a "bot added to group" event, so any group webhook is a deliberate interaction.
	if webhookCtx.isGroupChat {
		s.handleGroupLinking(ctx, managerID, webhookCtx)
	}

	if webhookCtx.eventName == "message.unsupported.received" {
		s.handleUnsupportedContactEvent(ctx, managerID, webhookCtx)
		return nil
	}

	// Help is answered before account linking and invoice handling so that a user who is not
	// connected yet still learns how to connect instead of getting a password or linking prompt.
	if isHelpCommandText(webhookCtx.text) {
		s.handleHelpCommand(ctx, managerID, user, webhookCtx)
		return nil
	}

	if user.ZaloUserID == nil || *user.ZaloUserID == "" {
		s.activateManagerByPassword(ctx, managerID, user, webhookCtx)
		return nil
	}

	if s.invoiceCommandService != nil && webhookCtx.text != "" {
		chatID := commandChatID(webhookCtx)
		if isInvoiceCommandText(webhookCtx.text) || s.invoiceCommandService.HasPendingState(ctx, managerID, chatID) {
			if err := s.invoiceCommandService.HandleInvoiceCommand(ctx, managerID, webhookCtx); err != nil {
				logger.Error(nil, 0, "zalo invoice command failed", err)
			}
			return nil
		}
	}

	if !webhookCtx.isGroupChat && webhookCtx.senderID != "" && (webhookCtx.text != "" || webhookCtx.contactPhone != "") {
		s.handleAccountLinkingByPhone(ctx, managerID, webhookCtx)
	}

	// Handle image attachment for transaction proof in group chat
	if webhookCtx.isGroupChat && webhookCtx.chatID != "" {
		if webhookCtx.photoURL != "" {
			err := s.processTransactionImage(ctx, managerID, webhookCtx.chatID, webhookCtx.photoURL)
			if err != nil {
				// We can just log it, don't fail the webhook
				logger.Error(nil, 0, "zalo webhook process transaction image failed", err)
			}
		}
	}

	return nil
}

// handleUnsupportedContactEvent replies to unsupported private messages (e.g. contact cards) from
// users who are not yet linked, guiding them to send their phone number as text instead.
func (s *zaloServiceImpl) handleUnsupportedContactEvent(ctx context.Context, managerID string, webhookCtx webhookMessageContext) {
	if webhookCtx.isGroupChat || webhookCtx.replyChatID == "" || webhookCtx.senderID == "" {
		return
	}
	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
	if err != nil || linkedUser == nil {
		botToken, err := s.getDecryptedToken(ctx, managerID)
		if err == nil {
			_ = s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Xin lỗi, Bot không hỗ trợ nhận danh thiếp (contact). Vui lòng gõ số điện thoại của bạn kèm theo \"abc\" (ví dụ: 0912345678 abc) để hệ thống nhận diện.")
		}
	}
}

// activateManagerByPassword links the manager's own Zalo account when they reply with their login
// password in a private chat, before any Zalo account has been linked to the manager profile.
func (s *zaloServiceImpl) activateManagerByPassword(ctx context.Context, managerID string, user *model.User, webhookCtx webhookMessageContext) {
	if webhookCtx.isGroupChat || webhookCtx.senderID == "" || webhookCtx.text == "" {
		return
	}
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

// handleAccountLinkingByPhone links a tenant or manager Zalo account from a phone number sent in a
// private chat. It enforces anti-hijack (already-linked phones), manager-ownership, and tenant-belongs
// -to-manager rules before saving the link, and greets users whose Zalo is already linked.
func (s *zaloServiceImpl) handleAccountLinkingByPhone(ctx context.Context, managerID string, webhookCtx webhookMessageContext) {
	cleanText := strings.ToLower(strings.ReplaceAll(webhookCtx.text, " ", ""))

	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		logger.Error(nil, 0, fmt.Sprintf("zalo webhook get token failed manager=%s", managerID), err)
		return
	}

	linkedUser, err := s.userRepo.GetByZaloUserID(ctx, webhookCtx.senderID)
	if err != nil {
		var cleanPhone string
		if webhookCtx.contactPhone != "" {
			cleanPhone = strings.ReplaceAll(webhookCtx.contactPhone, " ", "")
		} else {
			textWithoutSpaces := strings.ReplaceAll(webhookCtx.text, " ", "")
			re := regexp.MustCompile(`(?:\+84|84|0)[0-9]{9}`)
			cleanPhone = re.FindString(textWithoutSpaces)
		}

		isPhoneFormat := cleanPhone != ""

		if webhookCtx.contactPhone == "" && (cleanText == "botoi" || cleanText == "botơi") {
			if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Xin chào! Vui lòng nhập số điện thoại của bạn để liên kết tài khoản nhận thông báo."); err != nil {
				logger.Error(nil, 0, "zalo webhook send message failed", err)
			}
		} else if isPhoneFormat {
			if strings.HasPrefix(cleanPhone, "+84") {
				cleanPhone = "0" + cleanPhone[3:]
			} else if strings.HasPrefix(cleanPhone, "84") {
				cleanPhone = "0" + cleanPhone[2:]
			}

			user, err := s.userRepo.GetByPhone(ctx, cleanPhone)
			if err == nil && user != nil {
				// 1. Prevent hijacking an already linked account
				if user.ZaloUserID != nil && *user.ZaloUserID != "" {
					if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Số điện thoại này đã được liên kết với một tài khoản Zalo. Nếu có sai sót, vui lòng liên hệ quản lý."); err != nil {
						logger.Error(nil, 0, "zalo webhook send message failed", err)
					}
				} else if user.Role == model.RoleManager {
					// 2. Only allow the manager who owns the bot to link their manager account
					if user.ID != managerID {
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Bạn không thể liên kết tài khoản quản lý của người khác vào bot này."); err != nil {
							logger.Error(nil, 0, "zalo webhook send message failed", err)
						}
					} else {
						_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &webhookCtx.senderID})
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Liên kết tài khoản quản lý thành công! Từ giờ hệ thống sẽ gửi các thông báo quan trọng qua đây."); err != nil {
							logger.Error(nil, 0, "zalo webhook send message failed", err)
						}
					}
				} else {
					// 3. For Tenants, enforce that the Manager must be linked first
					managerUser, errMgr := s.userRepo.GetByUserID(ctx, managerID)
					if errMgr == nil && managerUser != nil && (managerUser.ZaloUserID == nil || *managerUser.ZaloUserID == "") {
						if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Hệ thống đang trong quá trình thiết lập. Quản lý cần liên kết tài khoản trước khi khách thuê có thể sử dụng."); err != nil {
							logger.Error(nil, 0, "zalo webhook send message failed", err)
						}
					} else {
						// First verify the tenant actually belongs to this manager
						fullInfoTenant, err2 := s.tenantRepo.GetFirstTenantByUserID(ctx, managerID, user.ID)
						if err2 != nil || fullInfoTenant == nil {
							if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Số điện thoại này không thuộc danh sách khách thuê của quản lý này. Vui lòng kiểm tra lại."); err != nil {
								logger.Error(nil, 0, "zalo webhook send message failed", err)
							}
						} else {
							// Tenant belongs to this manager, proceed to link
							_, _ = s.userRepo.UpdateUser(ctx, user.ID, model.UpdateUserInput{ZaloUserID: &webhookCtx.senderID})

							msg := fmt.Sprintf("Xin chào %s - %s. Zalo của bạn đã được liên kết hệ thống quản lý trọ thành công.", fullInfoTenant.FullName, fullInfoTenant.RoomName)
							if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, msg); err != nil {
								logger.Error(nil, 0, "zalo webhook send message failed", err)
							}

							// Notify Manager
							if managerUser != nil && managerUser.ZaloUserID != nil && *managerUser.ZaloUserID != "" {
								mgrMsg := fmt.Sprintf("Khách thuê %s ở phòng %s vừa liên kết Zalo nhận thông báo thành công.", fullInfoTenant.FullName, fullInfoTenant.RoomName)
								_ = s.client.SendMessage(ctx, botToken, *managerUser.ZaloUserID, mgrMsg)
							}
						}
					}
				}
			} else {
				if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Số điện thoại chưa được đăng ký trong hệ thống. Vui lòng kiểm tra lại."); err != nil {
					logger.Error(nil, 0, "zalo webhook send message failed", err)
				}
			}
		} else {
			if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Để bắt đầu kết nối, vui lòng gõ 'bot ơi'."); err != nil {
				logger.Error(nil, 0, "zalo webhook send message failed", err)
			}
		}
		return
	}

	// User is already linked
	if cleanText == "botoi" || cleanText == "botơi" {
		if linkedUser.Role == model.RoleManager {
			if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản quản lý của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
				logger.Error(nil, 0, "zalo webhook send message failed", err)
			}
		} else {
			if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản của bạn đã được liên kết thành công. Bạn sẽ nhận được thông báo từ hệ thống qua Zalo."); err != nil {
				logger.Error(nil, 0, "zalo webhook send message failed", err)
			}
		}
	} else {
		if err := s.client.SendMessage(ctx, botToken, webhookCtx.replyChatID, "Tài khoản của bạn đã được liên kết. Nếu cần hỗ trợ, vui lòng liên hệ quản lý."); err != nil {
			logger.Error(nil, 0, "zalo webhook send message failed", err)
		}
	}
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

	imageBytes, err := downloadZaloTransactionImage(ctx, imgURL)
	if err != nil {
		return err
	}

	// Ensure dir exists
	dir := "uploads/transactions"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.jpg", room.ID, uuid.Must(uuid.NewV7()).String())
	filePath := filepath.Join(dir, fileName)

	if err := os.WriteFile(filePath, imageBytes, 0600); err != nil {
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

// autoLinkRoom maps a Zalo group to a room when the group name follows the
// "<RoomName> <HouseName>" format. It returns true only when a room was matched and updated.
func (s *zaloServiceImpl) autoLinkRoom(ctx context.Context, managerID, groupChatID, groupName string) (bool, error) {
	// groupName is expected to be "<RoomName> <HouseName>"
	houses, err := s.houseRepo.ListHouseByManagerID(ctx, managerID, 1000, 0, "")
	if err != nil {
		return false, err
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
					if err := linkRoomToGroupChat(ctx, s.roomRepo, room, house.ID, groupChatID); err != nil {
						return false, err
					}
					return true, nil
				}
			}
		}
	}

	return false, nil
}

// isBotGreeting reports whether a group message is the "bot ơi" activation phrase.
func isBotGreeting(text string) bool {
	clean := strings.ToLower(strings.ReplaceAll(text, " ", ""))
	return clean == "botoi" || clean == "botơi"
}

// handleGroupLinking connects a Zalo group to a room. It first tries to auto-map by the group
// name format "<RoomName> <HouseName>" (Case 1); when the name is not in that format it replies
// with the group ID so the manager can connect the group to a room on the web (Case 2).
// It only talks to the group when the bot is just added or the manager greets it, to avoid spam.
func (s *zaloServiceImpl) handleGroupLinking(ctx context.Context, managerID string, webhookCtx webhookMessageContext) {
	if webhookCtx.chatID == "" {
		return
	}

	greeting := isBotGreeting(webhookCtx.text)

	// Already linked: confirm on greeting, stay silent otherwise.
	if room, err := s.roomRepo.GetRoomByGroupChatID(ctx, webhookCtx.chatID); err == nil && room != nil {
		if greeting {
			s.sendGroupMessage(ctx, managerID, webhookCtx.chatID, fmt.Sprintf("✅ Nhóm này đã được kết nối với phòng %s.", room.Name))
		}
		return
	}

	// Case 1: best-effort auto-map when the group name is available and follows "<RoomName> <HouseName>".
	// The Zalo Bot webhook payload does not carry the group name (chat only has id + chat_type), so this
	// rarely runs in production; the reliable path is manual connection via the group ID below (Case 2).
	if webhookCtx.groupName != "" {
		linked, err := s.autoLinkRoom(ctx, managerID, webhookCtx.chatID, webhookCtx.groupName)
		if err != nil {
			logger.Error(nil, 0, "zalo auto link room failed", err)
		}
		if linked {
			s.sendGroupMessage(ctx, managerID, webhookCtx.chatID, "✅ Bot đã tự động kết nối nhóm này với phòng thành công.")
			return
		}
	}

	// Case 2: group is not linked -> reply with the group ID so the manager can connect it on the web.
	// Help and the manager update commands answer with the same instructions plus the group ID, so
	// let their own replies stand instead of doubling up.
	if groupLinkingReplySuppressed(webhookCtx.text) {
		return
	}
	msg := fmt.Sprintf("⚠️ Nhóm này chưa được kết nối với phòng nào.\n\nQuản lý nhắn ngay trong nhóm này để kết nối:\n#update-room <mã nhà> <phòng>\nVD: #update-room 679qt P201\n\nHoặc kết nối trên web bằng mã nhóm (Group ID):\n%s", webhookCtx.chatID)
	s.sendGroupMessage(ctx, managerID, webhookCtx.chatID, msg)
}

// sendGroupMessage sends a best-effort text message to a group chat, logging failures without
// interrupting webhook processing.
func (s *zaloServiceImpl) sendGroupMessage(ctx context.Context, managerID, chatID, text string) {
	botToken, err := s.getDecryptedToken(ctx, managerID)
	if err != nil {
		logger.Error(nil, 0, "zalo webhook get token failed", err)
		return
	}
	if err := s.client.SendMessage(ctx, botToken, chatID, text); err != nil {
		logger.Error(nil, 0, "zalo webhook send group message failed", err)
	}
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
