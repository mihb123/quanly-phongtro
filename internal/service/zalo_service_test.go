package service_test

import (
	"context"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func TestZaloService_GetZaloConfigStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	houseRepo := mock_model.NewMockHouseRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef" // 32 bytes

	svc, err := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, nil, nil, encKey)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	ctx := context.Background()
	managerID := "manager1"

	// Test case: user exists, token is set
	token := "some-token"
	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken:    &token,
		IsZaloBotActive: true,
	}, nil)

	hasConfig, err := svc.GetZaloConfigStatus(ctx, managerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !hasConfig.HasConfig {
		t.Errorf("expected has_config true, got false")
	}
	if !hasConfig.IsActive {
		t.Errorf("expected is_active true, got false")
	}

	// Test case: user exists, token is nil
	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken: nil,
	}, nil)

	hasConfig, err = svc.GetZaloConfigStatus(ctx, managerID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if hasConfig.HasConfig {
		t.Errorf("expected has_config false, got true")
	}
}

func TestZaloService_SaveZaloConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	houseRepo := mock_model.NewMockHouseRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef"

	svc, err := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, nil, nil, encKey)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	ctx := context.Background()
	managerID := "manager1"
	botToken := "bot-token-123"

	// Mock GetMe to return success
	zaloClient.EXPECT().GetMe(ctx, botToken).Return(&service.ZaloAppInfo{
		AppID: "123",
	}, nil)

	zaloClient.EXPECT().SetWebhook(ctx, botToken, "http://test", "secret-xyz").Return(nil)

	// Mock UpdateUser to succeed. It should be called with encrypted tokens.
	userRepo.EXPECT().UpdateUser(ctx, managerID, gomock.Any()).DoAndReturn(func(ctx context.Context, id string, input model.UpdateUserInput) (*model.User, error) {
		if input.ZaloBotToken == nil {
			t.Errorf("expected tokens to be set")
		}
		// Verify we can decrypt it back
		decToken, _ := security.Decrypt(*input.ZaloBotToken, []byte(encKey))
		if decToken != botToken {
			t.Errorf("expected decrypted token %s, got %s", botToken, decToken)
		}
		return &model.User{}, nil
	})

	err = svc.SaveZaloConfig(ctx, managerID, botToken, "http://test", "secret-xyz")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestZaloService_HandleWebhook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	roomRepo := mock_model.NewMockRoomRepository(ctrl)
	tenantRepo := mock_model.NewMockTenantRepository(ctrl)
	houseRepo := mock_model.NewMockHouseRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	encKey := "z123456789abcdef0123456789abcdef"

	svc, err := service.NewZaloService(zaloClient, userRepo, roomRepo, tenantRepo, houseRepo, nil, nil, encKey)
	if err != nil {
		t.Fatalf("failed to init service: %v", err)
	}

	ctx := context.Background()
	managerID := "manager1"

	botToken, _ := security.Encrypt("bot123", []byte(encKey))

	userRepo.EXPECT().GetByUserID(ctx, managerID).Return(&model.User{
		ZaloBotToken:      &botToken,
	}, nil).AnyTimes()

	payload := []byte(`{
		"event_name": "group.bot.add",
		"group": {
			"id": "group_999",
			"name": "P101 Nhà A"
		}
	}`)

	houseRepo.EXPECT().ListHouseByManagerID(ctx, managerID, 1000, 0, "").Return([]model.House{
		{ID: "house1", Name: "Nhà A"},
	}, nil)

	roomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "house1").Return([]model.Room{
		{ID: "room1", Name: "P101"},
	}, nil)

	// Expect the room to be updated with group_999
	roomRepo.EXPECT().UpdateRoom(ctx, "room1", "house1", gomock.Any()).DoAndReturn(func(ctx context.Context, id, houseID string, input model.UpdateRoomParams) (*model.Room, error) {
		if input.GroupChatID == nil || *input.GroupChatID != "group_999" {
			t.Errorf("expected group chat ID to be group_999")
		}
		return &model.Room{}, nil
	})

	err = svc.HandleWebhook(ctx, managerID, payload, "some-secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
