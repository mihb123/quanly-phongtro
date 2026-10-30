package pdftext

import (
	"math"
	"sort"
	"strings"
)

type Item struct {
	X, Y, EndX, Size float64
	Text             string
}

type Page struct {
	Number        int
	Width, Height float64
	Items         []Item
}

type Line struct {
	Y     float64
	Items []Item
	Text  string
}

func Extract(data []byte) ([]Page, error) {
	doc, err := parseDocument(data)
	if err != nil {
		return nil, err
	}
	pageDicts := doc.pages()
	if len(pageDicts) == 0 {
		return nil, ErrNoPages
	}
	pages := make([]Page, 0, len(pageDicts))
	for i, pd := range pageDicts {
		page := Page{Number: i + 1}
		var llx, lly, urx, ury float64
		if box, ok := doc.resolve(pd["MediaBox"]).([]any); ok && len(box) == 4 {
			llx, lly = num(doc.resolve(box[0])), num(doc.resolve(box[1]))
			urx, ury = num(doc.resolve(box[2])), num(doc.resolve(box[3]))
		}
		page.Width, page.Height = urx-llx, ury-lly
		in := &interpreter{doc: doc, fonts: map[any]*font{}}
		in.gs.scale = 1
		switch (int(num(doc.resolve(pd["Rotate"])))%360 + 360) % 360 {
		case 90:
			in.gs.ctm = matrix{0, -1, 1, 0, -lly, urx}
			page.Width, page.Height = page.Height, page.Width
		case 180:
			in.gs.ctm = matrix{-1, 0, 0, -1, urx, ury}
		case 270:
			in.gs.ctm = matrix{0, 1, -1, 0, ury, -llx}
			page.Width, page.Height = page.Height, page.Width
		default:
			in.gs.ctm = matrix{1, 0, 0, 1, -llx, -lly}
		}
		in.run(doc.contents(pd["Contents"]), doc.dictOf(pd["Resources"]), 0)
		page.Items = in.items
		pages = append(pages, page)
	}
	return pages, nil
}

func (p Page) Text() string {
	lines := p.Lines()
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = l.Text
	}
	return strings.Join(out, "\n")
}

func (p Page) Lines() []Line {
	items := make([]Item, 0, len(p.Items))
	for _, it := range p.Items {
		if strings.TrimSpace(it.Text) != "" {
			items = append(items, it)
		}
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].Y > items[j].Y })

	var lines []Line
	for _, it := range items {
		tolerance := math.Max(it.Size, 1) * 0.4
		if n := len(lines); n > 0 && math.Abs(lines[n-1].Y-it.Y) <= tolerance {
			lines[n-1].Items = append(lines[n-1].Items, it)
			continue
		}
		lines = append(lines, Line{Y: it.Y, Items: []Item{it}})
	}
	for i := range lines {
		sort.SliceStable(lines[i].Items, func(a, b int) bool { return lines[i].Items[a].X < lines[i].Items[b].X })
		var sb strings.Builder
		for j, it := range lines[i].Items {
			if j > 0 {
				prev := lines[i].Items[j-1]
				if it.X-prev.EndX > math.Max(it.Size, 1)*0.2 && !strings.HasSuffix(sb.String(), " ") && !strings.HasPrefix(it.Text, " ") {
					sb.WriteByte(' ')
				}
			}
			sb.WriteString(it.Text)
		}
		lines[i].Text = strings.TrimSpace(sb.String())
	}
	return lines
}

type matrix [6]float64

var identity = matrix{1, 0, 0, 1, 0, 0}

func (m matrix) mul(n matrix) matrix {
	return matrix{
		m[0]*n[0] + m[1]*n[2],
		m[0]*n[1] + m[1]*n[3],
		m[2]*n[0] + m[3]*n[2],
		m[2]*n[1] + m[3]*n[3],
		m[4]*n[0] + m[5]*n[2] + n[4],
		m[4]*n[1] + m[5]*n[3] + n[5],
	}
}

