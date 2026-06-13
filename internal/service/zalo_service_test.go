package service_test

import (
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/mock/mock_service"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/mihb123/quanly-phongtro/internal/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func ptr[T any](v T) *T { return &v }

type zaloMocks struct {
	ctrl        *gomock.Controller
	zaloClient  *mock_service.MockZaloClient
	userRepo    *mock_model.MockUserRepository
	roomRepo    *mock_model.MockRoomRepository
	tenantRepo  *mock_model.MockTenantRepository
	houseRepo   *mock_model.MockHouseRepository
	invoiceRepo *mock_model.MockInvoiceRepository
	imgSvc      *mock_service.MockImageService
	svc         service.ZaloService
	tsvc        service.ZaloServiceTesting
	encKey      []byte
}

func setupZaloServiceTest(t *testing.T) *zaloMocks {
	t.Cleanup(func() {
		_ = os.RemoveAll("uploads/zalo-invoices")
	})

	ctrl := gomock.NewController(t)
	m := &zaloMocks{
		ctrl:        ctrl,
		zaloClient:  mock_service.NewMockZaloClient(ctrl),
		userRepo:    mock_model.NewMockUserRepository(ctrl),
		roomRepo:    mock_model.NewMockRoomRepository(ctrl),
		tenantRepo:  mock_model.NewMockTenantRepository(ctrl),
		houseRepo:   mock_model.NewMockHouseRepository(ctrl),
		invoiceRepo: mock_model.NewMockInvoiceRepository(ctrl),
		imgSvc:      mock_service.NewMockImageService(ctrl),
	}

	encKeyStr := "z123456789abcdef0123456789abcdef"
	svc, err := service.NewZaloService(m.zaloClient, m.userRepo, m.roomRepo, m.tenantRepo, m.houseRepo, m.invoiceRepo, m.imgSvc, encKeyStr)
	assert.NoError(t, err)

	m.svc = svc
	m.tsvc = service.CastToTesting(svc)
	m.encKey = service.GetEncryptionKey(svc)
	return m
}

func TestDecodeEncryptionKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid hex 32 bytes", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", false},
		{"valid raw 32 bytes", "z123456789abcdef0123456789abcdef", false},
		{"invalid hex length", "0123456789abcdef", true},
		{"invalid raw length", "short", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.DecodeEncryptionKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveZaloConfig(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "success",
			setup: func() {
				m.zaloClient.EXPECT().GetMe(ctx, "token").Return(&service.ZaloAppInfo{}, nil)
				m.zaloClient.EXPECT().SetWebhook(ctx, "token", "webhook", "secret").Return(nil)
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
			},
			wantErr: false,
		},
		{
			name: "get me error",
			setup: func() {
				m.zaloClient.EXPECT().GetMe(ctx, "token").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "set webhook error",
			setup: func() {
				m.zaloClient.EXPECT().GetMe(ctx, "token").Return(&service.ZaloAppInfo{}, nil)
				m.zaloClient.EXPECT().SetWebhook(ctx, "token", "webhook", "secret").Return(errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "db error",
			setup: func() {
				m.zaloClient.EXPECT().GetMe(ctx, "token").Return(&service.ZaloAppInfo{}, nil)
				m.zaloClient.EXPECT().SetWebhook(ctx, "token", "webhook", "secret").Return(nil)
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := m.svc.SaveZaloConfig(ctx, "m1", "token", "webhook", "secret")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetZaloConfigStatus(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	tests := []struct {
		name    string
		setup   func()
		want    service.ZaloBotStatus
		wantErr bool
	}{
		{
			name: "success",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, IsZaloBotActive: true, ZaloUserID: ptr("z1")}, nil)
				m.zaloClient.EXPECT().GetMe(ctx, "bot-token").Return(&service.ZaloAppInfo{AppID: "app1"}, nil)
			},
			want:    service.ZaloBotStatus{HasConfig: true, IsActive: true, IsLinked: true, BotID: "app1", ManagerID: "m1"},
			wantErr: false,
		},
		{
			name: "no config",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
			},
			want:    service.ZaloBotStatus{HasConfig: false, IsActive: false, IsLinked: false, ManagerID: "m1"},
			wantErr: false,
		},
		{
			name: "user repo error",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			res, err := m.svc.GetZaloConfigStatus(ctx, "m1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, res)
			}
		})
	}
}

func TestGetDecryptedToken(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
		want    string
	}{
		{
			name: "success",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
			},
			want: "bot-token",
		},
		{
			name: "db error",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "no token",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
			},
			wantErr: true,
		},
		{
			name: "bad token",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: ptr("bad")}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			res, err := m.tsvc.GetDecryptedToken(ctx, "m1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, res)
			}
		})
	}
}

