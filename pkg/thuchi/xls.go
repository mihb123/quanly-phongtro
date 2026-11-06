package thuchi

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"
	"unicode/utf16"
)

var (
	errNotXLS       = errors.New("không phải file Excel 97-2003 (.xls)")
	errXLSEncrypted = errors.New("file Excel đang đặt mật khẩu, hãy xuất lại không mật khẩu")
	oleMagic        = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
)

const (
	cfbEndOfChain = 0xFFFFFFFE
	cfbMaxRegSect = 0xFFFFFFFA
	xlsMaxRows    = 65536
	xlsMaxCols    = 256
)

type XLSSheet struct {
	Name string
	Rows [][]string
}

func IsXLS(data []byte) bool {
	return len(data) >= 512 && string(data[:8]) == string(oleMagic)
}

func ReadXLS(data []byte) ([]XLSSheet, error) {
	if !IsXLS(data) {
		return nil, errNotXLS
	}
	wb, err := cfbStream(data, "Workbook")
	if err != nil {
		if _, e2 := cfbStream(data, "Book"); e2 == nil {
			return nil, errors.New("file Excel 5.0/95 chưa được hỗ trợ")
		}
		return nil, err
	}
	return parseBIFF8(wb)
}

type cfb struct {
	data   []byte
	shift  uint
	fat    []uint32
	sector int
}

func (c *cfb) sectorBytes(id uint32) []byte {
	buf := make([]byte, c.sector)
	off := (int(id) + 1) << c.shift
	if off < len(c.data) {
		copy(buf, c.data[off:])
	}
	return buf
}

