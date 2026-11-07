package zalo

import (
	"context"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const zaloCronTask = "zalo_token_check"

// ZaloCronService runs periodic background jobs for Zalo integration.
// Currently it checks whether each manager's Zalo bot token is still valid every 4 hours.
type ZaloCronService struct {
	cron          *cron.Cron
	client        ZaloClient
	userRepo      model.UserRepository
	encryptionKey []byte
}

var TokenCheckCronSchedule = "0 */4 * * *"

// NewZaloCronService constructs the cron service and schedules the token health check.
func NewZaloCronService(client ZaloClient, userRepo model.UserRepository, encryptionKey []byte) *ZaloCronService {
	c := cron.New(cron.WithLocation(time.UTC))
	svc := &ZaloCronService{
		cron:          c,
		client:        client,
		userRepo:      userRepo,
		encryptionKey: encryptionKey,
	}

	_, err := c.AddFunc(TokenCheckCronSchedule, func() {
		svc.checkAllTokens()
	})
	if err != nil {
		zap.L().Error("schedule token health check failed", zap.String("task", zaloCronTask), zap.Error(err))
	}

	return svc
}

// Start begins the cron scheduler in a background goroutine.
func (s *ZaloCronService) Start() {
	s.cron.Start()
	zap.L().Info("cron started", zap.String("task", zaloCronTask), zap.String("schedule", TokenCheckCronSchedule))
}

// Stop gracefully shuts down the cron scheduler.
func (s *ZaloCronService) Stop() {
	s.cron.Stop()
	zap.L().Info("cron stopped", zap.String("task", zaloCronTask))
}

// checkAllTokens fetches all users with a Zalo token and verifies each one is still valid.
func (s *ZaloCronService) checkAllTokens() {
	ctx := logging.TaskContext(context.Background(), zaloCronTask)
	logging.FromCtx(ctx).Info("token health check started")

	users, err := s.userRepo.GetAllUsersWithZaloToken(ctx)
	if err != nil {
		logging.LogTaskError(ctx, "fetch users with zalo token failed", err)
		return
	}

	for _, u := range users {
		s.checkUserToken(ctx, u)
	}

	logging.FromCtx(ctx).Info("token health check completed", zap.Int("users", len(users)))
}

// checkUserToken decrypts and validates a single user's Zalo bot token.
// Updates is_zalo_bot_active in the DB based on the result.
func (s *ZaloCronService) checkUserToken(ctx context.Context, u model.User) {
	if u.ZaloBotToken == nil || *u.ZaloBotToken == "" {
		return
	}

	rawToken, err := security.Decrypt(*u.ZaloBotToken, s.encryptionKey)
	if err != nil {
		logging.FromCtx(ctx).Warn("decrypt zalo token failed", zap.String("user_id", u.ID), zap.Error(err))
		s.markInactive(ctx, u.ID)
		return
	}

	_, err = s.client.GetMe(ctx, rawToken)
	if err != nil {
		logging.FromCtx(ctx).Warn("zalo token invalid", zap.String("user_id", u.ID), zap.Error(err))
		s.markInactive(ctx, u.ID)
		return
	}

	// Token is alive — ensure it is marked active if it was previously inactive
	if !u.IsZaloBotActive {
		s.markActive(ctx, u.ID)
		logging.FromCtx(ctx).Info("zalo token restored to active", zap.String("user_id", u.ID))
	}
}

// markInactive sets is_zalo_bot_active = false for a given user.
func (s *ZaloCronService) markInactive(ctx context.Context, userID string) {
	inactive := false
	if _, err := s.userRepo.UpdateUser(ctx, userID, model.UpdateUserInput{IsZaloBotActive: &inactive}); err != nil {
		logging.LogTaskError(ctx, "mark zalo token inactive failed", err, zap.String("user_id", userID))
	}
}

// markActive sets is_zalo_bot_active = true for a given user.
func (s *ZaloCronService) markActive(ctx context.Context, userID string) {
	active := true
	if _, err := s.userRepo.UpdateUser(ctx, userID, model.UpdateUserInput{IsZaloBotActive: &active}); err != nil {
		logging.LogTaskError(ctx, "mark zalo token active failed", err, zap.String("user_id", userID))
	}
}

// RunNow immediately executes the token health check, bypassing the schedule.
// Useful for testing or triggering a manual check.
func (s *ZaloCronService) RunNow() {
	s.checkAllTokens()
}