type graphicsState struct {
	ctm         matrix
	font        *font
	size        float64
	charSpacing float64
	wordSpacing float64
	scale       float64
	leading     float64
	rise        float64
}

type interpreter struct {
	doc   *document
	fonts map[any]*font
	gs    graphicsState
	stack []graphicsState
	tm    matrix
	tlm   matrix
	items []Item
}

func (d *document) contents(v any) []byte {
	var parts []any
	switch t := d.resolve(v).(type) {
	case []any:
		parts = t
	case *stream:
		parts = []any{t}
	}
	var out []byte
	for _, p := range parts {
		if s, ok := d.resolve(p).(*stream); ok {
			if data, err := d.decode(s); err == nil {
				out = append(out, data...)
				out = append(out, '\n')
			}
		}
	}
	return out
}

func (in *interpreter) run(content []byte, resources dict, depth int) {
	if depth > 8 {
		return
	}
	l := &lexer{buf: content}
	var ops []any
	for {
		tok, ok := l.object()
		if !ok {
			return
		}
		op, isOp := tok.(keyword)
		if !isOp {
			ops = append(ops, tok)
			continue
		}
		if op == "BI" {
			l.skipInlineImage()
			ops = ops[:0]
			continue
		}
		in.apply(string(op), ops, resources, depth)
		ops = ops[:0]
	}
}

func (l *lexer) skipInlineImage() {
	for l.pos+2 < len(l.buf) {
		if l.buf[l.pos] == 'E' && l.buf[l.pos+1] == 'I' && isWhite(l.buf[l.pos-1]) && (isWhite(l.buf[l.pos+2]) || l.pos+2 == len(l.buf)) {
			l.pos += 2
			return
		}
		l.pos++
	}
	l.pos = len(l.buf)
}

func (in *interpreter) apply(op string, ops []any, resources dict, depth int) {
	f := func(i int) float64 {
		if i < len(ops) {
			return num(ops[i])
		}
		return 0
	}
	switch op {
	case "q":
		in.stack = append(in.stack, in.gs)
	case "Q":
		if n := len(in.stack); n > 0 {
			in.gs = in.stack[n-1]
			in.stack = in.stack[:n-1]
		}
	case "cm":
		if len(ops) == 6 {
			in.gs.ctm = matrix{f(0), f(1), f(2), f(3), f(4), f(5)}.mul(in.gs.ctm)
		}
	case "BT":
		in.tm, in.tlm = identity, identity
	case "Tf":
		if len(ops) == 2 {
			in.gs.font = in.font(resources, ops[0])
			in.gs.size = f(1)
		}
	case "Tc":
		in.gs.charSpacing = f(0)
	case "Tw":
		in.gs.wordSpacing = f(0)
	case "Tz":
		in.gs.scale = f(0) / 100
	case "TL":
		in.gs.leading = f(0)
	case "Ts":
		in.gs.rise = f(0)
	case "Td":
		in.moveLine(f(0), f(1))
	case "TD":
		in.gs.leading = -f(1)
		in.moveLine(f(0), f(1))
	case "Tm":
		if len(ops) == 6 {
			in.tm = matrix{f(0), f(1), f(2), f(3), f(4), f(5)}
			in.tlm = in.tm
		}
	case "T*":
		in.moveLine(0, -in.gs.leading)
	case "Tj":
		if len(ops) > 0 {
			in.show(ops[0])
		}
	case "'":
		in.moveLine(0, -in.gs.leading)
		if len(ops) > 0 {
			in.show(ops[0])
		}
	case "\"":
		if len(ops) == 3 {
			in.gs.wordSpacing, in.gs.charSpacing = f(0), f(1)
			in.moveLine(0, -in.gs.leading)
			in.show(ops[2])
		}
	case "TJ":
		if len(ops) > 0 {
			in.showArray(ops[0])
		}
	case "Do":
		if len(ops) > 0 {
			in.drawXObject(resources, ops[0], depth)
		}
	}
}

