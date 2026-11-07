package thuchi

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var VNZone = time.FixedZone("ICT", 7*3600)

type QuickItem struct {
	Desc       string
	Amount     int64
	Direction  string
	OccurredAt time.Time
	Line       string
}

var QuickKeywords = []MerchantEntry{
	{[]string{"CAFE", "CA PHE", "CAPHE", "CAFFE", "COFFEE", "CF", "TRA SUA", "TRA DA", "TRA CHANH", "NUOC MIA", "SINH TO", "NUOC EP", "NUOC NGOT", "NUOC SUOI", "BIA", "TRA DAO"}, "", CatAnUong},
	{[]string{"AN SANG", "AN TRUA", "AN TOI", "AN VAT", "AN DEM", "DO AN", "BANH MI", "PHO", "BUN", "BUN BO", "BUN CHA", "COM", "COM TAM", "COM GA", "CHAO", "MI", "MY", "HU TIEU", "XOI", "BANH", "BANH CUON", "TRAI CAY", "HOA QUA", "SNACK", "KEO", "GA RAN", "BURGER", "LAU", "OC"}, "", CatAnUong},
	{[]string{"NHA HANG", "BUFFET", "NUONG", "LAU NUONG", "PIZZA", "NHAU", "AN TIEC", "LIEN HOAN"}, "", CatBuaLon},
	{[]string{"GRAB", "XANH SM", "TAXI", "GUI XE", "VE XE", "XE OM", "XE BUYT", "XE BUS", "BUS", "VE MAY BAY", "VE TAU", "PHI DUONG BO", "PARKING", "RUA XE", "SUA XE", "VA XE", "BOM XE"}, "", CatDiLai},
	{[]string{"XANG", "DO XANG", "DAU NHOT", "THAY NHOT", "SAC XE", "SAC PIN XE"}, "", CatXangXe},
	{[]string{"SIEU THI", "DI CHO", "MUA DO", "QUAN AO", "AO", "AO KHOAC", "GIAY", "DEP", "TUI", "BALO", "DO DUNG", "DO GIA DUNG", "MUA SAM", "BACH HOA"}, "", CatMuaSam},
	{[]string{"MY PHAM", "SON MOI", "SON", "KEM CHONG NANG", "SUA RUA MAT", "NUOC HOA", "DUONG DA", "SKINCARE"}, "", CatMyPham},
	{[]string{"CAT TOC", "GOI DAU", "LAM TOC", "NHUOM TOC", "NAIL", "LAM NAIL", "SPA", "MASSAGE", "WAX"}, "", CatLamDep},
	{[]string{"THUOC", "NHA THUOC", "KHAM", "KHAM BENH", "BENH VIEN", "PHONG KHAM", "NHA KHOA", "VITAMIN", "XET NGHIEM"}, "", CatThuocMen},
	{[]string{"SACH", "HOC PHI", "KHOA HOC", "HOC THEM", "GIAO TRINH", "TAI LIEU"}, "", CatHocTap},
	{[]string{"ICLOUD", "GOOGLE ONE", "APP STORE", "CHPLAY", "CH PLAY", "VPN", "DOMAIN", "HOSTING", "SERVER", "CLAUDE"}, "", CatDichVuOnline},
	{[]string{"XEM PHIM", "PHIM", "GAME", "KARAOKE", "DU LICH", "BOI", "GYM", "CONCERT", "BIDA", "BI A", "VE CONCERT", "CAM TRAI"}, "", CatGiaiTri},
	{[]string{"TIEN DIEN", "TIEN NUOC", "INTERNET", "WIFI", "TIEN MANG", "NAP DIEN THOAI", "NAP CARD", "NAP TIEN DIEN THOAI", "THE CAO", "CUOC DIEN THOAI", "TIEN GAS", "GAS", "TIEN RAC", "PHI DICH VU", "PHI QUAN LY"}, "", CatHoaDon},
	{[]string{"TIEN NHA", "THUE NHA", "SUA NHA", "DIEN MAY", "NOI THAT", "DO NHA", "SUA DIEN", "SUA NUOC"}, "", CatNhaCua},
	{[]string{"DAM CUOI", "MUNG CUOI", "DAM HIEU", "PHONG BI", "LI XI", "BIEU", "MUNG SINH NHAT", "QUA TANG", "TANG QUA"}, "", CatChuyenNguoiKhac},
	{[]string{"PHI CHUYEN KHOAN", "PHI NGAN HANG", "PHI THUONG NIEN", "PHI SMS"}, "", CatPhiNganHang},
	{[]string{"LUONG", "TIEN LUONG", "SALARY", "THUONG TET", "TIEN THUONG", "BONUS"}, "", CatLuong},
	{[]string{"TIEN PHONG", "THU TIEN PHONG", "TIEN TRO"}, "", CatTienPhong},
	{[]string{"DUOC CHO", "DUOC CHUYEN", "NHAN TIEN", "DUOC LI XI", "ME CHO", "BO CHO"}, "", CatDuocChuyen},
	{[]string{"BAN DO", "THANH LY", "TIEN LAI", "LAI TIET KIEM", "HOAN TIEN", "CASHBACK", "THU NHAP"}, "", CatKhacThu},
}

