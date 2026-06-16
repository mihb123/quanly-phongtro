# quanly-phongtro (Clean Architecture)

A full-stack property management application with a Go backend and a React frontend.

## Tính năng chính

- **Xác thực và phân quyền:** đăng ký, đăng nhập, refresh token, xác minh email bằng OTP, quản lý thông tin cá nhân và phân quyền theo vai trò manager/tenant.
- **Quản lý nhà trọ và phòng:** tạo, cập nhật, xóa nhà/phòng; cấu hình giá thuê, điện, nước, wifi, gửi xe, dịch vụ và các khoản phụ thu theo từng nhà hoặc từng phòng.
- **Quản lý khách thuê:** đăng ký khách thuê theo phòng, cập nhật hồ sơ, lưu thông tin CCCD/hợp đồng và tra cứu khách thuê theo phòng hoặc theo nhà.
- **Quản lý hóa đơn:** tạo hóa đơn tiền phòng hằng tháng, tính điện nước/dịch vụ/phụ thu/giảm giá, theo dõi trạng thái thanh toán, lưu ảnh giao dịch và xuất ảnh hóa đơn.
- **Doanh thu và chi phí vận hành:** ghi nhận chi phí theo kỳ, tổng hợp doanh thu, tổng chi và lợi nhuận theo từng nhà trọ. Xem thêm [house_cost_implementation_plan.md](house_cost_implementation_plan.md).
- **Tích hợp Zalo và PayOS:** cấu hình bot/webhook, gửi hóa đơn qua Zalo, tạo link/QR thanh toán PayOS và cập nhật trạng thái hóa đơn từ webhook. Xem thêm [payos-plan.md](payos-plan.md).
- **Bảo mật API:** JWT kết hợp DPoP proof, cookie/session handling, bcrypt password hashing và kiểm soát quyền truy cập trên các endpoint quản trị. Xem thêm [docs/security_dpop.md](docs/security_dpop.md) và [SECURITY_REVIEW.md](SECURITY_REVIEW.md).

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
