#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 6] Deploy"
require_env SERVICE_NAME
require_env DEPLOY_USER
require_env DEPLOY_GROUP

INSTALL_CMD="$(find_command install)"
SYSTEMCTL_CMD="$(find_command systemctl)"
SUDOERS_FILE="/etc/sudoers.d/quanly-phongtro-deploy"
SUDOERS_COMMANDS="$INSTALL_CMD, $SYSTEMCTL_CMD"

# check_root_access verifies sudo before replacing deploy artifacts.
check_root_access() {
  check_sudo_access "$INSTALL_CMD" --version
  check_sudo_access "$SYSTEMCTL_CMD" --version
}

# activate_backend_binary promotes the freshly built backend binary.
activate_backend_binary() {
  if [[ -f "${SERVICE_NAME}.new" ]]; then
    mv "${SERVICE_NAME}.new" "$SERVICE_NAME"
  elif [[ -f "$SERVICE_NAME" ]]; then
    echo "Could not find ${SERVICE_NAME}.new, using existing binary"
  else
    echo "Could not find binary to deploy"
    return 1
  fi

  chmod +x "$SERVICE_NAME"
}

# install_systemd_unit renders and installs the backend service definition.
install_systemd_unit() {
  local service_template
  local service_unit

  service_template="$PROJECT_DIR/.github/systemd/${SERVICE_NAME}.service"
  service_unit="$(mktemp)"
  sed \
    -e "s/__DEPLOY_USER__/$(sed_escape "$DEPLOY_USER")/g" \
    -e "s/__DEPLOY_GROUP__/$(sed_escape "$DEPLOY_GROUP")/g" \
    -e "s/__PROJECT_DIR__/$(sed_escape "$PROJECT_DIR")/g" \
    "$service_template" > "$service_unit"

  run_sudo "$INSTALL_CMD" -m 0644 "$service_unit" "/etc/systemd/system/${SERVICE_NAME}.service"
  rm -f "$service_unit"
}

# restart_backend_service reloads systemd and restarts the backend service.
restart_backend_service() {
  run_sudo "$SYSTEMCTL_CMD" daemon-reload
  run_sudo "$SYSTEMCTL_CMD" enable "$SERVICE_NAME"
  run_sudo "$SYSTEMCTL_CMD" restart "$SERVICE_NAME"
  sleep 2
  run_sudo "$SYSTEMCTL_CMD" status "$SERVICE_NAME" --no-pager
}

check_root_access
activate_backend_binary
install_systemd_unit
restart_backend_service
