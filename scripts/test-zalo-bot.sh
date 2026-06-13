#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "Running focused Zalo bot tests..."
go test ./internal/service ./internal/repository ./internal/router -run 'TestZalo|TestHandleWebhook|TestSendInvoiceToZalo|TestProcessTransactionImage|TestSendTextMessage|TestIsZaloAuthError|TestNewZaloService|TestUserRepository|TestRouter'

echo "Running full backend test suite..."
go test ./...

if [[ "${RUN_AIR:-0}" != "1" ]]; then
	echo "Skipping air smoke test. Set RUN_AIR=1 to start a temporary server."
	exit 0
fi

if ! command -v air >/dev/null 2>&1; then
	echo "air is not installed or not on PATH" >&2
	exit 1
fi

APP_PORT="${APP_PORT:-18080}"
AIR_LOG="tmp/zalo-air-smoke.log"
mkdir -p tmp

echo "Starting air smoke server on port ${APP_PORT}..."
APP_PORT="$APP_PORT" APP_ENV=dev APP_URL_DEV="localhost:${APP_PORT}" air -c .air.toml >"$AIR_LOG" 2>&1 &
AIR_PID=$!

# cleanup stops the temporary air process started by this script.
cleanup() {
	if kill -0 "$AIR_PID" >/dev/null 2>&1; then
		kill "$AIR_PID" >/dev/null 2>&1 || true
		wait "$AIR_PID" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT

for _ in $(seq 1 30); do
	if curl -fsS "http://localhost:${APP_PORT}/health" >/dev/null 2>&1; then
		echo "air smoke server is healthy."
		exit 0
	fi
	sleep 1
done

echo "air smoke server did not become healthy. Last log lines:" >&2
tail -80 "$AIR_LOG" >&2 || true
exit 1
