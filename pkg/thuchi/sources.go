package thuchi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var vnZone = time.FixedZone("ICT", 7*3600)

type SePayPayload struct {
	ID              json.Number `json:"id"`
	Gateway         string      `json:"gateway"`
	TransactionDate string      `json:"transactionDate"`
	AccountNumber   string      `json:"accountNumber"`
	SubAccount      string      `json:"subAccount"`
	Code            string      `json:"code"`
	Content         string      `json:"content"`
	TransferType    string      `json:"transferType"`
	TransferAmount  json.Number `json:"transferAmount"`
	Accumulated     json.Number `json:"accumulated"`
	ReferenceCode   string      `json:"referenceCode"`
	Description     string      `json:"description"`
}

func ParseSePay(payload []byte) (*Incoming, error) {
	var p SePayPayload
	dec := json.NewDecoder(bytes.NewReader(payload))
	dec.UseNumber()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("sepay json: %w", err)
	}
	amount, err := numberToInt(p.TransferAmount)
	if err != nil || amount <= 0 {
		return nil, fmt.Errorf("sepay transferAmount %q", p.TransferAmount)
	}
	direction := strings.ToLower(strings.TrimSpace(p.TransferType))
	if direction != DirectionIn && direction != DirectionOut {
		return nil, fmt.Errorf("sepay transferType %q", p.TransferType)
	}
	occurred, err := time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(p.TransactionDate), vnZone)
	if err != nil {
		return nil, fmt.Errorf("sepay transactionDate %q", p.TransactionDate)
	}
	account := strings.TrimSpace(p.AccountNumber)
	if account == "" {
		return nil, fmt.Errorf("sepay accountNumber missing")
	}
	ref := strings.TrimSpace(p.ReferenceCode)
	if ref == "" {
		ref = "id:" + p.ID.String()
	}
	in := &Incoming{
		Source:        SourceSePay,
		SourceRef:     OnlyDigitsOr(account) + ":" + ref,
		BankCode:      gatewayToBankCode(p.Gateway),
		BankName:      strings.TrimSpace(p.Gateway),
		AccountNumber: account,
		Direction:     direction,
		Amount:        amount,
		OccurredAt:    occurred.UTC(),
		Content:       strings.TrimSpace(firstNonEmpty(p.Content, p.Description)),
	}
	if acc, err := numberToInt(p.Accumulated); err == nil && acc > 0 {
		in.BalanceAfter = &acc
	}
	return in, nil
}

func OnlyDigitsOr(s string) string {
	if d := OnlyDigits(s); d != "" {
		return d
	}
	return strings.TrimSpace(s)
}

