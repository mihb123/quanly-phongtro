package zalo

import (
	"context"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	tenantsvc "github.com/mihb123/quanly-phongtro/internal/service/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type recordingTenantService struct {
	tenantsvc.TenantService
	tenantID string
	input    tenantsvc.UpdateTenantInput
	err      error
}

// UpdateTenantInfo records the requested tenant patch for manager command tests.
func (s *recordingTenantService) UpdateTenantInfo(ctx context.Context, managerID, tenantID string, in tenantsvc.UpdateTenantInput) (*model.FullInfoTenant, error) {
	s.tenantID = tenantID
	s.input = in
	if s.err != nil {
		return nil, s.err
	}
	return &model.FullInfoTenant{TenantID: tenantID}, nil
}

// managerCommandFixture wires the command service with the mocks the manager commands need.
type managerCommandFixture struct {
	service    *zaloInvoiceCommandServiceImpl
	rooms      *mock_model.MockRoomRepository
	houses     *mock_model.MockHouseRepository
	tenants    *mock_model.MockTenantRepository
	users      *mock_model.MockUserRepository
	tenantSvc  *recordingTenantService
	zaloClient *recordingZaloClient
}

// newManagerCommandFixture builds the fixture with a linked manager sender by default.
func newManagerCommandFixture(t *testing.T, ctx context.Context, ctrl *gomock.Controller) *managerCommandFixture {
	t.Helper()

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)

	fixture := &managerCommandFixture{
		rooms:      mock_model.NewMockRoomRepository(ctrl),
		houses:     mock_model.NewMockHouseRepository(ctrl),
		tenants:    mock_model.NewMockTenantRepository(ctrl),
		users:      mock_model.NewMockUserRepository(ctrl),
		tenantSvc:  &recordingTenantService{},
		zaloClient: &recordingZaloClient{},
	}
	fixture.users.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()
	fixture.service = &zaloInvoiceCommandServiceImpl{
		roomRepo:      fixture.rooms,
		houseRepo:     fixture.houses,
		tenantRepo:    fixture.tenants,
		tenantService: fixture.tenantSvc,
		userRepo:      fixture.users,
		pendingRepo:   &memoryPendingInvoiceUpdateRepository{},
		zaloClient:    fixture.zaloClient,
		encryptionKey: encryptionKey,
	}
	return fixture
}

// expectManagerSender lets the fixture's sender pass the manager-only guard.
func (f *managerCommandFixture) expectManagerSender(ctx context.Context) {
	f.users.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "manager-1", Role: model.RoleManager}, nil)
}

// expectSingleHouseRoom resolves "P201" in the manager's only house.
func (f *managerCommandFixture) expectSingleHouseRoom(ctx context.Context, room model.Room) {
	f.houses.EXPECT().ListHouseByManagerID(ctx, "manager-1", 2, 0, "").Return([]model.House{*usageHouse()}, nil)
	f.rooms.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{room}, nil)
}

// privateManagerChat is the webhook context of the manager's private chat with the bot.
func privateManagerChat(text string) webhookMessageContext {
	return webhookMessageContext{chatID: "sender-1", replyChatID: "sender-1", senderID: "sender-1", text: text}
}

// TestUpdateTenantPhone_ReplacesExistingNumber verifies the manager can overwrite a tenant phone.
func TestUpdateTenantPhone_ReplacesExistingNumber(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.tenants.EXPECT().
		ListTenantByRoomID(ctx, "manager-1", "room-1").
		Return([]model.FullInfoTenant{{TenantID: "tenant-1", FullName: "Nguyễn An", Phone: "0900000000", ZaloUserID: "z1"}}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 0912345678"))

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", f.tenantSvc.tenantID)
	require.NotNil(t, f.tenantSvc.input.Phone)
	assert.Equal(t, "0912345678", *f.tenantSvc.input.Phone)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "0900000000 → 0912345678")
}

// TestUpdateTenantPhone_UsesGroupChatRoom verifies the short form works inside a room group chat.
func TestUpdateTenantPhone_UsesGroupChatRoom(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.rooms.EXPECT().GetRoomByGroupChatID(ctx, "chat-group-1").Return(&model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"}, nil)
	f.houses.EXPECT().GetByID(ctx, "house-1", "manager-1").Return(usageHouse(), nil)
	f.tenants.EXPECT().
		ListTenantByRoomID(ctx, "manager-1", "room-1").
		Return([]model.FullInfoTenant{{TenantID: "tenant-1", FullName: "Nguyễn An"}}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "chat-group-1",
		replyChatID: "chat-group-1",
		senderID:    "sender-1",
		isGroupChat: true,
		text:        "#update-tenant 0912 345 678",
	})

	require.NoError(t, err)
	require.NotNil(t, f.tenantSvc.input.Phone)
	assert.Equal(t, "0912345678", *f.tenantSvc.input.Phone)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "chưa có → 0912345678")
	assert.Contains(t, f.zaloClient.messages[0], "chưa liên kết Zalo")
}

