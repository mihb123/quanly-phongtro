#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 2] Lint & Test"

go mod download
go install go.uber.org/mock/mockgen@latest
make mocks
go vet ./...
go test ./...