func (c *cfb) chain(start uint32, table []uint32) []uint32 {
	var ids []uint32
	seen := map[uint32]bool{}
	for id := start; id <= cfbMaxRegSect && int(id) < len(table) && !seen[id]; id = table[id] {
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

func (c *cfb) read(start uint32) []byte {
	var out []byte
	for _, id := range c.chain(start, c.fat) {
		out = append(out, c.sectorBytes(id)...)
	}
	return out
}

func u32s(b []byte) []uint32 {
	out := make([]uint32, len(b)/4)
	for i := range out {
		out[i] = binary.LittleEndian.Uint32(b[i*4:])
	}
	return out
}

func cfbStream(data []byte, want string) ([]byte, error) {
	le := binary.LittleEndian
	shift := uint(le.Uint16(data[0x1E:]))
	if shift != 9 && shift != 12 {
		return nil, errNotXLS
	}
	c := &cfb{data: data, shift: shift, sector: 1 << shift}
	numFAT := int(le.Uint32(data[0x2C:]))
	var fatIDs []uint32
	for i := 0; i < 109 && len(fatIDs) < numFAT; i++ {
		fatIDs = append(fatIDs, le.Uint32(data[0x4C+i*4:]))
	}
	per := c.sector/4 - 1
	seen := map[uint32]bool{}
	for d := le.Uint32(data[0x44:]); d <= cfbMaxRegSect && len(fatIDs) < numFAT && !seen[d]; {
		seen[d] = true
		ent := u32s(c.sectorBytes(d))
		for i := 0; i < per && len(fatIDs) < numFAT; i++ {
			fatIDs = append(fatIDs, ent[i])
		}
		d = ent[per]
	}
	maxFAT := (len(data)>>shift + 1) * 4
	for _, id := range fatIDs {
		if id > cfbMaxRegSect || len(c.fat) > maxFAT {
			continue
		}
		c.fat = append(c.fat, u32s(c.sectorBytes(id))...)
	}
	dir := c.read(le.Uint32(data[0x30:]))
	if len(dir) < 128 {
		return nil, errNotXLS
	}
	rootStart := le.Uint32(dir[0x74:])
	for off := 0; off+128 <= len(dir); off += 128 {
		e := dir[off : off+128]
		if e[0x42] != 2 {
			continue
		}
		n := int(le.Uint16(e[0x40:]))
		if n < 2 || n > 64 {
			continue
		}
		name := make([]uint16, n/2-1)
		for i := range name {
			name[i] = le.Uint16(e[i*2:])
		}
		if string(utf16.Decode(name)) != want {
			continue
		}
		start := le.Uint32(e[0x74:])
		size := int(le.Uint32(e[0x78:]))
		var raw []byte
		if size < int(le.Uint32(data[0x38:])) {
			mini := c.read(rootStart)
			miniFAT := u32s(c.read(le.Uint32(data[0x3C:])))
			for _, id := range c.chain(start, miniFAT) {
				off := int(id) * 64
				if off+64 <= len(mini) {
					raw = append(raw, mini[off:off+64]...)
				}
			}
		} else {
			raw = c.read(start)
		}
		if size > len(raw) {
			return nil, fmt.Errorf("stream %s bị cắt cụt", want)
		}
		return raw[:size], nil
	}
	return nil, fmt.Errorf("không thấy stream %s", want)
}

type biffRec struct {
	id   uint16
	data []byte
	off  int
}

func biffRecords(wb []byte) []biffRec {
	var out []biffRec
	for off := 0; off+4 <= len(wb); {
		id := binary.LittleEndian.Uint16(wb[off:])
		n := int(binary.LittleEndian.Uint16(wb[off+2:]))
		if off+4+n > len(wb) {
			break
		}
		out = append(out, biffRec{id: id, data: wb[off+4 : off+4+n], off: off})
		off += 4 + n
	}
	return out
}

type biffCursor struct {
	recs [][]byte
	i    int
	pos  int
}

func (c *biffCursor) bytes(n int) ([]byte, bool) {
	out := make([]byte, 0, n)
	for len(out) < n {
		if c.i >= len(c.recs) {
			return out, false
		}
		if c.pos >= len(c.recs[c.i]) {
			c.i++
			c.pos = 0
			continue
		}
		take := min(n-len(out), len(c.recs[c.i])-c.pos)
		out = append(out, c.recs[c.i][c.pos:c.pos+take]...)
		c.pos += take
	}
	return out, true
}

func (c *biffCursor) chars(cch int, high bool) (string, bool) {
	units := make([]uint16, 0, cch)
	for len(units) < cch {
		if c.i >= len(c.recs) {
			return "", false
		}
		if c.pos >= len(c.recs[c.i]) {
			c.i++
			if c.i >= len(c.recs) || len(c.recs[c.i]) == 0 {
				return "", false
			}
			high = c.recs[c.i][0]&1 != 0
			c.pos = 1
			continue
		}
		rec := c.recs[c.i]
		if high {
			for c.pos+1 < len(rec) && len(units) < cch {
				units = append(units, binary.LittleEndian.Uint16(rec[c.pos:]))
				c.pos += 2
			}
			if c.pos+1 == len(rec) {
				c.pos++
			}
		} else {
			for c.pos < len(rec) && len(units) < cch {
				units = append(units, uint16(rec[c.pos]))
				c.pos++
			}
		}
	}
	return string(utf16.Decode(units)), true
}

func (c *biffCursor) unicodeString(cchSize int) (string, bool) {
	h, ok := c.bytes(cchSize + 1)
	if !ok {
		return "", false
	}
	cch := int(h[0])
	if cchSize == 2 {
		cch = int(binary.LittleEndian.Uint16(h))
	}
	flags := h[cchSize]
	runs, ext := 0, 0
	if flags&0x08 != 0 {
		b, ok := c.bytes(2)
		if !ok {
			return "", false
		}
		runs = int(binary.LittleEndian.Uint16(b))
	}
	if flags&0x04 != 0 {
		b, ok := c.bytes(4)
		if !ok {
			return "", false
		}
		ext = int(binary.LittleEndian.Uint32(b))
	}
	s, ok := c.chars(cch, flags&0x01 != 0)
	if !ok {
		return "", false
	}
	if _, ok := c.bytes(runs*4 + ext); !ok {
		return s, false
	}
	return s, true
}

func rkNumber(rk uint32) float64 {
	var v float64
	if rk&2 != 0 {
		v = float64(int32(rk) >> 2)
	} else {
		v = math.Float64frombits(uint64(rk&0xFFFFFFFC) << 32)
	}
	if rk&1 != 0 {
		v /= 100
	}
	return v
}

func formatNumber(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

type sheetRef struct {
	name string
	pos  int
}

func parseBIFF8(wb []byte) ([]XLSSheet, error) {
	recs := biffRecords(wb)
	if len(recs) == 0 || recs[0].id != 0x0809 || len(recs[0].data) < 4 || binary.LittleEndian.Uint16(recs[0].data) != 0x0600 {
		return nil, errors.New("chỉ hỗ trợ Excel 97-2003 (BIFF8)")
	}
	var sst []string
	var sheets []sheetRef
	for i := 0; i < len(recs); i++ {
		r := recs[i]
		switch r.id {
		case 0x002F:
			return nil, errXLSEncrypted
		case 0x0085:
			if len(r.data) < 8 {
				continue
			}
			cur := &biffCursor{recs: [][]byte{r.data[6:]}}
			name, _ := cur.unicodeString(1)
			sheets = append(sheets, sheetRef{name: name, pos: int(binary.LittleEndian.Uint32(r.data))})
		case 0x00FC:
			parts := [][]byte{r.data}
			for i+1 < len(recs) && recs[i+1].id == 0x003C {
				i++
				parts = append(parts, recs[i].data)
			}
			if len(r.data) < 8 {
				continue
			}
			unique := int(binary.LittleEndian.Uint32(r.data[4:]))
			cur := &biffCursor{recs: parts, pos: 8}
			sst = make([]string, 0, min(unique, 1<<16))
			for len(sst) < unique {
				s, ok := cur.unicodeString(2)
				if !ok {
					break
				}
				sst = append(sst, s)
			}
		case 0x000A:
			i = len(recs)
		}
	}
	byPos := map[int]int{}
	for i, r := range recs {
		byPos[r.off] = i
	}
	out := make([]XLSSheet, 0, len(sheets))
	for _, sh := range sheets {
		start, ok := byPos[sh.pos]
		if !ok {
			continue
		}
		out = append(out, XLSSheet{Name: sh.name, Rows: readSheet(recs, start, sst)})
	}
	if len(out) == 0 {
		return nil, errors.New("file Excel không có sheet nào")
	}
	return out, nil
}

func readSheet(recs []biffRec, start int, sst []string) [][]string {
	le := binary.LittleEndian
	cells := map[[2]int]string{}
	maxRow, maxCol := -1, -1
	set := func(row, col int, v string) {
		if row >= xlsMaxRows || col >= xlsMaxCols || v == "" {
			return
		}
		cells[[2]int{row, col}] = v
		maxRow, maxCol = max(maxRow, row), max(maxCol, col)
	}
	pendingFormula := [2]int{-1, -1}
	for i := start + 1; i < len(recs); i++ {
		r := recs[i]
		d := r.data
		if r.id == 0x000A {
			break
		}
		if r.id == 0x0207 {
			if pendingFormula[0] >= 0 {
				cur := &biffCursor{recs: [][]byte{d}}
				if s, ok := cur.unicodeString(2); ok {
					set(pendingFormula[0], pendingFormula[1], s)
				}
				pendingFormula = [2]int{-1, -1}
			}
			continue
		}
		if len(d) < 6 {
			continue
		}
		row, col := int(le.Uint16(d)), int(le.Uint16(d[2:]))
		switch r.id {
		case 0x00FD:
			if len(d) >= 10 {
				if k := int(le.Uint32(d[6:])); k < len(sst) {
					set(row, col, sst[k])
				}
			}
		case 0x0204:
			cur := &biffCursor{recs: [][]byte{d[6:]}}
			if s, ok := cur.unicodeString(2); ok {
				set(row, col, s)
			}
		case 0x0203:
			if len(d) >= 14 {
				set(row, col, formatNumber(math.Float64frombits(le.Uint64(d[6:]))))
			}
		case 0x027E:
			if len(d) >= 10 {
				set(row, col, formatNumber(rkNumber(le.Uint32(d[6:]))))
			}
		case 0x00BD:
			for k, c := 0, col; 4+k*6+6 <= len(d)-2; k, c = k+1, c+1 {
				set(row, c, formatNumber(rkNumber(le.Uint32(d[4+k*6+2:]))))
			}
		case 0x0006:
			if len(d) >= 14 {
				res := d[6:14]
				if res[6] == 0xFF && res[7] == 0xFF {
					if res[0] == 0 {
						pendingFormula = [2]int{row, col}
					}
				} else {
					set(row, col, formatNumber(math.Float64frombits(le.Uint64(res))))
				}
			}
		}
	}
	rows := make([][]string, maxRow+1)
	for i := range rows {
		rows[i] = make([]string, maxCol+1)
	}
	for k, v := range cells {
		rows[k[0]][k[1]] = v
	}
	return rows
}
