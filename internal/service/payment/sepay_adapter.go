package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/mihb123/quanly-phongtro/pkg/apperr"

	"github.com/mihb123/quanly-phongtro/internal/logging"
	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/pkg/sepay"
)

type SePayProvider struct{}

// NewSePayProvider initializes a new SePay provider adapter.
func NewSePayProvider() *SePayProvider {
	return &SePayProvider{}
}

// Name returns the standard string identifier for this provider.
func (p *SePayProvider) Name() string {
	return model.PaymentProviderSePay
}

// CreatePaymentLink generates a SePay QR transfer link for the invoice.
func (p *SePayProvider) CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error) {
	payment, err := sepay.CreatePayment(ctx, sepay.CredentialsFromMap(input.Credentials), input.Invoice.ID, int(input.Invoice.TotalAmount), input.ProviderOrderRefExists)
	if err != nil {
		return nil, mapSePayError(err)
	}
	return &PaymentProviderLink{
		ProviderOrderRef: payment.Code,
		PaymentLinkID:    payment.Code,
		CheckoutURL:      payment.QRURL,
		QRCode:           payment.QRURL,
		Amount:           payment.Amount,
	}, nil
}

// CancelPaymentLink is a no-op: a SePay QR transfer has no remote link to cancel.
func (p *SePayProvider) CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error {
	return nil
}

// VerifyWebhook authenticates the SePay webhook then maps an incoming transfer into a neutral event.
func (p *SePayProvider) VerifyWebhook(ctx context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	credentials := sepay.CredentialsFromMap(input.Credentials)
	if sepay.IsUnauthenticated(credentials) {
		logging.RecordWarning(ctx, "sepay webhook accepted without authentication (webhook_auth_method not set)", nil)
	}
	transfer, err := sepay.VerifyWebhook(input.Headers, input.Body, credentials, time.Now())
	if err != nil {
		return nil, mapSePayError(err)
	}
	return sePayTransferEvent(*transfer), nil
}

// reconciliationCredentialReader returns decrypted manager credentials; satisfied by PaymentCredentialService.
type reconciliationCredentialReader interface {
	GetCredentials(ctx context.Context, managerID, provider string) (map[string]string, error)
}

// verifiedTransactionProcessor records and settles a verified transaction; satisfied by PaymentService.
type verifiedTransactionProcessor interface {
	ProcessVerifiedTransaction(ctx context.Context, provider, managerID string, event *VerifiedPaymentEvent) error
}

// SePayReconciliationService pulls SePay transactions via API v2 to settle invoices missed by webhooks.
type SePayReconciliationService struct {
	reconciler        *sepay.Reconciler
	credentialService reconciliationCredentialReader
	processor         verifiedTransactionProcessor
}

// NewSePayReconciliationService wires API-v2 reconciliation for a manager's SePay account.
func NewSePayReconciliationService(client sepay.TransactionLister, credentialService reconciliationCredentialReader, processor verifiedTransactionProcessor) *SePayReconciliationService {
	reconciler := sepay.NewReconciler(client)
	reconciler.OnSkip = func(ctx context.Context, transactionID string, err error) {
		logging.RecordWarning(ctx, "sepay reconciliation skipped transaction "+transactionID, err)
	}
	return &SePayReconciliationService{
		reconciler:        reconciler,
		credentialService: credentialService,
		processor:         processor,
	}
}

// ReconcileManager pages through a manager's transactions in the window and processes incoming
// transfers that carry a payment code. Already-paid invoices are skipped by the shared processor.
func (s *SePayReconciliationService) ReconcileManager(ctx context.Context, managerID, dateFrom, dateTo string) (sepay.ReconcileResult, error) {
	credentials, err := s.credentialService.GetCredentials(ctx, managerID, model.PaymentProviderSePay)
	if err != nil {
		return sepay.ReconcileResult{}, apperr.Service("ReconcileManager", err)
	}

	result, err := s.reconciler.Reconcile(ctx, sepay.CredentialsFromMap(credentials), dateFrom, dateTo, func(ctx context.Context, transfer sepay.IncomingTransfer) error {
		if err := s.processor.ProcessVerifiedTransaction(ctx, model.PaymentProviderSePay, managerID, sePayTransferEvent(transfer)); err != nil {
			return apperr.Service("ReconcileManager", fmt.Errorf("process reconciled transaction %s for manager %s: %w", transfer.TransactionReference, managerID, err))
		}
		return nil
	})
	if result.Truncated {
		logging.RecordWarning(ctx, "sepay reconciliation hit the page cap; remaining transactions not pulled", nil)
	}
	return result, apperr.Service("ReconcileManager", mapSePayError(err))
}

func sePayTransferEvent(transfer sepay.IncomingTransfer) *VerifiedPaymentEvent {
	counterAccount := transfer.AccountNumber
	return &VerifiedPaymentEvent{
		ProviderOrderRef:     transfer.PaymentCode,
		Amount:               transfer.Amount,
		TransactionReference: transfer.TransactionReference,
		CounterAccount:       &counterAccount,
		RawPayload:           transfer.RawPayload,
		SignatureResult:      paymentSignatureValid,
		MatchingMethod:       paymentMatchOrderRef,
	}
}

func mapSePayError(err error) error {
	switch {
	case errors.Is(err, sepay.ErrIgnoredWebhook):
		return apperr.Service("mapSePayError", ErrPaymentWebhookIgnored)
	case errors.Is(err, sepay.ErrInvalidWebhook):
		return apperr.Service("mapSePayError", fmt.Errorf("%w: %w", ErrPaymentWebhookInvalid, err))
	case errors.Is(err, sepay.ErrMissingCredentials):
		return apperr.Service("mapSePayError", fmt.Errorf("%w: %w", ErrPaymentCredentialsNotFound, err))
	default:
		return apperr.Service("mapSePayError", err)
	}
}
