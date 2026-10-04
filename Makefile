.DEFAULT_GOAL := help

BINARY ?= quanly-phongtro-api
MIGRATIONS_DIR := migrations
GO_TEST_PKGS := ./internal/service/... ./internal/handler/... ./internal/repository/...

export PATH := $(shell go env GOPATH)/bin:$(PATH)

LOAD_ENV := set -a && . ./.env && set +a
MIGRATE := $(LOAD_ENV) && migrate -path $(MIGRATIONS_DIR) -database "$$POSTGRES_DSN"

.PHONY: help dev dev-fe run install build frontend clean \
	test test-fe vet fmt check mocks \
	migrate-up migrate-down migrate-create migrate-version migrate-force seed backup-db backup-db-ocl

help: ## Hiển thị danh sách lệnh
	@awk 'BEGIN {FS = ":.*?## "} \
		/^##@/ {printf "\n\033[1m%s\033[0m\n", substr($$0, 5); next} \
		/^[a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

##@ Phát triển

dev: ## Chạy backend với hot-reload (air)
	air

dev-fe: ## Chạy Vite dev server (frontend)
	cd frontend && pnpm dev

run: ## Chạy backend một lần (go run, không hot-reload)
	go run ./cmd/api

install: ## Cài dependency Go + frontend và các tool (mockgen, migrate, air)
	go mod download
	go install go.uber.org/mock/mockgen@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	go install github.com/air-verse/air@latest
	cd frontend && pnpm install --frozen-lockfile

##@ Build

build: frontend ## Build frontend rồi nhúng vào API thành một binary duy nhất
	rm -rf internal/web/dist
	cp -r frontend/dist internal/web/dist
	touch internal/web/dist/.gitkeep
	CGO_ENABLED=0 go build -o $(BINARY) ./cmd/api
	@echo "Built single binary: ./$(BINARY)"

frontend: ## Build bản tĩnh React (Vite) vào frontend/dist
	cd frontend && pnpm install --frozen-lockfile && pnpm build

clean: ## Xoá binary, tmp/ và bản build frontend
	rm -rf $(BINARY) tmp frontend/dist
	find internal/web/dist -mindepth 1 ! -name .gitkeep -exec rm -rf {} +

##@ Kiểm thử & chất lượng code

test: ## Chạy test backend (giống CI)
	go test -cover $(GO_TEST_PKGS)

test-fe: ## Lint + typecheck frontend
	cd frontend && pnpm test:frontend

vet: ## go vet toàn bộ backend
	go vet ./...

fmt: ## Format code Go (gofmt)
	gofmt -w cmd config internal

check: mocks vet test test-fe ## Chạy đủ bước kiểm tra như CI trước khi push

mocks: ## Sinh lại mock cho model và service (mockgen)
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

##@ Database

migrate-up: ## Áp dụng tất cả migration đang chờ
	$(MIGRATE) up

migrate-down: ## Rollback N migration gần nhất (mặc định N=1)
	$(MIGRATE) down $(or $(N),1)

migrate-create: ## Tạo cặp file migration mới: make migrate-create name=add_x_to_y
	@test -n "$(name)" || (echo "Thiếu tên: make migrate-create name=add_x_to_y" && exit 1)
	migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-version: ## Xem version migration hiện tại
	$(MIGRATE) version

migrate-force: ## Sửa trạng thái dirty: make migrate-force version=12
	@test -n "$(version)" || (echo "Thiếu version: make migrate-force version=12" && exit 1)
	$(MIGRATE) force $(version)

seed: ## Tạo dữ liệu mẫu
	go run ./cmd/seed

backup-db: ## Backup PostgreSQL vào backup/database (giữ 5 bản gần nhất)
	./scripts/backup-db.sh

backup-db-ocl: backup-db ## Backup PostgreSQL rồi chuyển bản mới nhất sang ocl:~/backup-quanly-phongtro
	./scripts/backup-db-offsite.sh
