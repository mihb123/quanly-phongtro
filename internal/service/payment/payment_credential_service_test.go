package payment

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

// fakePaymentCredentialRepo is a configurable stub for paymentCredentialRepository.
type fakePaymentCredentialRepo struct {
	byProvider map[string]*model.PaymentProviderCredential
	getErr     map[string]error
	upsertErr  error
	deleteErr  error
	staleErr   error

	upserted   []*model.PaymentProviderCredential
	deleted    []providerCall
	markedSate []providerCall
}

type providerCall struct {
	managerID string
	provider  string
}

func (r *fakePaymentCredentialRepo) GetActivePaymentProviderCredential(_ context.Context, _ string, provider string) (*model.PaymentProviderCredential, error) {
	if err, ok := r.getErr[provider]; ok && err != nil {
		return nil, err
	}
	return r.byProvider[provider], nil
}

func (r *fakePaymentCredentialRepo) UpsertPaymentProviderCredential(_ context.Context, c *model.PaymentProviderCredential) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.upserted = append(r.upserted, c)
	return nil
}

func (r *fakePaymentCredentialRepo) DeletePaymentProviderCredential(_ context.Context, managerID, provider string) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	r.deleted = append(r.deleted, providerCall{managerID, provider})
	return nil
}

func (r *fakePaymentCredentialRepo) MarkActivePaymentLinksStaleByManager(_ context.Context, managerID, provider string) error {
	if r.staleErr != nil {
		return r.staleErr
	}
	r.markedSate = append(r.markedSate, providerCall{managerID, provider})
	return nil
}

// testEncryptionKey returns a valid 32-byte AES key for round-trip encrypt/decrypt tests.
func testEncryptionKey() []byte {
	return bytes.Repeat([]byte{0xAB}, 32)
}

// encryptJSON encrypts arbitrary plaintext so it can be stored as EncryptedCredentials.
func encryptJSON(t *testing.T, key []byte, payload string) string {
	t.Helper()
	enc, err := security.Encrypt(payload, key)
	if err != nil {
		t.Fatalf("security.Encrypt: %v", err)
	}
	return enc
}

func validPayOSCreds() PayOSCredentials {
	return PayOSCredentials{ClientID: "client-1234567890", APIKey: "api-key", ChecksumKey: "checksum-key"}
}

func validSePayCreds() SePayCredentials {
	return SePayCredentials{
		BankShortName:     "MBBank",
		AccountNumber:     "0123456789",
		AccountName:       "NGUYEN VAN A",
		CodePrefix:        "PT",
		WebhookAuthMethod: "apikey",
		WebhookAPIKey:     "key",
		WebhookSecret:     "secret",
		APIToken:          "token",
	}
}

// TestNewPaymentCredentialServiceFallbackDetection verifies hasFallbackPayOS is set only for complete credentials.
func TestNewPaymentCredentialServiceFallbackDetection(t *testing.T) {
	key := testEncryptionKey()
	repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}

	t.Run("with fallback", func(t *testing.T) {
		s := NewPaymentCredentialService(repo, key, validPayOSCreds())
		_, err := s.GetCredentials(context.Background(), "", model.PaymentProviderPayOS)
		if err != nil {
			t.Fatalf("expected fallback credentials, got err = %v", err)
		}
	})
	t.Run("without fallback", func(t *testing.T) {
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "", model.PaymentProviderPayOS); err != ErrPaymentCredentialsNotFound {
			t.Fatalf("expected ErrPaymentCredentialsNotFound, got %v", err)
		}
	})
}

// TestGetCredentialsPayOS covers the PayOS branches of GetCredentials.
func TestGetCredentialsPayOS(t *testing.T) {
	key := testEncryptionKey()
	enc := encryptJSON(t, key, `{"client_id":"client-1234567890","api_key":"api-key","checksum_key":"checksum-key"}`)

	t.Run("manager credential found", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		creds, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if creds["client_id"] != "client-1234567890" {
			t.Errorf("client_id = %q", creds["client_id"])
		}
	})
	t.Run("manager credential nil with fallback", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, validPayOSCreds())
		creds, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if creds["checksum_key"] != "checksum-key" {
			t.Errorf("expected fallback map, got %v", creds)
		}
	})
	t.Run("manager credential nil without fallback", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS); err != ErrPaymentCredentialsNotFound {
			t.Fatalf("expected ErrPaymentCredentialsNotFound, got %v", err)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderPayOS: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
	t.Run("decrypt error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: "not-valid-base64-!!!", IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS); err == nil {
			t.Fatal("expected decrypt error, got nil")
		}
	})
	t.Run("unmarshal error", func(t *testing.T) {
		badEnc := encryptJSON(t, key, "not-json")
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: badEnc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderPayOS); err == nil {
			t.Fatal("expected unmarshal error, got nil")
		}
	})
}

