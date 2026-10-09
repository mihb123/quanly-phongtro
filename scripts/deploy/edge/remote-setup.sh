#!/usr/bin/env bash
set -euo pipefail

SITE_FILE=/etc/nginx/sites-available/qlpt-edge.conf
SITE_LINK=/etc/nginx/sites-enabled/qlpt-edge.conf
SNIPPET=/etc/nginx/snippets/qlpt-edge-proxy.conf
ACME_ROOT=/var/www/qlpt-acme
SSHD_CONF=/etc/ssh/sshd_config.d/60-qlpt-tunnel.conf

log() { printf '[edge] %s\n' "$*"; }
die() { printf '[edge] LỖI: %s\n' "$*" >&2; exit 1; }

ports() { for s in $SITES; do echo "${s##*:}"; done | sort -u; }

detect_tls_listen() {
  if [[ -n "${EDGE_TLS_LISTEN:-}" ]]; then
    TLS_LISTEN="$EDGE_TLS_LISTEN"
    CLIENT_IP="${EDGE_CLIENT_IP:-\$remote_addr}"
    return
  fi
  TLS_LISTEN="443 ssl http2"
  CLIENT_IP='$remote_addr'
  local conf backend addr
  conf="$(grep -ls 'ssl_preread[[:space:]]\+on' /etc/nginx/stream-enabled/*.conf 2>/dev/null | head -1 || true)"
  [[ -z "$conf" ]] && return
  backend="$(awk '$1 == "default" { gsub(";", "", $2); print $2; exit }' "$conf")"
  addr="$(awk -v b="$backend" '$1 == "upstream" && $2 == b { for (i = 3; i <= NF; i++) if ($i == "server") { gsub(";", "", $(i + 1)); print $(i + 1); exit } }' "$conf")"
  [[ -z "$addr" ]] && die "có SNI router ở $conf nhưng không đọc được backend mặc định; đặt EDGE_TLS_LISTEN"
  grep -q 'proxy_protocol[[:space:]]\+on' "$conf" || die "SNI router không bật proxy_protocol; đặt EDGE_TLS_LISTEN và EDGE_CLIENT_IP"
  TLS_LISTEN="$addr ssl proxy_protocol"
  CLIENT_IP='$proxy_protocol_addr'
}

