package payment

import (
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/pkg/sepay"
)

type fakeSePayBankAccountLister struct {
	response *sepay.BankAccountsResponse
	err      error
	calls    int
	params   sepay.ListBankAccountsParams
}

// ListBankAccounts records account-verification requests and returns configured test data.
func (f *fakeSePayBankAccountLister) ListBankAccounts(_ context.Context, _ string, params sepay.ListBankAccountsParams) (*sepay.BankAccountsResponse, error) {
	f.calls++
	f.params = params
	return f.response, f.err
}

// TestVerifySePayBankAccount verifies exact linked-account matching and canonical account data.
func TestVerifySePayBankAccount(t *testing.T) {
	lister := &fakeSePayBankAccountLister{response: &sepay.BankAccountsResponse{
		Data: []sepay.BankAccount{{
			BankShortName:     "Vietcombank",
			AccountNumber:     "0000000001",
			AccountHolderName: "CONG TY TEST",
		}},
	}}
	handler := &PaymentHandler{sePayAccountLister: lister}

	account, err := handler.verifySePayBankAccount(context.Background(), sepay.Credentials{
		Environment:   sepay.EnvironmentSandbox,
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
	if lister.params.Environment != sepay.EnvironmentSandbox {
		t.Fatalf("environment = %q", lister.params.Environment)
	}
}

// TestVerifySePayBankAccountWithoutToken preserves optional reconciliation configuration.
func TestVerifySePayBankAccountWithoutToken(t *testing.T) {
	lister := &fakeSePayBankAccountLister{}
	handler := &PaymentHandler{sePayAccountLister: lister}

	account, err := handler.verifySePayBankAccount(context.Background(), sepay.Credentials{})
	if err != nil || account != nil {
		t.Fatalf("account/error = %+v/%v, want nil/nil", account, err)
	}
	if lister.calls != 0 {
		t.Fatalf("calls = %d, want 0", lister.calls)
	}
}

// TestVerifySePayBankAccountRejectsMismatch ensures similar search results cannot validate a different account.
func TestVerifySePayBankAccountRejectsMismatch(t *testing.T) {
	lister := &fakeSePayBankAccountLister{response: &sepay.BankAccountsResponse{
		Data: []sepay.BankAccount{{BankShortName: "Vietcombank", AccountNumber: "9999999999"}},
	}}
	handler := &PaymentHandler{sePayAccountLister: lister}

	_, err := handler.verifySePayBankAccount(context.Background(), sepay.Credentials{
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

	_, err := handler.verifySePayBankAccount(context.Background(), sepay.Credentials{APIToken: "token"})
	if !errors.Is(err, upstreamErr) {
		t.Fatalf("error = %v, want upstream error", err)
	}
}
