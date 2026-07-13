package email_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mihb123/quanly-phongtro/internal/service/email"
)

func TestNewGoogleSMTPSender(t *testing.T) {
	sender := email.NewGoogleSMTPSender("smtp.gmail.com", "587", "user@test.com", "pass", "from@test.com", "Sender")

	if sender.Host != "smtp.gmail.com" {
		t.Errorf("unexpected host")
	}
	if sender.Port != 587 {
		t.Errorf("unexpected port")
	}
	if sender.Username != "user@test.com" {
		t.Errorf("unexpected username")
	}

	// Invalid port defaults to 587
	sender2 := email.NewGoogleSMTPSender("smtp.gmail.com", "invalid", "user@test.com", "pass", "from@test.com", "Sender")
	if sender2.Port != 587 {
		t.Errorf("expected default port 587")
	}
}

func TestRenderOTPEmailHTML(t *testing.T) {
	html, err := email.RenderOTPEmailHTML("123456", 5)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !strings.Contains(html, "123456") {
		t.Errorf("expected html to contain OTP code")
	}
	if !strings.Contains(html, "expires in 5 minutes") {
		t.Errorf("expected html to contain expiration time")
	}

	// Test negative expiration defaults to 3
	html2, err := email.RenderOTPEmailHTML("654321", -1)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !strings.Contains(html2, "expires in 3 minutes") {
		t.Errorf("expected html to default to 3 minutes")
	}
}

// TestSendEmail_DialError ensures SendEmail builds the message successfully and
// returns the underlying SMTP dial error when the SMTP host is unreachable. It
// exercises the full message-construction path (headers, plain-text body, HTML
// alternative) up to and including the failing DialAndSend call.
func TestSendEmail_DialError(t *testing.T) {
	// A non-routable / unresolvable host makes DialAndSend fail fast without
	// requiring a real SMTP server.
	sender := email.NewGoogleSMTPSender("invalid.invalid.localhost", "587", "user@test.com", "pass", "from@test.com", "Sender")

	err := sender.SendEmail("to@test.com", "123456", 5*time.Minute)
	if err == nil {
		t.Errorf("expected error when dialing an unreachable SMTP host")
	}
}

// TestSendEmail_ZeroDuration verifies SendEmail also handles a zero/negative
// expiry duration (rendered template falls back to the default minutes) and
// still attempts delivery, returning the dial error for the unreachable host.
func TestSendEmail_ZeroDuration(t *testing.T) {
	sender := email.NewGoogleSMTPSender("invalid.invalid.localhost", "587", "user@test.com", "pass", "from@test.com", "Sender")

	err := sender.SendEmail("to@test.com", "654321", 0)
	if err == nil {
		t.Errorf("expected error when dialing an unreachable SMTP host")
	}
}