var (
	reAmountFrac = regexp.MustCompile(`^(\d+)(k|tr|m|cu)(\d{1,3})$`)
	reAmountUnit = regexp.MustCompile(`^(\d+(?:[.,]\d+)?)(k|nghin|ngan|ng|tr|trieu|m|cu|lit)$`)
	reAmountSep  = regexp.MustCompile(`^\d{1,3}(?:[.,]\d{3})+$`)
	reAmountNum  = regexp.MustCompile(`^\d{1,10}$`)
	reQuickDate  = regexp.MustCompile(`^(\d{1,2})[/-](\d{1,2})(?:[/-](\d{2}|\d{4}))?$`)
	reLineSplit  = regexp.MustCompile(`[\n;]+`)
	reCommaSplit = regexp.MustCompile(`,\s+`)
)

var amountUnits = map[string]float64{"k": 1e3, "nghin": 1e3, "ngan": 1e3, "ng": 1e3, "tr": 1e6, "trieu": 1e6, "m": 1e6, "cu": 1e6, "lit": 1e5}

const maxQuickAmount = 10_000_000_000

func foldToken(s string) string {
	return strings.Trim(strings.ToLower(Fold(s)), ".,;:!?()[]{}\"'…")
}

func tokenAmount(raw string) (amount int64, strength int, sign int) {
	t := foldToken(raw)
	if strings.HasPrefix(t, "+") {
		sign, t = 1, t[1:]
	} else if strings.HasPrefix(t, "-") {
		sign, t = -1, t[1:]
	}
	currency := false
	for _, suf := range []string{"vnd", "dong", "d"} {
		if rest, ok := strings.CutSuffix(t, suf); ok && rest != "" && rest[len(rest)-1] != '.' {
			if reAmountFrac.MatchString(rest) || reAmountUnit.MatchString(rest) || reAmountSep.MatchString(rest) || reAmountNum.MatchString(rest) {
				t, currency = rest, true
				break
			}
		}
	}
	var v float64
	switch {
	case reAmountFrac.MatchString(t):
		m := reAmountFrac.FindStringSubmatch(t)
		base, _ := strconv.ParseFloat(m[1], 64)
		frac, _ := strconv.ParseFloat(m[3], 64)
		mul := amountUnits[m[2]]
		v, strength = base*mul+frac*mul/math.Pow10(len(m[3])), 3
	case reAmountUnit.MatchString(t):
		m := reAmountUnit.FindStringSubmatch(t)
		n, err := strconv.ParseFloat(strings.ReplaceAll(m[1], ",", "."), 64)
		if err != nil {
			return 0, 0, 0
		}
		v, strength = n*amountUnits[m[2]], 3
	case reAmountSep.MatchString(t):
		n, _ := strconv.ParseFloat(strings.NewReplacer(".", "", ",", "").Replace(t), 64)
		v, strength = n, 2
	case reAmountNum.MatchString(t):
		if len(t) > 1 && t[0] == '0' {
			return 0, 0, 0
		}
		n, _ := strconv.ParseFloat(t, 64)
		switch {
		case currency:
			v, strength = n, 2
		case n >= 1000 && int64(n)%100 == 0:
			v, strength = n, 2
		case n > 0 && n < 1000:
			v, strength = n*1000, 1
		default:
			return 0, 0, 0
		}
	default:
		return 0, 0, 0
	}
	amount = int64(math.Round(v))
	if amount <= 0 || amount > maxQuickAmount {
		return 0, 0, 0
	}
	return amount, strength, sign
}

