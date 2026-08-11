package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestSePayClient builds a client whose live and sandbox hosts are independently observable.
func newTestSePayClient(t *testing.T, productionHandler, sandboxHandler http.Handler) (*SePayClient, *httptest.Server, *httptest.Server) {
	t.Helper()
	productionServer := httptest.NewServer(productionHandler)
	sandboxServer := httptest.NewServer(sandboxHandler)
	t.Cleanup(productionServer.Close)
	t.Cleanup(sandboxServer.Close)
	return &SePayClient{
		httpClient:        http.DefaultClient,
		productionBaseURL: productionServer.URL,
		sandboxBaseURL:    sandboxServer.URL,
	}, productionServer, sandboxServer
}

// TestSePayClientListTransactionsSelectsEnvironment verifies sandbox tokens never leak to the live API.
func TestSePayClientListTransactionsSelectsEnvironment(t *testing.T) {
	productionCalls := 0
	sandboxCalls := 0
	productionHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		productionCalls++
		_, _ = w.Write([]byte(`{"status":"success","data":[],"meta":{"pagination":{"has_more":false}}}`))
	})
	sandboxHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sandboxCalls++
		if r.Header.Get("Authorization") != "Bearer sandbox-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("per_page") != "100" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"tx-1","amount_in":1000,"code":"PH1","transfer_type":"in"}],"meta":{"pagination":{"has_more":false}}}`))
	})
	client, _, _ := newTestSePayClient(t, productionHandler, sandboxHandler)

	response, err := client.ListTransactions(context.Background(), "sandbox-token", SePayListParams{
		Environment: SePayEnvironmentSandbox,
		Page:        2,
		PerPage:     100,
	})
	if err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}
	if sandboxCalls != 1 || productionCalls != 0 {
		t.Fatalf("sandbox/live calls = %d/%d, want 1/0", sandboxCalls, productionCalls)
	}
	if len(response.Data) != 1 || response.Data[0].ID != "tx-1" {
		t.Fatalf("response = %+v", response)
	}
}

// TestSePayClientListTransactionsDefaultsToProduction preserves existing stored credentials.
func TestSePayClientListTransactionsDefaultsToProduction(t *testing.T) {
	productionCalls := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		productionCalls++
		_, _ = w.Write([]byte(`{"status":"success","data":[],"meta":{"pagination":{"has_more":false}}}`))
	})
	client, _, _ := newTestSePayClient(t, handler, http.NotFoundHandler())

	if _, err := client.ListTransactions(context.Background(), "token", SePayListParams{}); err != nil {
		t.Fatalf("ListTransactions() error = %v", err)
	}
	if productionCalls != 1 {
		t.Fatalf("production calls = %d, want 1", productionCalls)
	}
}

// TestSePayClientListTransactionsErrors covers validation, upstream, decode, and cancellation failures.
func TestSePayClientListTransactionsErrors(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		if _, err := NewSePayClient().ListTransactions(context.Background(), "", SePayListParams{}); !errors.Is(err, ErrPaymentCredentialsNotFound) {
			t.Fatalf("error = %v, want ErrPaymentCredentialsNotFound", err)
		}
	})
	t.Run("unsupported environment", func(t *testing.T) {
		if _, err := NewSePayClient().ListTransactions(context.Background(), "token", SePayListParams{Environment: "custom"}); err == nil {
			t.Fatal("expected unsupported environment error")
		}
	})
	t.Run("upstream status", func(t *testing.T) {
		client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}), http.NotFoundHandler())
		_, err := client.ListTransactions(context.Background(), "token", SePayListParams{})
		if err == nil || !strings.Contains(err.Error(), "status 401") {
			t.Fatalf("error = %v, want status 401", err)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}), http.NotFoundHandler())
		if _, err := client.ListTransactions(context.Background(), "token", SePayListParams{}); err == nil {
			t.Fatal("expected JSON decode error")
		}
	})
	t.Run("cancelled request", func(t *testing.T) {
		client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}), http.NotFoundHandler())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := client.ListTransactions(ctx, "token", SePayListParams{}); err == nil {
			t.Fatal("expected cancelled request error")
		}
	})
}

// TestSePayClientListBankAccounts verifies environment isolation, filters, authentication, and response parsing.
func TestSePayClientListBankAccounts(t *testing.T) {
	productionCalls := 0
	sandboxHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sandbox-token" {
			t.Errorf("Authorization = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/bank-accounts" {
			t.Errorf("path = %q", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("bank_short_name") != "Vietcombank" || query.Get("q") != "0000000001" {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		if query.Get("active") != "true" || query.Get("per_page") != "100" {
			t.Errorf("query limits = %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"status":"success","data":[{"id":"bank-1","bank_short_name":"Vietcombank","account_number":"0000000001","account_holder_name":"CONG TY TEST","active":true}]}`))
	})
	client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		productionCalls++
	}), sandboxHandler)

	response, err := client.ListBankAccounts(context.Background(), "sandbox-token", SePayListBankAccountsParams{
		Environment:   SePayEnvironmentSandbox,
		BankShortName: "Vietcombank",
		AccountNumber: "0000000001",
	})
	if err != nil {
		t.Fatalf("ListBankAccounts() error = %v", err)
	}
	if productionCalls != 0 {
		t.Fatalf("production calls = %d, want 0", productionCalls)
	}
	if len(response.Data) != 1 || response.Data[0].AccountHolderName != "CONG TY TEST" {
		t.Fatalf("response = %+v", response)
	}
}

// TestSePayClientListBankAccountsErrors covers missing credentials and malformed upstream responses.
func TestSePayClientListBankAccountsErrors(t *testing.T) {
	t.Run("missing token", func(t *testing.T) {
		if _, err := NewSePayClient().ListBankAccounts(context.Background(), "", SePayListBankAccountsParams{}); !errors.Is(err, ErrPaymentCredentialsNotFound) {
			t.Fatalf("error = %v, want ErrPaymentCredentialsNotFound", err)
		}
	})
	t.Run("upstream status", func(t *testing.T) {
		client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}), http.NotFoundHandler())
		_, err := client.ListBankAccounts(context.Background(), "token", SePayListBankAccountsParams{})
		if err == nil || !strings.Contains(err.Error(), "status 401") {
			t.Fatalf("error = %v, want status 401", err)
		}
	})
	t.Run("invalid json", func(t *testing.T) {
		client, _, _ := newTestSePayClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not-json"))
		}), http.NotFoundHandler())
		if _, err := client.ListBankAccounts(context.Background(), "token", SePayListBankAccountsParams{}); err == nil {
			t.Fatal("expected JSON decode error")
		}
	})
}