// TestUpdateTenantPhone_SharedRoomIsReported verifies a room with several tenants is never guessed.
func TestUpdateTenantPhone_SharedRoomIsReported(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.tenants.EXPECT().
		ListTenantByRoomID(ctx, "manager-1", "room-1").
		Return([]model.FullInfoTenant{
			{TenantID: "tenant-1", FullName: "Nguyễn An", Phone: "0900000001"},
			{TenantID: "tenant-2", FullName: "Trần Bình"},
		}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 0912345678"))

	require.NoError(t, err)
	assert.Empty(t, f.tenantSvc.tenantID)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "Nguyễn An")
	assert.Contains(t, f.zaloClient.messages[0], "Trần Bình")
}

// TestUpdateTenantPhone_DuplicateNumberIsReported verifies the phone uniqueness rule is surfaced.
func TestUpdateTenantPhone_DuplicateNumberIsReported(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.tenantSvc.err = model.ErrPhoneAlreadyExists
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.tenants.EXPECT().
		ListTenantByRoomID(ctx, "manager-1", "room-1").
		Return([]model.FullInfoTenant{{TenantID: "tenant-1", FullName: "Nguyễn An", Phone: "0900000000"}}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 0912345678"))

	require.NoError(t, err)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "đã được dùng cho một tài khoản khác")
}

// TestUpdateTenantPhone_RejectsNonManager verifies tenants cannot edit phone numbers from chat.
func TestUpdateTenantPhone_RejectsNonManager(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.users.EXPECT().GetByZaloUserID(ctx, "sender-1").Return(&model.User{ID: "user-1", Role: model.RoleTenant}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 0912345678"))

	require.NoError(t, err)
	assert.Empty(t, f.tenantSvc.tenantID)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "Chỉ quản lý")
}

// TestUpdateTenantPhone_InvalidPhoneShowsSyntax verifies an unusable number is not written.
func TestUpdateTenantPhone_InvalidPhoneShowsSyntax(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 12345"))

	require.NoError(t, err)
	assert.Empty(t, f.tenantSvc.tenantID)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "#update-tenant <mã nhà> <phòng> <số điện thoại>")
}

// TestUpdateRoomGroup_ConnectsCurrentGroup verifies the in-group form links the chat it was sent from.
func TestUpdateRoomGroup_ConnectsCurrentGroup(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.rooms.EXPECT().GetRoomByGroupChatID(ctx, "9876543210").Return(nil, model.ErrRoomNotFound)

	var params model.UpdateRoomParams
	f.rooms.EXPECT().
		UpdateRoom(ctx, "room-1", "house-1", gomock.Any()).
		DoAndReturn(func(_ context.Context, _, _ string, in model.UpdateRoomParams) (*model.Room, error) {
			params = in
			return &model.Room{ID: "room-1"}, nil
		})

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "9876543210",
		replyChatID: "9876543210",
		senderID:    "sender-1",
		isGroupChat: true,
		text:        "#update-room P201",
	})

	require.NoError(t, err)
	require.NotNil(t, params.GroupChatID)
	assert.Equal(t, "9876543210", *params.GroupChatID)
	assert.Equal(t, "P201", params.Name)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "Đã kết nối phòng P201")
}

// TestUpdateRoomGroup_ExplicitIDFromPrivateChat verifies the copy-paste form works outside a group.
func TestUpdateRoomGroup_ExplicitIDFromPrivateChat(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.houses.EXPECT().ListHouseByManagerID(ctx, "manager-1", 2, 0, "").Return([]model.House{{ID: "house-1"}, {ID: "house-2"}}, nil)
	f.houses.EXPECT().GetHouseByCode(ctx, "manager-1", "679qt").Return(usageHouse(), nil)
	f.rooms.EXPECT().ListAllRoomsByHouseID(ctx, "house-1").Return([]model.Room{{ID: "room-1", HouseID: "house-1", Name: "P201", GroupChatID: ptrString("111111")}}, nil)
	f.rooms.EXPECT().GetRoomByGroupChatID(ctx, "9876543210").Return(nil, model.ErrRoomNotFound)
	f.rooms.EXPECT().UpdateRoom(ctx, "room-1", "house-1", gomock.Any()).Return(&model.Room{ID: "room-1"}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-room 679qt P201 9876543210"))

	require.NoError(t, err)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "111111 → 9876543210")
}

