# Source Code Review Report

Scope: full repository source review, with extra focus on current staged, unstaged, and untracked changes.

## 1. Logical Issues & Optimizations

### 🚨 Nghiêm trọng (High Severity)
* **[Backend]** File: `internal/handler/auth_handler.go`
  * [line 212] `GetMe` writes an `invalid token` response when `claims.GetSubject()` fails but does not return. Execution continues into `h.service.GetMe(...)` with the invalid/empty `userID`, which can produce a second response and misleading logs/status.
  * *Suggested Fix:* Return immediately after `writeError(w, http.StatusInternalServerError, "invalid token")`.
* **[Backend]** File: `internal/service/revenue_worker.go`
  * [lines 73-77] The worker continues and upserts a summary even when `CalculateRevenue` fails. That can overwrite a valid existing summary with `total_revenue = 0`, `profit = -cost`, making revenue reports wrong after a transient database error.
  * *Suggested Fix:* Treat revenue calculation failure as fatal for that event and return without upserting. Retry or re-enqueue if stale summaries are unacceptable.

### ⚠️ Cảnh báo (Medium Severity)
* **[Backend]** File: `internal/service/house_service.go`
  * [line 70] `CreateHouse` ignores `houseCostRepo.Create` errors while seeding the current month cost. A house can be created successfully but lack its initial cost record, so revenue/cost screens later show missing or inconsistent data with no error visible to the caller.
  * *Suggested Fix:* Return a wrapped error if seeding is part of the create-house invariant. If seeding is best-effort, log the error and add a reconciliation path that creates the missing cost row on first read.
* **[Backend]** File: `internal/service/house_cost_service.go`
  * [lines 185-190] `GetRevenueSummaries` checks ownership with one repository call per house ID and silently ignores ownership-check errors. With many selected houses, this becomes N+1 database work and can hide partial authorization/data-access failures.
  * *Suggested Fix:* Add a repository method that returns owned IDs in one query for `(managerID, houseIDs)`, and return/log unexpected repository errors instead of swallowing them.
* **[Frontend]** File: `frontend/src/components/home/modals/TenantRoomModal.tsx`
  * [lines 49-52, 67-70] `setHasChanged(true)` is followed immediately by `handleClose()`. React state updates are asynchronous, so `handleClose` can still read `hasChanged === false`, causing `TenantView` [line 228] to skip `fetchTenants(house.id)` after add/edit flows opened directly from a tenant row or add-room flow.
  * *Suggested Fix:* Pass the changed flag explicitly, e.g. `onClose(true)`, or change `handleClose` to accept `changed?: boolean` and call `onClose(changed || hasChanged)`.
* **[Frontend]** File: `frontend/src/components/home/modals/TenantListModal.tsx`
  * [lines 33-37] The delete flow has the same state timing risk when the last tenant is deleted: `onDataChange()` updates parent state, then `onClose()` can close before `TenantRoomModal` sees `hasChanged === true`.
  * *Suggested Fix:* Let `TenantListModal` call an explicit close callback with `changed=true`, or have `TenantRoomModal` provide an `onDeletedAndClose` handler that calls `onClose(true)` directly.
* **[Frontend]** File: `frontend/src/data/houseCostData.tsx`
  * [lines 82-83] `fetchSummaries` stores only rows returned by `/revenue-summary`. Houses without a summary row are omitted from totals until the user creates/edits a cost and the client inserts a synthetic summary locally [lines 108-117]. This makes dashboard/revenue totals depend on UI history rather than server state.
  * *Suggested Fix:* Have the backend return a zero summary for every requested owned house/period, or normalize missing rows in the store immediately after `getRevenueSummaries`.
* **[Frontend]** File: `frontend/src/components/home/DashboardView.tsx`
  * [lines 51-55] Dashboard loads rooms for every house and fires tenant fetches for every house on each house-list load. This creates a request burst and the tenant fetches are not awaited, so loading state can finish before tenant counts are actually current.
  * *Suggested Fix:* Add aggregate dashboard endpoints or store-level batch loading. If the current API remains, await tenant fetches and avoid refetching cached house data unless stale.

### 💡 Cải thiện (Low Severity / Optimization)
* **[Backend]** File: `internal/service/revenue_worker.go`
  * [lines 81-85] Missing-cost detection depends on `err.Error() == "house cost not found"`. This is fragile and will break if the repository wraps the error or changes the message.
  * *Suggested Fix:* Add a sentinel error such as `model.ErrHouseCostNotFound` and use `errors.Is`.
* **[Backend]** File: `internal/handler/house_cost_handler.go`
  * [lines 45-50, 75-80] Handler status mapping is based on `strings.Contains(err.Error(), ...)`, and one branch returns raw internal error text to the client. This couples API behavior to message wording and can leak database/service details.
  * *Suggested Fix:* Return typed/sentinel errors from the service and map them with `errors.Is`; keep client messages stable and generic for unexpected errors.

## 2. Validation

* `pnpm lint` in `frontend/`: passed.
* `go test ./cmd/... ./config ./internal/...`: passed.
* `go vet ./cmd/... ./config ./internal/...`: passed.
* `go test ./...` and `go vet ./...`: blocked by root-level helper programs (`test_db.go`, `test_bun_query.go`, `test_zalo.go`) that all declare `main`; `test_bun_query.go` also uses stale Bun APIs. These files should be moved under `cmd/`, renamed with build tags, or removed from package root.
* Build command skipped because project instructions say not to run build unless explicitly requested.

## 3. Suggested Commit Message

```text
fix(review): address revenue and tenant refresh issues

- prevent stale revenue summaries after calculation failures
- make tenant modal close paths pass changed state explicitly
- remove hardcoded local credentials and isolate helper programs
```

## 4. Validation Status

**Status: Validated**
I have reviewed the codebase and confirmed that the issues listed above are accurate and present in the code (with the exception of the root-level test scripts like `test_zalo.go` and `test_db.go`, which appear to have already been addressed or removed from the root directory). The identified logical issues and suggested fixes are appropriate and ready for implementation.
