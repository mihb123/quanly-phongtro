#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

if ! pw --raw eval "location.href" >/dev/null 2>&1; then
  pw open "$SEPAY_TESTMODE_URL/dashboard" --persistent --browser=chromium >/dev/null 2>&1
else
  pw goto "$SEPAY_TESTMODE_URL/dashboard" >/dev/null 2>&1
fi
sleep 2

if pw --raw eval "location.pathname" | grep -q login; then
  pw fill "getByRole('textbox', { name: 'Email hoặc số điện thoại' })" "$SEPAY_LOGIN_EMAIL" >/dev/null 2>&1
  pw fill "getByRole('textbox', { name: 'Mật khẩu' })" "$SEPAY_LOGIN_PASSWORD" >/dev/null 2>&1
  pw click "getByRole('button', { name: 'Đăng nhập' })" >/dev/null 2>&1
  sleep 4
  pw goto "$SEPAY_TESTMODE_URL/dashboard" >/dev/null 2>&1
  sleep 2
fi

url="$(pw --raw eval "location.href" | tr -d '"')"
if [[ "$url" != *"/testmode/"* ]]; then
  echo "Chưa vào được SePay Test Mode (url hiện tại: $url). Có thể cần captcha/OTP: mở trình duyệt bằng 'playwright-cli -s=$PW_SESSION show'." >&2
  exit 1
fi
echo "SePay Test Mode OK: $url"
