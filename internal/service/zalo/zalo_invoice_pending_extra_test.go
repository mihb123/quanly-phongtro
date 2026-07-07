package zalo

import (
	"context"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/mock/mock_model"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newPendingCommandTestService wires the minimal dependencies needed to send pending prompts.
func newPendingCommandTestService(t *testing.T, ctx context.Context) (*zaloInvoiceCommandServiceImpl, *recordingZaloClient) {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	encryptionKey := []byte("z123456789abcdef0123456789abcdef")
	token := encryptedToken(t, encryptionKey)
	userRepository := mock_model.NewMockUserRepository(ctrl)
	userRepository.EXPECT().GetByUserID(ctx, "manager-1").Return(&model.User{ZaloBotToken: &token}, nil).AnyTimes()

	zaloClient := &recordingZaloClient{}
	return &zaloInvoiceCommandServiceImpl{
		userRepo:      userRepository,
		pendingRepo:   &memoryPendingInvoiceUpdateRepository{},
		zaloClient:    zaloClient,
		encryptionKey: encryptionKey,
	}, zaloClient
}

// TestHandlePendingCommand_PromptBranches covers pending states that only reply with guidance.
func TestHandlePendingCommand_PromptBranches(t *testing.T) {
	ctx := context.Background()
	webhookCtx := webhookMessageContext{chatID: "chat-1", senderID: "sender-1"}

	t.Run("overwrite requires confirm", func(t *testing.T) {
		service, zaloClient := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandUtilitySingle}, &model.PendingInvoiceUpdate{
			ID:         "pending-1",
			ActionType: model.PendingActionConfirmOverwrite,
		})

		require.NoError(t, err)
		assert.True(t, handled)
		assert.Contains(t, zaloClient.messages[0], "#ok")
	})

	t.Run("await period requires month command", func(t *testing.T) {
		service, zaloClient := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandUnknown}, &model.PendingInvoiceUpdate{
			ID:         "pending-2",
			ActionType: model.PendingActionAwaitPeriod,
			PendingData: map[string]any{
				"period_options": map[string]any{"5": "2026-05"},
			},
		})

		require.NoError(t, err)
		assert.True(t, handled)
		assert.Contains(t, zaloClient.messages[0], "#<số tháng>")
	})

	t.Run("await period rejects unavailable month", func(t *testing.T) {
		service, zaloClient := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandPeriodSelect, PeriodMonth: 7}, &model.PendingInvoiceUpdate{
			ID:         "pending-3",
			ActionType: model.PendingActionAwaitPeriod,
			PendingData: map[string]any{
				"period_options": map[string]any{"5": "2026-05"},
			},
		})

		require.NoError(t, err)
		assert.True(t, handled)
		assert.Contains(t, zaloClient.messages[0], "không nằm")
	})

	t.Run("await utility missing expected utility is not handled", func(t *testing.T) {
		service, _ := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandUtilitySingle, UtilityType: "dien"}, &model.PendingInvoiceUpdate{
			ID:          "pending-4",
			ActionType:  model.PendingActionAwaitUtility,
			PendingData: map[string]any{},
		})

		require.NoError(t, err)
		assert.False(t, handled)
	})

	t.Run("await utility unknown text is not handled", func(t *testing.T) {
		service, _ := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandUnknown}, &model.PendingInvoiceUpdate{
			ID:         "pending-5",
			ActionType: model.PendingActionAwaitUtility,
			PendingData: map[string]any{
				"utility_type": "nuoc",
				"period":       "2026-05",
			},
		})

		require.NoError(t, err)
		assert.False(t, handled)
	})

	t.Run("await utility prompts for expected utility", func(t *testing.T) {
		service, zaloClient := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandUtilitySingle, UtilityType: "dien"}, &model.PendingInvoiceUpdate{
			ID:         "pending-6",
			ActionType: model.PendingActionAwaitUtility,
			PendingData: map[string]any{
				"utility_type": "nuoc",
				"period":       "2026-05",
			},
		})

		require.NoError(t, err)
		assert.True(t, handled)
		assert.Contains(t, zaloClient.messages[0], "#nuoc")
	})

	t.Run("unknown pending action falls through", func(t *testing.T) {
		service, _ := newPendingCommandTestService(t, ctx)
		handled, err := service.handlePendingCommand(ctx, "manager-1", "chat-1", webhookCtx, &ParsedCommand{Type: CommandConfirm}, &model.PendingInvoiceUpdate{
			ID:         "pending-7",
			ActionType: "other",
		})

		require.NoError(t, err)
		assert.False(t, handled)
	})
}

// TestCreateOverwritePending verifies overwrite confirmation data is persisted for later replay.
func TestCreateOverwritePending(t *testing.T) {
	pendingRepository := &memoryPendingInvoiceUpdateRepository{}
	service := &zaloInvoiceCommandServiceImpl{pendingRepo: pendingRepository}
	req := utilityUpdateRequest{
		UtilityType: "dien",
		NewIndex:    200,
		HasNewIndex: true,
		Room:        model.Room{ID: "room-1"},
	}

	err := service.createOverwritePending(context.Background(), "manager-1", webhookMessageContext{chatID: "chat-1", isGroupChat: true}, req, "2026-05", &model.Invoice{NewElectricityIndex: 150})

	require.NoError(t, err)
	require.Len(t, pendingRepository.created, 1)
	pending := pendingRepository.created[0]
	assert.Equal(t, model.PendingActionConfirmOverwrite, pending.ActionType)
	assert.Equal(t, "chat-1", pending.ChatID)
	assert.Equal(t, 150, pending.PendingData["old_value"])
}

// TestProcessPendingDataUnknownScope verifies malformed pending data is ignored.
func TestProcessPendingDataUnknownScope(t *testing.T) {
	service := &zaloInvoiceCommandServiceImpl{}
	err := service.processPendingData(context.Background(), "manager-1", webhookMessageContext{}, map[string]any{"command_scope": "bad"}, "", false)
	require.NoError(t, err)
}
