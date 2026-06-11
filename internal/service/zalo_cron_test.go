package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"go.uber.org/mock/gomock"
)

func TestZaloCronService_RunNow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)

	key := []byte("12345678901234567890123456789012") // 32 bytes
	cronSvc := service.NewZaloCronService(zaloClient, userRepo, key)

	validToken, _ := security.Encrypt("raw-token", key)
	badToken := "invalid-base64"
	nilTokenUser := model.User{ID: "user-nil", ZaloBotToken: nil}

	tests := []struct {
		name       string
		users      []model.User
		buildStubs func()
	}{
		{
			name: "Happy path - valid token and already active",
			users: []model.User{
				{ID: "user-1", ZaloBotToken: &validToken, IsZaloBotActive: true},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-1", ZaloBotToken: &validToken, IsZaloBotActive: true}}, nil)
				zaloClient.EXPECT().GetMe(gomock.Any(), "raw-token").Return(&service.ZaloAppInfo{}, nil)
				// Since it is already active, no UpdateUser should be called
			},
		},
		{
			name: "Happy path - valid token but inactive, should mark active",
			users: []model.User{
				{ID: "user-2", ZaloBotToken: &validToken, IsZaloBotActive: false},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-2", ZaloBotToken: &validToken, IsZaloBotActive: false}}, nil)
				zaloClient.EXPECT().GetMe(gomock.Any(), "raw-token").Return(&service.ZaloAppInfo{}, nil)
				userRepo.EXPECT().UpdateUser(gomock.Any(), "user-2", gomock.Any()).Return(&model.User{}, nil)
			},
		},
		{
			name: "API Error - should mark inactive",
			users: []model.User{
				{ID: "user-3", ZaloBotToken: &validToken, IsZaloBotActive: true},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-3", ZaloBotToken: &validToken, IsZaloBotActive: true}}, nil)
				zaloClient.EXPECT().GetMe(gomock.Any(), "raw-token").Return(nil, errors.New("api error"))
				userRepo.EXPECT().UpdateUser(gomock.Any(), "user-3", gomock.Any()).Return(&model.User{}, nil)
			},
		},
		{
			name: "Decrypt error - should mark inactive",
			users: []model.User{
				{ID: "user-4", ZaloBotToken: &badToken, IsZaloBotActive: true},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-4", ZaloBotToken: &badToken, IsZaloBotActive: true}}, nil)
				userRepo.EXPECT().UpdateUser(gomock.Any(), "user-4", gomock.Any()).Return(&model.User{}, nil)
			},
		},
		{
			name: "Nil token - should return early",
			users: []model.User{
				nilTokenUser,
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{nilTokenUser}, nil)
			},
		},
		{
			name: "GetAllUsersWithZaloToken returns error",
			users: nil,
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return(nil, errors.New("db error"))
			},
		},
		{
			name: "UpdateUser returns error when marking active",
			users: []model.User{
				{ID: "user-5", ZaloBotToken: &validToken, IsZaloBotActive: false},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-5", ZaloBotToken: &validToken, IsZaloBotActive: false}}, nil)
				zaloClient.EXPECT().GetMe(gomock.Any(), "raw-token").Return(&service.ZaloAppInfo{}, nil)
				userRepo.EXPECT().UpdateUser(gomock.Any(), "user-5", gomock.Any()).Return(nil, errors.New("update error"))
			},
		},
		{
			name: "UpdateUser returns error when marking inactive",
			users: []model.User{
				{ID: "user-6", ZaloBotToken: &validToken, IsZaloBotActive: true},
			},
			buildStubs: func() {
				userRepo.EXPECT().GetAllUsersWithZaloToken(gomock.Any()).Return([]model.User{{ID: "user-6", ZaloBotToken: &validToken, IsZaloBotActive: true}}, nil)
				zaloClient.EXPECT().GetMe(gomock.Any(), "raw-token").Return(nil, errors.New("api error"))
				userRepo.EXPECT().UpdateUser(gomock.Any(), "user-6", gomock.Any()).Return(nil, errors.New("update error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.buildStubs()
			cronSvc.RunNow()
		})
	}
}

func TestZaloCronService_StartStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mock_model.NewMockUserRepository(ctrl)
	zaloClient := mock_service.NewMockZaloClient(ctrl)
	key := []byte("12345678901234567890123456789012")

	svc := service.NewZaloCronService(zaloClient, userRepo, key)
	
	// Start in a goroutine
	go svc.Start()
	
	// Sleep a bit to let it start
	time.Sleep(50 * time.Millisecond)
	
	// Stop it
	svc.Stop()
}
