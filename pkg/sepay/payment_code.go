package sepay

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
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

// PaymentCodeSuffixLength is the fixed number of characters after the prefix. SePay's payment-code
// pattern must accept it (e.g. "PH[A-Za-z0-9]{6,8}"); longer codes get truncated by SePay.
const PaymentCodeSuffixLength = 6

// BuildPaymentCode creates a fixed-length code: prefix + the last 6 characters of the reference ID
// (the random tail of a UUIDv7, so it stays recognisable). Retries, and IDs too short to fill the
// suffix, use 6 base36 characters hashed from the ID and attempt so every code keeps the same length.
func BuildPaymentCode(prefix, referenceID string, attempt int) string {
	suffix := alphanumericUpper(referenceID)
	if attempt > 0 || len(suffix) < PaymentCodeSuffixLength {
		suffix = hashedSuffix(referenceID, attempt)
	} else {
		suffix = suffix[len(suffix)-PaymentCodeSuffixLength:]
	}

	// Uppercase the prefix too so a lowercase prefix still matches SePay's configured pattern.
	return alphanumericUpper(prefix) + suffix
}

func hashedSuffix(referenceID string, attempt int) string {
	digest := sha256.Sum256([]byte(referenceID + "#" + strconv.Itoa(attempt)))
	encoded := strings.ToUpper(strconv.FormatUint(binary.BigEndian.Uint64(digest[:8]), 36))
	encoded = strings.Repeat("0", PaymentCodeSuffixLength) + encoded
	return encoded[len(encoded)-PaymentCodeSuffixLength:]
}

// alphanumericUpper keeps only [A-Z0-9] so the code is safe inside a bank transfer memo.
func alphanumericUpper(value string) string {
	var safe strings.Builder
	for _, ch := range strings.ToUpper(value) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
			safe.WriteRune(ch)
		}
	}
	return safe.String()
}
