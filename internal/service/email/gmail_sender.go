package email

import (
	"bytes"
	"html/template"
	"strconv"
	"time"

	gomail "gopkg.in/mail.v2"
)

type GoogleSMTPSender struct {
	Host      string
	Port      int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

func NewGoogleSMTPSender(host string, port string, username string, password string, fromEmail string, fromName string) *GoogleSMTPSender {
	smtpPort, err := strconv.Atoi(port)
	if err != nil {
		smtpPort = 587
	}
	return &GoogleSMTPSender{
		Host:      host,
		Port:      smtpPort,
		Username:  username,
		Password:  password,
		FromEmail: fromEmail,
		FromName:  fromName,
	}
}

type otpEmailTemplateData struct {
	OTPCode       string
	ExpiresInMins int64
}

var otpEmailHTMLTemplate = template.Must(template.New("otp_email_html").Parse(`
<!doctype html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1.0" />
	<title>Your OTP Code</title>
</head>
<body style="margin:0;padding:0;background:#f6f8fb;font-family:Arial,Helvetica,sans-serif;color:#1f2937;">
	<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="padding:24px 12px;">
		<tr>
			<td align="center">
				<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="max-width:560px;background:#ffffff;border:1px solid #e5e7eb;border-radius:12px;overflow:hidden;">
					<tr>
						<td style="padding:24px 24px 16px 24px;">
							<p style="margin:0 0 8px 0;font-size:14px;color:#6b7280;">Email Verification</p>
							<h1 style="margin:0;font-size:22px;line-height:1.3;color:#111827;">Your one-time password</h1>
						</td>
					</tr>
					<tr>
						<td style="padding:8px 24px 0 24px;">
							<p style="margin:0;font-size:15px;line-height:1.6;color:#374151;">
								Use the code below to continue. This OTP expires in {{.ExpiresInMins}} minutes.
							</p>
						</td>
					</tr>
					<tr>
						<td style="padding:20px 24px 8px 24px;">
							<div style="display:inline-block;padding:12px 18px;border:1px dashed #d1d5db;border-radius:10px;background:#f9fafb;font-size:30px;letter-spacing:6px;font-weight:700;color:#111827;">
								{{.OTPCode}}
							</div>
						</td>
					</tr>
					<tr>
						<td style="padding:8px 24px 24px 24px;">
							<p style="margin:0;font-size:13px;line-height:1.6;color:#6b7280;">
								If you did not request this code, you can safely ignore this email.
							</p>
						</td>
					</tr>
				</table>
			</td>
		</tr>
	</table>
</body>
</html>
`))

func RenderOTPEmailHTML(otpCode string, expiresInMins int64) (string, error) {
	if expiresInMins <= 0 {
		expiresInMins = 3
	}

	var body bytes.Buffer
	err := otpEmailHTMLTemplate.Execute(&body, otpEmailTemplateData{
		OTPCode:       otpCode,
		ExpiresInMins: expiresInMins,
	})
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

func (s *GoogleSMTPSender) SendEmail(toEmail string, otpCode string, expiresIn time.Duration) error {

	messageBody, err := RenderOTPEmailHTML(otpCode, int64(expiresIn.Minutes()))
	if err != nil {
		return err
	}

	message := gomail.NewMessage()
	message.SetHeader("From", message.FormatAddress(s.FromEmail, s.FromName))
	message.SetHeader("To", toEmail)
	message.SetHeader("Subject", s.FromName+" - Mã xác thực tài khoản (OTP)")

	// Add plain text version first to reduce spam score
	message.SetBody("text/plain", "Mã xác thực (OTP) của bạn là: "+otpCode+".\nMã này sẽ hết hạn trong "+strconv.FormatInt(int64(expiresIn.Minutes()), 10)+" phút.\nNếu bạn không yêu cầu mã này, vui lòng bỏ qua email.")
	// Add HTML alternative
	message.AddAlternative("text/html", messageBody)
	dialer := gomail.NewDialer(s.Host, 587, s.Username, s.Password)
	return dialer.DialAndSend(message)

}
