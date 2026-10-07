package revenue

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/jobs"
)

type workerQueue struct {
	run func(context.Context, func(context.Context, json.RawMessage) error) (bool, error)
}

func (q workerQueue) ProcessNext(ctx context.Context, _ string, process func(context.Context, json.RawMessage) error) (bool, error) {
	return q.run(ctx, process)
}

func TestRevenueWorker_StartStopAndRetry(t *testing.T) {
	var attempts atomic.Int32
	done := make(chan struct{}, 1)
	queue := workerQueue{run: func(ctx context.Context, process func(context.Context, json.RawMessage) error) (bool, error) {
		if attempts.Add(1) == 1 {
			return true, errors.New("temporary database failure")
		}
		if err := process(ctx, []byte(`{"house_id":"h","period":"2026-10"}`)); err != nil {
			return true, err
		}
		select {
		case done <- struct{}{}:
		default:
		}
		return false, nil
	}}
	worker := jobs.NewWorker(queue, "revenue", func(_ context.Context, payload json.RawMessage) error {
		if string(payload) != `{"house_id":"h","period":"2026-10"}` {
			return errors.New("wrong payload")
		}
		return nil
	})
	worker.Start()
	defer worker.Stop()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("job was not retried")
	}
	if attempts.Load() < 2 {
		t.Fatal("expected retry")
	}
}

func TestRevenueWorker_StopCancelsActiveJob(t *testing.T) {
	entered := make(chan struct{})
	queue := workerQueue{run: func(ctx context.Context, _ func(context.Context, json.RawMessage) error) (bool, error) {
		close(entered)
		<-ctx.Done()
		return true, ctx.Err()
	}}
	worker := jobs.NewWorker(queue, "revenue", nil)
	worker.Start()
	<-entered
	stopped := make(chan struct{})
	go func() { worker.Stop(); worker.Stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel job")
	}
}
