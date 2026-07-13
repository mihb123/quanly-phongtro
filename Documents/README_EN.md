# quanly-phongtro

## Live Demo
- **URL:** https://quanly.ptro.site
- **Email:** `user@test.com`
- **Pass:** `test01234`

<p align="center">
  <img src="../assets/guide.jpg" width="28%" />&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
  <img src="../assets/mobile-1.jpg" width="28%" />&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
  <img src="../assets/mobile-2.jpg" width="28%" />
  <br /><br />
  <img src="../assets/mobile-3.jpg" width="28%" />&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
  <img src="../assets/mobile-4.jpg" width="28%" />&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
  <img src="../assets/mobile-5.jpg" width="28%" />
</p>

A full-stack property management application with a Go backend and a React frontend.

In production the React frontend is built and **embedded into the Go binary**, so the whole app ships and runs as a **single executable** that serves both the API and the UI from the same origin (no separate web server needed). During development the frontend and backend still run as two separate dev servers.

## Main Features

- **Authentication & Security:** Registration, login, refresh tokens, email verification, and RBAC permissions (manager/tenant). Secured via JWT + DPoP. See [security_dpop](feature/security_dpop.md).
- **House & Room Management:** CRUD for houses and rooms, flexible configuration of rent prices, electricity, water, Wi-Fi, parking spaces, and custom services/surcharges at both house and room levels.
- **Tenant Management:** Tenant registration by room, profile updates, saving ID cards (CCCD)/contracts, and searching by room or house.
- **Invoicing & Payments:** Automatically generate monthly invoices (utilities, services, surcharges, discounts), track payment status, save transaction images, and render invoice images.
- **Revenue & Expenses:** Record recurring expenses, aggregate revenue/expenses/profits per house via event-driven background workers (`EventBus` + `RevenueWorker`).
- **Zalo & PayOS Integration:** Create PayOS payment QR/links, update status via webhooks; Zalo bot sends invoices, creates/updates invoices via chat, and cron checks token health. See [zalo_bot_feature](feature/zalo_bot_feature.md), [zalo_automation](feature/zalo_automation_mark_paid_invoice.md).
- **SePay Integration:** Dynamic QR code for invoices, automatically mark as `PAID` via webhook or manual reconciliation via SePay API v2, and log all transactions to `payment_events`. See [sepay_integration](feature/sepay_integration.md).

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
