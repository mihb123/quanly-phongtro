package pdftext

import (
	"unicode/utf16"
)

type font struct {
	codeLengths  []int
	toUnicode    map[uint32]string
	widths       map[uint32]float64
	defaultWidth float64
	simple       bool
}

type glyph struct {
	code  uint32
	text  string
	width float64
	space bool
}

func (d *document) loadFont(v any) *font {
	fd := d.dictOf(v)
	f := &font{toUnicode: map[uint32]string{}, widths: map[uint32]float64{}, defaultWidth: 500}
	if fd == nil {
		f.simple = true
		f.codeLengths = []int{1}
		return f
	}

	if fd["Subtype"] == name("Type0") {
		f.codeLengths = []int{2}
		f.defaultWidth = 1000
		if descendants, ok := d.resolve(fd["DescendantFonts"]).([]any); ok && len(descendants) > 0 {
			cid := d.dictOf(descendants[0])
			if dw, ok := d.resolve(cid["DW"]).(float64); ok {
				f.defaultWidth = dw
			}
			d.loadCIDWidths(f, cid["W"])
		}
	} else {
		f.simple = true
		f.codeLengths = []int{1}
		firstChar := int(num(d.resolve(fd["FirstChar"])))
		if widths, ok := d.resolve(fd["Widths"]).([]any); ok {
			for i, w := range widths {
				f.widths[uint32(firstChar+i)] = num(d.resolve(w))
			}
		}
	}

	if s, ok := d.resolve(fd["ToUnicode"]).(*stream); ok {
		if data, err := d.decode(s); err == nil {
			f.parseCMap(data)
		}
	}
	return f
}

func (d *document) loadCIDWidths(f *font, v any) {
	arr, _ := d.resolve(v).([]any)
	for i := 0; i < len(arr); {
		first := uint32(num(d.resolve(arr[i])))
		if i+1 >= len(arr) {
			return
		}
		if list, ok := d.resolve(arr[i+1]).([]any); ok {
			for j, w := range list {
				f.widths[first+uint32(j)] = num(d.resolve(w))
			}
			i += 2
			continue
		}
		if i+2 >= len(arr) {
			return
		}
		last := uint32(num(d.resolve(arr[i+1])))
		w := num(d.resolve(arr[i+2]))
		for c := first; c <= last && c-first < 65536; c++ {
			f.widths[c] = w
		}
		i += 3
	}
}

func (f *font) parseCMap(data []byte) {
	l := &lexer{buf: data}
	var operands []any
	lengths := map[int]bool{}
	for {
		tok, ok := l.object()
		if !ok {
			break
		}
		kw, isKw := tok.(keyword)
		if !isKw {
			operands = append(operands, tok)
			continue
		}
		switch kw {
		case "endcodespacerange":
			for i := 0; i+1 < len(operands); i += 2 {
				if lo, ok := operands[i].([]byte); ok && len(lo) > 0 {
					lengths[len(lo)] = true
				}
			}
		case "endbfchar":
			for i := 0; i+1 < len(operands); i += 2 {
				src, ok1 := operands[i].([]byte)
				dst, ok2 := operands[i+1].([]byte)
				if ok1 && ok2 {
					f.toUnicode[codeOf(src)] = utf16BE(dst)
				}
			}
		case "endbfrange":
			for i := 0; i+2 < len(operands); i += 3 {
				lo, ok1 := operands[i].([]byte)
				hi, ok2 := operands[i+1].([]byte)
				if !ok1 || !ok2 {
					continue
				}
				start, end := codeOf(lo), codeOf(hi)
				if end < start || end-start > 65535 {
					continue
				}
				switch dst := operands[i+2].(type) {
				case []byte:
					base := []rune(utf16BE(dst))
					for c := start; c <= end; c++ {
						if len(base) == 0 {
							break
						}
						r := append([]rune(nil), base...)
						r[len(r)-1] += rune(c - start)
						f.toUnicode[c] = string(r)
					}
				case []any:
					for j, item := range dst {
						if b, ok := item.([]byte); ok && start+uint32(j) <= end {
							f.toUnicode[start+uint32(j)] = utf16BE(b)
						}
					}
				}
			}
		}
		operands = operands[:0]
	}
	if len(lengths) > 0 {
		f.codeLengths = f.codeLengths[:0]
		for n := 1; n <= 4; n++ {
			if lengths[n] {
				f.codeLengths = append(f.codeLengths, n)
			}
		}
	}
}

func (f *font) decode(s []byte) []glyph {
	var out []glyph
	for i := 0; i < len(s); {
		n := f.codeLengths[0]
		if len(f.codeLengths) > 1 {
			for _, candidate := range f.codeLengths {
				if _, ok := f.toUnicode[codeOf(s[i:min(len(s), i+candidate)])]; ok {
					n = candidate
					break
				}
			}
		}
		if i+n > len(s) {
			n = len(s) - i
		}
		code := codeOf(s[i : i+n])
		i += n

		text, ok := f.toUnicode[code]
		if !ok && f.simple {
			text = winAnsi(byte(code))
		}
		width, ok := f.widths[code]
		if !ok {
			width = f.defaultWidth
		}
		out = append(out, glyph{code: code, text: text, width: width, space: n == 1 && code == 32})
	}
	return out
}

func codeOf(b []byte) uint32 {
	var c uint32
	for _, x := range b {
		c = c<<8 | uint32(x)
	}
	return c
}

func utf16BE(b []byte) string {
	if len(b)%2 == 1 {
		b = append(b, 0)
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	return string(utf16.Decode(units))
}

var winAnsiHigh = map[byte]rune{
	0x80: '€', 0x82: '‚', 0x83: 'ƒ', 0x84: '„', 0x85: '…', 0x86: '†', 0x87: '‡', 0x88: 'ˆ',
	0x89: '‰', 0x8A: 'Š', 0x8B: '‹', 0x8C: 'Œ', 0x8E: 'Ž', 0x91: '‘', 0x92: '’', 0x93: '“',
	0x94: '”', 0x95: '•', 0x96: '–', 0x97: '—', 0x98: '˜', 0x99: '™', 0x9A: 'š', 0x9B: '›',
	0x9C: 'œ', 0x9E: 'ž', 0x9F: 'Ÿ',
}

func winAnsi(b byte) string {
	if r, ok := winAnsiHigh[b]; ok {
		return string(r)
	}
	if b < 0x20 {
		return ""
	}
	return string(rune(b))
}
