package jobs

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
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
				jobCtx, jobCancel := context.WithTimeout(ctx, 30*time.Second)
				found, err := w.queue.ProcessNext(jobCtx, w.kind, w.process)
				jobCancel()
				if err != nil && ctx.Err() == nil {
					log.Printf("background job %s: %v", w.kind, err)
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

func (w *Worker) Stop() {
	w.stop.Do(func() {
		if w.cancel != nil {
			w.cancel()
		}
		w.wg.Wait()
	})
}
