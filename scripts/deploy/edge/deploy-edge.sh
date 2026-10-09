#!/usr/bin/env bash
set -euo pipefail

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TUNNEL_USER="${TUNNEL_USER:-qlpt-tunnel}"
TUNNEL_KEY="${TUNNEL_KEY:-$HOME/.ssh/qlpt_tunnel_ed25519}"
CONF_DIR="$HOME/.config/qlpt-edge"
UNIT_DIR="$HOME/.config/systemd/user"
DNS_WAIT_SECONDS="${DNS_WAIT_SECONDS:-600}"

usage() {
  cat <<'EOF'
Triển khai edge cho quanly-phongtro: nginx trên máy chủ ngoài + tunnel SSH ngược từ máy này.

  deploy-edge.sh install <ssh-host>            cài/cập nhật edge và bật tunnel tới nó
  deploy-edge.sh cert <ssh-host> <domain>...   chờ DNS trỏ về edge, xin chứng chỉ, bật HTTPS
  deploy-edge.sh status                        tunnel đang chạy và DNS từng domain trỏ về đâu
  deploy-edge.sh remove <ssh-host>             gỡ edge khỏi máy chủ và tắt tunnel

<ssh-host> là alias trong ~/.ssh/config (vd: ocl); cần root hoặc sudo không mật khẩu, Debian/Ubuntu.
Domain -> cổng app lấy từ sites.conf (cạnh script này).
Chuyển một domain sang edge: install, tạo bản ghi A <domain> -> IP edge (DNS only), rồi cert.
EOF
}

die() { printf 'LỖI: %s\n' "$*" >&2; exit 1; }
sites() { awk 'NF == 2 && $1 !~ /^#/ { printf "%s:%s ", $1, $2 }' "$DIR/sites.conf"; }
site_ports() { for s in $(sites); do echo "${s##*:}"; done | sort -u; }
site_domains() { for s in $(sites); do echo "${s%%:*}"; done; }

edge_ip() {
  local name
  name="$(ssh -G "$1" | awk '$1 == "hostname" { print $2 }')"
  getent ahostsv4 "$name" | awk 'NR == 1 { print $1 }'
}

dns_a() { dig +short A "$1" @1.1.1.1 2>/dev/null | grep -E '^[0-9.]+$' | sort | tr '\n' ' ' | sed 's/ $//'; }

remote() {
  local host="$1"
  shift
  local overrides=()
  [[ -n "${EDGE_TLS_LISTEN:-}" ]] && overrides+=("EDGE_TLS_LISTEN=$EDGE_TLS_LISTEN")
  [[ -n "${EDGE_CLIENT_IP:-}" ]] && overrides+=("EDGE_CLIENT_IP=$EDGE_CLIENT_IP")
  { for kv in "$@" "${overrides[@]}"; do printf 'export %q\n' "$kv"; done; cat "$DIR/remote-setup.sh"; } |
    ssh -T -o BatchMode=yes "$host" 'if [ "$(id -u)" -eq 0 ]; then bash -s; else sudo -n bash -s; fi'
}

install_tunnel() {
  local host="$1" ip="$2" forwards=""
  for p in $(site_ports); do forwards+="-R 127.0.0.1:$p:127.0.0.1:$p "; done
  mkdir -p "$CONF_DIR" "$UNIT_DIR"
  install -m 644 "$DIR/qlpt-edge-tunnel@.service" "$UNIT_DIR/"
  printf 'EDGE_HOST=%s\nTUNNEL_USER=%s\nTUNNEL_KEY=%s\nFORWARDS="%s"\n' \
    "$ip" "$TUNNEL_USER" "$TUNNEL_KEY" "${forwards% }" >"$CONF_DIR/$host.env"
  systemctl --user daemon-reload
  systemctl --user enable "qlpt-edge-tunnel@$host" >/dev/null 2>&1
  systemctl --user restart "qlpt-edge-tunnel@$host"
}

verify_tunnel() {
  local host="$1" listening=""
  for _ in $(seq 1 20); do
    listening="$(ssh -o BatchMode=yes "$host" 'ss -ltnH' | awk '{ print $4 }')"
    local missing=0
    for p in $(site_ports); do grep -qx "127.0.0.1:$p" <<<"$listening" || missing=1; done
    ((missing == 0)) && break
    sleep 1
  done
  for p in $(site_ports); do
    grep -qx "127.0.0.1:$p" <<<"$listening" || die "tunnel chưa mở 127.0.0.1:$p trên $host (xem: journalctl --user -u qlpt-edge-tunnel@$host)"
    local code
    code="$(ssh -o BatchMode=yes "$host" "curl -s -o /dev/null -w '%{http_code}' --max-time 10 http://127.0.0.1:$p/health" || true)"
    printf '  tunnel 127.0.0.1:%s -> app: /health %s%s\n' "$p" "$code" "$([[ "$code" == 200 ]] || echo ' (app ở cổng này chưa chạy?)')"
  done
}

