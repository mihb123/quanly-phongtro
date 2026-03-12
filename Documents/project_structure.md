# Cấu trúc thư mục dự án

## Cây thư mục

```
quanly-phongtro/
├── main.go                         # Entry point: khởi tạo config, DB, server
├── go.mod / go.sum                 # Go module dependencies
├── .env.example                    # Mẫu biến môi trường
│
├── config/
│   └── config.go                   # Load cấu hình từ biến môi trường (.env)
│
├── src/                            # Source code của dự án
│   │
│   ├── Routes/
│   │   └── Routes.go               # Đăng ký toàn bộ routes, gắn middleware & controller
│   │
│   ├── Middleware/
│   │   └── Middleware.go           # CORS, JSONContentType, Auth...
│   │
│   ├── Request/
│   │   └── AuthRequest.go          # Request structs với binding và validation rules
│   │
│   ├── Controller/
│   │   ├── Controller.go           # Helpers: writeJSON, writeError, Home, HealthCheck
│   │   └── AuthController.go       # Xử lý HTTP /auth/register, /auth/login
│   │
│   ├── Service/
│   │   ├── AuthService.go          # Business logic: Register, Login, JWT signing
│   │   └── Errors.go               # Định nghĩa service-level errors
│   │
│   ├── Repository/
│   │   └── UserRepository.go       # SQL queries tương tác với bảng users (interface + impl)
│   │
│   ├── Model/
│   │   └── User.go                 # Domain model: User struct, UserRole enum
│   │
│   ├── Database/
│   │   └── Postgres.go             # Kết nối và ngắt kết nối PostgreSQL
│   │
│   └── Server/
│       └── Server.go               # Khởi tạo HTTP server, graceful shutdown
│
├── migrations/                     # SQL migration files
│
└── Documents/                      # Tài liệu đặc tả dự án
```

---

## Luồng xử lý request

```
HTTP Request
    → Routes        (định tuyến, gắn middleware)
    → Middleware    (CORS, Auth, ContentType...)
    → Request       (binding & validate input)
    → Controller    (parse request, gọi service, trả response)
    → Service       (business logic, không biết về HTTP)
    → Repository    (SQL queries, interface để dễ mock/test)
    → Database      (PostgreSQL)
```
