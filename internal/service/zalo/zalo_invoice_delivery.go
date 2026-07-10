package zalo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	invoicesvc "github.com/mihb123/quanly-phongtro/internal/service/invoice"
	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"

	"github.com/google/uuid"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service/logger"
)

type zaloInvoiceDeliveryDeps struct {
	client            ZaloClient
	userRepo          model.UserRepository
	roomRepo          model.RoomRepository
	tenantRepo        model.TenantRepository
	invoiceRepo       model.InvoiceRepository
	imageService      invoicesvc.ImageService
	paymentService    paymentsvc.PaymentService
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

	// Gather recipients
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

	var mainTenantName string
	for _, tenant := range tenants {
		if mainTenantName == "" {
			mainTenantName = tenant.FullName
		}
		if tenant.ZaloUserID == "" {
			continue
		}
		recipients = append(recipients, tenant.ZaloUserID)
		recipientNames[tenant.ZaloUserID] = tenant.FullName
	}

	if (room.GroupChatID == nil || *room.GroupChatID == "") && len(recipients) == 0 {
		return errors.New("room does not have a linked zalo group chat, linked manager, or linked tenant")
	}

	// Generate Invoice Image
	imageBytes, err := deps.imageService.GenerateInvoiceImage(ctx, invoice)
	if err != nil {
		return fmt.Errorf("generate image: %w", err)
	}
	photoURL, err := saveZaloInvoiceImage(deps.publicBaseURL, invoice.ID, imageBytes)
	if err != nil {
		return fmt.Errorf("save zalo invoice image: %w", err)
	}

	// Generate payment link and QR image.
	var qrPhotoURL string
	var checkoutURL string
	var qrProvider string
	if deps.paymentService != nil && invoice.Status == model.InvoiceStatusUnpaid {
		paymentLink, err := deps.paymentService.CreatePreferredPaymentLinkForInvoice(ctx, managerID, invoice, mainTenantName)
		if err != nil {
			logger.Error(nil, 0, "create preferred payment link failed", err)
		}
		if err == nil && paymentLink != nil {
			checkoutURL = paymentLink.CheckoutURL
			qrProvider = paymentLink.Provider
			qrBytes, err := fetchPaymentQRImage(paymentLink.Provider, paymentLink.QRCode)
			if err != nil {
				logger.Error(nil, 0, "fetch payment qr image failed", err)
			} else {
				qrPhotoURL, err = saveZaloInvoiceImage(deps.publicBaseURL, invoice.ID+"_qr", qrBytes)
				if err != nil {
					logger.Error(nil, 0, "save payment qr image failed", err)
				}
			}
		}
	}

	caption := fmt.Sprintf("Hóa đơn tiền nhà tháng %s cho phòng %s.\nTổng tiền: %s", invoice.Period, invoice.RoomName, invoicesvc.FormatCurrencyToVND(invoice.TotalAmount))
	qrCaption := ""
	if qrPhotoURL != "" {
		qrCaption = "Vui lòng quét mã QR để thanh toán."
		// Only PayOS exposes a real checkout page; SePay's URL is the QR image itself.
		if qrProvider == model.PaymentProviderPayOS && checkoutURL != "" {
			qrCaption += fmt.Sprintf(" Hoặc truy cập link: %s", checkoutURL)
		}
	}

	if room.GroupChatID != nil && *room.GroupChatID != "" {
		if err := deps.client.SendPhoto(ctx, botToken, *room.GroupChatID, photoURL, caption); err != nil {
			if isZaloAuthError(err) && deps.markTokenInactive != nil {
				deps.markTokenInactive(ctx, managerID)
				return fmt.Errorf("zalo bot token is invalid or expired")
			}
			return fmt.Errorf("group chat error: %w", err)
		}
		// Send QR
		if qrPhotoURL != "" {
			_ = deps.client.SendPhoto(ctx, botToken, *room.GroupChatID, qrPhotoURL, qrCaption)
		}
		return nil
	}

	var sendErrors []string
	for _, recipientID := range recipients {
		if err := deps.client.SendPhoto(ctx, botToken, recipientID, photoURL, caption); err != nil {
			if isZaloAuthError(err) && deps.markTokenInactive != nil {
				deps.markTokenInactive(ctx, managerID)
				return fmt.Errorf("zalo bot token is invalid or expired")
			}
			sendErrors = append(sendErrors, fmt.Sprintf("%s error: %v", recipientNames[recipientID], err))
		} else if qrPhotoURL != "" {
			// Send QR
			_ = deps.client.SendPhoto(ctx, botToken, recipientID, qrPhotoURL, qrCaption)
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

// saveZaloInvoiceImage stores a generated invoice image and returns a public URL Zalo can fetch.
func saveZaloInvoiceImage(publicBaseURL, invoiceID string, imageBytes []byte) (string, error) {
	dir := "uploads/zalo-invoices"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	fileName := fmt.Sprintf("%s_%s.png", invoiceID, uuid.Must(uuid.NewV7()).String())
	filePath := filepath.Join(dir, fileName)
	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	publicPath := fmt.Sprintf("/api/v1/uploads/zalo-invoices/%s", fileName)
	return publicBaseURL + publicPath, nil
}

// fetchPaymentQRImage returns QR image bytes for a payment link, honoring each provider's
// QR format: SePay's QRCode is already a ready-made QR image URL, while PayOS's QRCode is a
// raw VietQR string that must be rendered into an image before sending.
func fetchPaymentQRImage(provider, qrCode string) ([]byte, error) {
	if provider == model.PaymentProviderSePay {
		return httpGetImage(qrCode)
	}
	return downloadQRCodeImage(qrCode)
}

// downloadQRCodeImage renders a raw QR payload (e.g. a PayOS VietQR string) into a PNG via QuickChart.
func downloadQRCodeImage(qrCodeText string) ([]byte, error) {
	urlStr := fmt.Sprintf("https://quickchart.io/qr?text=%s&size=400", url.QueryEscape(qrCodeText))
	return httpGetImage(urlStr)
}

// httpGetImage downloads image bytes from an absolute URL.
func httpGetImage(urlStr string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image, status: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
