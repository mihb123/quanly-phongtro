# Task 1: Tạo Database và Dữ liệu mẫu (Test Data)

## Mục tiêu
- Khởi tạo cơ sở dữ liệu PostgreSQL theo cấu trúc đã định nghĩa trong `database_structure.md`.
- Viết script/file để tự động tạo dữ liệu mẫu (Test Data) dùng cho việc phát triển và kiểm thử.

## Công việc chi tiết
1. Viết script SQL hoặc thông qua Migration (GORM/Sequelize/Prisma, tùy Framework) để tạo các bảng (Users, Houses, Rooms, Tenants, Invoices).
2. Tạo script seed data chèn một số bản ghi mẫu:
   - 1-2 tài khoản Manager.
   - 2-3 nhà trọ cho mỗi Manager.
   - Một số phòng trọ (trạng thái AVAILABLE, OCCUPIED).
   - Tài khoản Tenant và dữ liệu liên kết phòng.
   - Hóa đơn mẫu cho các phòng.
