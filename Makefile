.PHONY: mocks

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
	@echo "Generating mocks for internal/service..."
	@mkdir -p internal/mock/mock_service
	mockgen -source=internal/service/auth_service.go -destination=internal/mock/mock_service/auth_service_mock.go -package=mock_service
	mockgen -source=internal/service/house_service.go -destination=internal/mock/mock_service/house_service_mock.go -package=mock_service
	mockgen -source=internal/service/image_service.go -destination=internal/mock/mock_service/image_service_mock.go -package=mock_service
	mockgen -source=internal/service/invoice_service.go -destination=internal/mock/mock_service/invoice_service_mock.go -package=mock_service
	mockgen -source=internal/service/room_service.go -destination=internal/mock/mock_service/room_service_mock.go -package=mock_service
	mockgen -source=internal/service/tenant_service.go -destination=internal/mock/mock_service/tenant_service_mock.go -package=mock_service
	mockgen -source=internal/service/zalo_client.go -destination=internal/mock/mock_service/zalo_client_mock.go -package=mock_service
	mockgen -source=internal/service/zalo_service.go -destination=internal/mock/mock_service/zalo_service_mock.go -package=mock_service
	@echo "Mocks generated successfully!"
