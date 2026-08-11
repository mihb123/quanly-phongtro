package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

// sePayReconcileThrottle spaces page requests to stay under SePay's 3 req/s API v2 limit.
const sePayReconcileThrottle = 350 * time.Millisecond

// sePayMaxReconcilePages caps a single reconciliation run so a runaway has_more never loops forever.
const sePayMaxReconcilePages = 100

// sePayTransactionLister fetches one page of API v2 transactions; satisfied by *SePayClient.
type sePayTransactionLister interface {
	ListTransactions(ctx context.Context, apiToken string, params SePayListParams) (*SePayTransactionsResponse, error)
}

// reconciliationCredentialReader returns decrypted manager credentials; satisfied by PaymentCredentialService.
type reconciliationCredentialReader interface {
	GetCredentials(ctx context.Context, managerID, provider string) (map[string]string, error)
}

// verifiedTransactionProcessor records and settles a verified transaction; satisfied by PaymentService.
type verifiedTransactionProcessor interface {
	ProcessVerifiedTransaction(ctx context.Context, provider, managerID string, event *VerifiedPaymentEvent) error
}

// SePayReconcileResult summarizes one reconciliation run for the caller/audit.
type SePayReconcileResult struct {
	PagesFetched int  `json:"pages_fetched"`
	Scanned      int  `json:"scanned"`
	Processed    int  `json:"processed"`
	Failed       int  `json:"failed"`
	Truncated    bool `json:"truncated"`
}

// SePayReconciliationService pulls SePay transactions via API v2 to settle invoices missed by webhooks.
type SePayReconciliationService struct {
	client            sePayTransactionLister
	credentialService reconciliationCredentialReader
	processor         verifiedTransactionProcessor
	throttle          time.Duration
}

// NewSePayReconciliationService wires API-v2 reconciliation for a manager's SePay account.
func NewSePayReconciliationService(client sePayTransactionLister, credentialService reconciliationCredentialReader, processor verifiedTransactionProcessor) *SePayReconciliationService {
	return &SePayReconciliationService{
		client:            client,
		credentialService: credentialService,
		processor:         processor,
		throttle:          sePayReconcileThrottle,
	}
}

// ReconcileManager pages through a manager's transactions in the window and processes incoming
// transfers that carry a payment code. Already-paid invoices are skipped by the shared processor.
func (s *SePayReconciliationService) ReconcileManager(ctx context.Context, managerID, dateFrom, dateTo string) (SePayReconcileResult, error) {
	var result SePayReconcileResult

	credentials, err := s.credentialService.GetCredentials(ctx, managerID, model.PaymentProviderSePay)
	if err != nil {
		return result, err
	}
	apiToken := credentials["api_token"]
	if apiToken == "" {
		return result, fmt.Errorf("%w: sepay api token not configured for reconciliation", ErrPaymentCredentialsNotFound)
	}

	for page := 1; page <= sePayMaxReconcilePages; page++ {
		response, err := s.client.ListTransactions(ctx, apiToken, SePayListParams{
			Environment: credentials["environment"],
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			Page:        page,
			PerPage:     sePayListMaxPerPage,
		})
		if err != nil {
			return result, err
		}
		result.PagesFetched++

		for i := range response.Data {
			result.Scanned++
			if err := s.processTransaction(ctx, managerID, response.Data[i], &result); err != nil {
				// A cancelled context aborts the whole run; a per-transaction failure is
				// logged and skipped so one bad transaction can't sink the entire batch.
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				result.Failed++
				log.Printf("SePay reconciliation manager %s: skipping transaction %s: %v", managerID, response.Data[i].ID, err)
				continue
			}
		}

		if !response.Meta.Pagination.HasMore {
			return result, nil
		}
		if page == sePayMaxReconcilePages {
			result.Truncated = true
			log.Printf("SePay reconciliation for manager %s hit the %d-page cap; remaining transactions not pulled", managerID, sePayMaxReconcilePages)
			break
		}

		// Respect SePay's 3 req/s rate limit between pages.
		if s.throttle > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(s.throttle):
			}
		}
	}

	return result, nil
}

// processTransaction settles a single incoming transaction that carries a payment code.
func (s *SePayReconciliationService) processTransaction(ctx context.Context, managerID string, tx SePayTransaction, result *SePayReconcileResult) error {
	if !strings.EqualFold(tx.TransferType, "in") || tx.AmountIn <= 0 {
		return nil
	}
	if tx.Code == "" {
		return nil
	}

	if err := s.processor.ProcessVerifiedTransaction(ctx, model.PaymentProviderSePay, managerID, mapSePayTransaction(tx)); err != nil {
		return fmt.Errorf("process reconciled transaction %s: %w", tx.ID, err)
	}
	result.Processed++
	return nil
}

// mapSePayTransaction converts an API v2 transaction into the provider-neutral verified event.
// The API v2 id (UUID) becomes the transaction reference; cross-source dedup is handled by the
// processor's already-paid guard, not by this reference.
func mapSePayTransaction(tx SePayTransaction) *VerifiedPaymentEvent {
	counterAccount := tx.AccountNumber
	rawPayload, _ := json.Marshal(tx)
	return &VerifiedPaymentEvent{
		ProviderOrderRef:     tx.Code,
		Amount:               tx.AmountIn,
		TransactionReference: tx.ID,
		CounterAccount:       &counterAccount,
		RawPayload:           string(rawPayload),
		SignatureResult:      paymentSignatureValid,
		MatchingMethod:       paymentMatchOrderRef,
	}
}
