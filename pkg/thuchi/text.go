package thuchi

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"strings"
	"time"
)

var vnFold = map[rune]rune{}

func init() {
	groups := map[rune]string{
		'a': "àáạảãâầấậẩẫăằắặẳẵ",
		'e': "èéẹẻẽêềếệểễ",
		'i': "ìíịỉĩ",
		'o': "òóọỏõôồốộổỗơờớợởỡ",
		'u': "ùúụủũưừứựửữ",
		'y': "ỳýỵỷỹ",
		'd': "đ",
	}
	for base, chars := range groups {
		for _, c := range chars {
			vnFold[c] = base
		}
		upper := []rune(strings.ToUpper(chars))
		for _, c := range upper {
			vnFold[c] = base - 'a' + 'A'
		}
	}
}

var (
	nonAlnum   = regexp.MustCompile(`[^A-Z0-9]+`)
	digitRuns  = regexp.MustCompile(`\d{6,19}`)
	multiSpace = regexp.MustCompile(`\s+`)
)

func Fold(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if f, ok := vnFold[r]; ok {
			b.WriteRune(f)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Normalize(s string) string {
	s = strings.ToUpper(Fold(s))
	s = nonAlnum.ReplaceAllString(s, " ")
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}

func OnlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func DigitRuns(s string) []string {
	return digitRuns.FindAllString(s, -1)
}

func AccountHash(ownerID, accountNumber string) string {
	digits := OnlyDigits(accountNumber)
	if digits == "" {
		digits = strings.ToUpper(strings.TrimSpace(accountNumber))
	}
	if digits == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(ownerID + ":" + digits))
	return hex.EncodeToString(sum[:])
}

func Last4(accountNumber string) string {
	d := OnlyDigits(accountNumber)
	if len(d) > 4 {
		return d[len(d)-4:]
	}
	return d
}

func NewID() string {
	var b [16]byte
	ms := uint64(time.Now().UnixMilli())
	binary.BigEndian.PutUint64(b[0:8], ms<<16)
	_, _ = rand.Read(b[6:])
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:32]
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func ValidID(s string) bool {
	return uuidPattern.MatchString(s)
}

func containsWord(haystack, needle string) bool {
	if needle == "" {
		return false
	}
	return strings.Contains(" "+haystack+" ", " "+needle+" ")
}

var merchantStopwords = set(
	"THANH", "TOAN", "TT", "QR", "QRCODE", "VIETQR", "CHUYEN", "TIEN", "CK", "CHUYENKHOAN", "NHAN", "TU", "DEN", "DI", "GD", "GIAO", "DICH",
	"LAN", "NOI", "DUNG", "ND", "FT", "IBFT", "MBVCB", "TKTHE", "TK", "STK", "SO", "TAI", "KHOAN", "THE", "POS", "PAY", "PAYMENT", "TRACE",
	"REF", "VND", "VNPAY", "NAPAS", "TRANSFER", "FROM", "TO", "VA", "CUA", "CHO", "TAN", "MUA", "HANG", "DON", "ORDER", "MA", "CODE", "TRANG",
)

func DeriveMerchant(content string) string {
	words := strings.Fields(Normalize(content))
	var out []string
	for _, w := range words {
		if len(out) >= 4 {
			break
		}
		if strings.IndexFunc(w, func(r rune) bool { return r >= '0' && r <= '9' }) >= 0 || len(w) < 2 && len(out) == 0 {
			continue
		}
		if merchantStopwords[w] {
			if len(out) > 0 {
				break
			}
			continue
		}
		out = append(out, w)
	}
	if len(out) == 0 {
		return ""
	}
	return TitleCase(strings.Join(out, " "))
}

func TitleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r := []rune(w)
		r[0] = []rune(strings.ToUpper(string(r[0])))[0]
		words[i] = string(r)
	}
	return strings.Join(words, " ")
}
