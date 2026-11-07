package apperr

import (
	"errors"
	"strings"
	"testing"
)

var errSentinel = errors.New("sentinel")

//go:noinline
func rootFailure() error { return Repo("rootFailure", errors.New("connection refused")) }

func TestWrapNil(t *testing.T) {
	if Wrap(LayerRepo, "X", nil) != nil || Repo("X", nil) != nil || Service("X", nil) != nil {
		t.Fatal("wrapping nil must return nil")
	}
	if Wrapf(LayerRepo, "X", nil, "ctx") != nil {
		t.Fatal("Wrapf(nil) must return nil")
	}
}

func TestChainMessageAndIs(t *testing.T) {
	err := Service("Get", Repo("GetByID", errSentinel))
	if got, want := err.Error(), "[service:Get] [repo:GetByID] sentinel"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if !errors.Is(err, errSentinel) {
		t.Fatal("errors.Is must reach the sentinel")
	}
}

func TestWrapfKeepsCause(t *testing.T) {
	err := Wrapf(LayerRepo, "Q", errSentinel, "id=%d", 7)
	if !strings.Contains(err.Error(), "id=7: sentinel") || !errors.Is(err, errSentinel) {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestRootMetadata(t *testing.T) {
	err := Service("Get", rootFailure())
	if FailedLayer(err) != LayerRepo {
		t.Fatalf("FailedLayer = %q", FailedLayer(err))
	}
	if !strings.HasPrefix(Origin(err), "apperr/apperr_test.go:") {
		t.Fatalf("Origin = %q", Origin(err))
	}
	if !strings.Contains(string(Stack(err)), "rootFailure") {
		t.Fatalf("stack must contain root function, got:\n%s", Stack(err))
	}
}

func TestStackCapturedOnce(t *testing.T) {
	outer := Service("Get", rootFailure())
	if outer.(*Error).stack != nil {
		t.Fatal("outer link must not capture a stack")
	}
}

func TestNoApperr(t *testing.T) {
	err := errors.New("plain")
	if FailedLayer(err) != "" || Origin(err) != "" || Stack(err) != nil {
		t.Fatal("plain error must yield empty metadata")
	}
}

func TestNewIsRoot(t *testing.T) {
	err := New(LayerHandler, "Parse", "bad id")
	if FailedLayer(err) != LayerHandler || len(Stack(err)) == 0 {
		t.Fatal("New must create a root error with stack")
	}
}

func TestMessageStripsPrefixes(t *testing.T) {
	if got := Message(Service("A", Repo("B", errSentinel))); got != "sentinel" {
		t.Fatalf("got %q", got)
	}
	if got := Message(errSentinel); got != "sentinel" {
		t.Fatalf("got %q", got)
	}
	if Message(nil) != "" {
		t.Fatal("nil -> empty")
	}
}
