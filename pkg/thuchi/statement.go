package thuchi

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type StatementParser func(rows [][]string) ([]*Incoming, error)

var StatementParsers = map[string]StatementParser{
	"TIMO": parseTimoStatement,
}

var errNotStatement = errors.New("không đúng mẫu sao kê")

func ParseStatementFile(name string, data []byte) ([]*Incoming, error) {
	if !IsXLS(data) {
		return nil, fmt.Errorf("%s: chưa hỗ trợ định dạng này, cần file .xls", name)
	}
	sheets, err := ReadXLS(data)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	banks := make([]string, 0, len(StatementParsers))
	for b := range StatementParsers {
		banks = append(banks, b)
	}
	sort.Strings(banks)
	for _, sh := range sheets {
		for _, bank := range banks {
			ins, err := StatementParsers[bank](sh.Rows)
			if errors.Is(err, errNotStatement) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			for _, in := range ins {
				in.BankCode = bank
			}
			return ins, nil
		}
	}
	return nil, fmt.Errorf("%s: %w đã hỗ trợ", name, errNotStatement)
}

func cellKey(s string) string {
	first, _, _ := strings.Cut(s, "\n")
	return strings.ToLower(strings.TrimSpace(multiSpace.ReplaceAllString(Fold(first), " ")))
}

func findHeader(rows [][]string, want map[string]string) (int, map[string]int) {
	for r, row := range rows {
		cols := map[string]int{}
		for c, v := range row {
			k := cellKey(v)
			for field, prefix := range want {
				if _, ok := cols[field]; !ok && strings.HasPrefix(k, prefix) {
					cols[field] = c
				}
			}
		}
		if len(cols) == len(want) {
			return r, cols
		}
	}
	return -1, nil
}

func labelValue(rows [][]string, prefix string) string {
	for _, row := range rows {
		for c, v := range row {
			if !strings.HasPrefix(cellKey(v), prefix) {
				continue
			}
			for _, next := range row[c+1:] {
				if strings.TrimSpace(next) != "" {
					return strings.TrimSpace(next)
				}
			}
		}
	}
	return ""
}

var decimalTail = regexp.MustCompile(`[.,]\d{1,2}$`)

func ParseVND(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "VND"), "đ")
	s = decimalTail.ReplaceAllString(strings.TrimSpace(s), "")
	d := OnlyDigits(s)
	if d == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func parseTimoStatement(rows [][]string) ([]*Incoming, error) {
	head, cols := findHeader(rows, map[string]string{
		"date":    "ngay gd",
		"ref":     "ma giao dich",
		"credit":  "phat sinh co",
		"debit":   "phat sinh no",
		"balance": "so du",
		"desc":    "dien giai",
	})
	if head < 0 {
		return nil, errNotStatement
	}
	acctCol := -1
	for c, v := range rows[head] {
		if strings.HasPrefix(cellKey(v), "so tai khoan truy van") {
			acctCol = c
		}
	}
	mainAcct := OnlyDigits(labelValue(rows[:head], "so tai khoan chinh"))
	cell := func(row []string, c int) string {
		if c < 0 || c >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[c])
	}
	var out []*Incoming
	for _, row := range rows[head+1:] {
		at, err := time.ParseInLocation("02/01/2006 15:04:05", cell(row, cols["date"]), vnZone)
		if err != nil {
			continue
		}
		acct := firstNonEmpty(OnlyDigits(cell(row, acctCol)), mainAcct)
		if acct == "" {
			return nil, errors.New("sao kê thiếu số tài khoản")
		}
		in := &Incoming{
			Source:        SourceEmail,
			AccountNumber: acct,
			OccurredAt:    at.UTC(),
			Content:       cell(row, cols["desc"]),
		}
		if v, ok := ParseVND(cell(row, cols["credit"])); ok && v > 0 {
			in.Direction, in.Amount = DirectionIn, v
		} else if v, ok := ParseVND(cell(row, cols["debit"])); ok && v > 0 {
			in.Direction, in.Amount = DirectionOut, v
		} else {
			continue
		}
		if b, ok := ParseVND(cell(row, cols["balance"])); ok {
			if strings.HasPrefix(cell(row, cols["balance"]), "-") {
				b = -b
			}
			in.BalanceAfter = &b
		}
		ref := cell(row, cols["ref"])
		if ref == "" {
			sum := sha256.Sum256([]byte(strings.Join(row, "\x1f")))
			ref = "h:" + hex.EncodeToString(sum[:8])
		}
		in.SourceRef = "stmt:" + acct + ":" + ref
		out = append(out, in)
	}
	if len(out) == 0 {
		return nil, errors.New("sao kê không có giao dịch nào")
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].OccurredAt.Before(out[j].OccurredAt) })
	return out, nil
}
