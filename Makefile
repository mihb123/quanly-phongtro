.PHONY: build frontend mocks

# build: biên dịch frontend rồi nhúng vào API để ra một file thực thi duy nhất.
BINARY ?= quanly-phongtro-api
build: frontend
	rm -rf internal/web/dist
	cp -r frontend/dist internal/web/dist
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/api
	@echo "Built single binary: ./$(BINARY)"

# frontend: build bản tĩnh React (Vite) vào frontend/dist.
frontend:
	cd frontend && pnpm install --frozen-lockfile && pnpm build

mocks:
	@echo "Generating mocks for internal/model..."
	@mkdir -p internal/mock/mock_model
	mockgen -source=internal/model/email_verification.go -destination=internal/mock/mock_model/email_verification_mock.go -package=mock_model
	mockgen -source=internal/model/house.go -destination=internal/mock/mock_model/house_mock.go -package=mock_model
	mockgen -source=internal/model/invoice.go -destination=internal/mock/mock_model/invoice_mock.go -package=mock_model
	mockgen -source=internal/model/jwt.go -destination=internal/mock/mock_model/jwt_mock.go -package=mock_model
	mockgen -source=internal/model/room.go -destination=internal/mock/mock_model/room_mock.go -package=mock_model
	mockgen -source=internal/model/tenant.go -destination=internal/mock/mock_model/tenant_mock.go -package=mock_model
	mockgen -source=internal/model/user.go -destination=internal/mock/mock_model/user_mock.go -package=mock_model
	mockgen -source=internal/model/operating_cost.go -destination=internal/mock/mock_model/operating_cost_mock.go -package=mock_model
	@echo "Generating mocks for internal/service..."
	@mkdir -p internal/mock/mock_service
	mockgen -source=internal/service/auth/auth_service.go -destination=internal/mock/mock_service/auth_service_mock.go -package=mock_service
	mockgen -source=internal/service/house/house_service.go -destination=internal/mock/mock_service/house_service_mock.go -package=mock_service
	mockgen -source=internal/service/invoice/image_service.go -destination=internal/mock/mock_service/image_service_mock.go -package=mock_service
	mockgen -source=internal/service/invoice/invoice_service.go -destination=internal/mock/mock_service/invoice_service_mock.go -package=mock_service
	mockgen -source=internal/service/room/room_service.go -destination=internal/mock/mock_service/room_service_mock.go -package=mock_service
	mockgen -source=internal/service/tenant/tenant_service.go -destination=internal/mock/mock_service/tenant_service_mock.go -package=mock_service
	mockgen -source=internal/service/zalo/zalo_client.go -destination=internal/mock/mock_service/zalo_client_mock.go -package=mock_service
	mockgen -source=internal/service/zalo/zalo_service.go -destination=internal/mock/mock_service/zalo_service_mock.go -package=mock_service
	mockgen -source=internal/service/house/house_cost_service.go -destination=internal/mock/mock_service/house_cost_service_mock.go -package=mock_service
	mockgen -source=internal/service/revenue/event_bus.go -destination=internal/mock/mock_service/event_bus_mock.go -package=mock_service
	@echo "Mocks generated successfully!"