// TestUpdateRoomGroup_RejectsIDUsedByAnotherRoom verifies one group chat maps to a single room.
func TestUpdateRoomGroup_RejectsIDUsedByAnotherRoom(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.rooms.EXPECT().GetRoomByGroupChatID(ctx, "9876543210").Return(&model.Room{ID: "room-9", Name: "P101"}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-room P201 9876543210"))

	require.NoError(t, err)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "đang được kết nối với phòng P101")
}

// TestUpdateRoomGroup_UnlinkedGroupWithoutRoomShowsGroupID verifies the failure reply carries the
// group ID, replacing the separate "group not connected" notice.
func TestUpdateRoomGroup_UnlinkedGroupWithoutRoomShowsGroupID(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.expectManagerSender(ctx)
	f.rooms.EXPECT().GetRoomByGroupChatID(ctx, "9876543210").Return(nil, model.ErrRoomNotFound)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", webhookMessageContext{
		chatID:      "9876543210",
		replyChatID: "9876543210",
		senderID:    "sender-1",
		isGroupChat: true,
		text:        "#update-room",
	})

	require.NoError(t, err)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "9876543210")
	assert.Contains(t, f.zaloClient.messages[0], "#update-room <mã nhà> <phòng>")
}

// TestUpdateRoomGroup_PrivateChatNeedsGroupID verifies the private-chat form requires an ID.
func TestUpdateRoomGroup_PrivateChatNeedsGroupID(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-room 679qt P201"))

	require.NoError(t, err)
	require.Len(t, f.zaloClient.messages, 1)
	assert.Contains(t, f.zaloClient.messages[0], "Vui lòng nhập mã nhóm")
}

// TestManagerCommandsIgnorePendingState verifies a manager edit is not swallowed by a pending prompt.
func TestManagerCommandsIgnorePendingState(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	f := newManagerCommandFixture(t, ctx, ctrl)
	f.service.pendingRepo = &memoryPendingInvoiceUpdateRepository{
		pending: &model.PendingInvoiceUpdate{
			ID:          "pending-1",
			ChatID:      "sender-1",
			ActionType:  model.PendingActionAwaitPeriod,
			PendingData: map[string]any{"period_options": map[string]any{"9": "2026-09"}},
		},
	}
	f.expectManagerSender(ctx)
	f.expectSingleHouseRoom(ctx, model.Room{ID: "room-1", HouseID: "house-1", Name: "P201"})
	f.tenants.EXPECT().
		ListTenantByRoomID(ctx, "manager-1", "room-1").
		Return([]model.FullInfoTenant{{TenantID: "tenant-1", FullName: "Nguyễn An", Phone: "0900000000"}}, nil)

	err := f.service.HandleInvoiceCommand(ctx, "manager-1", privateManagerChat("#update-tenant P201 0912345678"))

	require.NoError(t, err)
	assert.Equal(t, "tenant-1", f.tenantSvc.tenantID)
}

// TestGroupLinkingReplySuppressed verifies which commands answer the unlinked-group case themselves.
func TestGroupLinkingReplySuppressed(t *testing.T) {
	assert.True(t, groupLinkingReplySuppressed("#help"))
	assert.True(t, groupLinkingReplySuppressed("#update-room 679qt P201"))
	assert.True(t, groupLinkingReplySuppressed("#update-tenant 0912345678"))
	assert.False(t, groupLinkingReplySuppressed("#dien 661"))
	assert.False(t, groupLinkingReplySuppressed("bot ơi"))
}

// TestIsZaloGroupChatID verifies group IDs are told apart from room names in the same position.
func TestIsZaloGroupChatID(t *testing.T) {
	assert.True(t, IsZaloGroupChatID("9876543210"))
	assert.False(t, IsZaloGroupChatID("P201"))
	assert.False(t, IsZaloGroupChatID("201"))
	assert.False(t, IsZaloGroupChatID(""))
}

// TestNormalizePhone covers the accepted local and international formats.
func TestNormalizePhone(t *testing.T) {
	for _, raw := range []string{"0912345678", "0912 345 678", "+84912345678", "84912345678", "0912.345.678"} {
		phone, ok := NormalizePhone(raw)
		require.Truef(t, ok, "expected %q to be a phone", raw)
		assert.Equal(t, "0912345678", phone)
	}
	for _, raw := range []string{"", "12345", "091234567", "abc0912345678"} {
		_, ok := NormalizePhone(raw)
		assert.Falsef(t, ok, "expected %q to be rejected", raw)
	}
}
