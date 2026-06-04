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
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", []byte("img"), gomock.Any()).Return(nil)
				m.tenantRepo.EXPECT().ListTenantByRoomID(ctx, "m1", "r1").Return([]model.FullInfoTenant{{ZaloUserID: "t1"}}, nil)
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "t1", []byte("img"), gomock.Any()).Return(nil)
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
				m.zaloClient.EXPECT().SendPhoto(ctx, "bot-token", "g1", []byte("img"), gomock.Any()).Return(errors.New("invalid access token -216"))
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := m.tsvc.ProcessTransactionImage(ctx, "m1", "g1", ts.URL)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
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
			name: "missing secret",
			payload: `{}`,
			setup: func() {},
			wantErr: true,
		},
		{
			name: "group.bot.add autolink",
			payload: `{"event_name":"group.bot.add","group":{"id":"g1","name":"P101 N1"}}`,
			setup: func() {
				m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 1000, 0, "").Return([]model.House{{ID: "h1", Name: "N1"}}, nil)
				m.roomRepo.EXPECT().ListAllRoomsByHouseID(ctx, "h1").Return([]model.Room{{ID: "r1", Name: "P101"}}, nil)
				m.roomRepo.EXPECT().UpdateRoom(ctx, "r1", "h1", gomock.Any()).Return(&model.Room{}, nil)
			},
			wantErr: false,
		},
		{
			name: "private chat kich hoat success",
			payload: `{"message":{"text":"Kich hoat Manager One","from":{"id":"u1"}}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{FullName: "Manager One"}, nil)
				m.userRepo.EXPECT().UpdateUser(ctx, "m1", gomock.Any()).Return(&model.User{}, nil)
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "private chat botoi triggers phone request",
			payload: `{"message":{"text":"botoi","from":{"id":"u1"}}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
				m.zaloClient.EXPECT().SendMessage(ctx, "bot-token", "u1", gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
        {
			name: "private chat phone number links manager",
			payload: `{"message":{"text":"0912345678","from":{"id":"u1"}}}`,
			setup: func() {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil)
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
