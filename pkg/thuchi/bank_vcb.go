package thuchi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

func init() {
	EmailParsers["vietcombank.com.vn"] = parseVCBEmail
}

var (
	tdOpen     = regexp.MustCompile(`(?i)<td[^>]*>`)
	tdClose    = regexp.MustCompile(`(?i)</td\s*>`)
	brTag      = regexp.MustCompile(`(?i)<br\s*/?>|</p\s*>|</div\s*>`)
	vcbTime    = regexp.MustCompile(`(\d{1,2}):(\d{2})(?::(\d{2}))?`)
	vcbDate    = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})`)
	senderPays = []string{"nguoi chuyen tra", "ben chuyen tra"}
)

func HTMLCells(h string) [][]string {
	parts := tdOpen.Split(h, -1)
	if len(parts) < 2 {
		return nil
	}
	out := make([][]string, 0, len(parts)-1)
	for _, p := range parts[1:] {
		if loc := tdClose.FindStringIndex(p); loc != nil {
			p = p[:loc[0]]
		}
		p = brTag.ReplaceAllString(p, "\n")
		p = html.UnescapeString(htmlTag.ReplaceAllString(p, ""))
		var lines []string
		for _, l := range strings.Split(p, "\n") {
			l = strings.TrimSpace(multiSpace.ReplaceAllString(strings.ReplaceAll(l, " ", " "), " "))
			if l != "" {
				lines = append(lines, l)
			}
		}
		if len(lines) > 0 {
			out = append(out, lines)
		}
	}
	return out
}

func cellValue(cells [][]string, labels ...string) []string {
	for i := 0; i+1 < len(cells); i++ {
		k := cellKey(cells[i][0])
		for _, l := range labels {
			if k == l {
				return cells[i+1]
			}
		}
	}
	return nil
}

func parseVCBEmail(m *EmailMessage) (*Incoming, error) {
	cells := HTMLCells(m.HTML)
	if len(cells) == 0 {
		return nil, errors.New("email Vietcombank không có nội dung HTML")
	}
	first := func(labels ...string) string {
		if v := cellValue(cells, labels...); len(v) > 0 {
			return v[0]
		}
		return ""
	}
	debit := OnlyDigits(first("tai khoan nguon", "tai khoan trich no", "so tai khoan nguon"))
	amountText := first("so tien", "so tien giao dich", "so tien chuyen")
	amount, ok := ParseVND(amountText)
	if debit == "" || !ok || amount <= 0 {
		return nil, fmt.Errorf("email Vietcombank %q chưa đọc được (thiếu tài khoản nguồn/số tiền)", m.Subject)
	}
	when := strings.Join(cellValue(cells, "ngay, gio giao dich", "ngay gio giao dich", "thoi gian giao dich"), " ")
	occurred, err := parseVCBTime(when)
	if err != nil {
		if m.Date.IsZero() {
			return nil, fmt.Errorf("email Vietcombank: thời gian %q", when)
		}
		occurred = m.Date
	}
	if fee, ok := ParseVND(first("so tien phi", "phi giao dich")); ok && fee > 0 {
		chargeCode := cellKey(first("loai phi"))
		for _, p := range senderPays {
			if strings.Contains(chargeCode, p) {
				amount += fee
				break
			}
		}
	}
	content := strings.Join(cellValue(cells, "noi dung chuyen tien", "noi dung", "noi dung giao dich"), " ")
	in := &Incoming{
		BankCode:            "VCB",
		AccountNumber:       debit,
		Direction:           DirectionOut,
		Amount:              amount,
		OccurredAt:          occurred.UTC(),
		Content:             content,
		CounterpartyName:    first("ten nguoi huong", "ten nguoi thu huong", "ten don vi thu huong"),
		CounterpartyAccount: OnlyDigits(first("tai khoan nguoi huong", "tai khoan thu huong", "so tai khoan nguoi huong", "the nguoi huong")),
	}
	if bank := first("ten ngan hang huong", "ngan hang huong", "ngan hang thu huong"); bank != "" {
		in.Content = strings.TrimSpace(in.Content + " - " + bank)
	}
	ref := OnlyDigits(first("so lenh giao dich", "so tham chieu", "ma giao dich"))
	if ref == "" {
		sum := sha256.Sum256([]byte(debit + "|" + amountText + "|" + when + "|" + content))
		ref = "h:" + hex.EncodeToString(sum[:8])
	}
	in.SourceRef = "vcb:" + debit + ":" + ref
	return in, nil
}

func parseVCBTime(s string) (time.Time, error) {
	d := vcbDate.FindStringSubmatch(s)
	if d == nil {
		return time.Time{}, errors.New("no date")
	}
	clock := "00:00:00"
	if t := vcbTime.FindStringSubmatch(s); t != nil {
		clock = t[1] + ":" + t[2] + ":" + firstNonEmpty(t[3], "00")
	}
	return time.ParseInLocation("2/1/2006 15:04:05", d[1]+"/"+d[2]+"/"+d[3]+" "+clock, vnZone)
}
