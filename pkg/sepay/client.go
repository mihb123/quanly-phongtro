package sepay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// ProductionAPIBaseURL is the production base URL for SePay User API v2.
	ProductionAPIBaseURL = "https://userapi.sepay.vn/v2"
	// SandboxAPIBaseURL is the Test Mode base URL for SePay User API v2.
	SandboxAPIBaseURL = "https://userapi-sandbox.sepay.vn/v2"
)

// MaxPerPage is the maximum page size accepted by the SePay transactions endpoint.
const MaxPerPage = 100

// Transaction is one transaction object from the API v2 list endpoint.
// Field names are snake_case (API v2) and differ from the camelCase webhook payload; notably
// the id here is a UUID, not the integer id sent by webhooks.
type Transaction struct {
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

// Pagination is the meta.pagination block of the list response.
type Pagination struct {
	CurrentPage int  `json:"current_page"`
	LastPage    int  `json:"last_page"`
	HasMore     bool `json:"has_more"`
}

// TransactionsResponse is the enveloped list-transactions response.
type TransactionsResponse struct {
	Status string        `json:"status"`
	Data   []Transaction `json:"data"`
	Meta   struct {
		Pagination Pagination `json:"pagination"`
	} `json:"meta"`
}

// BankAccount is one company-linked account returned by the API v2 bank-account list.
type BankAccount struct {
	BankShortName     string `json:"bank_short_name"`
	AccountNumber     string `json:"account_number"`
	AccountHolderName string `json:"account_holder_name"`
}

// BankAccountsResponse is the enveloped API v2 bank-account list response.
type BankAccountsResponse struct {
	Status string        `json:"status"`
	Data   []BankAccount `json:"data"`
}

// ListBankAccountsParams identifies one linked account in a fixed SePay environment.
type ListBankAccountsParams struct {
	Environment   string
	BankShortName string
	AccountNumber string
}

// ListTransactionsParams filters a list-transactions request.
type ListTransactionsParams struct {
	Environment string
	DateFrom    string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	DateTo      string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	Page        int
	PerPage     int
}

// Client calls the SePay User API v2 over HTTP.
type Client struct {
	httpClient        *http.Client
	productionBaseURL string
	sandboxBaseURL    string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{
		httpClient:        httpClient,
		productionBaseURL: ProductionAPIBaseURL,
		sandboxBaseURL:    SandboxAPIBaseURL,
	}
}

// ListTransactions fetches one page of transactions authenticated with the API token.
func (c *Client) ListTransactions(ctx context.Context, apiToken string, params ListTransactionsParams) (*TransactionsResponse, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("%w: sepay api token is required", ErrMissingCredentials)
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

	var parsed TransactionsResponse
	if err := c.get(ctx, apiToken, params.Environment, "/transactions?"+query.Encode(), "transactions", &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// ListBankAccounts fetches active company-linked accounts matching the supplied bank and account number.
func (c *Client) ListBankAccounts(ctx context.Context, apiToken string, params ListBankAccountsParams) (*BankAccountsResponse, error) {
	if apiToken == "" {
		return nil, fmt.Errorf("%w: sepay api token is required", ErrMissingCredentials)
	}

	query := url.Values{}
	query.Set("active", "true")
	query.Set("per_page", strconv.Itoa(MaxPerPage))
	if params.BankShortName != "" {
		query.Set("bank_short_name", params.BankShortName)
	}
	if params.AccountNumber != "" {
		query.Set("q", params.AccountNumber)
	}

	var parsed BankAccountsResponse
	if err := c.get(ctx, apiToken, params.Environment, "/bank-accounts?"+query.Encode(), "bank accounts", &parsed); err != nil {
		return nil, err
	}
	return &parsed, nil
}

// FindLinkedAccount returns the account that exactly matches bank + account number.
func FindLinkedAccount(accounts []BankAccount, bankShortName, accountNumber string) (*BankAccount, error) {
	for index := range accounts {
		account := &accounts[index]
		if strings.EqualFold(account.BankShortName, bankShortName) && account.AccountNumber == accountNumber {
			return account, nil
		}
	}
	return nil, ErrAccountNotLinked
}

func (c *Client) get(ctx context.Context, apiToken, environment, pathAndQuery, resource string, dst any) error {
	baseURL, err := c.baseURL(environment)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+pathAndQuery, nil)
	if err != nil {
		return fmt.Errorf("build sepay %s request: %w", resource, err)
	}
	request.Header.Set("Authorization", "Bearer "+apiToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("call sepay %s: %w", resource, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read sepay %s response: %w", resource, err)
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("sepay %s returned status %d: %s", resource, response.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, dst); err != nil {
		return fmt.Errorf("parse sepay %s response: %w", resource, err)
	}
	return nil
}

// baseURL resolves the configured SePay environment without allowing arbitrary hosts.
func (c *Client) baseURL(environment string) (string, error) {
	switch NormalizeEnvironment(environment) {
	case EnvironmentProduction:
		return c.productionBaseURL, nil
	case EnvironmentSandbox:
		return c.sandboxBaseURL, nil
	default:
		return "", fmt.Errorf("unsupported sepay environment: %s", environment)
	}
}