func numberToInt(n json.Number) (int64, error) {
	if n == "" {
		return 0, fmt.Errorf("empty")
	}
	if i, err := n.Int64(); err == nil {
		return i, nil
	}
	f, err := n.Float64()
	if err != nil {
		return 0, err
	}
	return int64(f + 0.5), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

type NotifyPayload struct {
	Package  string   `json:"package"`
	Title    string   `json:"title"`
	Text     string   `json:"text"`
	PostedAt string   `json:"posted_at"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

func DecodeNotify(body []byte) (*NotifyPayload, error) {
	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("notify json: %w", err)
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				switch t := v.(type) {
				case string:
					if t != "" {
						return t
					}
				case float64:
					return strconv.FormatFloat(t, 'f', -1, 64)
				}
			}
		}
		return ""
	}
	num := func(keys ...string) *float64 {
		for _, k := range keys {
			switch t := raw[k].(type) {
			case float64:
				return &t
			case string:
				if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
					return &f
				}
			}
		}
		return nil
	}
	p := &NotifyPayload{
		Package:  str("package", "package_name", "from", "app"),
		Title:    str("title"),
		Text:     str("text", "content", "msg", "message", "body"),
		PostedAt: str("posted_at", "receive_time", "time", "timestamp"),
		Lat:      num("lat", "latitude"),
		Lng:      num("lng", "lon", "longitude"),
	}
	if p.Package == "" && p.Text == "" {
		return nil, fmt.Errorf("notify payload thiếu package/text")
	}
	return p, nil
}

type NotifyParser func(p *NotifyPayload, receivedAt time.Time) (*Incoming, error)

var NotifyPackages = map[string]string{
	"com.VCB":                   "VCB",
	"com.mservice.momotransfer": "MOMO",
	"com.tpb.mb.gprsandroid":    "TPB",
	"io.lifestyle.plus":         "TIMO",
	"com.mbmobile":              "MB",
	"mobile.acb.com.vn":         "ACB",
	"vn.com.techcombank.bb.app": "TCB",
	"xyz.be.cake":               "CAKE",
	"vn.com.vng.zalopay":        "ZALOPAY",
}

var NotifyParsers = map[string]NotifyParser{}

func ParseNotify(body []byte, receivedAt time.Time) (*Incoming, string, error) {
	p, err := DecodeNotify(body)
	if err != nil {
		return nil, "", err
	}
	bank := NotifyPackages[p.Package]
	parser, ok := NotifyParsers[bank]
	if bank == "" || !ok {
		return nil, p.Package, fmt.Errorf("chưa có parser cho thông báo của %s", firstNonEmpty(BankNames[bank], p.Package))
	}
	in, err := parser(p, receivedAt)
	if err != nil {
		return nil, p.Package, err
	}
	in.Source = SourceNotify
	in.BankCode = bank
	in.Lat, in.Lng = p.Lat, p.Lng
	return in, p.Package, nil
}

func NotifyPostedAt(p *NotifyPayload, receivedAt time.Time) time.Time {
	v := strings.TrimSpace(p.PostedAt)
	if v == "" {
		return receivedAt
	}
	if n, err := strconv.ParseInt(v, 10, 64); err == nil {
		if n > 1e12 {
			return time.UnixMilli(n)
		}
		return time.Unix(n, 0)
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "02/01/2006 15:04:05", "2006/01/02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, v, vnZone); err == nil {
			return t
		}
	}
	return receivedAt
}

type EmailMessage struct {
	From        string
	Subject     string
	Date        time.Time
	Text        string
	HTML        string
	Attachments []EmailAttachment
}

type EmailAttachment struct {
	Name        string
	ContentType string
	Data        []byte
}

const (
	maxEmailText        = 2 << 20
	maxEmailAttachment  = 8 << 20
	maxEmailAttachments = 10
)

var htmlTag = regexp.MustCompile(`(?s)<style.*?</style>|<script.*?</script>|<[^>]+>`)

func ParseEmail(raw []byte) (*EmailMessage, error) {
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("email: %w", err)
	}
	dec := new(mime.WordDecoder)
	subject, _ := dec.DecodeHeader(msg.Header.Get("Subject"))
	from, _ := dec.DecodeHeader(msg.Header.Get("From"))
	out := &EmailMessage{From: from, Subject: subject}
	if d, err := msg.Header.Date(); err == nil {
		out.Date = d
	}
	out.readPart(msg.Header, msg.Body, 0)
	if out.Text == "" && out.HTML != "" {
		out.Text = htmlToText(out.HTML)
	}
	out.Text = strings.TrimSpace(out.Text)
	return out, nil
}

type headerGetter interface{ Get(string) string }

func (m *EmailMessage) readPart(h headerGetter, body io.Reader, depth int) {
	if depth > 5 {
		return
	}
	mediaType, params, err := mime.ParseMediaType(h.Get("Content-Type"))
	if err != nil {
		mediaType = "text/plain"
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(body, params["boundary"])
		for {
			part, err := mr.NextPart()
			if err != nil {
				break
			}
			m.readPart(part.Header, part, depth+1)
		}
		return
	}
	var r io.Reader = body
	switch strings.ToLower(strings.TrimSpace(h.Get("Content-Transfer-Encoding"))) {
	case "quoted-printable":
		r = quotedprintable.NewReader(body)
	case "base64":
		r = base64.NewDecoder(base64.StdEncoding, newlineStripper{body})
	}
	disposition, dparams, _ := mime.ParseMediaType(h.Get("Content-Disposition"))
	name := firstNonEmpty(dparams["filename"], params["name"])
	if decoded, err := new(mime.WordDecoder).DecodeHeader(name); err == nil {
		name = decoded
	}
	isText := mediaType == "text/plain" || mediaType == "text/html"
	if isText && name == "" && disposition != "attachment" {
		data, _ := io.ReadAll(io.LimitReader(r, maxEmailText))
		if mediaType == "text/plain" && m.Text == "" {
			m.Text = string(data)
		} else if mediaType == "text/html" && m.HTML == "" {
			m.HTML = string(data)
		}
		return
	}
	if (name == "" && disposition != "attachment") || len(m.Attachments) >= maxEmailAttachments {
		return
	}
	data, _ := io.ReadAll(io.LimitReader(r, maxEmailAttachment))
	m.Attachments = append(m.Attachments, EmailAttachment{Name: strings.TrimSpace(name), ContentType: mediaType, Data: data})
}

type newlineStripper struct{ r io.Reader }

func (n newlineStripper) Read(p []byte) (int, error) {
	buf := make([]byte, len(p))
	c, err := n.r.Read(buf)
	j := 0
	for i := 0; i < c; i++ {
		if buf[i] != '\r' && buf[i] != '\n' {
			p[j] = buf[i]
			j++
		}
	}
	return j, err
}

func htmlToText(h string) string {
	h = strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "</p>", "\n", "</tr>", "\n", "</div>", "\n", "</td>", " ").Replace(h)
	h = htmlTag.ReplaceAllString(h, "")
	h = strings.NewReplacer("&nbsp;", " ", "&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#39;", "'").Replace(h)
	lines := strings.Split(h, "\n")
	out := lines[:0]
	for _, l := range lines {
		if l = strings.TrimSpace(multiSpace.ReplaceAllString(l, " ")); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

type EmailParser func(m *EmailMessage) (*Incoming, error)

var EmailParsers = map[string]EmailParser{}

var gmailConfirmLink = regexp.MustCompile(`https://mail(?:-settings)?\.google\.com/mail/[^\s"<>]+`)

func ParseEmailEvents(raw []byte) ([]*Incoming, string, error) {
	m, err := ParseEmail(raw)
	if err != nil {
		return nil, "", err
	}
	key := m.From
	if addr, err := mail.ParseAddress(m.From); err == nil {
		key = addr.Address
	}
	lower := strings.ToLower(key)
	if strings.Contains(lower, "forwarding-noreply@google.com") {
		link := gmailConfirmLink.FindString(m.Text)
		return nil, key, fmt.Errorf("Gmail cần xác nhận địa chỉ chuyển tiếp: %s", firstNonEmpty(link, m.Subject))
	}
	var attErr error
	for _, a := range m.Attachments {
		if !IsXLS(a.Data) && !strings.HasSuffix(strings.ToLower(a.Name), ".xls") && !strings.HasSuffix(strings.ToLower(a.Name), ".xlsx") {
			continue
		}
		ins, err := ParseStatementFile(firstNonEmpty(a.Name, "tệp đính kèm"), a.Data)
		if err != nil {
			attErr = err
			continue
		}
		for _, in := range ins {
			in.Source = SourceEmail
		}
		return ins, key, nil
	}
	domain := lower
	if i := strings.LastIndex(lower, "@"); i >= 0 {
		domain = lower[i+1:]
	}
	for suffix, parser := range EmailParsers {
		if domain == suffix || strings.HasSuffix(domain, "."+suffix) {
			in, err := parser(m)
			if err != nil {
				return nil, key, err
			}
			in.Source = SourceEmail
			return []*Incoming{in}, key, nil
		}
	}
	if attErr != nil {
		return nil, key, attErr
	}
	return nil, key, fmt.Errorf("chưa có parser cho email từ %s (%s)", key, m.Subject)
}
