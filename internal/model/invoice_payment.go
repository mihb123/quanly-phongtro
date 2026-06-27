package model

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

const (
	PaymentProviderPayOS = "payos"
	PaymentProviderSePay = "sepay"

	PaymentLinkStatusActive    = "ACTIVE"
	PaymentLinkStatusStale     = "STALE"
	PaymentLinkStatusCancelled = "CANCELLED"
	PaymentLinkStatusPaid      = "PAID"

	PaymentEventStatusPending   = "PENDING"
	PaymentEventStatusProcessed = "PROCESSED"
	PaymentEventStatusUnmatched = "UNMATCHED"
)

// InvoicePaymentLink tracks payment links generated for an invoice by a provider.
type InvoicePaymentLink struct {
	bun.BaseModel    `bun:"table:invoice_payment_links"`
	ID               string    `json:"id"`
	InvoiceID        string    `json:"invoice_id"`
	Provider         string    `json:"provider"`
	ProviderOrderRef string    `json:"provider_order_ref"`
	OrderCode        int64     `json:"order_code"` // PayOS compatibility field
	PaymentLinkID    string    `json:"payment_link_id"`
	CheckoutURL      string    `json:"checkout_url"`
	QRCode           string    `json:"qr_code"`
	Amount           int       `json:"amount"` // payment providers use integer VND amounts
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// PaymentProviderCredential stores encrypted manager-specific provider credentials.
type PaymentProviderCredential struct {
	bun.BaseModel        `bun:"table:payment_provider_credentials"`
	ID                   string    `json:"id"`
	ManagerID            string    `json:"manager_id"`
	Provider             string    `json:"provider"`
	EncryptedCredentials string    `json:"-"`
	IsActive             bool      `json:"is_active"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// PaymentEvent logs verified webhook callbacks from payment providers.
type PaymentEvent struct {
	bun.BaseModel        `bun:"table:payment_events"`
	ID                   string    `json:"id"`
	Provider             string    `json:"provider"`
	ManagerID            *string   `json:"manager_id"`
	InvoiceID            *string   `json:"invoice_id"`
	ProviderOrderRef     string    `json:"provider_order_ref"`
	OrderCode            int64     `json:"order_code"` // PayOS compatibility field
	Amount               int       `json:"amount"`
	TransactionReference string    `json:"transaction_reference"`
	PayerAccount         *string   `json:"payer_account"`
	CounterAccount       *string   `json:"counter_account"`
	RawPayload           string    `json:"raw_payload"`
	SignatureResult      string    `json:"signature_result"` // e.g. "VALID", "INVALID"
	MatchingMethod       *string   `json:"matching_method"`  // e.g. "ORDER_CODE", "TENANT_NAME", "AMOUNT"
	Status               string    `json:"status"`           // e.g. "PROCESSED", "UNMATCHED"
	CreatedAt            time.Time `json:"created_at"`
}

// PayOSPaymentEvent is kept as a compatibility alias for older tests and call sites.
type PayOSPaymentEvent = PaymentEvent

// PaymentRepository defines provider-neutral payment persistence operations.
type PaymentRepository interface {
	CreatePaymentLink(ctx context.Context, link *InvoicePaymentLink) error
	GetActivePaymentLinkByProvider(ctx context.Context, invoiceID, provider string) (*InvoicePaymentLink, error)
	GetPaymentLinkByProviderOrderRef(ctx context.Context, provider, providerOrderRef string) (*InvoicePaymentLink, error)
	GetPaymentLinkByProviderOrderRefForManager(ctx context.Context, managerID, provider, providerOrderRef string) (*InvoicePaymentLink, error)
	UpdatePaymentLinkStatus(ctx context.Context, id, status string) error
	MarkActivePaymentLinksStaleByManager(ctx context.Context, managerID, provider string) error
	CheckProviderEventExists(ctx context.Context, provider, providerOrderRef, transactionRef string) (bool, error)
	CreatePaymentEvent(ctx context.Context, event *PaymentEvent) error
	GetActivePaymentProviderCredential(ctx context.Context, managerID, provider string) (*PaymentProviderCredential, error)
	UpsertPaymentProviderCredential(ctx context.Context, credential *PaymentProviderCredential) error
	DeletePaymentProviderCredential(ctx context.Context, managerID, provider string) error
}