// TestGetCredentialsSePay covers the SePay branches of GetCredentials.
func TestGetCredentialsSePay(t *testing.T) {
	key := testEncryptionKey()
	enc := encryptJSON(t, key, `{"bank_short_name":"MBBank","account_number":"0123456789","account_name":"NGUYEN VAN A","code_prefix":"PT","webhook_auth_method":"apikey","webhook_api_key":"key","webhook_secret":"secret","api_token":"token"}`)

	t.Run("manager credential found", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		creds, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderSePay)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if creds["account_number"] != "0123456789" {
			t.Errorf("account_number = %q", creds["account_number"])
		}
	})
	t.Run("manager empty returns not found", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "", model.PaymentProviderSePay); err != ErrPaymentCredentialsNotFound {
			t.Fatalf("expected ErrPaymentCredentialsNotFound, got %v", err)
		}
	})
	t.Run("manager credential nil returns not found", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderSePay); err != ErrPaymentCredentialsNotFound {
			t.Fatalf("expected ErrPaymentCredentialsNotFound, got %v", err)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderSePay: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderSePay); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
	t.Run("decrypt error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {EncryptedCredentials: "!!!invalid!!!", IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderSePay); err == nil {
			t.Fatal("expected decrypt error, got nil")
		}
	})
	t.Run("unmarshal error", func(t *testing.T) {
		badEnc := encryptJSON(t, key, "not-json")
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {EncryptedCredentials: badEnc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetCredentials(context.Background(), "mgr", model.PaymentProviderSePay); err == nil {
			t.Fatal("expected unmarshal error, got nil")
		}
	})
}

// TestGetCredentialsUnknownProvider ensures unknown providers are rejected.
func TestGetCredentialsUnknownProvider(t *testing.T) {
	repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
	s := NewPaymentCredentialService(repo, testEncryptionKey(), PayOSCredentials{})
	if _, err := s.GetCredentials(context.Background(), "mgr", "unknown"); err != ErrPaymentProviderNotFound {
		t.Fatalf("expected ErrPaymentProviderNotFound, got %v", err)
	}
}

