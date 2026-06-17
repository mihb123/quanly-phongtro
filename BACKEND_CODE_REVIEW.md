# Backend Code & Business Logic Review

Date: 2026-06-17
Scope reviewed: Go backend under `cmd/`, `config/`, `internal/{handler,router,service,repository,model,security}`, plus migrations where they affect business logic.

Validation run:

- `go test ./...` ✅
- `go vet ./...` ✅

## Executive summary

The backend has a clear package split (`handler`, `service`, `repository`, `model`, `router`, `security`) and broad unit-test coverage across handlers, services, repositories, router, and security utilities. The main risks are not compile-time problems; they are business-rule and authorization gaps around ownership checks, payment/webhook state transitions, and a schema/model mismatch in invoice status handling.

## High-priority findings

### 1. Room creation can bypass manager ownership

- Location: `internal/service/room_service.go:62-71`
- Severity: High
- Issue: `CreateRoom` calls `s.houseRepo.GetByID(ctx, room.HouseID, managerID)` but ignores the returned house and error. Because `RoomRepository.CreateRoom` explicitly says ownership must be verified by the caller, a manager who knows another manager's `house_id` can create a room under that house.
- Impact: Cross-manager data pollution and unauthorized mutation of another manager's house inventory.
- Recommendation: Return immediately on `GetByID`/ownership error before calling `CreateRoom`.

### 2. Tenant registration does not verify room ownership

- Location: `internal/service/tenant_service.go:128-211`, `internal/repository/tenant_repository.go:23-72`
- Severity: High
- Issue: `RegisterTenant` checks room capacity by `roomID`, then inserts a tenant with the caller's `ManagerID` and updates the room status. It does not verify that the room belongs to that manager.
- Impact: A manager can attach tenants to another manager's room, consume its capacity, and change its status to `OCCUPIED`.
- Recommendation: Before capacity checks and insertion, verify the room's house belongs to `in.ManagerID` or add a repository operation that joins `rooms -> houses` and enforces ownership atomically.

### 3. Zalo transaction proof flow writes an invoice status not allowed by the DB

- Location: `internal/model/invoice.go:16-20`, `internal/service/zalo_webhook_handler.go:264-270`, `migrations/000008_create_invoices_table.up.sql`
- Severity: High
- Issue: The model uses `InvoiceStatusPendingVerification = "PENDING_VERIFICATION"`, but the `invoice_status` enum only allows `UNPAID`, `PARTIALLY_PAID`, and `PAID`.
- Impact: When a tenant posts a transaction image in a linked Zalo group, `processTransactionImage` will fail updating the invoice status in PostgreSQL, so the proof image path/status will not persist.
- Recommendation: Either add a migration that extends the enum with `PENDING_VERIFICATION`, or map transaction-proof state into an existing allowed status plus a separate verification field.

### 4. Payment webhooks acknowledge most processing failures as success

- Location: `internal/handler/payment_handler.go:173-180`
- Severity: High
- Issue: `handleWebhook` returns HTTP 200 for every `paymentService.HandleWebhook` error except `ErrPayOSVerifiedDataNil`. That includes DB write failures, invoice update failures, credential errors, and signature mismatch errors.
- Impact: A payment provider may stop retrying even though the invoice was not marked paid or the event was not recorded, causing permanent payment reconciliation gaps.
- Recommendation: Return non-2xx for retryable processing errors, return 400/401 for signature/credential problems, and only return success after the event is safely recorded or intentionally ignored.

### 5. Payment webhook processing marks invoices paid without validating paid amount/status

- Location: `internal/service/payment_service.go:199-213`, `internal/service/payment_service.go:225-237`, `internal/service/payos_provider.go:107-135`
- Severity: High
- Issue: A matched provider order reference is enough to mark the invoice `PAID`; the code does not compare `verifiedEvent.Amount` to the stored payment link amount or invoice total, and does not check the provider webhook success/payment code before processing.
- Impact: A valid but non-success/partial/mismatched payment event could incorrectly close an invoice as paid.
- Recommendation: Validate provider success semantics and require `verifiedEvent.Amount == paymentLink.Amount` (or an explicit accepted tolerance/business rule) before calling `SystemUpdateInvoiceStatusAndMethod`.

## Medium-priority findings

### 6. Active payment links can become stale when invoice amounts change

- Location: `internal/service/invoice_service.go:213-234`, `internal/service/payment_service.go:93-99`
- Severity: Medium
- Issue: `CreateInvoice` acts as an upsert and can change `TotalAmount`, but existing active payment links for that invoice are not cancelled or marked stale. Later Zalo invoice delivery reuses the old active link without checking whether `activeLink.Amount` still equals `invoice.TotalAmount`.
- Impact: Tenants may receive or pay an old PayOS amount after a manager edits fees, discounts, tenants, vehicles, room prices, or house defaults.
- Recommendation: Mark active links stale on invoice updates/recalculations, or have `CreatePaymentLinkForInvoice` refuse/recreate an active link when its amount differs from the current invoice.

### 7. Recalculation errors are swallowed after room/house updates

- Location: `internal/service/invoice_service.go:346-388`, callers in `internal/handler/room_handler.go:234-238` and `internal/handler/house_handler.go:240-244`
- Severity: Medium
- Issue: `RecalculateUnpaidInvoicesByRoom` logs per-invoice errors with `fmt.Printf` and always returns nil; `RecalculateUnpaidInvoicesByHouse` also suppresses room-level errors.
- Impact: Managers can update pricing and receive a successful response while unpaid invoices remain stale.
- Recommendation: Return an aggregated error or a structured partial-failure result, and surface that to the handler or background job monitoring.

