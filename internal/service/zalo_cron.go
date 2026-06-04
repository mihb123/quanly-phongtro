package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
	"github.com/robfig/cron/v3"
)

// ZaloCronService runs periodic background jobs for Zalo integration.
// Currently it checks whether each manager's Zalo bot token is still valid every 4 hours.
type ZaloCronService struct {
	cron          *cron.Cron
	client        ZaloClient
	userRepo      model.UserRepository
	encryptionKey []byte
}

// NewZaloCronService constructs the cron service and schedules the token health check.
func NewZaloCronService(client ZaloClient, userRepo model.UserRepository, encryptionKey []byte) *ZaloCronService {
	c := cron.New(cron.WithLocation(time.UTC))
	svc := &ZaloCronService{
		cron:          c,
		client:        client,
		userRepo:      userRepo,
		encryptionKey: encryptionKey,
	}

	// Schedule every 4 hours: "0 */4 * * *"
	_, err := c.AddFunc("0 */4 * * *", func() {
		svc.checkAllTokens()
	})
	if err != nil {
		log.Printf("[ZaloCron] Failed to schedule token health check: %v", err)
	}

	return svc
}

// Start begins the cron scheduler in a background goroutine.
func (s *ZaloCronService) Start() {
	s.cron.Start()
	log.Println("[ZaloCron] Started Zalo token health check scheduler (every 4 hours).")
}

// Stop gracefully shuts down the cron scheduler.
func (s *ZaloCronService) Stop() {
	s.cron.Stop()
	log.Println("[ZaloCron] Stopped Zalo token health check scheduler.")
}

// checkAllTokens fetches all users with a Zalo token and verifies each one is still valid.
func (s *ZaloCronService) checkAllTokens() {
	ctx := context.Background()
	log.Println("[ZaloCron] Running Zalo token health check...")

	users, err := s.userRepo.GetAllUsersWithZaloToken(ctx)
	if err != nil {
		log.Printf("[ZaloCron] Failed to fetch users with Zalo token: %v", err)
		return
	}

	for _, u := range users {
		s.checkUserToken(ctx, u)
	}

	log.Printf("[ZaloCron] Token health check completed for %d users.", len(users))
}

// checkUserToken decrypts and validates a single user's Zalo bot token.
// Updates is_zalo_bot_active in the DB based on the result.
func (s *ZaloCronService) checkUserToken(ctx context.Context, u model.User) {
	if u.ZaloBotToken == nil || *u.ZaloBotToken == "" {
		return
	}

	rawToken, err := security.Decrypt(*u.ZaloBotToken, s.encryptionKey)
	if err != nil {
		log.Printf("[ZaloCron] Failed to decrypt token for user %s: %v", u.ID, err)
		s.markInactive(ctx, u.ID)
		return
	}

	_, err = s.client.GetMe(ctx, rawToken)
	if err != nil {
		log.Printf("[ZaloCron] Token invalid for user %s (%s): %v", u.ID, u.Email, err)
		s.markInactive(ctx, u.ID)
		return
	}

	// Token is alive — ensure it is marked active if it was previously inactive
	if !u.IsZaloBotActive {
		s.markActive(ctx, u.ID)
		log.Printf("[ZaloCron] Token restored to active for user %s (%s)", u.ID, u.Email)
	}
}

// markInactive sets is_zalo_bot_active = false for a given user.
func (s *ZaloCronService) markInactive(ctx context.Context, userID string) {
	inactive := false
	if _, err := s.userRepo.UpdateUser(ctx, userID, model.UpdateUserInput{IsZaloBotActive: &inactive}); err != nil {
		log.Printf("[ZaloCron] Failed to mark token inactive for user %s: %v", userID, err)
	}
}

// markActive sets is_zalo_bot_active = true for a given user.
func (s *ZaloCronService) markActive(ctx context.Context, userID string) {
	active := true
	if _, err := s.userRepo.UpdateUser(ctx, userID, model.UpdateUserInput{IsZaloBotActive: &active}); err != nil {
		log.Printf("[ZaloCron] Failed to mark token active for user %s: %v", userID, err)
	}
}

// RunNow immediately executes the token health check, bypassing the schedule.
// Useful for testing or triggering a manual check.
func (s *ZaloCronService) RunNow() {
	fmt.Println("[ZaloCron] Manual token check triggered.")
	s.checkAllTokens()
}
