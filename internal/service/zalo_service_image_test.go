package service_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func TestZaloService_HandleWebhook_ImageProcessing_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	houseRepo := mock_model.NewMockHouseRepository(ctrl)
	invoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef"

	svc, err := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, invoiceRepo, nil, encKey)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	// Create a dummy image server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-image-data"))
	}))
	defer ts.Close()

	ctx := context.Background()
	managerID := "manager1"

	botToken, _ := security.Encrypt("bot123", []byte(encKey))

	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken:      &botToken,
	}, nil).AnyTimes()

	roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "group_123").Return(&model.Room{
		ID:      "room1",
		HouseID: "house1",
	}, nil)

	houseRepo.EXPECT().ListHouseByManagerID(ctx, managerID, 1000, 0, "").Return([]model.House{}, nil)
	houseRepo.EXPECT().GetByID(ctx, "house1", managerID).Return(&model.House{}, nil)

	invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "room1").Return(&model.Invoice{
		ID:     "inv1",
		Period: "2023-10",
		Status: "UNPAID",
	}, nil)

	invoiceRepo.EXPECT().UpdateInvoice(ctx, managerID, gomock.Any()).DoAndReturn(func(ctx context.Context, mgrID string, inv *model.Invoice) error {
		if inv.Status != model.InvoiceStatusPendingVerification {
			t.Errorf("expected status PENDING_VERIFICATION, got %s", inv.Status)
		}
		if inv.TransactionImagePath == nil {
			t.Errorf("expected transaction image path to be set")
		}
		return nil
	})

	zaloClient.EXPECT().SendMessage(ctx, "bot123", "group_123", gomock.Any()).Return(nil)

	payload := []byte(`{
		"event_name": "user_send_image",
		"message": {
			"chat": {
				"id": "group_123",
				"title": "House 1 Room 1"
			},
			"attachments": [
				{
					"type": "image",
					"payload": {
						"url": "` + ts.URL + `"
					}
				}
			]
		}
	}`)

	err = svc.HandleWebhook(ctx, managerID, payload, "some-secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestZaloService_HandleWebhook_ImageProcessing_RoomNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	houseRepo := mock_model.NewMockHouseRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef"
	svc, _ := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, nil, nil, encKey)

	ctx := context.Background()
	managerID := "manager1"

	botToken, _ := security.Encrypt("bot123", []byte(encKey))

	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken:      &botToken,
	}, nil).AnyTimes()

	houseRepo.EXPECT().ListHouseByManagerID(ctx, managerID, 1000, 0, "").Return([]model.House{}, nil)
	roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "group_123").Return(nil, errors.New("not found"))

	payload := []byte(`{
		"event_name": "user_send_image",
		"message": {
			"chat": {
				"id": "group_123",
				"title": "House 1 Room 1"
			},
			"attachments": [
				{
					"type": "image",
					"payload": {
						"url": "http://fake.url"
					}
				}
			]
		}
	}`)

	err := svc.HandleWebhook(ctx, managerID, payload, "some-secret")
	// Webhook should not fail entirely if processing image fails, it logs error but returns nil
	if err != nil {
		t.Fatalf("expected no error from webhook, got %v", err)
	}
}

func TestZaloService_SendInvoiceToZalo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	invoiceRepo := mock_model.NewMockInvoiceRepository(ctrl)
	imageService := mock_service.NewMockImageService(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef"
	svc, _ := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, nil, invoiceRepo, imageService, encKey)

	ctx := context.Background()
	managerID := "manager1"
	invoiceID := "inv1"

	botToken, _ := security.Encrypt("bot123", []byte(encKey))

	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken:      &botToken,
	}, nil)

	invoice := &model.InvoiceWithRoom{
		Invoice: model.Invoice{
			ID:     invoiceID,
			RoomID: "room1",
		},
	}
	invoiceRepo.EXPECT().GetInvoiceByID(ctx, managerID, invoiceID).Return(invoice, nil)

	groupID := "group_123"
	roomRepo.EXPECT().GetRoomByID(ctx, managerID, "room1").Return(&model.Room{
		ID:          "room1",
		GroupChatID: &groupID,
	}, nil)

	imageService.EXPECT().GenerateInvoiceImage(ctx, invoice).Return([]byte("fake-png"), nil)

	zaloClient.EXPECT().SendPhoto(ctx, "bot123", "group_123", []byte("fake-png"), gomock.Any()).Return(nil)

	tenantRepo.EXPECT().ListTenantByRoomID(ctx, managerID, "room1").Return([]model.FullInfoTenant{
		{ZaloUserID: "user_123", FullName: "Tenant A"},
	}, nil)

	zaloClient.EXPECT().SendPhoto(ctx, "bot123", "user_123", []byte("fake-png"), gomock.Any()).Return(nil)

	err := svc.SendInvoiceToZalo(ctx, managerID, invoiceID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
