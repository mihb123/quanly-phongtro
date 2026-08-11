package payment

import (
	"context"
	"errors"
	"testing"

	paymentsvc "github.com/mihb123/quanly-phongtro/internal/service/payment"
)

type fakeSePayBankAccountLister struct {
	response *paymentsvc.SePayBankAccountsResponse
	err      error
	calls    int
	params   paymentsvc.SePayListBankAccountsParams
}

// ListBankAccounts records account-verification requests and returns configured test data.
func (f *fakeSePayBankAccountLister) ListBankAccounts(_ context.Context, _ string, params paymentsvc.SePayListBankAccountsParams) (*paymentsvc.SePayBankAccountsResponse, error) {
	f.calls++
	f.params = params
	return f.response, f.err
}

// TestVerifySePayBankAccount verifies exact linked-account matching and canonical account data.
func TestVerifySePayBankAccount(t *testing.T) {
	lister := &fakeSePayBankAccountLister{response: &paymentsvc.SePayBankAccountsResponse{
		Data: []paymentsvc.SePayBankAccount{{
			BankShortName:     "Vietcombank",
			AccountNumber:     "0000000001",
			AccountHolderName: "CONG TY TEST",
		}},
	}}
	handler := &PaymentHandler{sePayAccountLister: lister}

	account, err := handler.verifySePayBankAccount(context.Background(), paymentsvc.SePayCredentials{
		Environment:   paymentsvc.SePayEnvironmentSandbox,
		BankShortName: "VIETCOMBANK",
		AccountNumber: "0000000001",
		APIToken:      "token",
	})
	if err != nil {
		t.Fatalf("verifySePayBankAccount() error = %v", err)
	}
	if account.AccountHolderName != "CONG TY TEST" {
		t.Fatalf("account = %+v", account)
	}
	if lister.params.Environment != paymentsvc.SePayEnvironmentSandbox {
		t.Fatalf("environment = %q", lister.params.Environment)
	}
}

// TestVerifySePayBankAccountWithoutToken preserves optional reconciliation configuration.
func TestVerifySePayBankAccountWithoutToken(t *testing.T) {
	lister := &fakeSePayBankAccountLister{}
	handler := &PaymentHandler{sePayAccountLister: lister}

	account, err := handler.verifySePayBankAccount(context.Background(), paymentsvc.SePayCredentials{})
	if err != nil || account != nil {
		t.Fatalf("account/error = %+v/%v, want nil/nil", account, err)
	}
	if lister.calls != 0 {
		t.Fatalf("calls = %d, want 0", lister.calls)
	}
}

// TestVerifySePayBankAccountRejectsMismatch ensures similar search results cannot validate a different account.
func TestVerifySePayBankAccountRejectsMismatch(t *testing.T) {
	lister := &fakeSePayBankAccountLister{response: &paymentsvc.SePayBankAccountsResponse{
		Data: []paymentsvc.SePayBankAccount{{BankShortName: "Vietcombank", AccountNumber: "9999999999"}},
	}}
	handler := &PaymentHandler{sePayAccountLister: lister}

	_, err := handler.verifySePayBankAccount(context.Background(), paymentsvc.SePayCredentials{
		BankShortName: "Vietcombank",
		AccountNumber: "0000000001",
		APIToken:      "token",
	})
	if err == nil {
		t.Fatal("expected unlinked account error")
	}
}

// TestVerifySePayBankAccountPropagatesAPIError keeps invalid tokens and upstream failures visible to the save flow.
func TestVerifySePayBankAccountPropagatesAPIError(t *testing.T) {
	upstreamErr := errors.New("unauthorized")
	handler := &PaymentHandler{sePayAccountLister: &fakeSePayBankAccountLister{err: upstreamErr}}

	_, err := handler.verifySePayBankAccount(context.Background(), paymentsvc.SePayCredentials{APIToken: "token"})
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("error = %v, want upstream error", err)
	}
}
