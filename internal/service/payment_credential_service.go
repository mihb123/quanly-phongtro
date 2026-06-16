package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"github.com/mihb123/quanly-phongtro/internal/security"
)

type PaymentCredentialService interface {
	GetCredentials(ctx context.Context, managerID, provider string) (map[string]string, error)
	GetPayOSConfig(ctx context.Context, managerID, appURL string) (PayOSConfigStatus, error)
	SavePayOSConfig(ctx context.Context, managerID string, credentials PayOSCredentials) error
	DeletePayOSConfig(ctx context.Context, managerID string) error
}

type PayOSCredentials struct {
	ClientID    string `json:"client_id"`
	APIKey      string `json:"api_key"`
	ChecksumKey string `json:"checksum_key"`
}

type PayOSConfigStatus struct {
	HasConfig      bool   `json:"has_config"`
	IsActive       bool   `json:"is_active"`
	Provider       string `json:"provider"`
	MaskedClientID string `json:"masked_client_id"`
	WebhookURL     string `json:"webhook_url"`
}

type paymentCredentialRepository interface {
	GetActivePaymentProviderCredential(ctx context.Context, managerID, provider string) (*model.PaymentProviderCredential, error)
	UpsertPaymentProviderCredential(ctx context.Context, credential *model.PaymentProviderCredential) error
	DeletePaymentProviderCredential(ctx context.Context, managerID, provider string) error
	MarkActivePaymentLinksStaleByManager(ctx context.Context, managerID, provider string) error
}

type paymentCredentialService struct {
	repository       paymentCredentialRepository
	encryptionKey    []byte
	fallbackPayOS    PayOSCredentials
	hasFallbackPayOS bool
}

// NewPaymentCredentialService creates encrypted credential storage with optional app-level fallback credentials.
func NewPaymentCredentialService(repository paymentCredentialRepository, encryptionKey []byte, fallbackPayOS PayOSCredentials) PaymentCredentialService {
	return &paymentCredentialService{
		repository:       repository,
		encryptionKey:    encryptionKey,
		fallbackPayOS:    fallbackPayOS,
		hasFallbackPayOS: validPayOSCredentials(fallbackPayOS),
	}
}

// GetCredentials returns manager credentials first, then configured app-level fallback credentials.
func (s *paymentCredentialService) GetCredentials(ctx context.Context, managerID, provider string) (map[string]string, error) {
	if provider != model.PaymentProviderPayOS {
		return nil, ErrPaymentProviderNotFound
	}
	if managerID != "" {
		credential, err := s.repository.GetActivePaymentProviderCredential(ctx, managerID, provider)
		if err != nil {
			return nil, err
		}
		if credential != nil {
			return s.decryptCredentials(credential.EncryptedCredentials)
		}
	}
	if s.hasFallbackPayOS {
		return payOSCredentialsMap(s.fallbackPayOS), nil
	}
	return nil, ErrPaymentCredentialsNotFound
}

// GetPayOSConfig returns masked manager PayOS configuration details and webhook URL.
func (s *paymentCredentialService) GetPayOSConfig(ctx context.Context, managerID, appURL string) (PayOSConfigStatus, error) {
	status := PayOSConfigStatus{
		Provider:   model.PaymentProviderPayOS,
		WebhookURL: appURL + "/api/v1/payments/providers/payos/managers/" + managerID + "/webhook",
	}
	credential, err := s.repository.GetActivePaymentProviderCredential(ctx, managerID, model.PaymentProviderPayOS)
	if err != nil {
		return PayOSConfigStatus{}, err
	}
	if credential == nil {
		return status, nil
	}

	credentials, err := s.decryptPayOSCredentials(credential.EncryptedCredentials)
	if err != nil {
		return PayOSConfigStatus{}, err
	}
	status.HasConfig = true
	status.IsActive = credential.IsActive
	status.MaskedClientID = maskSecret(credentials.ClientID)
	return status, nil
}

// SavePayOSConfig encrypts and stores manager-specific PayOS credentials.
func (s *paymentCredentialService) SavePayOSConfig(ctx context.Context, managerID string, credentials PayOSCredentials) error {
	payload, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("marshal payos credentials: %w", err)
	}
	encryptedCredentials, err := security.Encrypt(string(payload), s.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt payos credentials: %w", err)
	}

	if err := s.repository.UpsertPaymentProviderCredential(ctx, &model.PaymentProviderCredential{
		ManagerID:            managerID,
		Provider:             model.PaymentProviderPayOS,
		EncryptedCredentials: encryptedCredentials,
		IsActive:             true,
	}); err != nil {
		return err
	}
	return s.repository.MarkActivePaymentLinksStaleByManager(ctx, managerID, model.PaymentProviderPayOS)
}

// DeletePayOSConfig deactivates manager-specific PayOS credentials.
func (s *paymentCredentialService) DeletePayOSConfig(ctx context.Context, managerID string) error {
	if err := s.repository.DeletePaymentProviderCredential(ctx, managerID, model.PaymentProviderPayOS); err != nil {
		return err
	}
	return s.repository.MarkActivePaymentLinksStaleByManager(ctx, managerID, model.PaymentProviderPayOS)
}

// decryptCredentials returns generic provider credentials from encrypted JSON.
func (s *paymentCredentialService) decryptCredentials(encryptedCredentials string) (map[string]string, error) {
	payOSCredentials, err := s.decryptPayOSCredentials(encryptedCredentials)
	if err != nil {
		return nil, err
	}
	return payOSCredentialsMap(payOSCredentials), nil
}

// decryptPayOSCredentials decrypts stored PayOS credential JSON.
func (s *paymentCredentialService) decryptPayOSCredentials(encryptedCredentials string) (PayOSCredentials, error) {
	plaintext, err := security.Decrypt(encryptedCredentials, s.encryptionKey)
	if err != nil {
		return PayOSCredentials{}, fmt.Errorf("decrypt payment credentials: %w", err)
	}
	var credentials PayOSCredentials
	if err := json.Unmarshal([]byte(plaintext), &credentials); err != nil {
		return PayOSCredentials{}, fmt.Errorf("unmarshal payos credentials: %w", err)
	}
	return credentials, nil
}

// payOSCredentialsMap converts typed PayOS credentials to the provider adapter format.
func payOSCredentialsMap(credentials PayOSCredentials) map[string]string {
	return map[string]string{
		"client_id":    credentials.ClientID,
		"api_key":      credentials.APIKey,
		"checksum_key": credentials.ChecksumKey,
	}
}

// validPayOSCredentials checks whether fallback PayOS credentials are complete.
func validPayOSCredentials(credentials PayOSCredentials) bool {
	return credentials.ClientID != "" && credentials.APIKey != "" && credentials.ChecksumKey != ""
}

// maskSecret exposes only a short suffix so managers can identify configured credentials.
func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return "****" + value[len(value)-4:]
}
