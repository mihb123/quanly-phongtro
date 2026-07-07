package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

// fakeSePayLister serves pre-canned transaction pages keyed by requested page number.
type fakeSePayLister struct {
	pages map[int]*SePayTransactionsResponse
	calls []SePayListParams
	err   error
}

// ListTransactions records the request params and returns the configured page.
func (f *fakeSePayLister) ListTransactions(_ context.Context, _ string, params SePayListParams) (*SePayTransactionsResponse, error) {
	f.calls = append(f.calls, params)
	if f.err != nil {
		return nil, f.err
	}
	if resp, ok := f.pages[params.Page]; ok {
		return resp, nil
	}
	return &SePayTransactionsResponse{}, nil
}

// fakeReconcileProcessor records every verified event it is asked to process.
type fakeReconcileProcessor struct {
	events []*VerifiedPaymentEvent
	err    error
}

// ProcessVerifiedTransaction stores the event for assertions.
func (f *fakeReconcileProcessor) ProcessVerifiedTransaction(_ context.Context, _, _ string, event *VerifiedPaymentEvent) error {
	f.events = append(f.events, event)
	return f.err
}

type endlessSePayLister struct {
	calls int
}

// ListTransactions returns has_more forever so the reconciliation cap can be verified.
func (f *endlessSePayLister) ListTransactions(_ context.Context, _ string, params SePayListParams) (*SePayTransactionsResponse, error) {
	f.calls = params.Page
	return pageWith(true), nil
}

// pageWith builds a one-page response with the given transactions and has_more flag.
func pageWith(hasMore bool, txs ...SePayTransaction) *SePayTransactionsResponse {
	resp := &SePayTransactionsResponse{Status: "success", Data: txs}
	resp.Meta.Pagination.HasMore = hasMore
	return resp
}

// TestSePayReconcileProcessesIncomingWithCodeAcrossPages verifies paging, filtering, and mapping.
func TestSePayReconcileProcessesIncomingWithCodeAcrossPages(t *testing.T) {
	lister := &fakeSePayLister{pages: map[int]*SePayTransactionsResponse{
		1: pageWith(true,
			SePayTransaction{ID: "uuid-in-1", Code: "PTAAA", AmountIn: 100000, TransferType: "in", AccountNumber: "123"},
			SePayTransaction{ID: "uuid-out", Code: "PTOUT", AmountOut: 50000, TransferType: "out"},
			SePayTransaction{ID: "uuid-nocode", Code: "", AmountIn: 70000, TransferType: "in"},
		),
		2: pageWith(false,
			SePayTransaction{ID: "uuid-in-2", Code: "PTBBB", AmountIn: 200000, TransferType: "in", AccountNumber: "456"},
		),
	}}
	processor := &fakeReconcileProcessor{}
	creds := fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok"}}

	svc := NewSePayReconciliationService(lister, creds, processor)
	svc.throttle = 0 // keep the test fast

	result, err := svc.ReconcileManager(context.Background(), "manager-1", "2026-06-01 00:00:00", "2026-06-27 00:00:00")
	if err != nil {
		t.Fatalf("ReconcileManager() error = %v", err)
	}

	if result.PagesFetched != 2 {
		t.Errorf("PagesFetched = %d, want 2", result.PagesFetched)
	}
	if result.Scanned != 4 {
		t.Errorf("Scanned = %d, want 4", result.Scanned)
	}
	if result.Processed != 2 {
		t.Errorf("Processed = %d, want 2 (only incoming with code)", result.Processed)
	}
	if len(processor.events) != 2 {
		t.Fatalf("processed events = %d, want 2", len(processor.events))
	}

	first := processor.events[0]
	if first.ProviderOrderRef != "PTAAA" || first.Amount != 100000 || first.TransactionReference != "uuid-in-1" {
		t.Errorf("first event = {%q, %d, %q}, want {PTAAA, 100000, uuid-in-1}", first.ProviderOrderRef, first.Amount, first.TransactionReference)
	}
	if processor.events[1].ProviderOrderRef != "PTBBB" {
		t.Errorf("second event ref = %q, want PTBBB", processor.events[1].ProviderOrderRef)
	}
	// Page size must request the documented maximum.
	if lister.calls[0].PerPage != sePayListMaxPerPage {
		t.Errorf("per_page = %d, want %d", lister.calls[0].PerPage, sePayListMaxPerPage)
	}
}

// TestSePayReconcileRequiresAPIToken verifies a missing api_token is rejected.
func TestSePayReconcileRequiresAPIToken(t *testing.T) {
	svc := NewSePayReconciliationService(
		&fakeSePayLister{},
		fakePaymentCredentialService{credentials: map[string]string{}},
		&fakeReconcileProcessor{},
	)
	svc.throttle = 0

	if _, err := svc.ReconcileManager(context.Background(), "manager-1", "", ""); !errors.Is(err, ErrPaymentCredentialsNotFound) {
		t.Errorf("error = %v, want ErrPaymentCredentialsNotFound", err)
	}
}

