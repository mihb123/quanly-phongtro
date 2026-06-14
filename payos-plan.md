payOS + Zalo Invoice Payment Integration Plan

  Summary

  - Use payOS Go SDK v2 (github.com/payOSHQ/payos-lib-golang/v2) with single app-level credentials in v1; keep per-manager payOS
    credentials as phase 2.

  - Create a payOS payment link automatically for every new unpaid invoice. When the link is created, send the existing invoice image
    through Zalo with the payOS checkout/QR payment instruction.

  - payOS webhook is the trusted payment source. A valid signed successful webhook auto-marks the invoice PAID, records transaction
    details, updates revenue summaries, and sends Zalo notifications to manager and tenant.

  - Create Documents/plan/payos_integration_alignment.md with the aligned questions, selected answers, payOS doc links, and final
    decisions.

  Key Changes

  - Add config/env: PAYOS_CLIENT_ID, PAYOS_API_KEY, PAYOS_CHECKSUM_KEY; use APP_URL/APP_URL_DEV for payOS webhook and required return/
    cancel URLs.

  - Add payment persistence, separate from invoices:
      - invoice_payment_links: invoice id, payOS orderCode, paymentLinkId, checkoutUrl, qrCode, amount, status (ACTIVE, STALE,
        CANCELLED, PAID), timestamps.

      - payos_payment_events: raw webhook payload, signature result, matching method, transaction reference, amount, payer/counter
        account fields, processing status.

  - Backend API:
      - Public POST /api/v1/payos/webhook verifies payOS signature and processes payment events.
      - Public GET /api/v1/payos/return and GET /api/v1/payos/cancel return simple text only; they do not update invoice status.
      - Authenticated POST /api/v1/invoice/{id}/payment-link manually recreates a payOS link after invoice edits and sends Zalo.

  - Invoice behavior:
      - New invoice: create payment link, then send Zalo invoice image + payment link/QR instruction when Zalo channel exists.
      - Edited unpaid invoice with existing link: cancel old payOS link, mark it CANCELLED/STALE, leave invoice unpaid, require manual recreate.
      - Paid invoice remains non-editable under current behavior.

  - Matching priority for webhook events:
      - First: exact active orderCode/paymentLinkId/payment description match.
      - Second: transfer name matches an active tenant
      - Third: exact amount uniquely matches one unpaid invoice under the manager.
      - Ambiguous or mismatched events are recorded as unmatched and notify the manager, but do not mark paid.

  - Zalo notifications:
      - On payment success: notify manager and linked tenant/group with invoice room, period, amount, and transaction reference.
      - On unsuccessful/cancelled payment: notify tenant/group that payment was not completed and invoice remains unpaid.
      - Existing manual /zalo/invoices/{id}/send should include payOS payment details and create a link if missing.

  Questions MD File

  - Create Documents/plan/payos_integration_alignment.md containing:
      - Docs reviewed: payOS intro, API payment link, webhook payload/signature, Go SDK.
      - Decisions selected: phase 1 single account then per-manager later; auto mark paid; create link on invoice create; cancel old
        link on invoice edit; Zalo-centered UX.

      - Operational setup questions for deployment: payOS account credentials, webhook URL configured in payOS dashboard, test merchant
        account availability, and Zalo manager/tenant linking readiness.

  Test Plan

  - Unit test payOS service: create link payload, cancel stale link, webhook signature rejection, success webhook marks invoice paid,
    duplicate webhook idempotency.

  - Unit test matching priority: direct order code, tenant name + amount, unique amount, ambiguous amount stays unmatched.
  - Handler tests for public webhook/return/cancel routes and authenticated recreate-link route.
  - Repository tests for payment link/event insert, active-link lookup, stale/cancel transitions.
  - Zalo service tests that invoice sending includes payOS info and success/failure notifications are sent to expected chat IDs.

  Assumptions

  - v1 uses one payOS merchant account for the whole app; per-manager payOS credentials are explicitly out of v1 implementation.
  - Amounts sent to payOS are integer VND derived from invoice.total_amount; decimal invoice totals must be rejected or rounded by a
    clearly defined helper before link creation.

  - Return/cancel URLs exist only because payOS requires them; Zalo messages are the tenant-facing flow.
  - Official docs used: https://payos.vn/docs/, https://payos.vn/docs/api/, https://payos.vn/docs/du-lieu-tra-ve/webhook/,
    https://payos.vn/docs/tich-hop-webhook/kiem-tra-du-lieu-voi-signature/, https://payos.vn/docs/sdks/back-end/golang/.
