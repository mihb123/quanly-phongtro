package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	// SePayEnvironmentProduction selects the live SePay User API.
	SePayEnvironmentProduction = "production"
	// SePayEnvironmentSandbox selects the isolated SePay Test Mode API.
	SePayEnvironmentSandbox = "sandbox"
	// sePayAPIBaseURL is the production base URL for SePay User API v2.
	sePayAPIBaseURL = "https://userapi.sepay.vn/v2"
	// sePaySandboxAPIBaseURL is the Test Mode base URL for SePay User API v2.
	sePaySandboxAPIBaseURL = "https://userapi-sandbox.sepay.vn/v2"
)

// sePayListMaxPerPage is the maximum page size accepted by the SePay transactions endpoint.
const sePayListMaxPerPage = 100

// SePayTransaction is one transaction object from the API v2 list endpoint.
// Field names are snake_case (API v2) and differ from the camelCase webhook payload; notably
// the id here is a UUID, not the integer id sent by webhooks.
type SePayTransaction struct {
	ID                 string `json:"id"`
	TransactionDate    string `json:"transaction_date"`
	AccountNumber      string `json:"account_number"`
	AmountIn           int    `json:"amount_in"`
	AmountOut          int    `json:"amount_out"`
	TransactionContent string `json:"transaction_content"`
	ReferenceNumber    string `json:"reference_number"`
	Code               string `json:"code"`
	BankBrandName      string `json:"bank_brand_name"`
	TransferType       string `json:"transfer_type"`
}

// sePayPagination is the meta.pagination block of the list response.
type sePayPagination struct {
	CurrentPage int  `json:"current_page"`
	LastPage    int  `json:"last_page"`
	HasMore     bool `json:"has_more"`
}

// SePayTransactionsResponse is the enveloped list-transactions response.
type SePayTransactionsResponse struct {
	Status string             `json:"status"`
	Data   []SePayTransaction `json:"data"`
	Meta   struct {
		Pagination sePayPagination `json:"pagination"`
	} `json:"meta"`
}

// SePayBankAccount is one company-linked account returned by the API v2 bank-account list.
type SePayBankAccount struct {
	BankShortName     string `json:"bank_short_name"`
	AccountNumber     string `json:"account_number"`
	AccountHolderName string `json:"account_holder_name"`
}

// SePayBankAccountsResponse is the enveloped API v2 bank-account list response.
type SePayBankAccountsResponse struct {
	Status string             `json:"status"`
	Data   []SePayBankAccount `json:"data"`
}

// SePayListBankAccountsParams identifies one linked account in a fixed SePay environment.
type SePayListBankAccountsParams struct {
	Environment   string
	BankShortName string
	AccountNumber string
}

// SePayListParams filters a list-transactions request.
type SePayListParams struct {
	Environment string
	DateFrom    string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	DateTo      string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	Page        int
	PerPage     int
}

// SePayClient calls the SePay User API v2 over HTTP.
type SePayClient struct {
	httpClient        *http.Client
	productionBaseURL string
	sandboxBaseURL    string
}

// NewSePayClient builds a SePay API v2 client against the production base URL.
func NewSePayClient() *SePayClient {
	return &SePayClient{
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		productionBaseURL: sePayAPIBaseURL,
		sandboxBaseURL:    sePaySandboxAPIBaseURL,
	}
}

// ListTransactions fetches one page of transactions authenticated with the manager API token.
func (c *SePayClient) ListTransactions(ctx context.Context, apiToken string, params SePayListParams) (*SePayTransactionsResponse, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("%w: sepay api token is required", ErrPaymentCredentialsNotFound)
	}

	query := url.Values{}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}
	if params.DateFrom != "" {
		query.Set("transaction_date_from", params.DateFrom)
	}
	if params.DateTo != "" {
		query.Set("transaction_date_to", params.DateTo)
	}

	baseURL, err := c.baseURL(params.Environment)
	if err != nil {
		return nil, err
	}
	endpoint := baseURL + "/transactions?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build sepay transactions request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+apiToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call sepay transactions: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read sepay transactions response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sepay transactions returned status %d: %s", response.StatusCode, string(body))
	}

	var parsed SePayTransactionsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse sepay transactions response: %w", err)
	}
	return &parsed, nil
}

// ListBankAccounts fetches active company-linked accounts matching the supplied bank and account number.
func (c *SePayClient) ListBankAccounts(ctx context.Context, apiToken string, params SePayListBankAccountsParams) (*SePayBankAccountsResponse, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("%w: sepay api token is required", ErrPaymentCredentialsNotFound)
	}

	query := url.Values{}
	query.Set("active", "true")
	query.Set("per_page", strconv.Itoa(sePayListMaxPerPage))
	if params.BankShortName != "" {
		query.Set("bank_short_name", params.BankShortName)
	}
	if params.AccountNumber != "" {
		query.Set("q", params.AccountNumber)
	}

	baseURL, err := c.baseURL(params.Environment)
	if err != nil {
		return nil, err
	}
	endpoint := baseURL + "/bank-accounts?" + query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build sepay bank accounts request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+apiToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call sepay bank accounts: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read sepay bank accounts response: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sepay bank accounts returned status %d: %s", response.StatusCode, string(body))
	}

	var parsed SePayBankAccountsResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse sepay bank accounts response: %w", err)
	}
	return &parsed, nil
}

// baseURL resolves the configured SePay environment without allowing arbitrary hosts.
func (c *SePayClient) baseURL(environment string) (string, error) {
	switch normalizeSePayEnvironment(environment) {
	case SePayEnvironmentProduction:
		return c.productionBaseURL, nil
	case SePayEnvironmentSandbox:
		return c.sandboxBaseURL, nil
	default:
		return "", fmt.Errorf("unsupported sepay environment: %s", environment)
	}
}

// normalizeSePayEnvironment keeps existing credentials on production by default.
func normalizeSePayEnvironment(environment string) string {
	if environment == "" {
		return SePayEnvironmentProduction
	}
	return environment
}
