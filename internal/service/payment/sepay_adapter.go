package payment

import (
	"context"
	"errors"
	"fmt"
	"time"

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
func (p *SePayProvider) VerifyWebhook(_ context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error) {
	transfer, err := sepay.VerifyWebhook(input.Headers, input.Body, sepay.CredentialsFromMap(input.Credentials), time.Now())
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
	return &SePayReconciliationService{
		reconciler:        sepay.NewReconciler(client),
		credentialService: credentialService,
		processor:         processor,
	}
}

// ReconcileManager pages through a manager's transactions in the window and processes incoming
// transfers that carry a payment code. Already-paid invoices are skipped by the shared processor.
func (s *SePayReconciliationService) ReconcileManager(ctx context.Context, managerID, dateFrom, dateTo string) (sepay.ReconcileResult, error) {
	credentials, err := s.credentialService.GetCredentials(ctx, managerID, model.PaymentProviderSePay)
	if err != nil {
		return sepay.ReconcileResult{}, err
	}

	result, err := s.reconciler.Reconcile(ctx, sepay.CredentialsFromMap(credentials), dateFrom, dateTo, func(ctx context.Context, transfer sepay.IncomingTransfer) error {
		if err := s.processor.ProcessVerifiedTransaction(ctx, model.PaymentProviderSePay, managerID, sePayTransferEvent(transfer)); err != nil {
			return fmt.Errorf("process reconciled transaction %s for manager %s: %w", transfer.TransactionReference, managerID, err)
		}
		return nil
	})
	return result, mapSePayError(err)
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
		return ErrPaymentWebhookIgnored
	case errors.Is(err, sepay.ErrInvalidWebhook):
		return fmt.Errorf("%w: %w", ErrPaymentWebhookInvalid, err)
	case errors.Is(err, sepay.ErrMissingCredentials):
		return fmt.Errorf("%w: %w", ErrPaymentCredentialsNotFound, err)
	default:
		return err
	}
}