// TestGetPayOSConfig covers masking, active flag, and error paths.
func TestGetPayOSConfig(t *testing.T) {
	key := testEncryptionKey()

	t.Run("no config returns empty status", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetPayOSConfig(context.Background(), "mgr", "https://app.example.com")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if status.HasConfig {
			t.Errorf("expected HasConfig=false")
		}
		if status.WebhookURL != "https://app.example.com/api/v1/payments/providers/payos/managers/mgr/webhook" {
			t.Errorf("unexpected webhook url: %s", status.WebhookURL)
		}
	})
	t.Run("active config masks client id", func(t *testing.T) {
		enc := encryptJSON(t, key, `{"client_id":"client-1234567890","api_key":"k","checksum_key":"c"}`)
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetPayOSConfig(context.Background(), "mgr", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !status.HasConfig || !status.IsActive {
			t.Errorf("expected HasConfig+IsActive, got %+v", status)
		}
		if status.MaskedClientID != "****7890" {
			t.Errorf("masked client id = %q, want ****7890", status.MaskedClientID)
		}
	})
	t.Run("short client id fully masked", func(t *testing.T) {
		enc := encryptJSON(t, key, `{"client_id":"abc","api_key":"k","checksum_key":"c"}`)
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetPayOSConfig(context.Background(), "mgr", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if status.MaskedClientID != "****" {
			t.Errorf("masked client id = %q, want ****", status.MaskedClientID)
		}
	})
	t.Run("empty client id returns empty mask", func(t *testing.T) {
		enc := encryptJSON(t, key, `{"client_id":"","api_key":"k","checksum_key":"c"}`)
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetPayOSConfig(context.Background(), "mgr", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if status.MaskedClientID != "" {
			t.Errorf("masked client id = %q, want empty", status.MaskedClientID)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderPayOS: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetPayOSConfig(context.Background(), "mgr", ""); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
	t.Run("decrypt error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {EncryptedCredentials: "!!!bad!!!", IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetPayOSConfig(context.Background(), "mgr", ""); err == nil {
			t.Fatal("expected decrypt error, got nil")
		}
	})
}

// TestSavePayOSConfig verifies encryption, upsert, and stale marking.
func TestSavePayOSConfig(t *testing.T) {
	key := testEncryptionKey()

	t.Run("success", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SavePayOSConfig(context.Background(), "mgr", validPayOSCreds()); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(repo.upserted) != 1 || repo.upserted[0].Provider != model.PaymentProviderPayOS || !repo.upserted[0].IsActive {
			t.Errorf("expected one active payos upsert, got %+v", repo.upserted)
		}
		if len(repo.markedSate) != 1 || repo.markedSate[0].provider != model.PaymentProviderPayOS {
			t.Errorf("expected stale marking, got %+v", repo.markedSate)
		}
	})
	t.Run("upsert error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			upsertErr:  errors.New("write failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SavePayOSConfig(context.Background(), "mgr", validPayOSCreds()); err == nil {
			t.Fatal("expected upsert error, got nil")
		}
		if len(repo.markedSate) != 0 {
			t.Errorf("stale marking should not run on upsert failure")
		}
	})
	t.Run("stale error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			staleErr:   errors.New("stale failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SavePayOSConfig(context.Background(), "mgr", validPayOSCreds()); err == nil {
			t.Fatal("expected stale error, got nil")
		}
	})
}

// TestDeletePayOSConfig verifies delete + stale marking and error propagation.
func TestDeletePayOSConfig(t *testing.T) {
	key := testEncryptionKey()
	t.Run("success", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeletePayOSConfig(context.Background(), "mgr"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(repo.deleted) != 1 || len(repo.markedSate) != 1 {
			t.Errorf("expected delete + stale call, got %+v/%+v", repo.deleted, repo.markedSate)
		}
	})
	t.Run("delete error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			deleteErr:  errors.New("delete failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeletePayOSConfig(context.Background(), "mgr"); err == nil {
			t.Fatal("expected delete error, got nil")
		}
	})
	t.Run("stale error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			staleErr:   errors.New("stale failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeletePayOSConfig(context.Background(), "mgr"); err == nil {
			t.Fatal("expected stale error, got nil")
		}
	})
}

// TestGetSePayConfig covers masking, fields, and error paths.
func TestGetSePayConfig(t *testing.T) {
	key := testEncryptionKey()

	t.Run("no config returns empty status", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetSePayConfig(context.Background(), "mgr", "https://app.example.com")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if status.HasConfig {
			t.Errorf("expected HasConfig=false")
		}
		if status.WebhookURL != "https://app.example.com/api/v1/payments/providers/sepay/managers/mgr/webhook" {
			t.Errorf("unexpected webhook url: %s", status.WebhookURL)
		}
	})
	t.Run("active config returns masked fields", func(t *testing.T) {
		enc := encryptJSON(t, key, `{"bank_short_name":"MBBank","account_number":"0123456789","account_name":"A","code_prefix":"PT","webhook_auth_method":"hmac","webhook_api_key":"k","webhook_secret":"s","api_token":"t"}`)
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {EncryptedCredentials: enc, IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		status, err := s.GetSePayConfig(context.Background(), "mgr", "")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !status.HasConfig || !status.IsActive {
			t.Errorf("expected HasConfig+IsActive, got %+v", status)
		}
		if status.MaskedAccountNumber != "****6789" || status.BankShortName != "MBBank" || status.CodePrefix != "PT" || status.WebhookAuthMethod != "hmac" {
			t.Errorf("unexpected sepay status: %+v", status)
		}
	})
	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderSePay: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetSePayConfig(context.Background(), "mgr", ""); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
	t.Run("decrypt error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {EncryptedCredentials: "!!!bad!!!", IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.GetSePayConfig(context.Background(), "mgr", ""); err == nil {
			t.Fatal("expected decrypt error, got nil")
		}
	})
}

