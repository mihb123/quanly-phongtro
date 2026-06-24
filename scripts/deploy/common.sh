#!/usr/bin/env bash

# require_env stops a deploy phase when a required environment variable is missing.
require_env() {
  local name="$1"

  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: $name"
    return 1
  fi
}

# start_deploy_step attaches deploy notifications and enters the project directory.
start_deploy_step() {
  local step_name="$1"

  require_env PROJECT_DIR
  source "$PROJECT_DIR/scripts/setup-trap.sh" "$step_name"
  cd "$PROJECT_DIR"
}

# find_command resolves command paths for sudoers rules on each server.
find_command() {
  local command_name="$1"
  local command_path

  command_path="$(command -v "$command_name" 2>/dev/null || true)"
  if [[ -n "$command_path" ]]; then
    printf '%s\n' "$command_path"
    return 0
  fi

  for command_path in "/usr/bin/$command_name" "/usr/sbin/$command_name" "/bin/$command_name" "/sbin/$command_name"; do
    if [[ -x "$command_path" ]]; then
      printf '%s\n' "$command_path"
      return 0
    fi
  done

  echo "Command not found on server: $command_name"
  return 1
}

# print_sudo_fix shows the one-time server setup needed for CI sudo.
print_sudo_fix() {
  local current_user

  require_env SUDOERS_FILE
  require_env SUDOERS_COMMANDS

  current_user="$(id -un)"
  cat <<EOF
Sudo cannot run in GitHub Actions for user '$current_user'.
Run once on the server with a sudo user:
  sudo visudo
Add the following line:
  $current_user ALL=(root) NOPASSWD: $SUDOERS_COMMANDS
Then check again:
  sudo -l -U $current_user
EOF
}

# is_sudo_auth_error detects sudo policy or authentication failures.
is_sudo_auth_error() {
  grep -qiE 'terminal is required|password is required|no tty present|not allowed|may not run sudo|authenticate'
}

# check_sudo_access fails early before deploy files are changed.
check_sudo_access() {
  local output
  local status

  if output="$(sudo -n "$@" 2>&1 >/dev/null)"; then
    return 0
  else
    status=$?
  fi

  printf '%s\n' "$output"
  if printf '%s\n' "$output" | is_sudo_auth_error; then
    print_sudo_fix
  fi
  return "$status"
}

# run_sudo runs root commands without interactive password prompts.
run_sudo() {
  local output
  local status

  if output="$(sudo -n "$@" 2>&1)"; then
    if [[ -n "$output" ]]; then
      printf '%s\n' "$output"
    fi
    return 0
  else
    status=$?
    if [[ -n "$output" ]]; then
      printf '%s\n' "$output"
    fi
  fi

  if printf '%s\n' "$output" | is_sudo_auth_error; then
    print_sudo_fix
  fi
  return "$status"
}

# sed_escape escapes replacement text for sed substitutions.
sed_escape() {
  printf '%s' "$1" | sed 's/[\/&]/\\&/g'
}