func isAmountUnitWord(s string) bool {
	_, ok := amountUnits[s]
	return ok || s == "d" || s == "dong" || s == "vnd"
}

type amountSpan struct {
	start, end int
	amount     int64
	strength   int
	sign       int
}

func findAmounts(toks []string) []amountSpan {
	var out []amountSpan
	for i := 0; i < len(toks); i++ {
		amount, strength, sign := tokenAmount(toks[i])
		if strength == 0 {
			continue
		}
		end := i + 1
		if strength < 3 && i+1 < len(toks) && isAmountUnitWord(foldToken(toks[i+1])) {
			if a2, s2, sg2 := tokenAmount(foldToken(toks[i]) + foldToken(toks[i+1])); s2 >= 2 {
				amount, strength, sign, end = a2, s2, sg2, i+2
			}
		}
		out = append(out, amountSpan{start: i, end: end, amount: amount, strength: strength, sign: sign})
		i = end - 1
	}
	return out
}

func ParseAmount(text string) (int64, bool) {
	toks := strings.Fields(text)
	if len(toks) == 0 || len(toks) > 2 {
		return 0, false
	}
	spans := findAmounts(toks)
	if len(spans) != 1 || spans[0].start != 0 || spans[0].end != len(toks) {
		return 0, false
	}
	return spans[0].amount, true
}

func ParseQuick(text string, now time.Time) []QuickItem {
	var out []QuickItem
	for _, line := range reLineSplit.Split(text, -1) {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "•*-–"))
		if line == "" || strings.HasPrefix(line, "/") {
			continue
		}
		parts := reCommaSplit.Split(line, -1)
		var items []QuickItem
		ok := len(parts) > 1
		if ok {
			for _, p := range parts {
				got := parseQuickLine(p, now)
				if len(got) == 0 {
					ok = false
					break
				}
				items = append(items, got...)
			}
		}
		if !ok {
			items = parseQuickLine(line, now)
		}
		out = append(out, items...)
	}
	return out
}

func quickDay(now time.Time, offset int, hour int) time.Time {
	local := now.In(VNZone)
	d := time.Date(local.Year(), local.Month(), local.Day(), hour, 0, 0, 0, VNZone).AddDate(0, 0, offset)
	return d.UTC()
}

func extractQuickDate(toks []string, now time.Time) ([]string, time.Time) {
	at := now.UTC()
	var rest []string
	for i := 0; i < len(toks); i++ {
		f := foldToken(toks[i])
		next := ""
		if i+1 < len(toks) {
			next = foldToken(toks[i+1])
		}
		pair := f + " " + next
		switch {
		case pair == "hom qua" || pair == "toi qua" || pair == "dem qua":
			hour := 12
			if f != "hom" {
				hour = 20
			}
			at = quickDay(now, -1, hour)
			i++
			continue
		case pair == "hom kia":
			at = quickDay(now, -2, 12)
			i++
			continue
		case pair == "hom nay":
			i++
			continue
		case f == "hqua":
			at = quickDay(now, -1, 12)
			continue
		case f == "hkia":
			at = quickDay(now, -2, 12)
			continue
		case f == "hnay":
			continue
		}
		if m := reQuickDate.FindStringSubmatch(f); m != nil {
			local := now.In(VNZone)
			day, _ := strconv.Atoi(m[1])
			month, _ := strconv.Atoi(m[2])
			year := local.Year()
			if m[3] != "" {
				year, _ = strconv.Atoi(m[3])
				if year < 100 {
					year += 2000
				}
			}
			if day >= 1 && day <= 31 && month >= 1 && month <= 12 {
				d := time.Date(year, time.Month(month), day, 12, 0, 0, 0, VNZone)
				if m[3] == "" && d.After(local.AddDate(0, 0, 1)) {
					d = d.AddDate(-1, 0, 0)
				}
				if d.Year() == local.Year() && d.YearDay() == local.YearDay() {
					d = now
				}
				at = d.UTC()
				continue
			}
		}
		rest = append(rest, toks[i])
	}
	return rest, at
}

func cleanDesc(toks []string) string {
	s := strings.Join(toks, " ")
	s = strings.Trim(s, " :-–,.+=")
	return strings.TrimSpace(s)
}

