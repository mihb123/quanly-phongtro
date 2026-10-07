package sepay

import (
	"context"
	"errors"
	"testing"
)

// fakeLister serves pre-canned transaction pages keyed by requested page number.
type fakeLister struct {
	pages map[int]*TransactionsResponse
	calls []ListTransactionsParams
	err   error
}

// ListTransactions records the request params and returns the configured page.
func (f *fakeLister) ListTransactions(_ context.Context, _ string, params ListTransactionsParams) (*TransactionsResponse, error) {
	f.calls = append(f.calls, params)
	if f.err != nil {
		return nil, f.err
	}
	if resp, ok := f.pages[params.Page]; ok {
		return resp, nil
	}
	return &TransactionsResponse{}, nil
}

type endlessLister struct {
	calls int
}

// ListTransactions returns has_more forever so the reconciliation cap can be verified.
func (f *endlessLister) ListTransactions(_ context.Context, _ string, params ListTransactionsParams) (*TransactionsResponse, error) {
	f.calls = params.Page
	return pageWith(true), nil
}

// pageWith builds a one-page response with the given transactions and has_more flag.
func pageWith(hasMore bool, txs ...Transaction) *TransactionsResponse {
	resp := &TransactionsResponse{Status: "success", Data: txs}
	resp.Meta.Pagination.HasMore = hasMore
	return resp
}

func testReconciler(lister TransactionLister) *Reconciler {
	reconciler := NewReconciler(lister)
	reconciler.Throttle = 0
	return reconciler
}

var reconcileCredentials = Credentials{APIToken: "tok", Environment: EnvironmentSandbox}

// TestReconcileProcessesIncomingWithCodeAcrossPages verifies paging, filtering, and mapping.
func TestReconcileProcessesIncomingWithCodeAcrossPages(t *testing.T) {
	lister := &fakeLister{pages: map[int]*TransactionsResponse{
		1: pageWith(true,
			Transaction{ID: "uuid-in-1", Code: "PTAAA", AmountIn: 100000, TransferType: "in", AccountNumber: "123"},
			Transaction{ID: "uuid-out", Code: "PTOUT", AmountOut: 50000, TransferType: "out"},
			Transaction{ID: "uuid-nocode", Code: "", AmountIn: 70000, TransferType: "in"},
		),
		2: pageWith(false,
			Transaction{ID: "uuid-in-2", Code: "PTBBB", AmountIn: 200000, TransferType: "in", AccountNumber: "456"},
		),
	}}
	var transfers []IncomingTransfer

	result, err := testReconciler(lister).Reconcile(context.Background(), reconcileCredentials, "2026-06-01 00:00:00", "2026-06-27 00:00:00", func(_ context.Context, transfer IncomingTransfer) error {
		transfers = append(transfers, transfer)
		return nil
	})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}

	if result.PagesFetched != 2 || result.Scanned != 4 || result.Processed != 2 {
		t.Errorf("result = %+v, want pages=2 scanned=4 processed=2", result)
	}
	if len(transfers) != 2 {
		t.Fatalf("handled transfers = %d, want 2", len(transfers))
	}
	first := transfers[0]
	if first.PaymentCode != "PTAAA" || first.Amount != 100000 || first.TransactionReference != "uuid-in-1" || first.AccountNumber != "123" {
		t.Errorf("first transfer = %+v", first)
	}
	if transfers[1].PaymentCode != "PTBBB" {
		t.Errorf("second transfer code = %q, want PTBBB", transfers[1].PaymentCode)
	}
	if lister.calls[0].PerPage != MaxPerPage {
		t.Errorf("per_page = %d, want %d", lister.calls[0].PerPage, MaxPerPage)
	}
	if lister.calls[0].Environment != EnvironmentSandbox || lister.calls[0].DateFrom != "2026-06-01 00:00:00" {
		t.Errorf("params = %+v", lister.calls[0])
	}
}

// TestReconcileRequiresAPIToken verifies a missing api_token is rejected before any request.
func TestReconcileRequiresAPIToken(t *testing.T) {
	lister := &fakeLister{}
	_, err := testReconciler(lister).Reconcile(context.Background(), Credentials{}, "", "", func(context.Context, IncomingTransfer) error { return nil })
	if !errors.Is(err, ErrMissingCredentials) {
		t.Errorf("error = %v, want ErrMissingCredentials", err)
	}
	if len(lister.calls) != 0 {
		t.Errorf("calls = %d, want 0", len(lister.calls))
	}
}

// TestReconcileReturnsListError verifies page fetch failures abort the run.
func TestReconcileReturnsListError(t *testing.T) {
	listErr := errors.New("sepay unavailable")
	result, err := testReconciler(&fakeLister{err: listErr}).Reconcile(context.Background(), reconcileCredentials, "", "", func(context.Context, IncomingTransfer) error { return nil })
	if !errors.Is(err, listErr) {
		t.Fatalf("error = %v, want list error", err)
	}
	if result.PagesFetched != 0 {
		t.Errorf("PagesFetched = %d, want 0", result.PagesFetched)
	}
}

// TestReconcileCountsTransactionFailure verifies one bad transaction is skipped.
func TestReconcileCountsTransactionFailure(t *testing.T) {
	lister := &fakeLister{pages: map[int]*TransactionsResponse{
		1: pageWith(false, Transaction{ID: "uuid-in", Code: "PTAAA", AmountIn: 100000, TransferType: "in"}),
	}}
	result, err := testReconciler(lister).Reconcile(context.Background(), reconcileCredentials, "", "", func(context.Context, IncomingTransfer) error {
		return errors.New("processor failed")
	})
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if result.Failed != 1 || result.Processed != 0 || result.Scanned != 1 {
		t.Errorf("result = %+v, want failed=1 processed=0 scanned=1", result)
	}
}

// TestReconcileReturnsContextErrorDuringThrottle verifies cancellation between pages aborts.
func TestReconcileReturnsContextErrorDuringThrottle(t *testing.T) {
	lister := &fakeLister{pages: map[int]*TransactionsResponse{1: pageWith(true)}}
	reconciler := NewReconciler(lister)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := reconciler.Reconcile(ctx, reconcileCredentials, "", "", func(context.Context, IncomingTransfer) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if result.PagesFetched != 1 {
		t.Errorf("PagesFetched = %d, want 1", result.PagesFetched)
	}
}

// TestReconcileTruncatesAtPageCap verifies a runaway has_more is capped.
func TestReconcileTruncatesAtPageCap(t *testing.T) {
	lister := &endlessLister{}
	result, err := testReconciler(lister).Reconcile(context.Background(), reconcileCredentials, "", "", func(context.Context, IncomingTransfer) error { return nil })
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !result.Truncated || result.PagesFetched != DefaultReconcileMaxPages {
		t.Errorf("result = %+v, want truncated at %d pages", result, DefaultReconcileMaxPages)
	}
	if lister.calls != DefaultReconcileMaxPages {
		t.Errorf("last page call = %d, want %d", lister.calls, DefaultReconcileMaxPages)
	}
}
