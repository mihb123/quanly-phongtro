package bankstatement

import (
	"math"
	"regexp"
	"strings"

	"github.com/mihb123/quanly-phongtro/pkg/pdftext"
)

type tpBankParser struct{}

func (tpBankParser) Bank() string { return "TPBANK" }

var (
	tpAdviceTitle   = regexp.MustCompile(`PHIẾU BÁO CÓ`)
	tpAdviceRef     = regexp.MustCompile(`Số tham chiếu:\s*(\S+)`)
	tpAdviceAccount = regexp.MustCompile(`ghi có tài khoản của Quý khách hàng số:\s*(\d+)`)
	tpAdviceAmount  = regexp.MustCompile(`Số tiền ghi có\s*:\s*VND\s*([\d.,]+)`)
	tpAdviceDate    = regexp.MustCompile(`Ngày giá trị:\s*(\d{2}/\d{2}/\d{4})`)
	tpAdviceContent = regexp.MustCompile(`(?s)Nội dung:\s*(.*?)\s*(?:Phiếu báo Có được in|$)`)
	tpLedgerTitle   = regexp.MustCompile(`SỔ PHỤ TÀI KHOẢN`)
	tpLedgerAccount = regexp.MustCompile(`\b(\d{8,20})\b`)
	tpLedgerDate    = regexp.MustCompile(`^\d{2}/\d{2}/\d{4}$`)
	tpLedgerSeq     = regexp.MustCompile(`^\d+$`)
)

func (p tpBankParser) ParseCredits(file StatementFile) ([]Transaction, bool) {
	for _, page := range file.Pages {
		if tpAdviceTitle.MatchString(page.Text()) {
			return p.parseAdvices(file), true
		}
	}
	for _, page := range file.Pages {
		if tpLedgerTitle.MatchString(page.Text()) {
			return p.parseLedger(file), false
		}
	}
	return nil, false
}

func (p tpBankParser) parseAdvices(file StatementFile) []Transaction {
	var out []Transaction
	for _, page := range file.Pages {
		text := page.Text()
		if !tpAdviceTitle.MatchString(text) {
			continue
		}
		amountMatch := tpAdviceAmount.FindStringSubmatch(text)
		dateMatch := tpAdviceDate.FindStringSubmatch(text)
		if amountMatch == nil || dateMatch == nil {
			continue
		}
		amount, err := parseMoney(amountMatch[1])
		if err != nil {
			continue
		}
		date, err := parseDate(dateMatch[1])
		if err != nil {
			continue
		}
		tx := Transaction{Bank: p.Bank(), Amount: amount, Date: date}
		if m := tpAdviceRef.FindStringSubmatch(text); m != nil {
			tx.Reference = m[1]
		}
		if m := tpAdviceAccount.FindStringSubmatch(text); m != nil {
			tx.Account = m[1]
		}
		if m := tpAdviceContent.FindStringSubmatch(text); m != nil {
			tx.Description = strings.Join(strings.Fields(m[1]), " ")
		}
		out = append(out, tx)
	}
	return out
}

type ledgerColumns struct {
	dateX, refX, descX, debitEnd, creditEnd float64
	found                                   bool
}

type ledgerRow struct {
	date        string
	reference   strings.Builder
	description []string
	credit      string
}

func (p tpBankParser) parseLedger(file StatementFile) []Transaction {
	var (
		out     []Transaction
		account string
		cols    ledgerColumns
		row     *ledgerRow
	)
	flush := func() {
		if row == nil || row.credit == "" {
			row = nil
			return
		}
		amount, err1 := parseMoney(row.credit)
		date, err2 := parseDate(row.date)
		if err1 == nil && err2 == nil {
			out = append(out, Transaction{
				Bank:        p.Bank(),
				Account:     account,
				Reference:   row.reference.String(),
				Date:        date,
				Amount:      amount,
				Description: strings.Join(strings.Fields(strings.Join(row.description, " ")), " "),
			})
		}
		row = nil
	}

	for _, page := range file.Pages {
		lines := page.Lines()
		for i, line := range lines {
			if account == "" && strings.Contains(line.Text, "Số tài khoản") && i+1 < len(lines) {
				if m := tpLedgerAccount.FindStringSubmatch(lines[i+1].Text); m != nil {
					account = m[1]
				}
			}
			if header, ok := ledgerHeader(line); ok {
				cols = header
				continue
			}
			if !cols.found {
				continue
			}
			if strings.HasPrefix(line.Text, "Tổng phát sinh") || strings.HasPrefix(line.Text, "Số dư cuối kỳ") {
				flush()
				continue
			}
			if startsLedgerRow(line, cols) {
				flush()
				row = &ledgerRow{}
			}
			if row == nil {
				continue
			}
			for _, item := range line.Items {
				text := strings.TrimSpace(item.Text)
				switch {
				case item.X < cols.dateX-2:
				case item.X < cols.refX-2:
					if tpLedgerDate.MatchString(text) {
						row.date = text
					}
				case item.X < cols.descX-2:
					row.reference.WriteString(text)
				case moneyPattern.MatchString(text) && item.EndX > cols.descX+(cols.debitEnd-cols.descX)*0.6:
					if math.Abs(item.EndX-cols.creditEnd) < math.Abs(item.EndX-cols.debitEnd) {
						row.credit = text
					}
				default:
					row.description = append(row.description, text)
				}
			}
		}
	}
	flush()
	return out
}

func ledgerHeader(line pdftext.Line) (ledgerColumns, bool) {
	var cols ledgerColumns
	var hasDate, hasRef, hasDesc, hasDebit, hasCredit bool
	for _, item := range line.Items {
		switch strings.TrimSpace(item.Text) {
		case "Ngày giá trị":
			cols.dateX, hasDate = item.X, true
		case "Số giao dịch":
			cols.refX, hasRef = item.X, true
		case "Diễn giải":
			cols.descX, hasDesc = item.X, true
		case "Nợ":
			cols.debitEnd, hasDebit = item.EndX, true
		case "Có":
			cols.creditEnd, hasCredit = item.EndX, true
		}
	}
	cols.found = hasDate && hasRef && hasDesc && hasDebit && hasCredit
	return cols, cols.found
}

func startsLedgerRow(line pdftext.Line, cols ledgerColumns) bool {
	var hasSeq, hasDate bool
	for _, item := range line.Items {
		text := strings.TrimSpace(item.Text)
		if item.X < cols.dateX-2 && tpLedgerSeq.MatchString(text) {
			hasSeq = true
		}
		if item.X >= cols.dateX-2 && item.X < cols.refX-2 && tpLedgerDate.MatchString(text) {
			hasDate = true
		}
	}
	return hasSeq && hasDate
}
