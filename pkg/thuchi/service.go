package thuchi

import (
	"context"
	"log"
	"sync"
	"time"
)

type Crypto interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
}

type Service struct {
	Store  Store
	Crypto Crypto
	Now    func() time.Time

	Telegram *TelegramBot

	kickOnce sync.Once
	kick     chan struct{}
}

func NewService(store Store, crypto Crypto) *Service {
	return &Service{Store: store, Crypto: crypto}
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) kickChan() chan struct{} {
	s.kickOnce.Do(func() { s.kick = make(chan struct{}, 1) })
	return s.kick
}

func (s *Service) Kick() {
	select {
	case s.kickChan() <- struct{}{}:
	default:
	}
}

func (s *Service) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		s.ProcessPending(ctx)
		s.ProcessRecurrences(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.kickChan():
		}
	}
}

func (s *Service) ProcessPending(ctx context.Context) {
	for ctx.Err() == nil {
		events, err := s.Store.PendingRawEvents(ctx, 50)
		if err != nil {
			log.Printf("thuchi: list pending raw events: %v", err)
			return
		}
		if len(events) == 0 {
			return
		}
		for _, ev := range events {
			if err := s.ProcessRawEvent(ctx, ev.OwnerID, ev.ID); err != nil {
				log.Printf("thuchi: process raw event %s: %v", ev.ID, err)
				s.markFailed(ctx, ev.OwnerID, ev.ID, err)
			}
		}
	}
}

func (s *Service) markFailed(ctx context.Context, ownerID, id string, cause error) {
	err := s.Store.RunOwner(ctx, ownerID, func(ctx context.Context, tx Tx) error {
		ev, err := tx.GetRawEvent(ctx, id)
		if err != nil {
			return err
		}
		ev.Attempts++
		ev.Error = cause.Error()
		if ev.Attempts >= 5 {
			ev.Status = RawError
		}
		return tx.SaveRawEvent(ctx, ev)
	})
	if err != nil {
		log.Printf("thuchi: mark raw event %s failed: %v", id, err)
	}
}