### 8. House creation ignores failure to seed the default monthly cost

- Location: `internal/service/house_service.go:57-73`
- Severity: Medium
- Issue: After a house is created, `h.houseCostRepo.Create(ctx, cost)` is explicitly ignored.
- Impact: New houses may lack their initial cost row, causing revenue/profit summaries to be incomplete without any visible error.
- Recommendation: Either return the seed error, run it in an explicit best-effort path with logging/metric, or document and reconcile missing cost rows later.

### 9. Refresh-token rotation does not revoke the old refresh token

- Location: `internal/service/auth_service.go:257-305`
- Severity: Medium
- Issue: `RefreshToken` validates the old token and creates a new refresh token but does not revoke the old session.
- Impact: A stolen refresh token remains usable until expiry even after the legitimate client refreshes.
- Recommendation: Revoke the consumed refresh token as part of rotation, ideally in the same transaction as creating the new session.

### 10. Invoice and cost periods are not validated as `YYYY-MM`

- Location: `internal/handler/invoice_handler.go:25-35`, `internal/handler/house_cost_handler.go`, `internal/repository/invoice_repository.go:227-245`
- Severity: Medium
- Issue: Period fields are only `required`; repositories compare periods lexicographically to find previous invoices.
- Impact: Inputs like `06-2026` or `2026-6` can break previous-invoice detection, ordering, uniqueness expectations, and monthly revenue summaries.
- Recommendation: Add a strict `YYYY-MM` validation tag/helper for invoice and house-cost periods.

### 11. Public Zalo invoice image URLs are unsigned despite signed-route support

- Location: `internal/service/zalo_invoice_delivery.go:167-181`, `internal/router/upload_handler.go:13-36`
- Severity: Medium
- Issue: `saveZaloInvoiceImage` returns `/api/v1/uploads/zalo-invoices/{file}` without `expires`/`sig`. The upload handler only validates signatures when those query params are present, so generated invoice images are effectively public by unguessable filename.
- Impact: Anyone with a leaked URL can fetch invoice images indefinitely.
- Recommendation: Generate signed, expiring URLs with `security.SignPath`, and consider requiring signatures for this route if Zalo can fetch signed URLs reliably.

## Lower-priority findings and maintainability notes

### 12. Config has an OTP env-var typo and misleading error

- Location: `config/config.go:69`, `config/config.go:103-106`
- Severity: Low/Medium
- Issue: The variable is read as `OTP_EXPIRE_MINIUTES` and the validation error says `JWT_TTL_MINUTES must be a positive integer`.
- Impact: Deployments using the expected spelling (`OTP_EXPIRE_MINUTES`) will fail to start with a confusing message.
- Recommendation: Support the correctly spelled env var, keep backward compatibility for the typo if needed, and fix the error message.

### 13. API responses leak persistence models in several handlers

- Location examples: `internal/handler/house_handler.go`, `internal/handler/room_handler.go`, `internal/handler/invoice_handler.go`, `internal/handler/tenant_handler.go`
- Severity: Low/Medium
- Issue: Handlers commonly return `model.*` values directly instead of DTOs.
- Impact: Database shape and API contract are coupled, making future schema changes risky and increasing chance of exposing internal fields.
- Recommendation: Introduce response DTOs for externally visible objects, starting with invoices and tenant documents.

### 14. Error handling conventions are inconsistent

- Location examples: `internal/handler/house_cost_handler.go` string-matches errors; invoice handler compares `err.Error()`; room/house handlers use sentinels.
- Severity: Low/Medium
- Issue: Some handlers use `errors.Is`, while others parse error strings.
- Impact: Refactoring messages can accidentally change HTTP status behavior.
- Recommendation: Add package-level sentinel errors or typed domain errors for common cases (`forbidden`, `already exists`, `paid invoice`, etc.).

### 15. Some background/best-effort paths only print to stdout

- Location examples: `internal/service/invoice_service.go:370`, `internal/service/zalo_webhook_handler.go`
- Severity: Low
- Issue: Several business flows use `fmt.Printf` instead of the project logger/structured logging.
- Impact: Production observability is harder, especially for reconciliation and webhook failures.
- Recommendation: Route these through `internal/service/logger` or a shared structured logger with manager/house/room/invoice identifiers.

## Positive observations

- Tests are broad and currently pass across all packages.
- Repository queries generally use parameterized Bun APIs, which reduces SQL-injection risk.
- File download paths for tenant and transaction uploads use filename normalization and ownership checks.
- DPoP proof validation includes `htu`, `htm`, `iat`, `jti`, and `ath` checks for bound access tokens.
- Payment credentials and Zalo bot tokens are encrypted at rest with AES-GCM.
- The router mostly enforces role checks consistently for manager-only business routes.

## Recommended fix order

1. Fix cross-manager ownership gaps in room creation and tenant registration.
2. Align invoice status enum/model for Zalo transaction proof flow.
3. Harden payment webhook acknowledgement, success/status validation, and amount validation.
4. Stale/cancel payment links when invoice totals change.
5. Make recalculation and house-cost seeding failures visible.
6. Add strict period validation and clean up config/env typo.
7. Gradually introduce DTOs and consistent domain errors.