// TestSaveSePayConfig verifies encryption, upsert, and stale marking.
func TestSaveSePayConfig(t *testing.T) {
	key := testEncryptionKey()

	t.Run("success", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SaveSePayConfig(context.Background(), "mgr", validSePayCreds()); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(repo.upserted) != 1 || repo.upserted[0].Provider != model.PaymentProviderSePay || !repo.upserted[0].IsActive {
			t.Errorf("expected one active sepay upsert, got %+v", repo.upserted)
		}
		if len(repo.markedSate) != 1 || repo.markedSate[0].provider != model.PaymentProviderSePay {
			t.Errorf("expected stale marking, got %+v", repo.markedSate)
		}
	})
	t.Run("upsert error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			upsertErr:  errors.New("write failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SaveSePayConfig(context.Background(), "mgr", validSePayCreds()); err == nil {
			t.Fatal("expected upsert error, got nil")
		}
	})
	t.Run("stale error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			staleErr:   errors.New("stale failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.SaveSePayConfig(context.Background(), "mgr", validSePayCreds()); err == nil {
			t.Fatal("expected stale error, got nil")
		}
	})
}

// TestDeleteSePayConfig verifies delete + stale marking and error propagation.
func TestDeleteSePayConfig(t *testing.T) {
	key := testEncryptionKey()
	t.Run("success", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeleteSePayConfig(context.Background(), "mgr"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(repo.deleted) != 1 || len(repo.markedSate) != 1 {
			t.Errorf("expected delete + stale call, got %+v/%+v", repo.deleted, repo.markedSate)
		}
	})
	t.Run("delete error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			deleteErr:  errors.New("delete failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeleteSePayConfig(context.Background(), "mgr"); err == nil {
			t.Fatal("expected delete error, got nil")
		}
	})
	t.Run("stale error", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			staleErr:   errors.New("stale failed"),
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if err := s.DeleteSePayConfig(context.Background(), "mgr"); err == nil {
			t.Fatal("expected stale error, got nil")
		}
	})
}

// TestResolvePreferredProvider covers provider priority and fallback logic.
func TestResolvePreferredProvider(t *testing.T) {
	key := testEncryptionKey()

	t.Run("sepay preferred when active", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderSePay: {IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, validPayOSCreds())
		provider, err := s.ResolvePreferredProvider(context.Background(), "mgr")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if provider != model.PaymentProviderSePay {
			t.Errorf("provider = %q, want sepay", provider)
		}
	})
	t.Run("payos when sepay nil", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{
			model.PaymentProviderPayOS: {IsActive: true},
		}}
		s := NewPaymentCredentialService(repo, key, validPayOSCreds())
		provider, err := s.ResolvePreferredProvider(context.Background(), "mgr")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if provider != model.PaymentProviderPayOS {
			t.Errorf("provider = %q, want payos", provider)
		}
	})
	t.Run("payos fallback when no manager credentials", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, validPayOSCreds())
		provider, err := s.ResolvePreferredProvider(context.Background(), "mgr")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if provider != model.PaymentProviderPayOS {
			t.Errorf("provider = %q, want payos", provider)
		}
	})
	t.Run("not found when nothing configured", func(t *testing.T) {
		repo := &fakePaymentCredentialRepo{byProvider: map[string]*model.PaymentProviderCredential{}}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.ResolvePreferredProvider(context.Background(), "mgr"); err != ErrPaymentCredentialsNotFound {
			t.Fatalf("expected ErrPaymentCredentialsNotFound, got %v", err)
		}
	})
	t.Run("sepay repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderSePay: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.ResolvePreferredProvider(context.Background(), "mgr"); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
	t.Run("payos repo error", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &fakePaymentCredentialRepo{
			byProvider: map[string]*model.PaymentProviderCredential{},
			getErr:     map[string]error{model.PaymentProviderPayOS: repoErr},
		}
		s := NewPaymentCredentialService(repo, key, PayOSCredentials{})
		if _, err := s.ResolvePreferredProvider(context.Background(), "mgr"); !errors.Is(err, repoErr) {
			t.Fatalf("expected repo error, got %v", err)
		}
	})
}