ensure_packages() {
  local missing=()
  command -v nginx >/dev/null || missing+=(nginx)
  command -v certbot >/dev/null || missing+=(certbot)
  command -v curl >/dev/null || missing+=(curl)
  ((${#missing[@]} == 0)) && return
  command -v apt-get >/dev/null || die "chỉ hỗ trợ Debian/Ubuntu (apt-get), thiếu: ${missing[*]}"
  log "cài ${missing[*]}"
  DEBIAN_FRONTEND=noninteractive apt-get update -qq
  DEBIAN_FRONTEND=noninteractive apt-get install -y -qq "${missing[@]}" >/dev/null
}

ensure_tunnel_user() {
  [[ -n "${TUNNEL_PUBKEY:-}" ]] || die "thiếu TUNNEL_PUBKEY"
  id "$TUNNEL_USER" >/dev/null 2>&1 ||
    useradd --system --create-home --home-dir "/var/lib/$TUNNEL_USER" --shell /usr/sbin/nologin "$TUNNEL_USER"
  local home listens=""
  home="$(getent passwd "$TUNNEL_USER" | cut -d: -f6)"
  for p in $(ports); do listens+="permitlisten=\"127.0.0.1:$p\","; done
  install -d -m 700 -o "$TUNNEL_USER" -g "$TUNNEL_USER" "$home/.ssh"
  printf 'restrict,port-forwarding,%scommand="/usr/sbin/nologin" %s\n' "$listens" "$TUNNEL_PUBKEY" >"$home/.ssh/authorized_keys"
  chown "$TUNNEL_USER:$TUNNEL_USER" "$home/.ssh/authorized_keys"
  chmod 600 "$home/.ssh/authorized_keys"

  cat >"$SSHD_CONF" <<EOF
Match User $TUNNEL_USER
    PasswordAuthentication no
    AllowTcpForwarding remote
    GatewayPorts no
    X11Forwarding no
    PermitTTY no
    ClientAliveInterval 15
    ClientAliveCountMax 3
EOF
  sshd -t || { rm -f "$SSHD_CONF"; die "sshd -t thất bại, đã bỏ $SSHD_CONF"; }
  [[ "$(sshd -T -C user=root,host=localhost,addr=127.0.0.1 | awk '$1 == "permittty" { print $2 }')" == "yes" ]] ||
    { rm -f "$SSHD_CONF"; sshd -t && systemctl reload ssh 2>/dev/null; die "Match block lan sang user khác, đã gỡ"; }
  systemctl reload ssh 2>/dev/null || systemctl reload sshd
  log "user tunnel $TUNNEL_USER chỉ được mở: $(ports | sed 's/^/127.0.0.1:/' | tr '\n' ' ')"
}

ensure_firewall() {
  if command -v ufw >/dev/null && ufw status | grep -q '^Status: active'; then
    ufw allow 80/tcp >/dev/null
    ufw allow 443/tcp >/dev/null
  fi
}

write_snippet() {
  install -d /etc/nginx/snippets
  cat >"$SNIPPET" <<EOF
proxy_http_version 1.1;
proxy_set_header Connection "";
proxy_set_header Host \$host;
proxy_set_header X-Forwarded-Proto https;
proxy_set_header X-Forwarded-Host \$host;
proxy_set_header X-Forwarded-For $CLIENT_IP;
proxy_set_header X-Real-IP $CLIENT_IP;
proxy_set_header CF-Connecting-IP $CLIENT_IP;
proxy_set_header True-Client-IP "";
proxy_set_header CF-Visitor "";
proxy_request_buffering off;
proxy_read_timeout 300s;
proxy_send_timeout 300s;
EOF
}

render_site() {
  local ssl_opts=""
  [[ -f /etc/letsencrypt/options-ssl-nginx.conf ]] && ssl_opts="    include /etc/letsencrypt/options-ssl-nginx.conf;"
  {
    for p in $(ports); do
      printf 'upstream qlpt_edge_%s {\n    server 127.0.0.1:%s;\n    keepalive 16;\n}\n\n' "$p" "$p"
    done
    for s in $SITES; do
      local domain="${s%%:*}" port="${s##*:}"
      cat <<EOF
server {
    listen 80;
    server_name $domain;
    location ^~ /.well-known/acme-challenge/ { root $ACME_ROOT; }
    location / { return 308 https://\$host\$request_uri; }
}

EOF
      [[ -f "/etc/letsencrypt/live/$domain/fullchain.pem" ]] || continue
      cat <<EOF
server {
    listen $TLS_LISTEN;
    server_name $domain;
    ssl_certificate /etc/letsencrypt/live/$domain/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$domain/privkey.pem;
$ssl_opts
    client_max_body_size 20m;
    gzip on;
    gzip_proxied any;
    gzip_min_length 512;
    gzip_types text/css application/javascript text/javascript application/json image/svg+xml application/manifest+json;

    location / {
        proxy_pass http://qlpt_edge_$port;
        include $SNIPPET;
    }

    location /assets/ {
        proxy_pass http://qlpt_edge_$port;
        include $SNIPPET;
        add_header Cache-Control "public, max-age=31536000, immutable" always;
    }
}

EOF
    done
  } >"$SITE_FILE.new"

  local backup=""
  [[ -f "$SITE_FILE" ]] && backup="$(mktemp)" && cp "$SITE_FILE" "$backup"
  mv "$SITE_FILE.new" "$SITE_FILE"
  ln -sfn "$SITE_FILE" "$SITE_LINK"
  if ! nginx -t >/dev/null 2>&1; then
    nginx -t 2>&1 | tail -3 >&2
    if [[ -n "$backup" ]]; then mv "$backup" "$SITE_FILE"; else rm -f "$SITE_FILE" "$SITE_LINK"; fi
    die "nginx -t thất bại, đã trả lại cấu hình cũ"
  fi
  [[ -n "$backup" ]] && rm -f "$backup"
  systemctl reload nginx
  local https=()
  for s in $SITES; do [[ -f "/etc/letsencrypt/live/${s%%:*}/fullchain.pem" ]] && https+=("${s%%:*}"); done
  log "nginx: TLS listen '$TLS_LISTEN', IP client lấy từ $CLIENT_IP, HTTPS đã bật cho: ${https[*]:-(chưa có, chạy lệnh cert sau khi đổi DNS)}"
}

issue_certs() {
  for domain in $CERT_DOMAINS; do
    log "xin chứng chỉ cho $domain"
    certbot certonly --webroot -w "$ACME_ROOT" -d "$domain" --non-interactive --agree-tos \
      --register-unsafely-without-email --keep-until-expiring --deploy-hook "systemctl reload nginx" -q
  done
}

remove_all() {
  rm -f "$SITE_LINK" "$SITE_FILE" "$SNIPPET"
  nginx -t >/dev/null 2>&1 && systemctl reload nginx
  rm -f "$SSHD_CONF"
  sshd -t && { systemctl reload ssh 2>/dev/null || systemctl reload sshd; }
  pkill -u "$TUNNEL_USER" 2>/dev/null || true
  for _ in $(seq 1 10); do pgrep -u "$TUNNEL_USER" >/dev/null 2>&1 || break; sleep 0.5; done
  if id "$TUNNEL_USER" >/dev/null 2>&1; then
    userdel -r "$TUNNEL_USER" 2>/dev/null || true
    ! id "$TUNNEL_USER" >/dev/null 2>&1 || die "không xoá được user $TUNNEL_USER"
  fi
  rm -rf "$ACME_ROOT"
  log "đã gỡ site nginx, user tunnel và cấu hình sshd (chứng chỉ trong /etc/letsencrypt giữ nguyên)"
}

[[ "$(id -u)" -eq 0 ]] || die "cần chạy bằng root"
case "$ACTION" in
  install)
    ensure_packages
    ensure_tunnel_user
    ensure_firewall
    install -d "$ACME_ROOT"
    detect_tls_listen
    write_snippet
    render_site
    ;;
  render)
    detect_tls_listen
    write_snippet
    render_site
    ;;
  cert)
    issue_certs
    detect_tls_listen
    write_snippet
    render_site
    ;;
  remove) remove_all ;;
  *) die "ACTION không hợp lệ: $ACTION" ;;
esac
