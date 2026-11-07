package logging

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLevels(t *testing.T) {
	for _, env := range []string{"prod", "dev"} {
		l, err := New(env, "warn")
		if err != nil {
			t.Fatal(err)
		}
		if l.Core().Enabled(zap.InfoLevel) || !l.Core().Enabled(zap.WarnLevel) {
			t.Fatalf("%s: level warn not applied", env)
		}
	}
	l, err := New("prod", "bogus")
	if err != nil || !l.Core().Enabled(zap.InfoLevel) {
		t.Fatal("invalid level must fall back to default info")
	}
}

func TestTraceIDAndLogger(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	ctx := WithTraceID(context.Background(), zap.New(core), "abc")
	if TraceID(ctx) != "abc" {
		t.Fatal("trace id missing")
	}
	FromCtx(ctx).Info("hello")
	if logs.Len() != 1 || logs.All()[0].ContextMap()["trace_id"] != "abc" {
		t.Fatalf("logger must carry trace_id: %+v", logs.All())
	}
	if TraceID(nil) != "" || FromCtx(nil) == nil { //nolint:staticcheck // nil ctx is tolerated by design
		t.Fatal("nil ctx must be tolerated")
	}
}

func TestDetachSurvivesCancel(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	parent, cancel := context.WithTimeout(WithTraceID(context.Background(), zap.New(core), "t1"), time.Hour)
	bg := Detach(parent)
	cancel()
	if bg.Err() != nil {
		t.Fatal("detached ctx must not be cancelled")
	}
	FromCtx(bg).Info("bg")
	if TraceID(bg) != "t1" || logs.Len() != 1 {
		t.Fatal("detached ctx must keep trace id and logger")
	}
	if Detach(context.Background()).Value(traceKey{}) != nil {
		t.Fatal("no trace id -> nothing to carry")
	}
}

func TestGoLogsErrorAndPanic(t *testing.T) {
	core, logs := observer.New(zap.InfoLevel)
	ctx := WithTraceID(context.Background(), zap.New(core), "bg-1")
	done := make(chan struct{}, 2)
	Go(ctx, "fail", time.Second, func(context.Context) error {
		defer func() { done <- struct{}{} }()
		return context.Canceled
	})
	Go(ctx, "boom", time.Second, func(context.Context) error {
		defer func() { done <- struct{}{} }()
		panic("boom")
	})
	<-done
	<-done
	deadline := time.Now().Add(time.Second)
	for logs.Len() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if logs.Len() != 2 {
		t.Fatalf("want 2 logs, got %d", logs.Len())
	}
	for _, e := range logs.All() {
		if e.ContextMap()["trace_id"] != "bg-1" {
			t.Fatalf("missing trace id: %+v", e.ContextMap())
		}
	}
}
