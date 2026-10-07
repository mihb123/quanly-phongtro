#!/usr/bin/env bash
. "$(dirname "$0")/env.sh"

if [[ $# -lt 2 ]]; then
  echo "Dùng: $0 <payment_code> <amount VND>" >&2
  exit 1
fi
code="$1" amount="$2"
[[ "$code" =~ ^[A-Z0-9]+$ ]] || { echo "payment_code chỉ gồm A-Z0-9" >&2; exit 1; }
[[ "$amount" =~ ^[0-9]+$ ]] || { echo "amount phải là số nguyên" >&2; exit 1; }
[[ "$SEPAY_TEST_ACCOUNT" =~ ^[0-9]+$ ]] || { echo "SEPAY_TEST_ACCOUNT không hợp lệ" >&2; exit 1; }

pw run-code "async page => {
  await page.goto('$SEPAY_TESTMODE_URL/transaction/simulate');
  if (!page.url().includes('/testmode/')) throw new Error('Chưa đăng nhập SePay Test Mode, chạy sepay-login.sh trước');
  await page.getByText('Chọn tài khoản ngân hàng...').click();
  await page.getByRole('option', { name: /$SEPAY_TEST_ACCOUNT/ }).click();
  const va = page.getByText('Chọn VA (bắt buộc)...');
  if (await va.isVisible()) {
    await va.click();
    await page.getByRole('option').filter({ visible: true }).first().click();
  }
  await page.getByRole('radio', { name: 'Tiền vào' }).check();
  await page.getByRole('textbox', { name: 'Số tiền (VND) *' }).fill('$amount');
  await page.getByRole('textbox', { name: 'Nội dung chuyển khoản' }).fill('Thanh toan $code');
  await page.getByRole('button', { name: 'Gửi giao dịch thử' }).click();
  await page.waitForTimeout(5000);
  const text = await page.locator('body').innerText();
  return (text.match(/Quy trình xử lý[\\s\\S]*?(?=Hướng dẫn mô phỏng|$)/) || [text.slice(0, 800)])[0];
}" 2>&1 | sed -n '/### Result/,/### Ran/p' | sed '$d'
