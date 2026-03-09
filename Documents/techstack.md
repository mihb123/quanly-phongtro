# Tech Stack

Dự án sử dụng các công nghệ hiện đại nhằm đảm bảo hiệu năng cao, dễ dàng bảo trì và mở rộng trong tương lai.

## 1. Backend
- **Ngôn ngữ/Framework**: Go (Golang) - Lựa chọn hàng đầu cho các hệ thống đòi hỏi hiệu năng cao và xử lý đồng thời tốt. Có thể sử dụng framework/router như Gin, Echo hoặc Standard Library `net/http`.
- **ORM / Database Driver**: GORM hoặc sqlc, pgx (tùy thuộc vào design pattern cụ thể muốn hướng tới).
- **Authentication**: JWT (JSON Web Tokens) cho cơ chế Stateless Authentication.
- **Quản lý biến môi trường**: Viper hoặc qua file `.env`.

## 2. Frontend
- **Thư viện chính**: ReactJS.
- **Styling**: TailwindCSS hoặc Vanilla CSS tùy chọn.
- **State Management**: Redux Toolkit, Zustand hoặc React Context API.
- **Routing**: React Router DOM.
- **HTTP Client**: Axios hoặc Fetch API.

## 3. Database
- **Hệ quản trị CSDL**: PostgreSQL. Đây là hệ quản trị RDBMS mạnh mẽ, hỗ trợ tốt các ràng buộc khóa ngoại, tính toán transaction và toàn vẹn dữ liệu cho hóa đơn.

## 4. Công cụ hỗ trợ và Tooling
- **Version Control**: Git / GitHub.
- **Đóng gói dự án**: Docker / Docker Compose (Thiết lập database local dễ dàng).
- **Thiết kế API**: Swagger / OpenAPI.
