package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/pkg/apperr"
)

type Queue interface {
	ProcessNext(context.Context, string, func(context.Context, json.RawMessage) error) (bool, error)
}

type Worker struct {
	queue   Queue
	kind    string
	process func(context.Context, json.RawMessage) error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	start   sync.Once
	stop    sync.Once
}

func NewWorker(queue Queue, kind string, process func(context.Context, json.RawMessage) error) *Worker {
	return &Worker{queue: queue, kind: kind, process: process}
}

func (w *Worker) Start() {
	w.start.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		w.cancel = cancel
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for ctx.Err() == nil {
				jobCtx := logging.TaskContext(ctx, "job:"+w.kind)
				found, err := w.runOnce(jobCtx)
				if err != nil && ctx.Err() == nil {
					logging.LogTaskError(jobCtx, "background job failed", err)
				}
				if found && err == nil {
					continue
				}
				timer := time.NewTimer(time.Second)
				select {
				case <-ctx.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	})
}

// runOnce xử lý tối đa một job; panic trong process được chuyển thành error để goroutine
// worker không chết. apperr chụp stack ngay trong defer nên stack vẫn trỏ tới điểm panic.
func (w *Worker) runOnce(ctx context.Context) (found bool, err error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	defer func() {
		if r := recover(); r != nil {
			found, err = false, apperr.Service("runOnce", fmt.Errorf("panic: %v", r))
		}
	}()
	return w.queue.ProcessNext(ctx, w.kind, w.process)
}

func (w *Worker) Stop() {
	w.stop.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		w.wg.Wait()
	})
}
