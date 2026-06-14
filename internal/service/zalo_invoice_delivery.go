package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type zaloInvoiceDeliveryDeps struct {
	client            ZaloClient
	userRepo          model.UserRepository
	roomRepo          model.RoomRepository
	tenantRepo        model.TenantRepository
	invoiceRepo       model.InvoiceRepository
	imageService      ImageService
	encryptionKey     []byte
	publicBaseURL     string
	markTokenInactive func(ctx context.Context, managerID string)
}

// deliverInvoiceToZalo generates and sends an invoice image using the current Zalo recipient rules.
func deliverInvoiceToZalo(ctx context.Context, deps zaloInvoiceDeliveryDeps, managerID, invoiceID string) error {
	if deps.client == nil || deps.userRepo == nil || deps.roomRepo == nil || deps.tenantRepo == nil || deps.invoiceRepo == nil || deps.imageService == nil {
		return errors.New("zalo invoice delivery is not configured")
	}

	manager, botToken, err := getDecryptedZaloToken(ctx, deps.userRepo, managerID, deps.encryptionKey)
	if err != nil {
		return err
	}

	invoice, err := deps.invoiceRepo.GetInvoiceByID(ctx, managerID, invoiceID)
	if err != nil {
		return err
	}

	room, err := deps.roomRepo.GetRoomByID(ctx, invoice.RoomID, invoice.HouseID)
	if err != nil {
		return err
	}

	imageBytes, err := deps.imageService.GenerateInvoiceImage(ctx, invoice)
	if err != nil {
		return fmt.Errorf("generate image: %w", err)
	}
	photoURL, err := saveZaloInvoiceImage(deps.publicBaseURL, invoice.ID, imageBytes)
	if err != nil {
		return fmt.Errorf("save zalo invoice image: %w", err)
	}

	caption := fmt.Sprintf("Hóa đơn tiền nhà tháng %s cho phòng %s.\nTổng tiền: %s", invoice.Period, invoice.RoomName, formatCurrencyToVND(invoice.TotalAmount))
	if room.GroupChatID != nil && *room.GroupChatID != "" {
		if err := deps.client.SendPhoto(ctx, botToken, *room.GroupChatID, photoURL, caption); err != nil {
			if isZaloAuthError(err) && deps.markTokenInactive != nil {
				deps.markTokenInactive(ctx, managerID)
				return fmt.Errorf("zalo bot token is invalid or expired")
			}
			return fmt.Errorf("group chat error: %w", err)
		}
		return nil
	}

	recipients := make([]string, 0, 4)
	recipientNames := make(map[string]string)
	if manager.ZaloUserID != nil && *manager.ZaloUserID != "" {
		recipients = append(recipients, *manager.ZaloUserID)
		recipientNames[*manager.ZaloUserID] = "manager"
	}

	tenants, err := deps.tenantRepo.ListTenantByRoomID(ctx, managerID, invoice.RoomID)
	if err != nil {
		return err
	}
	for _, tenant := range tenants {
		if tenant.ZaloUserID == "" {
			continue
		}
		recipients = append(recipients, tenant.ZaloUserID)
		recipientNames[tenant.ZaloUserID] = tenant.FullName
	}

	if len(recipients) == 0 {
		return errors.New("room does not have a linked zalo group chat, linked manager, or linked tenant")
	}

	var sendErrors []string
	for _, recipientID := range recipients {
		if err := deps.client.SendPhoto(ctx, botToken, recipientID, photoURL, caption); err != nil {
			if isZaloAuthError(err) && deps.markTokenInactive != nil {
				deps.markTokenInactive(ctx, managerID)
				return fmt.Errorf("zalo bot token is invalid or expired")
			}
			sendErrors = append(sendErrors, fmt.Sprintf("%s error: %v", recipientNames[recipientID], err))
		}
	}
	if len(sendErrors) > 0 {
		return fmt.Errorf("some messages failed: %s", strings.Join(sendErrors, ", "))
	}

	return nil
}

// getDecryptedZaloToken returns the manager account and decrypted bot token.
func getDecryptedZaloToken(ctx context.Context, userRepo model.UserRepository, managerID string, encryptionKey []byte) (*model.User, string, error) {
	user, err := userRepo.GetByUserID(ctx, managerID)
	if err != nil {
		return nil, "", err
	}
	if user.ZaloBotToken == nil || *user.ZaloBotToken == "" {
		return nil, "", errors.New("zalo config not found for manager")
	}
	botToken, err := security.Decrypt(*user.ZaloBotToken, encryptionKey)
	if err != nil {
		return nil, "", fmt.Errorf("decrypt bot token failed: %v", err)
	}
	return user, botToken, nil
}

// saveZaloInvoiceImage stores a generated invoice image at a public URL that Zalo can fetch.
func saveZaloInvoiceImage(publicBaseURL, invoiceID string, imageBytes []byte) (string, error) {
	dir := "uploads/zalo-invoices"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.png", invoiceID, uuid.NewString())
	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return fmt.Sprintf("%s/api/v1/uploads/zalo-invoices/%s", publicBaseURL, fileName), nil
}