cmd_install() {
  local host="${1:?thiếu <ssh-host>}" ip
  ip="$(edge_ip "$host")"
  [[ -n "$ip" ]] || die "không phân giải được IP của $host"
  [[ -f "$TUNNEL_KEY" ]] || ssh-keygen -q -t ed25519 -N '' -C "qlpt-tunnel@$(hostname)" -f "$TUNNEL_KEY"
  echo "== Cài edge trên $host ($ip)"
  remote "$host" ACTION=install "SITES=$(sites)" "TUNNEL_USER=$TUNNEL_USER" "TUNNEL_PUBKEY=$(cat "$TUNNEL_KEY.pub")"
  echo "== Bật tunnel qlpt-edge-tunnel@$host"
  install_tunnel "$host" "$ip"
  verify_tunnel "$host"
  echo "== Việc tiếp theo cho từng domain muốn chuyển sang $host:"
  for d in $(site_domains); do
    echo "  Cloudflare DNS: $d -> A $ip, Proxy status: DNS only; rồi chạy: $0 cert $host $d"
  done
}

cmd_cert() {
  local host="${1:?thiếu <ssh-host>}"
  shift
  (($# > 0)) || die "thiếu domain"
  local ip
  ip="$(edge_ip "$host")"
  for d in "$@"; do
    site_domains | grep -qx "$d" || die "$d không có trong sites.conf"
    local waited=0
    until [[ "$(dns_a "$d")" == "$ip" ]]; do
      ((waited >= DNS_WAIT_SECONDS)) && die "$d vẫn trỏ về '$(dns_a "$d")', chưa phải $ip"
      ((waited % 30 == 0)) && echo "  chờ DNS $d -> $ip (hiện: $(dns_a "$d"))"
      sleep 5
      waited=$((waited + 5))
    done
  done
  echo "== Xin chứng chỉ trên $host: $*"
  remote "$host" ACTION=cert "SITES=$(sites)" "CERT_DOMAINS=$*"
  for d in "$@"; do
    printf '  https://%s/health -> %s\n' "$d" "$(curl -s -o /dev/null -w '%{http_code} (%{http_version}, %{time_total}s)' --max-time 15 --resolve "$d:443:$ip" "https://$d/health" || true)"
  done
}

cmd_status() {
  echo "== Tunnel"
  systemctl --user list-units 'qlpt-edge-tunnel@*' --all --no-legend --plain 2>/dev/null | awk '{ printf "  %s %s\n", $1, $4 }'
  echo "== DNS (1.1.1.1)"
  for d in $(site_domains); do
    local a edge="Cloudflare/khác"
    a="$(dns_a "$d")"
    for f in "$CONF_DIR"/*.env; do
      [[ -f "$f" ]] || continue
      [[ "$a" == "$(sed -n 's/^EDGE_HOST=//p' "$f")" ]] && edge="edge $(basename "$f" .env)"
    done
    printf '  %-20s %-32s %s\n' "$d" "$a" "$edge"
  done
}

cmd_remove() {
  local host="${1:?thiếu <ssh-host>}" ip
  ip="$(edge_ip "$host")"
  for d in $(site_domains); do
    [[ "$(dns_a "$d")" == "$ip" && "${FORCE:-}" != 1 ]] && die "$d vẫn trỏ về $host ($ip); đổi DNS trước hoặc chạy với FORCE=1"
  done
  remote "$host" ACTION=remove "SITES=$(sites)" "TUNNEL_USER=$TUNNEL_USER"
  systemctl --user disable --now "qlpt-edge-tunnel@$host" >/dev/null 2>&1 || true
  rm -f "$CONF_DIR/$host.env"
  echo "== Đã gỡ edge $host"
}

case "${1:-}" in
  install) shift; cmd_install "$@" ;;
  cert) shift; cmd_cert "$@" ;;
  status) cmd_status ;;
  remove) shift; cmd_remove "$@" ;;
  *) usage; exit 1 ;;
esac
