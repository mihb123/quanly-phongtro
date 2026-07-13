# quanly-phongtro

A full-stack property management application with a Go backend and a React frontend.

In production the React frontend is built and **embedded into the Go binary**, so the whole app ships and runs as a **single executable** that serves both the API and the UI from the same origin (no separate web server needed). During development the frontend and backend still run as two separate dev servers.

## Main Features

- **Authentication & Security:** Registration, login, refresh tokens, OTP-based email verification, and profile management with RBAC (manager/tenant roles). Secured via JWT with DPoP proofs, secure HTTP-only cookies, bcrypt password hashing, and AES-256 encryption for application secrets and bot tokens. Includes GeoIP and Geocoding integration for tracking. See [Documents/feature/security_dpop.md](Documents/feature/security_dpop.md).
- **House & Room Management:** Complete CRUD operations for houses and rooms. Flexible configuration for rent prices, electricity, water, Wi-Fi, parking, and custom services/surcharges at both house and room levels.
- **Tenant Management:** Room-based tenant registration, profile updates, storage of ID cards (CCCD) and contracts, along with comprehensive tenant search by room or house.
- **Invoice & Payment Processing:** Automated monthly room invoicing encompassing utilities, services, surcharges, and discounts. Features include payment status tracking, transaction image storage, and dynamic invoice image generation.
- **Revenue & Operating Costs:** Periodic expense recording and automated asynchronous aggregation of revenue, expenses, and profits per house using event-driven background workers (`EventBus` + `RevenueWorker`).
- **Zalo & PayOS Integrations:** Automated payment links/QR code generation via PayOS with webhook-driven status updates. Deep Zalo integration featuring bot configurations, webhook handling, sending invoices directly via Zalo, Zalo bot commands (creating and updating invoices via chat), and a background cron service for active token health checks. See [Documents/feature/zalo_bot_feature.md](Documents/feature/zalo_bot_feature.md) and [Documents/feature/zalo_automation_mark_paid_invoice.md](Documents/feature/zalo_automation_mark_paid_invoice.md).
- **SePay Integration:** A bank-transfer payment provider built on the adapter architecture (SePay is preferred over PayOS). Managers configure their receiving account per manager (AES-256 encrypted at rest, secrets transported via RSA), generate dynamic transfer QR codes for invoices, and have invoices marked `PAID` automatically via the SePay webhook (API Key or HMAC authenticated) or through manual reconciliation against the SePay User API v2. Every transaction is written to `payment_events` for auditing. See [Documents/feature/sepay_integration.md](Documents/feature/sepay_integration.md).

## Project Structure

This is a monorepo containing both the backend and frontend:

```text
├── cmd/
│   ├── api/main.go               # Go Backend entrypoint
│   └── seed/main.go              # Database seeder
├── internal/                     # Go Backend (Clean Architecture)
│   ├── model/                    # Entities and Interfaces
│   ├── handler/                  # HTTP Handlers (chi router)
│   ├── service/                  # Business Logic & Rules
│   ├── repository/               # Data Access (Bun ORM)
│   ├── db/                       # Database connection setup
│   ├── router/                   # API Routes configuration + SPA fallback
│   ├── security/                 # JWT, bcrypt, encryption
│   ├── assets/                   # Embedded assets (fonts, GeoIP DB)
│   ├── web/                      # Embedded frontend build (dist) served by the API
│   └── mock/                     # Mocks for unit testing
├── migrations/                   # SQL Schema migrations (golang-migrate)
└── frontend/                     # React Frontend
    ├── src/components/           # Reusable UI components (shadcn)
    ├── src/pages/                # Application views
    ├── src/router/               # React Router configuration
    └── src/data/                 # API integration and State (Zustand)
```

## Tech Stack

### Backend
- **Language:** Go 1.26
- **Framework:** chi/v5 (Router)
- **Database:** PostgreSQL with Bun ORM
- **Migrations:** golang-migrate
- **Validation:** go-playground/validator
- **Security:** bcrypt + JWT

### Frontend
- **Framework:** React 19 + Vite
- **Styling:** Tailwind CSS v4 + Base UI
- **State Management:** Zustand
- **Forms:** React Hook Form + Zod
- **Routing:** React Router DOM v7

## Environment Setup

Use `.env.example` as a reference to create your `.env` file in the root directory:
```bash
cp .env.example .env
```

## Running the Application

### 1. Database Migrations

The project uses `golang-migrate` to manage the PostgreSQL database schema.

Apply all pending migrations:
```bash
# Load environment variables from .env
set -a && source .env && set +a

# Run up migrations
migrate -path migrations -database "$POSTGRES_DSN" up
```

*(Optional)* Seed test data:
```bash
go run ./cmd/seed
```

### 2. Development (backend + frontend run separately)

Start the backend:
```bash
go run ./cmd/api
```
The API server will start on port `8080` (or whatever `APP_PORT` is set to).

Open a new terminal and start the Vite dev server (it proxies `/api` to the backend automatically):
```bash
cd frontend
pnpm install
pnpm dev
```
The frontend is served at `http://localhost:5173`.

> In dev mode the API serves **only** the API — the embedded frontend (`internal/web/dist`) is just a placeholder. The UI comes from the Vite dev server.

### 3. Production (single binary)

Build the frontend, embed it into the backend, and produce one self-contained executable:
```bash
make build
```
This runs `pnpm build`, copies `frontend/dist` into `internal/web/dist`, then `go build`. The resulting `quanly-phongtro-api` binary embeds the React app **and** the GeoIP database, and serves the UI plus the API on `APP_PORT`:
```bash
./quanly-phongtro-api
```
Open `http://localhost:8080` — the React UI is served directly; `/api/v1/*` is the API on the same origin.

> The binary is self-contained for code, frontend, and GeoIP. At runtime it still needs: a reachable **PostgreSQL**, the **`.env`** config file next to it, and an **`uploads/`** directory for stored files.

## APIs Summary

The backend exposes a RESTful API under the `/api/v1/` prefix. Key namespaces include:

- **`Auth`** (`/api/v1/auth/*`): Registration, login, token refresh, and email verification.
- **`House`** (`/api/v1/house/*`): CRUD operations for properties/houses (Managers only).
- **`Room`** (`/api/v1/room/*`): CRUD operations for individual rooms within a house, including custom pricing and surcharges.
- **`Tenant`** (`/api/v1/tenant/*`): Tenant registration and management per room/house.
- **`Invoice`** (`/api/v1/invoice/*`): Generation, tracking, image generation, and payment status of monthly rent/utility invoices.
- **`Zalo`** (`/api/v1/zalo/*`): Integration with Zalo mini-apps, bots, and webhook handling for notifications.
- **`Payments`** (`/api/v1/payments/*`): SePay provider configuration, payment QR/link generation, transfer webhook handling, and transaction reconciliation via the SePay API v2.
- **`Health`** (`/health`): Public health-check endpoint.

*Note: All endpoints (except public auth & health) require a valid JWT Bearer token and appropriate role permissions.*
