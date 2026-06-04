# 🧠 Code Review Report

**1. Logical Issues & Optimizations**

* File: `internal/service/zalo_service.go`
  * [line 260] **Invoice Status Query Logic**: The `GetLatestUnpaidInvoiceByRoomID` method is called when processing an image from the Zalo Webhook. However, this repository method strictly filters by `status = 'UNPAID'`. If a tenant uploads a second image (e.g., correcting a blurry photo), the invoice's status is already `PENDING_VERIFICATION`, causing the query to return `ErrInvoiceNotFound` and silently ignore the new image.
  * *Suggested Fix:* Modify the query or create a new repository method `GetLatestPendingOrUnpaidInvoiceByRoomID` that looks for invoices with either `UNPAID` or `PENDING_VERIFICATION` statuses, allowing tenants to resend proofs before manager confirmation.

  * [line 172] **Webhook Security (Timing Attack)**: The webhook secret token validation uses standard string equality comparison (`secretTokenHeader != webhookSecret`).
  * *Suggested Fix:* Use `crypto/subtle.ConstantTimeCompare([]byte(secretTokenHeader), []byte(webhookSecret))` to prevent timing attacks on the webhook validation endpoint.

* File: `frontend/src/components/home/modals/InvoiceDetailModal.tsx`
  * [line 346] **Hardcoded API Base URL**: The image `src` for the transaction proof is hardcoded to `http://localhost:8080/api/v1/uploads/...`. This will break when the application is deployed or accessed from another device (like testing on a mobile phone via local IP).
  * *Suggested Fix:* Remove the hardcoded domain and use a dynamic base URL from your environment variables (e.g., `import.meta.env.VITE_API_URL`) or read from `apiClient.defaults.baseURL`.

**2. Suggested Commit Message**

```text
feat(zalo): implement zalo bot invoice sending and webhook processing

- add rsa-encrypted webhook settings form for manager configuration
- integrate auto-linking of zalo group chats and rooms via webhook event
- add image generation and sending for invoices via custom zalo client
- process tenant transaction images from webhooks and update invoice status
```
