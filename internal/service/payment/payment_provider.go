package payment

import (
	"context"
	"errors"
	"net/http"

	"github.com/mihb123/quanly-phongtro/internal/model"
)

const (
	paymentSignatureValid   = "VALID"
	paymentSignatureInvalid = "INVALID"
	paymentMatchOrderRef    = "ORDER_CODE"
	paymentMatchUnmatched   = "UNMATCHED"
	defaultTenantName       = "Khách thuê"
)

var (
	ErrPaymentProviderNotFound    = errors.New("payment provider not found")
	ErrPaymentCredentialsNotFound = errors.New("payment provider credentials not found")
	ErrPaymentWebhookIgnored      = errors.New("payment webhook ignored")
	ErrPaymentWebhookInvalid      = errors.New("invalid payment webhook")
)

type PaymentCreateInput struct {
	Invoice                *model.InvoiceWithRoom
	TenantName             string
	ManagerID              string
	AppURL                 string
	Credentials            map[string]string
	ProviderOrderRefExists func(ctx context.Context, providerOrderRef string) (bool, error)
}

type PaymentProviderLink struct {
	ProviderOrderRef string
	OrderCode        int64
	PaymentLinkID    string
	CheckoutURL      string
	QRCode           string
	Amount           int
}

type PaymentCancelInput struct {
	Credentials      map[string]string
	ProviderOrderRef string
	Reason           string
}

type PaymentWebhookInput struct {
	Body        []byte
	Headers     http.Header
	Credentials map[string]string
}

type VerifiedPaymentEvent struct {
	ProviderOrderRef     string
	OrderCode            int64
	Amount               int
	TransactionReference string
	PayerAccount         *string
	CounterAccount       *string
	RawPayload           string
	SignatureResult      string
	MatchingMethod       string
}

type PaymentProvider interface {
	Name() string
	CreatePaymentLink(ctx context.Context, input PaymentCreateInput) (*PaymentProviderLink, error)
	CancelPaymentLink(ctx context.Context, input PaymentCancelInput) error
	VerifyWebhook(ctx context.Context, input PaymentWebhookInput) (*VerifiedPaymentEvent, error)
}

type PaymentProviderRegistry struct {
	providers map[string]PaymentProvider
}

// NewPaymentProviderRegistry indexes configured provider adapters by provider name.
func NewPaymentProviderRegistry(providers ...PaymentProvider) *PaymentProviderRegistry {
	registry := &PaymentProviderRegistry{
		providers: make(map[string]PaymentProvider, len(providers)),
	}
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		registry.providers[provider.Name()] = provider
	}
	return registry
}

// Get returns the provider adapter for the given provider name.
func (r *PaymentProviderRegistry) Get(provider string) (PaymentProvider, bool) {
	if r == nil {
		return nil, false
	}
	providerAdapter, ok := r.providers[provider]
	return providerAdapter, ok
}
