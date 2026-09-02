package zalo_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	helpPrivatePayload = `{"message":{"text":"#help","from":{"id":"u1"}}}`
	helpGroupPayload   = `{"message":{"text":"#help","from":{"id":"u1"},"chat":{"id":"g1","chat_type":"GROUP"}}}`
)

// TestHandleWebhookHelpCommand covers the help reply for every sender role and connection state.
func TestHandleWebhookHelpCommand(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		chatID   string
		setup    func(m *zaloMocks, ctx context.Context, encToken string)
		contains []string
	}{
		{
			name:    "manager not linked yet gets activation instructions",
			payload: helpPrivatePayload,
			chatID:  "u1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
			},
			contains: []string{"chưa được liên kết", "mật khẩu đăng nhập"},
		},
		{
			name:    "unlinked tenant gets linking instructions only",
			payload: helpPrivatePayload,
			chatID:  "u1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(nil, errors.New("not linked"))
			},
			contains: []string{"chưa được liên kết", "bot ơi", "số điện thoại"},
		},
		{
			name:    "linked manager with one house sees the short syntax",
			payload: helpPrivatePayload,
			chatID:  "u1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("u1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "m1", Role: model.RoleManager}, nil)
				m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 2, 0, "").Return([]model.House{{ID: "h1", HouseCode: "679qt"}}, nil)
			},
			contains: []string{"QUẢN LÝ", "#dien <mã nhà> <phòng> <số mới>", "679qt", "#dien P201 661", "#help"},
		},
		{
			name:    "linked manager with several houses keeps the house code required",
			payload: helpPrivatePayload,
			chatID:  "u1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("u1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "m1", Role: model.RoleManager}, nil)
				m.houseRepo.EXPECT().ListHouseByManagerID(ctx, "m1", 2, 0, "").Return([]model.House{{ID: "h1"}, {ID: "h2"}}, nil)
			},
			contains: []string{"QUẢN LÝ", "#dien 679qt P201 661"},
		},
		{
			name:    "linked tenant sees the short room commands",
			payload: helpPrivatePayload,
			chatID:  "u1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.userRepo.EXPECT().GetByZaloUserID(ctx, "u1").Return(&model.User{ID: "t1", Role: model.RoleTenant}, nil)
				m.tenantRepo.EXPECT().GetFirstTenantByUserID(ctx, "m1", "t1").Return(&model.FullInfoTenant{RoomName: "P201"}, nil)
			},
			contains: []string{"KHÁCH THUÊ", "P201", "#dien <số mới>", "#nuoc <số mới>"},
		},
		{
			name:    "connected group lists the room commands",
			payload: helpGroupPayload,
			chatID:  "g1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(&model.Room{ID: "r1", Name: "P201"}, nil).Times(2)
			},
			contains: []string{"NHÓM P201", "#dien <số mới>", "@ bot"},
		},
		{
			name:    "unconnected group gets the group id instructions once",
			payload: helpGroupPayload,
			chatID:  "g1",
			setup: func(m *zaloMocks, ctx context.Context, encToken string) {
				m.userRepo.EXPECT().GetByUserID(ctx, "m1").Return(&model.User{ZaloBotToken: &encToken, ZaloUserID: ptr("z1")}, nil).Times(2)
				m.roomRepo.EXPECT().GetRoomByGroupChatID(ctx, "g1").Return(nil, errors.New("not found")).Times(2)
			},
			contains: []string{"chưa được kết nối", "g1", "Group Chat ID"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := setupZaloServiceTest(t)
			defer m.ctrl.Finish()
			ctx := context.Background()
			encToken, err := security.Encrypt("bot-token", m.encKey)
			require.NoError(t, err)

			tt.setup(m, ctx, encToken)

			var sent []string
			m.zaloClient.EXPECT().
				SendMessage(ctx, "bot-token", tt.chatID, gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _, text string) error {
					sent = append(sent, text)
					return nil
				})

			require.NoError(t, m.svc.HandleWebhook(ctx, "m1", []byte(tt.payload), "secret"))
			require.Len(t, sent, 1)
			for _, want := range tt.contains {
				assert.Contains(t, sent[0], want)
			}
		})
	}
}
