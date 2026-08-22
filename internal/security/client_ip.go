package security

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ClientIP trả về IP của client, chỉ đọc header chuyển tiếp khi peer trực tiếp
// nằm trong dải proxy tin cậy. Nếu không parse được IP thì trả về RemoteAddr thô
// để caller vẫn có một khóa định danh ổn định.
func ClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	remoteIP := remoteAddrIP(r.RemoteAddr)
	if remoteIP != "" && fromTrustedProxy(r, trustedProxies) {
		// Cloudflare/cloudflared luôn ghi đè CF-Connecting-IP nên client không giả mạo được.
		if ip := parseIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
			return ip
		}
		// Proxy *nối thêm* IP client vào cuối X-Forwarded-For, giữ nguyên phần client tự gửi.
		// Vì vậy chỉ mục cuối — do proxy tin cậy ghi — mới không giả mạo được.
		if xff := lastForwardedIP(r.Header.Get("X-Forwarded-For")); xff != "" {
			return xff
		}
	}

	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

// remoteAddrIP extracts and validates the IP portion of a request RemoteAddr.
func remoteAddrIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		if _, parseErr := netip.ParseAddr(remoteAddr); parseErr == nil {
			return remoteAddr
		}
		return ""
	}

	if _, err := netip.ParseAddr(host); err != nil {
		return ""
	}

	return host
}

// lastForwardedIP extracts the last syntactically valid X-Forwarded-For IP,
// tức mục do proxy tin cậy gần nhất ghi vào.
func lastForwardedIP(header string) string {
	parts := strings.Split(header, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if ip := parseIP(parts[i]); ip != "" {
			return ip
		}
	}
	return ""
}

// parseIP trims and validates a single IP value from a header.
func parseIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if _, err := netip.ParseAddr(value); err != nil {
		return ""
	}
	return value
}
