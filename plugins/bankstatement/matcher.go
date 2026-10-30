package bankstatement

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/mihb123/quanly-phongtro/internal/model"
	"golang.org/x/text/unicode/norm"
)

const (
	MatchAmount           = "AMOUNT"
	MatchAmountHinted     = "AMOUNT_HINT"
	MatchAmountNearHinted = "AMOUNT_NEAR_HINT"
)

const minIdentityHint = 2

type matchResult struct {
	Transaction Transaction
	Invoice     *model.OutstandingInvoice
	Method      string
	Candidates  []model.OutstandingInvoice
}

type scoredInvoice struct {
	index int
	score int
}

func matchTransactions(txs []Transaction, invoices []model.OutstandingInvoice, maxAmountDiff int64) []matchResult {
	results := make([]matchResult, len(txs))
	assigned := make([]bool, len(invoices))
	resolved := make([]bool, len(txs))
	for i, tx := range txs {
		results[i].Transaction = tx
	}

	candidates := func(tx Transaction) []int {
		var out []int
		month := tx.Date.Format("2006-01")
		for i, inv := range invoices {
			if !assigned[i] && int64(math.Round(inv.TotalAmount)) == tx.Amount && inv.Period == month {
				out = append(out, i)
			}
		}
		return out
	}
	assign := func(txIndex, invIndex int, method string) {
		inv := invoices[invIndex]
		results[txIndex].Invoice = &inv
		results[txIndex].Method = method
		assigned[invIndex] = true
		resolved[txIndex] = true
	}

	for progress := true; progress; {
		progress = false
		proposals := map[int][]int{}
		for i, tx := range txs {
			if resolved[i] {
				continue
			}
			if best, ok := bestHinted(tx, invoices, candidates(tx)); ok {
				proposals[best] = append(proposals[best], i)
			}
		}
		for invIndex, txIndexes := range proposals {
			if len(txIndexes) == 1 {
				assign(txIndexes[0], invIndex, MatchAmountHinted)
				progress = true
			}
		}
	}

	for i, tx := range txs {
		if resolved[i] {
			continue
		}
		cands := candidates(tx)
		switch len(cands) {
		case 0:
		case 1:
			assign(i, cands[0], MatchAmount)
		default:
			for _, c := range cands {
				results[i].Candidates = append(results[i].Candidates, invoices[c])
			}
		}
	}

	nearCandidates := func(tx Transaction) []int {
		var out []int
		month := tx.Date.Format("2006-01")
		for i, inv := range invoices {
			diff := tx.Amount - int64(math.Round(inv.TotalAmount))
			if !assigned[i] && diff != 0 && diff > -maxAmountDiff && diff < maxAmountDiff && inv.Period == month {
				out = append(out, i)
			}
		}
		return out
	}
	for progress := true; progress; {
		progress = false
		proposals := map[int][]int{}
		for i, tx := range txs {
			if resolved[i] || len(results[i].Candidates) > 0 {
				continue
			}
			if best, ok := bestIdentified(tx, invoices, nearCandidates(tx)); ok {
				proposals[best] = append(proposals[best], i)
			}
		}
		for invIndex, txIndexes := range proposals {
			if len(txIndexes) == 1 {
				assign(txIndexes[0], invIndex, MatchAmountNearHinted)
				progress = true
			}
		}
	}
	return results
}

func bestHinted(tx Transaction, invoices []model.OutstandingInvoice, cands []int) (int, bool) {
	tokens := foldTokens(tx.Description)
	return uniqueBest(cands, 1, func(c int) int { return hintScore(tokens, invoices[c]) })
}

func bestIdentified(tx Transaction, invoices []model.OutstandingInvoice, cands []int) (int, bool) {
	tokens := foldTokens(tx.Description)
	return uniqueBest(cands, minIdentityHint, func(c int) int { return identityScore(tokens, invoices[c]) })
}

func uniqueBest(cands []int, minScore int, score func(int) int) (int, bool) {
	var scored []scoredInvoice
	for _, c := range cands {
		scored = append(scored, scoredInvoice{index: c, score: score(c)})
	}
	best, bestScore, ties := -1, 0, 0
	for _, s := range scored {
		switch {
		case s.score > bestScore:
			best, bestScore, ties = s.index, s.score, 1
		case s.score == bestScore && s.score > 0:
			ties++
		}
	}
	return best, bestScore >= minScore && ties == 1
}

func hintScore(tokens []string, inv model.OutstandingInvoice) int {
	score := identityScore(tokens, inv)
	if mentionsMonth(tokens, inv.Period) {
		score++
	}
	return score
}

func identityScore(tokens []string, inv model.OutstandingInvoice) int {
	score := 0
	if mentionsRoom(tokens, inv.RoomName) {
		score += 4
	}
	best := 0
	for _, name := range inv.TenantNames {
		best = max(best, nameScore(tokens, foldTokens(name)))
	}
	return score + best
}

func nameScore(tokens, nameTokens []string) int {
	if len(nameTokens) < 2 {
		return 0
	}
	if containsSequence(tokens, nameTokens) {
		return 3
	}
	inName := map[string]bool{}
	for _, t := range nameTokens {
		inName[t] = true
	}
	given := nameTokens[len(nameTokens)-1]
	for i := 0; i+1 < len(tokens); i++ {
		a, b := tokens[i], tokens[i+1]
		if a != b && inName[a] && inName[b] && (a == given || b == given) {
			return 2
		}
	}
	return 0
}

var roomPrefixes = map[string]bool{"phong": true, "p": true, "ph": true, "room": true, "r": true}

func mentionsRoom(tokens []string, roomName string) bool {
	nameTokens := foldTokens(roomName)
	for len(nameTokens) > 1 && roomPrefixes[nameTokens[0]] {
		nameTokens = nameTokens[1:]
	}
	key := strings.Join(nameTokens, "")
	if key == "" || (len(nameTokens) == 1 && roomPrefixes[key]) {
		return false
	}
	standalone := len(key) >= 3 && !isYear(key)
	for i, t := range tokens {
		if t == "p"+key || t == "phong"+key || t == "room"+key {
			return true
		}
		if i+len(nameTokens) <= len(tokens) && strings.Join(tokens[i:i+len(nameTokens)], "") == key {
			if standalone || (i > 0 && roomPrefixes[tokens[i-1]]) {
				return true
			}
		}
	}
	return false
}

func mentionsMonth(tokens []string, period string) bool {
	if len(period) != 7 {
		return false
	}
	month, err := strconv.Atoi(period[5:])
	if err != nil {
		return false
	}
	for i := 0; i+1 < len(tokens); i++ {
		if tokens[i] != "thang" && tokens[i] != "t" {
			continue
		}
		if m, err := strconv.Atoi(tokens[i+1]); err == nil && m == month {
			return true
		}
	}
	for _, t := range tokens {
		if strings.HasPrefix(t, "thang") {
			if m, err := strconv.Atoi(t[len("thang"):]); err == nil && m == month {
				return true
			}
		}
	}
	return false
}

func isYear(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 1900 && n <= 2100
}

func containsSequence(tokens, seq []string) bool {
	for i := 0; i+len(seq) <= len(tokens); i++ {
		match := true
		for j := range seq {
			if tokens[i+j] != seq[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func foldTokens(s string) []string {
	var sb strings.Builder
	for _, r := range norm.NFD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == 'đ' || r == 'Đ':
			sb.WriteRune('d')
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
		default:
			sb.WriteRune(' ')
		}
	}
	return strings.Fields(sb.String())
}
