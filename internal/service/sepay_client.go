package service

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

// sePayAPIBaseURL is the production base URL for SePay User API v2.
const sePayAPIBaseURL = "https://userapi.sepay.vn/v2"

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

// SePayListParams filters a list-transactions request.
type SePayListParams struct {
	DateFrom string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	DateTo   string // "YYYY-MM-DD HH:MM:SS", inclusive (optional)
	Page     int
	PerPage  int
}

// SePayClient calls the SePay User API v2 over HTTP.
type SePayClient struct {
	httpClient *http.Client
	baseURL    string
}

// NewSePayClient builds a SePay API v2 client against the production base URL.
func NewSePayClient() *SePayClient {
	return &SePayClient{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    sePayAPIBaseURL,
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

	endpoint := c.baseURL + "/transactions?" + query.Encode()
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
