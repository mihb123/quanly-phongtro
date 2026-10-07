package sepay

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const QRImageBaseURL = "https://qr.sepay.vn/img"

const DefaultCodeAttempts = 5

type Payment struct {
	Code   string
	QRURL  string
	Amount int
}

// CreatePayment generates a unique payment code and the SePay QR transfer URL for it.
func CreatePayment(ctx context.Context, credentials Credentials, referenceID string, amount int, exists func(context.Context, string) (bool, error)) (*Payment, error) {
	if amount <= 0 {
		return nil, errors.New("sepay: amount must be greater than 0")
	}
	if credentials.BankShortName == "" || credentials.AccountNumber == "" || credentials.CodePrefix == "" {
		return nil, fmt.Errorf("%w: bank, account number and code prefix are required", ErrMissingCredentials)
	}
	code, err := GenerateUniquePaymentCode(ctx, credentials.CodePrefix, referenceID, DefaultCodeAttempts, exists)
	if err != nil {
		return nil, err
	}
	return &Payment{
		Code:   code,
		QRURL:  BuildQRURL(credentials.AccountNumber, credentials.BankShortName, amount, code),
		Amount: amount,
	}, nil
}

// BuildQRURL generates a SePay VietQR image URL for a bank transfer.
func BuildQRURL(accountNumber, bankShortName string, amount int, paymentCode string) string {
	values := url.Values{}
	values.Add("acc", accountNumber)
	values.Add("bank", bankShortName)
	values.Add("amount", strconv.Itoa(amount))
	values.Add("des", paymentCode)
	return QRImageBaseURL + "?" + values.Encode()
}

// GenerateUniquePaymentCode generates a new SePay payment code avoiding collisions.
func GenerateUniquePaymentCode(ctx context.Context, prefix, referenceID string, maxAttempts int, exists func(context.Context, string) (bool, error)) (string, error) {
	if maxAttempts <= 0 {
		maxAttempts = DefaultCodeAttempts
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code := BuildPaymentCode(prefix, referenceID, attempt)
		if exists == nil {
			return code, nil
		}
		found, err := exists(ctx, code)
		if err != nil {
			return "", err
		}
		if !found {
			return code, nil
		}
	}
	return "", errors.New("could not generate unique SePay payment code")
}

// BuildPaymentCode creates a formatted string from prefix and an invoice identifier.
func BuildPaymentCode(prefix, referenceID string, attempt int) string {
	cleanID := strings.ReplaceAll(referenceID, "-", "")
	cleanID = strings.ToUpper(cleanID)
	if len(cleanID) > 8 {
		cleanID = cleanID[:8]
	}

	// Uppercase the whole code so a lowercase prefix still survives the [A-Z0-9] filter below;
	// SePay must receive the prefix intact to auto-extract the payment code.
	code := strings.ToUpper(prefix + cleanID)
	if attempt > 0 {
		code += strconv.Itoa(attempt)
	}

	// Keep only [A-Z0-9] so the code is safe inside a bank transfer memo.
	var safeCode strings.Builder
	for _, ch := range code {
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			safeCode.WriteRune(ch)
		}
	}

	return safeCode.String()
}
