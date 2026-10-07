package sepay

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// DefaultReconcileThrottle spaces page requests to stay under SePay's 3 req/s API v2 limit.
const DefaultReconcileThrottle = 350 * time.Millisecond

// DefaultReconcileMaxPages caps a single reconciliation run so a runaway has_more never loops forever.
const DefaultReconcileMaxPages = 100

// TransactionLister fetches one page of API v2 transactions; satisfied by *Client.
type TransactionLister interface {
	ListTransactions(ctx context.Context, apiToken string, params ListTransactionsParams) (*TransactionsResponse, error)
}

// ReconcileResult summarizes one reconciliation run for the caller/audit.
type ReconcileResult struct {
	PagesFetched int  `json:"pages_fetched"`
	Scanned      int  `json:"scanned"`
	Processed    int  `json:"processed"`
	Failed       int  `json:"failed"`
	Truncated    bool `json:"truncated"`
}

type Reconciler struct {
	Lister   TransactionLister
	Throttle time.Duration
	MaxPages int
}

func NewReconciler(lister TransactionLister) *Reconciler {
	return &Reconciler{Lister: lister, Throttle: DefaultReconcileThrottle, MaxPages: DefaultReconcileMaxPages}
}

// Reconcile pages through transactions in the window and hands every incoming transfer that
// carries a payment code to handle. Settlement idempotency is the handler's responsibility.
func (r *Reconciler) Reconcile(ctx context.Context, credentials Credentials, dateFrom, dateTo string, handle func(context.Context, IncomingTransfer) error) (ReconcileResult, error) {
	var result ReconcileResult
	if credentials.APIToken == "" {
		return result, fmt.Errorf("%w: sepay api token not configured for reconciliation", ErrMissingCredentials)
	}
	maxPages := r.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultReconcileMaxPages
	}

	for page := 1; page <= maxPages; page++ {
		response, err := r.Lister.ListTransactions(ctx, credentials.APIToken, ListTransactionsParams{
			Environment: credentials.Environment,
			DateFrom:    dateFrom,
			DateTo:      dateTo,
			Page:        page,
			PerPage:     MaxPerPage,
		})
		if err != nil {
			return result, err
		}
		result.PagesFetched++

		for i := range response.Data {
			result.Scanned++
			tx := response.Data[i]
			if !strings.EqualFold(tx.TransferType, "in") || tx.AmountIn <= 0 || tx.Code == "" {
				continue
			}
			if err := handle(ctx, TransferFromTransaction(tx)); err != nil {
				// A cancelled context aborts the whole run; a per-transaction failure is
				// logged and skipped so one bad transaction can't sink the entire batch.
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				result.Failed++
				log.Printf("SePay reconciliation: skipping transaction %s: %v", tx.ID, err)
				continue
			}
			result.Processed++
		}

		if !response.Meta.Pagination.HasMore {
			return result, nil
		}
		if page == maxPages {
			result.Truncated = true
			log.Printf("SePay reconciliation hit the %d-page cap; remaining transactions not pulled", maxPages)
			break
		}

		// Respect SePay's 3 req/s rate limit between pages.
		if r.Throttle > 0 {
			select {
			case <-ctx.Done():
				return result, ctx.Err()
			case <-time.After(r.Throttle):
			}
		}
	}

	return result, nil
}

// TransferFromTransaction converts an API v2 transaction into an incoming transfer.
// The API v2 id (UUID) becomes the transaction reference; it differs from the webhook integer id,
// so cross-source dedup must rely on the settled state of the payment, not on this reference.
func TransferFromTransaction(tx Transaction) IncomingTransfer {
	rawPayload, _ := json.Marshal(tx)
	return IncomingTransfer{
		PaymentCode:          tx.Code,
		Amount:               tx.AmountIn,
		TransactionReference: tx.ID,
		AccountNumber:        tx.AccountNumber,
		RawPayload:           string(rawPayload),
	}
}
