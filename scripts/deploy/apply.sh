#!/usr/bin/env bash
set -euo pipefail

source "$(dirname "$0")/common.sh"
start_deploy_step "[Phase 6] Deploy"
require_env SERVICE_NAME
require_env DEPLOY_USER
require_env DEPLOY_GROUP
require_env FRONTEND_DIR
require_env NGINX_SITE_NAME
require_env NGINX_AVAILABLE_DIR
require_env NGINX_ENABLED_DIR

INSTALL_CMD="$(find_command install)"
MKDIR_CMD="$(find_command mkdir)"
CP_CMD="$(find_command cp)"
SYSTEMCTL_CMD="$(find_command systemctl)"
NGINX_CMD="$(find_command nginx)"
SUDOERS_FILE="/etc/sudoers.d/quanly-phongtro-deploy"
SUDOERS_COMMANDS="$INSTALL_CMD, $MKDIR_CMD, $CP_CMD, $SYSTEMCTL_CMD, $NGINX_CMD"

# check_root_access verifies sudo before replacing deploy artifacts.
check_root_access() {
  check_sudo_access "$INSTALL_CMD" --version
  check_sudo_access "$MKDIR_CMD" --version
  check_sudo_access "$CP_CMD" --version
  check_sudo_access "$SYSTEMCTL_CMD" --version
  check_sudo_access "$NGINX_CMD" -v
}

# activate_backend_binary promotes the freshly built backend binary.
activate_backend_binary() {
  if [[ -f "${SERVICE_NAME}.new" ]]; then
    mv "${SERVICE_NAME}.new" "$SERVICE_NAME"
  elif [[ -f "$SERVICE_NAME" ]]; then
    echo "Khong tim thay ${SERVICE_NAME}.new, dung binary hien co"
  else
    echo "Khong tim thay binary de deploy"
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

# install_nginx_config renders the configured frontend root into nginx.
install_nginx_config() {
  local nginx_config
  local app_url="quanly-phongtro.mvpc.site"

  if [[ -f "$PROJECT_DIR/.env" ]]; then
    local env_app_url
    env_app_url=$(grep -E '^APP_URL=' "$PROJECT_DIR/.env" | cut -d '=' -f 2- | tr -d '"'\'' ' || true)
    if [[ -n "$env_app_url" ]]; then
      app_url="$env_app_url"
    fi
  fi

  nginx_config="$(mktemp)"
  sed -E \
    -e "s#^[[:space:]]*root[[:space:]]+[^;]+;#    root $(printf '%s' "$FRONTEND_DIR" | sed 's/[#&]/\\&/g');#" \
    -e "s#^[[:space:]]*server_name[[:space:]]+[^;]+;#    server_name $(printf '%s' "$app_url" | sed 's/[#&]/\\&/g');#" \
    "$PROJECT_DIR/nginx.conf" > "$nginx_config"

  run_sudo "$MKDIR_CMD" -p "$NGINX_AVAILABLE_DIR" "$NGINX_ENABLED_DIR"
  run_sudo "$INSTALL_CMD" -m 0644 "$nginx_config" "$NGINX_AVAILABLE_DIR/$NGINX_SITE_NAME"
  run_sudo "$INSTALL_CMD" -m 0644 "$nginx_config" "$NGINX_ENABLED_DIR/$NGINX_SITE_NAME"
  rm -f "$nginx_config"
  run_sudo "$NGINX_CMD" -t
}

# publish_frontend replaces the served frontend files with the latest build.
publish_frontend() {
  run_sudo "$MKDIR_CMD" -p "$FRONTEND_DIR"
  run_sudo "$CP_CMD" -aT "$PROJECT_DIR/frontend/dist" "$FRONTEND_DIR"
}

check_root_access
activate_backend_binary
install_systemd_unit
restart_backend_service
install_nginx_config
publish_frontend
run_sudo "$SYSTEMCTL_CMD" reload nginx
