package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	database "github.com/mihb123/quanly-phongtro/internal/db"
	"github.com/uptrace/bun"
)

type Repository struct{ db *bun.DB }

func NewRepository(db *bun.DB) *Repository { return &Repository{db: db} }

func (r *Repository) ProcessNext(ctx context.Context, kind string, process func(context.Context, json.RawMessage) error) (bool, error) {
	var job struct {
		ID      int64
		Payload json.RawMessage
	}
	err := database.WithinTransaction(ctx, r.db, func(ctx context.Context) error {
		executor := database.Executor(ctx, r.db)
		err := executor.NewRaw("SELECT id, payload FROM background_jobs WHERE kind = ? AND available_at <= NOW() ORDER BY available_at, id LIMIT 1 FOR UPDATE SKIP LOCKED", kind).Scan(ctx, &job)
		if err != nil {
			return err
		}
		if err := process(ctx, job.Payload); err != nil {
			return err
		}
		_, err = executor.ExecContext(ctx, "DELETE FROM background_jobs WHERE id = ?", job.ID)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) && job.ID == 0 {
		return false, nil
	}
	if err != nil && job.ID != 0 {
		retryCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, retryErr := r.db.ExecContext(retryCtx, "UPDATE background_jobs SET attempts = attempts + 1, last_error = ?, available_at = NOW() + LEAST(attempts + 1, 60) * INTERVAL '5 seconds' WHERE id = ?", err.Error(), job.ID)
		if retryErr != nil {
			return false, errors.Join(err, fmt.Errorf("schedule job retry: %w", retryErr))
		}
	}
	return job.ID != 0, err
}
