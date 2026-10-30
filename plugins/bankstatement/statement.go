package bankstatement

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mihb123/quanly-phongtro/pkg/pdftext"
)

type Transaction struct {
	Bank        string    `json:"bank"`
	Account     string    `json:"account"`
	Reference   string    `json:"reference"`
	Date        time.Time `json:"date"`
	Amount      int64     `json:"amount"`
	Description string    `json:"description"`
}

type StatementFile struct {
	Name  string
	Pages []pdftext.Page
}

type StatementParser interface {
	Bank() string
	ParseCredits(file StatementFile) ([]Transaction, bool)
}

var parsers = []StatementParser{tpBankParser{}}

var moneyPattern = regexp.MustCompile(`^\d{1,3}(?:[.,]\d{3})+$|^\d+$`)

func parseMoney(raw string) (int64, error) {
	digits := strings.NewReplacer(",", "", ".", "", " ", "").Replace(strings.TrimSpace(raw))
	return strconv.ParseInt(digits, 10, 64)
}

func parseDate(raw string) (time.Time, error) {
	return time.ParseInLocation("02/01/2006", strings.TrimSpace(raw), time.Local)
}

func parseCredits(files []StatementFile) ([]Transaction, error) {
	var advices, ledgers []Transaction
	for _, file := range files {
		for _, p := range parsers {
			txs, isAdvice := p.ParseCredits(file)
			if isAdvice {
				advices = append(advices, txs...)
			} else {
				ledgers = append(ledgers, txs...)
			}
		}
	}
	credits := advices
	if len(credits) == 0 {
		credits = ledgers
	}

	seen := map[string]bool{}
	out := make([]Transaction, 0, len(credits))
	for _, tx := range credits {
		if tx.Amount <= 0 {
			continue
		}
		if tx.Reference == "" {
			sum := sha1.Sum([]byte(fmt.Sprintf("%s|%s|%d|%s", tx.Account, tx.Date.Format("2006-01-02"), tx.Amount, tx.Description)))
			tx.Reference = "H" + hex.EncodeToString(sum[:])[:20]
		}
		if seen[tx.Reference] {
			continue
		}
		seen[tx.Reference] = true
		out = append(out, tx)
	}
	return out, nil
}