func TestSendInvoiceToZalo(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)
	inv := &model.InvoiceWithRoom{Invoice: model.Invoice{ID: "i1", RoomID: "r1", Period: "09-2023", TotalAmount: 1000}, RoomName: "101"}
	room := &model.Room{ID: "r1", GroupChatID: ptr("g1")}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "success to group and tenant",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(room, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", gomock.Any(), gomock.Any()).Return(nil)
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{ZaloUserID: "t1"}}, nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "t1", gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "zalo auth error marks token inactive",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(room, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", gomock.Any(), gomock.Any()).Return(errors.New("invalid access token -216"))
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
			},
			wantErr: true,
		},
		{
			name: "no linked group or tenant",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1"}, nil) // no group
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{}}, nil) // no tenant
			},
			wantErr: true,
		},
		{
			name: "GenerateInvoiceImage fails",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1", GroupChatID: ptr("g1")}, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return(nil, errors.New("img err"))
			},
			wantErr: true,
		},
		{
			name: "SendPhoto fails",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1", GroupChatID: ptr("g1")}, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", gomock.Any(), gomock.Any()).Return(errors.New("api err"))
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{}, nil)
			},
			wantErr: true,
		},
		{
			name: "SendPhoto auth error",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1", GroupChatID: ptr("g1")}, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", gomock.Any(), gomock.Any()).Return(errors.New("invalid access token"))
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
			},
			wantErr: true,
		},
		{
			name: "Send to tenant successfully",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1"}, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{ZaloUserID: "u1"}}, nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "u1", gomock.Any(), gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "Send to tenant fails",
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
				m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(&model.Room{ID: "r1"}, nil)
				m.imgSvc.EXPECT().GenerateInvoiceImage(ctx, inv).Return([]byte("img"), nil)
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{ZaloUserID: "u1"}}, nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "u1", gomock.Any(), gomock.Any()).Return(errors.New("tenant err"))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := m.svc.SendInvoiceToZalo(ctx, "m1", "i1")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessTransactionImage(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	// mock image server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("fake-img"))
	}))
	defer ts.Close()

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "success",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(&model.Invoice{ID: "i1", Status: "UNPAID"}, nil)
				m.invoiceRepo.EXPECT().UpdateInvoice(ctx, "m1", gomock.Any()).Return(nil)
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "g1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "room not found",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "no unpaid invoice ignores",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(nil, model.ErrInvoiceNotFound)
			},
			wantErr: false,
		},
		{
			name: "house not owned",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(nil, errors.New("err"))
			},
			wantErr: true,
		},
		{
			name: "update invoice success",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(&model.Invoice{ID: "i1", Status: "UNPAID", Period: "05/2026"}, nil)
				m.invoiceRepo.EXPECT().UpdateInvoice(ctx, "m1", gomock.Any()).Return(nil)
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "g1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "update invoice error",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(&model.Invoice{ID: "i1", Status: "UNPAID"}, nil)
				m.invoiceRepo.EXPECT().UpdateInvoice(ctx, "m1", gomock.Any()).Return(errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name: "invoice lookup error",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(nil, errors.New("db err"))
			},
			wantErr: true,
		},
		{
			name: "image server status error",
			setup: func() {
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
				m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
				m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(&model.Invoice{ID: "i1", Status: "UNPAID"}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			imageURL := ts.URL
			if tt.name == "image server status error" {
				badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				defer badServer.Close()
				imageURL = badServer.URL
			}
			err := m.tsvc.ProcessTransactionImage(ctx, "m1", "g1", imageURL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

	t.Run("image download fails", func(t *testing.T) {
		m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", HouseID: "h1"}, nil)
		m.houseRepo.EXPECT().GetByID(ctx, "h1", "m1").Return(&model.House{}, nil)
		m.invoiceRepo.EXPECT().GetLatestUnpaidInvoiceByRoomID(ctx, "r1").Return(&model.Invoice{ID: "i1", Status: "UNPAID"}, nil)
		err := m.tsvc.ProcessTransactionImage(ctx, "m1", "g1", "http://invalid-url-that-fails")
		assert.Error(t, err)
	})
}

func TestHandleWebhook(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	tests := []struct {
		name    string
		payload string
		setup   func()
		wantErr bool
	}{
		{
			name:    "missing secret",
			payload: `{}`,
			setup:   func() {},
			wantErr: true,
		},
		{
			name:    "group.bot.add autolink",
			payload: `{"event_name":"group.bot.add","group":{"id":"g1","name":"P101 N1"}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
				m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return([]model.House{{ID: "h1", Name: "N1"}}, nil)
				m.roomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "h1").Return([]model.Room{{ID: "r1", Name: "P101"}}, nil)
				m.roomRepo.EXPECT().UpdateRoom(ctx, "r1", "h1", gomock.Any()).Return(&model.Room{}, nil)
			},
			wantErr: false,
		},
		{
			name:    "private chat password success",
			payload: `{"message":{"text":"password123","from":{"id":"u1"}}}`,
			setup: func() {
				hashedPassword, _ := security.NewBcryptHasher().Hash("password123")
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{PasswordHash: hashedPassword, ZaloBotToken: &encToken}, nil).Times(2)
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "private chat botoi triggers phone request",
			payload: `{"message":{"text":"botoi","from":{"id":"u1"}}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "private chat phone number links manager",
			payload: `{"message":{"text":"0912345678","from":{"id":"u1"}}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
				m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "m1", Role: model.RoleManager}, nil)
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			secretHeader := "secret"
			if tt.name == "missing secret" {
				secretHeader = ""
			}
			err := m.svc.HandleWebhook(ctx, "m1", []byte(tt.payload), secretHeader)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHandleWebhook_OfficialZaloPayload(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)
	encSecret, _ := security.Encrypt("secret", m.encKey)

	t.Run("private text under result replies to chat id", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloWebhookSecret: &encSecret, ZaloUserID: ptr("z1")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", "Xin chào! Vui lòng nhập số điện thoại của bạn để liên kết tài khoản nhận thông báo.").Return(nil)

		payload := `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"u1","chat_type":"PRIVATE"},"text":"botoi"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("group text under result replies to group chat id", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloWebhookSecret: &encSecret, ZaloUserID: ptr("z1")}, nil).Times(2)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "group-1", "Bot đã kết nối thành công").Return(nil)

		payload := `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"group-1","chat_type":"GROUP"},"text":"bot ơi"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("invalid saved secret rejects webhook", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloWebhookSecret: &encSecret}, nil)

		payload := `{"ok":true,"result":{"event_name":"message.text.received","message":{"from":{"id":"u1"},"chat":{"id":"u1","chat_type":"PRIVATE"},"text":"botoi"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "wrong")
		assert.Error(t, err)
	})
}

// TestAutoLinkRoom_ListHouseError covers room autolink repository failure.
func TestAutoLinkRoom_ListHouseError(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()

	m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return(nil, errors.New("db error"))

	err := m.tsvc.AutoLinkRoom(ctx, "m1", "g1", "P101 N1")
	assert.Error(t, err)
}

// TestHandleWebhook_PrivateChatBranches covers linked-manager private chat flows.
func TestHandleWebhook_PrivateChatBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("invalid json", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()

		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{bad json`), "secret")
		assert.Error(t, err)
	})

	t.Run("user repo error", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("db error"))

		payload := `{"message":{"text":"bot oi","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.Error(t, err)
	})

	t.Run("token decrypt error is logged", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: ptr("bad-token"), ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)

		payload := `{"message":{"text":"bot oi","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("phone belongs to already linked account", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "tenant-1", ZaloUserID: ptr("existing-zalo")}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"0912345678","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("manager phone belongs to another manager", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "other-manager", Role: model.RoleManager}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"+84912345678","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("tenant waits for manager link", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "tenant-1", Role: model.RoleTenant}, nil)
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"0912345678","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("tenant links and notifies manager", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "tenant-1", Role: model.RoleTenant}, nil)
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloUserID: ptr("mgr-zalo")}, nil)
		m.userRepo.EXPECT().UpdateUser(ctx, "tenant-1", gomock.Any()).Return(&model.User{}, nil)
		m.tenantRepo.EXPECT().GetFirstTenantByUserID(ctx, "m1", "tenant-1").Return(&model.FullInfoTenant{FullName: "Tenant A", RoomName: "101"}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "mgr-zalo", gomock.Any()).Return(nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"0912345678","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("already linked manager asks bot", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{Role: model.RoleManager}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"bot oi","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("unregistered phone reports missing account", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(nil, errors.New("not found"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"0912345678","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("unlinked user sends unknown text", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"hello","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("already linked tenant asks bot", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{Role: model.RoleTenant}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"bot oi","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})

	t.Run("already linked user sends other text", func(t *testing.T) {
		m := setupZaloServiceTest(t)
		defer m.ctrl.Finish()
		encToken, _ := security.Encrypt("bot-token", m.encKey)

		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("mgr-zalo")}, nil).Times(2)
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{Role: model.RoleTenant}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(errors.New("send error"))

		payload := `{"message":{"text":"help","from":{"id":"u1"}}}`
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "secret")
		assert.NoError(t, err)
	})
}

func TestNewZaloService(t *testing.T) {
	_, err := service.NewZaloService(nil, nil, nil, nil, nil, nil, nil, "short")
	assert.Error(t, err)

	hexKey := hex.EncodeToString([]byte("12345678901234567890123456789012"))
	svc, err := service.NewZaloService(nil, nil, nil, nil, nil, nil, nil, hexKey)
	assert.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestSendTextMessage(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		encToken, _ := security.Encrypt("bot-token", m.encKey)
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "c1", "hello").Return(nil)
		err := m.svc.SendTextMessage(ctx, "m1", "c1", "hello")
		assert.NoError(t, err)
	})

	t.Run("get token error", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(nil, errors.New("err"))
		err := m.svc.SendTextMessage(ctx, "m1", "c1", "hello")
		assert.Error(t, err)
	})
}

func TestHandleWebhook_AutoLink_Branches(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()

	t.Run("list rooms error", func(t *testing.T) {
		payload := `{"event_name":"group.bot.add","group":{"id":"g1","name":"P101 N1"}}`
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
		m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return([]model.House{{ID: "h1", Name: "N1"}}, nil)
		m.roomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "h1").Return(nil, errors.New("err"))
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "test-secret")
		assert.NoError(t, err)
	})

	t.Run("no match", func(t *testing.T) {
		payload := `{"event_name":"group.bot.add","group":{"id":"g1","name":"P999 N1"}}`
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{}, nil)
		m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return([]model.House{{ID: "h1", Name: "N1"}}, nil)
		m.roomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "h1").Return([]model.Room{{ID: "r1", Name: "101"}}, nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(payload), "test-secret")
		assert.NoError(t, err)
	})
}

func TestSendInvoiceToZalo_Extra(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)
	inv := &model.InvoiceWithRoom{
		Invoice: model.Invoice{ID: "i1", RoomID: "r1", Status: "UNPAID", Period: "05/2026", TotalAmount: 1500000},
	}

	t.Run("get room fails", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
		m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(inv, nil)
		m.roomRepo.EXPECT().GetRoomByID(ctx, "m1", "r1").Return(nil, errors.New("db err"))
		err := m.svc.SendInvoiceToZalo(ctx, "m1", "i1")
		assert.Error(t, err)
	})

	t.Run("get invoice fails", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
		m.invoiceRepo.EXPECT().GetInvoiceByID(ctx, "m1", "i1").Return(nil, errors.New("db err"))
		err := m.svc.SendInvoiceToZalo(ctx, "m1", "i1")
		assert.Error(t, err)
	})
}

func TestProcessTransactionImage_Extra(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()

	t.Run("room error", func(t *testing.T) {
		m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(nil, errors.New("err"))
		err := m.tsvc.ProcessTransactionImage(ctx, "m1", "g1", "img")
		assert.Error(t, err)
	})
}

func TestHandleWebhook_PasswordActivation(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)
	hashedPassword, _ := security.NewBcryptHasher().Hash("password123")

	t.Run("incorrect password", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{PasswordHash: hashedPassword, ZaloBotToken: &encToken}, nil).Times(2)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", "Manager cần nhập mật khẩu để kích hoạt tài khoản").Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"wrongpass","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("update error", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{PasswordHash: hashedPassword, ZaloBotToken: &encToken}, nil).Times(2)
		m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(nil, errors.New("db err"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"password123","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})
}

func TestHandleWebhook_Phone(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	t.Run("phone format manager diff id", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("err"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(&model.User{ID: "m2", Role: model.RoleManager}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"0912345678","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

}

func TestIsZaloAuthError(t *testing.T) {
	assert.False(t, service.IsZaloAuthError(nil))
	assert.False(t, service.IsZaloAuthError(errors.New("random error")))
	assert.True(t, service.IsZaloAuthError(errors.New("some error -216")))
	assert.True(t, service.IsZaloAuthError(errors.New("INVALID ACCESS TOKEN")))
	assert.True(t, service.IsZaloAuthError(errors.New("unauthorized request")))
}

func TestHandleWebhook_OtherBranches(t *testing.T) {
	m := setupZaloServiceTest(t)
	defer m.ctrl.Finish()
	ctx := context.Background()
	encToken, _ := security.Encrypt("bot-token", m.encKey)

	t.Run("user already linked says botoi as manager", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "m1", Role: model.RoleManager}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"botoi","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("user already linked says botoi as tenant", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "t1", Role: model.RoleTenant}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"botơi","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("user already linked random message", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "t1", Role: model.RoleTenant}, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"hello bot","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("botoi group chat", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return(nil, nil)
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "g1", "Bot đã kết nối thành công").Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"event_name":"user_send_text","message":{"text":"botoi","chat":{"id":"g1","title":"G1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("user not linked, random message", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not found"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", "Để bắt đầu kết nối, vui lòng gõ 'bot ơi'.").Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"what","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("user not linked, phone not found", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not found"))
		m.userRepo.EXPECT().GetByPhone(ctx, "0912345678").Return(nil, errors.New("not found"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", "Số điện thoại chưa được đăng ký trong hệ thống. Vui lòng kiểm tra lại.").Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"0912345678","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})

	t.Run("user not linked, says botoi", func(t *testing.T) {
		m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).AnyTimes()
		m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not found"))
		m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", "Xin chào! Vui lòng nhập số điện thoại của bạn để liên kết tài khoản nhận thông báo.").Return(nil)
		err := m.svc.HandleWebhook(ctx, "m1", []byte(`{"message":{"text":"botoi","from":{"id":"u1"}}}`), "secret")
		assert.NoError(t, err)
	})
}
