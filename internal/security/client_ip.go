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
//
// Hàm này luôn tính lại từ request + trustedProxies, không đọc context — nhờ vậy
// caller truyền danh sách hẹp hơn vẫn nhận đúng ngữ nghĩa mình yêu cầu. Nếu muốn
// dùng lại giá trị middleware đã resolve thì gọi ResolvedClientIP.
func ClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	if r == nil {
		return ""
	}

	remoteIP := remoteAddrIP(r.RemoteAddr)
	if remoteIP != "" && FromTrustedProxy(r, trustedProxies) {
		// 1. Cloudflare/cloudflared luôn ghi đè CF-Connecting-IP nên client không giả mạo được.
		if ip := parseIP(r.Header.Get("CF-Connecting-IP")); ip != "" {
			return ip
		}
		// 2. Akamai / Cloudflare Enterprise ghi True-Client-IP.
		if ip := parseIP(r.Header.Get("True-Client-IP")); ip != "" {
			return ip
		}
		// 3. Proxy *nối thêm* IP client vào cuối X-Forwarded-For, giữ nguyên phần client tự gửi.
		// Vì vậy chỉ mục cuối — do proxy tin cậy ghi — mới không giả mạo được.
		// Ưu tiên hơn X-Real-IP vì X-Real-IP chỉ có một giá trị, không diễn tả được chuỗi proxy.
		if xff := lastForwardedIP(r.Header.Get("X-Forwarded-For")); xff != "" {
			return xff
		}
		// 4. Chuẩn RFC 7239 (Traefik, Envoy, Caddy...): Forwarded: for=...
		if fwd := lastForwardedRFC7239(r.Header.Get("Forwarded")); fwd != "" {
			return fwd
		}
		// 5. Nginx / reverse proxy thiết lập X-Real-IP ($remote_addr).
		if ip := parseIP(r.Header.Get("X-Real-IP")); ip != "" {
			return ip
		}
	}

	if remoteIP != "" {
		return remoteIP
	}
	return r.RemoteAddr
}

// ResolvedClientIP trả về IP client mà middleware đã resolve và gắn vào context,
// hoặc tự tính bằng ClientIP nếu request chưa đi qua middleware đó.
func ResolvedClientIP(r *http.Request, trustedProxies []netip.Prefix) string {
	if r == nil {
		return ""
	}
	if ip, ok := ClientIPFromContext(r.Context()); ok {
		return ip
	}
	return ClientIP(r, trustedProxies)
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

// lastForwardedRFC7239 lấy IP client từ header Forwarded (RFC 7239). Giống
// X-Forwarded-For, phần tử cuối là phần tử do proxy gần nhất ghi vào.
func lastForwardedRFC7239(header string) string {
	if header == "" {
		return ""
	}
	elements := splitOutsideQuotes(header, ',')
	for i := len(elements) - 1; i >= 0; i-- {
		for _, param := range splitOutsideQuotes(elements[i], ';') {
			param = strings.TrimSpace(param)
			if len(param) < 4 || !strings.EqualFold(param[:4], "for=") {
				continue
			}
			if ip := parseIP(unwrapForwardedNode(param[4:])); ip != "" {
				return ip
			}
		}
	}
	return ""
}

// unwrapForwardedNode bóc lớp quote, port và ngoặc vuông IPv6 của một node RFC 7239,
// ví dụ `"[2001:db8::1]:8080"` -> `2001:db8::1`.
func unwrapForwardedNode(value string) string {
	value = strings.Trim(strings.TrimSpace(value), `"`)
	if host, _, err := net.SplitHostPort(value); err == nil {
		return host
	}
	return strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
}

// splitOutsideQuotes tách chuỗi theo sep nhưng bỏ qua sep nằm trong cặp nháy kép,
// vì node RFC 7239 được phép quote và chứa cả dấu phẩy.
func splitOutsideQuotes(value string, sep byte) []string {
	var (
		parts   []string
		start   int
		inQuote bool
	)
	for i := 0; i < len(value); i++ {
		switch {
		case value[i] == '\\' && inQuote:
			i++
		case value[i] == '"':
			inQuote = !inQuote
		case value[i] == sep && !inQuote:
			parts = append(parts, value[start:i])
			start = i + 1
		}
	}
	return append(parts, value[start:])
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