func (in *interpreter) moveLine(tx, ty float64) {
	in.tlm = matrix{1, 0, 0, 1, tx, ty}.mul(in.tlm)
	in.tm = in.tlm
}

func (in *interpreter) font(resources dict, key any) *font {
	fonts := in.doc.dictOf(resources["Font"])
	n, _ := key.(name)
	fontRef := fonts[n]
	cacheKey := fontRef
	if _, isRef := fontRef.(ref); !isRef {
		cacheKey = n
	}
	if cached, ok := in.fonts[cacheKey]; ok {
		return cached
	}
	loaded := in.doc.loadFont(fontRef)
	in.fonts[cacheKey] = loaded
	return loaded
}

func (in *interpreter) showArray(v any) {
	arr, _ := v.([]any)
	var item *Item
	for _, el := range arr {
		switch t := el.(type) {
		case []byte:
			next := in.showRun(t)
			if next == nil {
				continue
			}
			if item == nil {
				item = next
			} else {
				if next.X-item.EndX > math.Max(item.Size, 1)*0.2 && !strings.HasSuffix(item.Text, " ") {
					item.Text += " "
				}
				item.Text += next.Text
				item.EndX = next.EndX
			}
		case float64:
			in.advance(-t / 1000 * in.gs.size * in.gs.scale)
		}
	}
	if item != nil {
		in.items = append(in.items, *item)
	}
}

func (in *interpreter) show(v any) {
	if s, ok := v.([]byte); ok {
		if item := in.showRun(s); item != nil {
			in.items = append(in.items, *item)
		}
	}
}

func (in *interpreter) showRun(s []byte) *Item {
	if in.gs.font == nil {
		return nil
	}
	start, size := in.origin()
	var sb strings.Builder
	for _, g := range in.gs.font.decode(s) {
		sb.WriteString(g.text)
		tx := g.width/1000*in.gs.size + in.gs.charSpacing
		if g.space {
			tx += in.gs.wordSpacing
		}
		in.advance(tx * in.gs.scale)
	}
	end, _ := in.origin()
	if sb.Len() == 0 {
		return nil
	}
	return &Item{X: start[0], Y: start[1], EndX: end[0], Size: size, Text: sb.String()}
}

func (in *interpreter) advance(tx float64) {
	in.tm = matrix{1, 0, 0, 1, tx, 0}.mul(in.tm)
}

func (in *interpreter) origin() ([2]float64, float64) {
	trm := matrix{1, 0, 0, 1, 0, in.gs.rise}.mul(in.tm).mul(in.gs.ctm)
	size := in.gs.size * math.Hypot(trm[2], trm[3])
	return [2]float64{trm[4], trm[5]}, math.Abs(size)
}

func (in *interpreter) drawXObject(resources dict, key any, depth int) {
	n, _ := key.(name)
	s, ok := in.doc.resolve(in.doc.dictOf(resources["XObject"])[n]).(*stream)
	if !ok || s.dict["Subtype"] != name("Form") {
		return
	}
	data, err := in.doc.decode(s)
	if err != nil {
		return
	}
	saved, savedTM, savedTLM := in.gs, in.tm, in.tlm
	if m, ok := in.doc.resolve(s.dict["Matrix"]).([]any); ok && len(m) == 6 {
		var fm matrix
		for i := range fm {
			fm[i] = num(in.doc.resolve(m[i]))
		}
		in.gs.ctm = fm.mul(in.gs.ctm)
	}
	formResources := in.doc.dictOf(s.dict["Resources"])
	if formResources == nil {
		formResources = resources
	}
	in.run(data, formResources, depth+1)
	in.gs, in.tm, in.tlm = saved, savedTM, savedTLM
}