func capitalize(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	r[0] = []rune(strings.ToUpper(string(r[0])))[0]
	return string(r)
}

func parseQuickLine(line string, now time.Time) []QuickItem {
	toks, at := extractQuickDate(strings.Fields(line), now)
	lineSign := 0
	if len(toks) > 0 && (toks[0] == "+" || toks[0] == "-") {
		lineSign = map[string]int{"+": 1, "-": -1}[toks[0]]
		toks = toks[1:]
	}
	spans := findAmounts(toks)
	var strong []amountSpan
	for _, sp := range spans {
		if sp.strength >= 2 {
			strong = append(strong, sp)
		}
	}
	if len(strong) == 0 {
		for _, sp := range spans {
			if sp.end == len(toks) {
				strong = append(strong, sp)
			}
		}
	}
	if len(strong) == 0 {
		return nil
	}
	direction := func(sp amountSpan) string {
		sign := sp.sign
		if sign == 0 {
			sign = lineSign
		}
		switch sign {
		case 1:
			return DirectionIn
		case -1:
			return DirectionOut
		}
		return ""
	}
	item := func(desc string, sp amountSpan) QuickItem {
		return QuickItem{Desc: capitalize(desc), Amount: sp.amount, Direction: direction(sp), OccurredAt: at, Line: strings.TrimSpace(line)}
	}
	if len(strong) > 1 {
		var items []QuickItem
		leading := strong[0].start == 0
		valid := true
		for k, sp := range strong {
			var seg []string
			if leading {
				end := len(toks)
				if k+1 < len(strong) {
					end = strong[k+1].start
				}
				seg = toks[sp.end:end]
			} else {
				start := 0
				if k > 0 {
					start = strong[k-1].end
				}
				seg = toks[start:sp.start]
			}
			desc := cleanDesc(seg)
			if desc == "" {
				valid = false
				break
			}
			items = append(items, item(desc, sp))
		}
		if valid {
			if !leading {
				if tail := cleanDesc(toks[strong[len(strong)-1].end:]); tail != "" {
					items[len(items)-1].Desc += " " + tail
				}
			}
			return items
		}
	}
	sp := strong[len(strong)-1]
	var rest []string
	rest = append(rest, toks[:sp.start]...)
	rest = append(rest, toks[sp.end:]...)
	desc := cleanDesc(rest)
	if desc == "" {
		return nil
	}
	return []QuickItem{item(desc, sp)}
}

func kindOf(direction string) string {
	if direction == DirectionIn {
		return "income"
	}
	return "expense"
}

func (ic *ingestCtx) keywordCategory(text string, fits func(*string) bool) *string {
	var best *string
	bestLen := 0
	for _, dict := range [][]MerchantEntry{ic.keywords, MerchantDictionary} {
		for _, m := range dict {
			for _, k := range m.Keywords {
				if len(k) <= bestLen || !containsWord(text, k) {
					continue
				}
				if id := ic.catID(m.Category); fits(id) {
					best, bestLen = id, len(k)
				}
			}
		}
	}
	return best
}

func (ic *ingestCtx) classifyQuick(desc, direction string) (*string, *Rule) {
	key := Normalize(desc)
	if key == "" {
		return nil, nil
	}
	fits := func(id *string) bool {
		if id == nil {
			return false
		}
		c, ok := ic.categories[*id]
		return ok && (direction == "" || c.Kind == kindOf(direction))
	}
	var contained *Rule
	containedLen := 0
	for _, r := range ic.rules {
		if r.Match != MatchMerchant || r.Origin != OriginLearned || !fits(r.CategoryID) {
			continue
		}
		p := Normalize(r.Pattern)
		if p == key {
			return r.CategoryID, r
		}
		if len(p) >= 2 && len(p) > containedLen && containsWord(key, p) {
			contained, containedLen = r, len(p)
		}
	}
	if contained != nil {
		return contained.CategoryID, contained
	}
	for _, r := range ic.rules {
		if r.Origin == OriginLearned || r.Match == MatchOwnAccount || !fits(r.CategoryID) {
			continue
		}
		if ruleMatches(r, key, key) {
			return r.CategoryID, r
		}
	}
	if id := ic.keywordCategory(key, fits); id != nil {
		return id, nil
	}
	if direction == DirectionIn {
		return ic.catID(CatKhacThu), nil
	}
	return nil, nil
}
