# quanly-phongtro (Clean Architecture)

A full-stack property management application with a Go backend and a React frontend.

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
│   ├── router/                   # API Routes configuration
│   ├── security/                 # JWT, bcrypt, encryption
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

### 2. Start the Backend (Go)

```bash
go run ./cmd/api
```
The API server will start on port `8080` (or whatever `APP_PORT` is set to).

### 3. Start the Frontend (React)

Open a new terminal window, navigate to the `frontend` directory, and start the Vite development server:

```bash
cd frontend
npm install
npm run dev
```

## APIs Summary

The backend exposes a RESTful API under the `/api/v1/` prefix. Key namespaces include:

- **`Auth`** (`/api/v1/auth/*`): Registration, login, token refresh, and email verification.
- **`House`** (`/api/v1/house/*`): CRUD operations for properties/houses (Managers only).
- **`Room`** (`/api/v1/room/*`): CRUD operations for individual rooms within a house, including custom pricing and surcharges.
- **`Tenant`** (`/api/v1/tenant/*`): Tenant registration and management per room/house.
- **`Invoice`** (`/api/v1/invoice/*`): Generation, tracking, image generation, and payment status of monthly rent/utility invoices.
- **`Zalo`** (`/api/v1/zalo/*`): Integration with Zalo mini-apps, bots, and webhook handling for notifications.
- **`Health`** (`/health`): Public health-check endpoint.

*Note: All endpoints (except public auth & health) require a valid JWT Bearer token and appropriate role permissions.*
