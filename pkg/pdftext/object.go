package pdftext

import (
	"bytes"
	"strconv"
)

type name string

type ref struct{ num, gen int }

type dict map[name]any

type stream struct {
	dict dict
	data []byte
}

type keyword string

type lexer struct {
	buf []byte
	pos int
}

func isWhite(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t' || c == '\f' || c == 0
}

func isDelim(c byte) bool {
	return bytes.IndexByte([]byte("()<>[]{}/%"), c) >= 0
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.buf) {
		c := l.buf[l.pos]
		if isWhite(c) {
			l.pos++
			continue
		}
		if c == '%' {
			for l.pos < len(l.buf) && l.buf[l.pos] != '\n' && l.buf[l.pos] != '\r' {
				l.pos++
			}
			continue
		}
		return
	}
}

func (l *lexer) next() (any, bool) {
	l.skipSpace()
	if l.pos >= len(l.buf) {
		return nil, false
	}
	c := l.buf[l.pos]
	switch {
	case c == '/':
		l.pos++
		start := l.pos
		for l.pos < len(l.buf) && !isWhite(l.buf[l.pos]) && !isDelim(l.buf[l.pos]) {
			l.pos++
		}
		return name(decodeNameEscapes(l.buf[start:l.pos])), true
	case c == '(':
		return l.literalString(), true
	case c == '<':
		if l.pos+1 < len(l.buf) && l.buf[l.pos+1] == '<' {
			l.pos += 2
			return keyword("<<"), true
		}
		return l.hexString(), true
	case c == '>':
		if l.pos+1 < len(l.buf) && l.buf[l.pos+1] == '>' {
			l.pos += 2
			return keyword(">>"), true
		}
		l.pos++
		return keyword(">"), true
	case c == '[' || c == ']' || c == '{' || c == '}':
		l.pos++
		return keyword(string(c)), true
	}
	start := l.pos
	for l.pos < len(l.buf) && !isWhite(l.buf[l.pos]) && !isDelim(l.buf[l.pos]) {
		l.pos++
	}
	if start == l.pos {
		l.pos++
		return keyword(string(c)), true
	}
	word := string(l.buf[start:l.pos])
	if n, err := strconv.ParseFloat(word, 64); err == nil && (word[0] == '-' || word[0] == '+' || word[0] == '.' || (word[0] >= '0' && word[0] <= '9')) {
		return n, true
	}
	switch word {
	case "true":
		return true, true
	case "false":
		return false, true
	case "null":
		return nil, true
	}
	return keyword(word), true
}

func decodeNameEscapes(raw []byte) string {
	if bytes.IndexByte(raw, '#') < 0 {
		return string(raw)
	}
	out := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] == '#' && i+2 < len(raw) {
			if v, err := strconv.ParseUint(string(raw[i+1:i+3]), 16, 8); err == nil {
				out = append(out, byte(v))
				i += 2
				continue
			}
		}
		out = append(out, raw[i])
	}
	return string(out)
}

func (l *lexer) literalString() []byte {
	l.pos++
	var out []byte
	depth := 1
	for l.pos < len(l.buf) {
		c := l.buf[l.pos]
		l.pos++
		switch c {
		case '(':
			depth++
			out = append(out, c)
		case ')':
			depth--
			if depth == 0 {
				return out
			}
			out = append(out, c)
		case '\\':
			if l.pos >= len(l.buf) {
				return out
			}
			e := l.buf[l.pos]
			l.pos++
			switch e {
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case '\r':
				if l.pos < len(l.buf) && l.buf[l.pos] == '\n' {
					l.pos++
				}
			case '\n':
			default:
				if e >= '0' && e <= '7' {
					v := int(e - '0')
					for i := 0; i < 2 && l.pos < len(l.buf) && l.buf[l.pos] >= '0' && l.buf[l.pos] <= '7'; i++ {
						v = v*8 + int(l.buf[l.pos]-'0')
						l.pos++
					}
					out = append(out, byte(v))
				} else {
					out = append(out, e)
				}
			}
		default:
			out = append(out, c)
		}
	}
	return out
}

func (l *lexer) hexString() []byte {
	l.pos++
	var digits []byte
	for l.pos < len(l.buf) && l.buf[l.pos] != '>' {
		c := l.buf[l.pos]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			digits = append(digits, c)
		}
		l.pos++
	}
	l.pos++
	if len(digits)%2 == 1 {
		digits = append(digits, '0')
	}
	out := make([]byte, len(digits)/2)
	for i := range out {
		v, _ := strconv.ParseUint(string(digits[2*i:2*i+2]), 16, 8)
		out[i] = byte(v)
	}
	return out
}

func (l *lexer) object() (any, bool) {
	tok, ok := l.next()
	if !ok {
		return nil, false
	}
	switch t := tok.(type) {
	case keyword:
		switch t {
		case "<<":
			d := dict{}
			for {
				key, ok := l.object()
				if !ok {
					return d, true
				}
				if k, isKw := key.(keyword); isKw && k == ">>" {
					return d, true
				}
				keyName, isName := key.(name)
				value, ok := l.object()
				if !ok {
					return d, true
				}
				if isName {
					d[keyName] = value
				}
			}
		case "[":
			var arr []any
			for {
				item, ok := l.object()
				if !ok {
					return arr, true
				}
				if k, isKw := item.(keyword); isKw && k == "]" {
					return arr, true
				}
				arr = append(arr, item)
			}
		}
		return t, true
	case float64:
		save := l.pos
		gen, ok := l.next()
		if g, isNum := gen.(float64); ok && isNum && isInt(t) && isInt(g) {
			if r, ok := l.next(); ok {
				if k, isKw := r.(keyword); isKw && k == "R" {
					return ref{int(t), int(g)}, true
				}
			}
		}
		l.pos = save
		return t, true
	}
	return tok, true
}

func isInt(f float64) bool {
	return f >= 0 && f == float64(int(f))
}
