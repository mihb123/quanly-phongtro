package pdftext

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"regexp"
	"strconv"
)

var (
	ErrNotPDF    = errors.New("pdftext: not a PDF file")
	ErrEncrypted = errors.New("pdftext: encrypted PDFs are not supported")
	ErrNoPages   = errors.New("pdftext: no pages found")
)

const maxStreamSize = 64 << 20

var objHeader = regexp.MustCompile(`(\d+)\s+(\d+)\s+obj\b`)

type document struct {
	data    []byte
	objects map[int]any
	trailer dict
}

func parseDocument(data []byte) (*document, error) {
	if !bytes.HasPrefix(bytes.TrimLeft(data[:min(len(data), 1024)], "\x00\r\n\t "), []byte("%PDF")) {
		return nil, ErrNotPDF
	}
	doc := &document{data: data, objects: map[int]any{}, trailer: dict{}}
	parsedUntil := 0
	for _, m := range objHeader.FindAllSubmatchIndex(data, -1) {
		if m[0] < parsedUntil || (m[0] > 0 && !isWhite(data[m[0]-1]) && !isDelim(data[m[0]-1])) {
			continue
		}
		num, _ := strconv.Atoi(string(data[m[2]:m[3]]))
		l := &lexer{buf: data, pos: m[1]}
		obj, ok := l.object()
		if !ok {
			continue
		}
		if d, isDict := obj.(dict); isDict {
			if s, isStream := doc.readStream(l, d); isStream {
				obj = s
			}
		}
		doc.objects[num] = obj
		parsedUntil = l.pos
	}

	for _, idx := range regexp.MustCompile(`trailer\s*<<`).FindAllIndex(data, -1) {
		l := &lexer{buf: data, pos: idx[1] - 2}
		if d, ok := l.object(); ok {
			for k, v := range d.(dict) {
				doc.trailer[k] = v
			}
		}
	}
	for _, obj := range doc.objects {
		if s, ok := obj.(*stream); ok && s.dict["Type"] == name("XRef") {
			for _, k := range []name{"Root", "Encrypt"} {
				if v, has := s.dict[k]; has {
					if _, set := doc.trailer[k]; !set {
						doc.trailer[k] = v
					}
				}
			}
		}
	}
	if _, encrypted := doc.trailer["Encrypt"]; encrypted {
		return nil, ErrEncrypted
	}
	doc.expandObjectStreams()
	return doc, nil
}

func (d *document) readStream(l *lexer, header dict) (*stream, bool) {
	save := l.pos
	l.skipSpace()
	if !bytes.HasPrefix(l.buf[l.pos:], []byte("stream")) {
		l.pos = save
		return nil, false
	}
	start := l.pos + len("stream")
	if start < len(l.buf) && l.buf[start] == '\r' {
		start++
	}
	if start < len(l.buf) && l.buf[start] == '\n' {
		start++
	}
	end := -1
	if n, ok := header["Length"].(float64); ok && start+int(n) <= len(l.buf) {
		candidate := start + int(n)
		rest := bytes.TrimLeft(l.buf[candidate:min(len(l.buf), candidate+20)], "\r\n \t")
		if bytes.HasPrefix(rest, []byte("endstream")) {
			end = candidate
		}
	}
	if end < 0 {
		idx := bytes.Index(l.buf[start:], []byte("endstream"))
		if idx < 0 {
			return nil, false
		}
		end = start + idx
		for end > start && (l.buf[end-1] == '\n' || l.buf[end-1] == '\r') {
			end--
		}
	}
	l.pos = end
	return &stream{dict: header, data: l.buf[start:end]}, true
}

func (d *document) expandObjectStreams() {
	for _, obj := range d.objects {
		s, ok := obj.(*stream)
		if !ok || s.dict["Type"] != name("ObjStm") {
			continue
		}
		data, err := d.decode(s)
		if err != nil {
			continue
		}
		count := int(num(d.resolve(s.dict["N"])))
		first := int(num(d.resolve(s.dict["First"])))
		l := &lexer{buf: data}
		type entry struct{ num, offset int }
		entries := make([]entry, 0, count)
		for i := 0; i < count; i++ {
			n, ok1 := l.next()
			off, ok2 := l.next()
			if !ok1 || !ok2 {
				break
			}
			entries = append(entries, entry{int(num(n)), int(num(off))})
		}
		for _, e := range entries {
			if _, exists := d.objects[e.num]; exists || first+e.offset >= len(data) {
				continue
			}
			ol := &lexer{buf: data, pos: first + e.offset}
			if v, ok := ol.object(); ok {
				d.objects[e.num] = v
			}
		}
	}
}

func (d *document) resolve(v any) any {
	for i := 0; i < 32; i++ {
		r, ok := v.(ref)
		if !ok {
			return v
		}
		v = d.objects[r.num]
	}
	return nil
}

func (d *document) dictOf(v any) dict {
	switch t := d.resolve(v).(type) {
	case dict:
		return t
	case *stream:
		return t.dict
	}
	return nil
}

func (d *document) decode(s *stream) ([]byte, error) {
	data := s.data
	if n, ok := d.resolve(s.dict["Length"]).(float64); ok && int(n) < len(data) && int(n) >= 0 {
		data = data[:int(n)]
	}
	var filters []any
	switch f := d.resolve(s.dict["Filter"]).(type) {
	case name:
		filters = []any{f}
	case []any:
		filters = f
	}
	for _, f := range filters {
		switch d.resolve(f) {
		case name("FlateDecode"), name("Fl"):
			r, err := zlib.NewReader(bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			out, err := io.ReadAll(io.LimitReader(r, maxStreamSize))
			if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && len(out) == 0 {
				return nil, err
			}
			data = out
		default:
			return nil, errors.New("pdftext: unsupported filter")
		}
	}
	return data, nil
}

func (d *document) pages() []dict {
	root := d.dictOf(d.trailer["Root"])
	if root == nil {
		for _, obj := range d.objects {
			if dd, ok := obj.(dict); ok && dd["Type"] == name("Catalog") {
				root = dd
			}
		}
	}
	var out []dict
	inheritable := []name{"Resources", "MediaBox", "Rotate"}
	var walk func(node, inherited dict, depth int)
	walk = func(node, inherited dict, depth int) {
		if node == nil || depth > 64 {
			return
		}
		attrs := dict{}
		for _, k := range inheritable {
			if v, ok := node[k]; ok {
				attrs[k] = v
			} else if v, ok := inherited[k]; ok {
				attrs[k] = v
			}
		}
		if node["Type"] == name("Page") {
			page := dict{}
			for k, v := range node {
				page[k] = v
			}
			for k, v := range attrs {
				page[k] = v
			}
			out = append(out, page)
			return
		}
		kids, _ := d.resolve(node["Kids"]).([]any)
		for _, kid := range kids {
			walk(d.dictOf(kid), attrs, depth+1)
		}
	}
	if root != nil {
		walk(d.dictOf(root["Pages"]), dict{}, 0)
	}
	return out
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}