// TestSePayReconcileReturnsListError verifies page fetch failures abort the run.
func TestSePayReconcileReturnsListError(t *testing.T) {
	listErr := errors.New("sepay unavailable")
	svc := NewSePayReconciliationService(
		&fakeSePayLister{err: listErr},
		fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok"}},
		&fakeReconcileProcessor{},
	)
	svc.throttle = 0

	result, err := svc.ReconcileManager(context.Background(), "manager-1", "", "")
	if !errors.Is(err, listErr) {
		t.Fatalf("error = %v, want list error", err)
	}
	if result.PagesFetched != 0 {
		t.Errorf("PagesFetched = %d, want 0", result.PagesFetched)
	}
}

// TestSePayReconcileCountsTransactionFailure verifies one bad transaction is skipped.
func TestSePayReconcileCountsTransactionFailure(t *testing.T) {
	lister := &fakeSePayLister{pages: map[int]*SePayTransactionsResponse{
		1: pageWith(false, SePayTransaction{ID: "uuid-in", Code: "PTAAA", AmountIn: 100000, TransferType: "in"}),
	}}
	processor := &fakeReconcileProcessor{err: errors.New("processor failed")}
	svc := NewSePayReconciliationService(
		lister,
		fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok"}},
		processor,
	)
	svc.throttle = 0

	result, err := svc.ReconcileManager(context.Background(), "manager-1", "", "")
	if err != nil {
		t.Fatalf("ReconcileManager() error = %v", err)
	}
	if result.Failed != 1 || result.Processed != 0 || result.Scanned != 1 {
		t.Errorf("result = %+v, want failed=1 processed=0 scanned=1", result)
	}
}

// TestSePayReconcileReturnsContextErrorDuringThrottle verifies cancellation between pages aborts.
func TestSePayReconcileReturnsContextErrorDuringThrottle(t *testing.T) {
	lister := &fakeSePayLister{pages: map[int]*SePayTransactionsResponse{
		1: pageWith(true),
	}}
	svc := NewSePayReconciliationService(
		lister,
		fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok"}},
		&fakeReconcileProcessor{},
	)
	svc.throttle = sePayReconcileThrottle
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := svc.ReconcileManager(ctx, "manager-1", "", "")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
	if result.PagesFetched != 1 {
		t.Errorf("PagesFetched = %d, want 1", result.PagesFetched)
	}
}

// TestSePayReconcileTruncatesAtPageCap verifies a runaway has_more is capped.
func TestSePayReconcileTruncatesAtPageCap(t *testing.T) {
	lister := &endlessSePayLister{}
	svc := NewSePayReconciliationService(
		lister,
		fakePaymentCredentialService{credentials: map[string]string{"api_token": "tok"}},
		&fakeReconcileProcessor{},
	)
	svc.throttle = 0

	result, err := svc.ReconcileManager(context.Background(), "manager-1", "", "")
	if err != nil {
		t.Fatalf("ReconcileManager() error = %v", err)
	}
	if !result.Truncated || result.PagesFetched != sePayMaxReconcilePages {
		t.Errorf("result = %+v, want truncated at %d pages", result, sePayMaxReconcilePages)
	}
	if lister.calls != sePayMaxReconcilePages {
		t.Errorf("last page call = %d, want %d", lister.calls, sePayMaxReconcilePages)
	}
}

// TestProcessVerifiedTransactionSkipsPaidLink verifies the cross-source guard: an already-paid
// link (settled by a webhook) is recorded for audit but not re-settled by reconciliation.
func TestProcessVerifiedTransactionSkipsPaidLink(t *testing.T) {
	svc, paymentRepository, invoiceRepository := newSePayWebhookService(
		&model.InvoicePaymentLink{ID: "link-1", InvoiceID: "invoice-1", ProviderOrderRef: "PTABC123", Amount: 500000, Status: model.PaymentLinkStatusPaid},
	)

	event := &VerifiedPaymentEvent{
		ProviderOrderRef:     "PTABC123",
		Amount:               500000,
		TransactionReference: "uuid-from-api-v2",
		MatchingMethod:       paymentMatchOrderRef,
		SignatureResult:      paymentSignatureValid,
	}
	if err := svc.ProcessVerifiedTransaction(context.Background(), model.PaymentProviderSePay, "manager-1", event); err != nil {
		t.Fatalf("ProcessVerifiedTransaction() error = %v", err)
	}

	if invoiceRepository.updatedStatus == model.InvoiceStatusPaid {
		t.Error("already-paid invoice was settled again by reconciliation")
	}
	if paymentRepository.event == nil {
		t.Error("expected the reconciled transaction to be recorded for audit")
	}
}
