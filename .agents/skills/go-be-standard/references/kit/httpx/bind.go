package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
)

const maxBodyBytes = 1 << 20

type Validatable interface {
	Validate(errs *FieldErrors)
}

type FieldErrors struct {
	order []string
	msgs  map[string]string
}

func (e *FieldErrors) Add(field, message string) {
	if e.msgs == nil {
		e.msgs = map[string]string{}
	}
	if _, ok := e.msgs[field]; ok {
		return
	}
	e.order = append(e.order, field)
	e.msgs[field] = message
}

func (e *FieldErrors) Has(field string) bool {
	_, ok := e.msgs[field]
	return ok
}

func (e *FieldErrors) Empty() bool { return len(e.order) == 0 }

func (e *FieldErrors) First(fallback string) string {
	if e.Empty() {
		return fallback
	}
	return e.msgs[e.order[0]]
}

func (e *FieldErrors) Map() map[string]string { return e.msgs }

func Bind(w http.ResponseWriter, r *http.Request, dst any) bool {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("httpx.Bind: dst phải là con trỏ tới struct, nhận %T", dst))
	}
	var errs FieldErrors
	if status, message := decodeBody(w, r, dst, &errs); status != 0 {
		WriteError(w, status, message)
		return false
	}
	decodeValues(rv.Elem(), "query", r.URL.Query(), &errs)
	decodeValues(rv.Elem(), "path", pathValues(r, rv.Elem().Type()), &errs)
	trimStrings(rv.Elem())
	validateTags(dst, &errs)
	if v, ok := dst.(Validatable); ok && errs.Empty() {
		v.Validate(&errs)
	}
	if !errs.Empty() {
		ValidationFailed(w, &errs)
		return false
	}
	return true
}

func DayRange(from, to string, loc *time.Location) (start, end time.Time) {
	if t, err := time.ParseInLocation(time.DateOnly, from, loc); err == nil {
		start = t
	}
	if t, err := time.ParseInLocation(time.DateOnly, to, loc); err == nil {
		end = t.AddDate(0, 0, 1)
	}
	return start, end
}

func decodeBody(w http.ResponseWriter, r *http.Request, dst any, errs *FieldErrors) (int, string) {
	if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Body == nil || r.Body == http.NoBody {
		return 0, ""
	}
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		return 0, ""
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(dst)
	if errors.Is(err, io.EOF) {
		return 0, ""
	}
	var tooLarge *http.MaxBytesError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &tooLarge):
		return http.StatusRequestEntityTooLarge, "Dữ liệu gửi lên quá lớn"
	case errors.As(err, &typeErr) && typeErr.Field != "":
		errs.Add(typeErr.Field, typeMessage(labelFor(reflect.TypeOf(dst), typeErr.Field), typeErr.Type))
	case err != nil:
		return http.StatusBadRequest, "Dữ liệu gửi lên không đúng định dạng JSON"
	}
	if !errors.Is(dec.Decode(&struct{}{}), io.EOF) {
		return http.StatusBadRequest, "Dữ liệu gửi lên không đúng định dạng JSON"
	}
	return 0, ""
}

func pathValues(r *http.Request, t reflect.Type) url.Values {
	values := url.Values{}
	for i := range t.NumField() {
		if name := tagName(t.Field(i), "path"); name != "" {
			values.Set(name, r.PathValue(name))
		}
	}
	return values
}

func decodeValues(v reflect.Value, tag string, src url.Values, errs *FieldErrors) {
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		name := tagName(sf, tag)
		if name == "" || !sf.IsExported() {
			continue
		}
		bracket := src[name+"[]"]
		vals := slices.Concat(src[name], bracket)
		for j := range vals {
			vals[j] = strings.TrimSpace(vals[j])
		}
		vals = slices.DeleteFunc(vals, func(s string) bool { return s == "" })
		if len(vals) == 0 {
			continue
		}
		label := labelOf(sf, name)
		field := v.Field(i)
		if field.Kind() == reflect.Slice {
			list := reflect.MakeSlice(field.Type(), len(vals), len(vals))
			for j, raw := range vals {
				if !setScalar(list.Index(j), raw) {
					errs.Add(name, typeMessage(label, field.Type().Elem()))
					break
				}
			}
			field.Set(list)
			continue
		}
		if len(vals) > 1 || len(bracket) > 0 {
			errs.Add(name, label+" chỉ nhận một giá trị")
			continue
		}
		if !setScalar(field, vals[0]) {
			errs.Add(name, typeMessage(label, field.Type()))
		}
	}
}

func setScalar(f reflect.Value, raw string) bool {
	if f.Kind() == reflect.Pointer {
		p := reflect.New(f.Type().Elem())
		if !setScalar(p.Elem(), raw) {
			return false
		}
		f.Set(p)
		return true
	}
	switch f.Kind() {
	case reflect.String:
		f.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return false
		}
		f.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, f.Type().Bits())
		if err != nil {
			return false
		}
		f.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, f.Type().Bits())
		if err != nil {
			return false
		}
		f.SetUint(n)
	case reflect.Float32, reflect.Float64:
		n, err := strconv.ParseFloat(raw, f.Type().Bits())
		if err != nil {
			return false
		}
		f.SetFloat(n)
	default:
		panic(fmt.Sprintf("httpx: kiểu %s không dùng được cho query/path", f.Type()))
	}
	return true
}

func trimStrings(v reflect.Value) {
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			trimStrings(v.Elem())
		}
	case reflect.String:
		if v.CanSet() {
			v.SetString(strings.TrimSpace(v.String()))
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			if sf := t.Field(i); sf.IsExported() && sf.Tag.Get("bind") != "raw" {
				trimStrings(v.Field(i))
			}
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			trimStrings(v.Index(i))
		}
	}
}

func tagName(sf reflect.StructField, tag string) string {
	name, _, _ := strings.Cut(sf.Tag.Get(tag), ",")
	if name == "-" {
		return ""
	}
	return name
}

func fieldKey(sf reflect.StructField) string {
	for _, tag := range []string{"path", "query", "json"} {
		if name := tagName(sf, tag); name != "" {
			return name
		}
	}
	return sf.Name
}

func labelOf(sf reflect.StructField, fallback string) string {
	if label := sf.Tag.Get("label"); label != "" {
		return label
	}
	return fallback
}

func labelFor(t reflect.Type, key string) string {
	segs := strings.Split(key, ".")
	label := segs[len(segs)-1]
	for _, seg := range segs {
		t = indirect(t)
		if t.Kind() == reflect.Map {
			t = t.Elem()
			continue
		}
		if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
			t = indirect(t.Elem())
			if _, err := strconv.Atoi(seg); err == nil {
				continue
			}
		}
		if t.Kind() != reflect.Struct {
			return label
		}
		sf, ok := fieldByKey(t, seg)
		if !ok {
			return label
		}
		label = labelOf(sf, seg)
		t = sf.Type
	}
	return label
}

func fieldByKey(t reflect.Type, key string) (reflect.StructField, bool) {
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Anonymous && indirect(sf.Type).Kind() == reflect.Struct && tagName(sf, "json") == "" {
			if inner, ok := fieldByKey(indirect(sf.Type), key); ok {
				return inner, true
			}
			continue
		}
		if sf.IsExported() && fieldKey(sf) == key {
			return sf, true
		}
	}
	return reflect.StructField{}, false
}

func indirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func typeMessage(label string, t reflect.Type) string {
	switch indirect(t).Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return label + " phải là số nguyên"
	case reflect.Float32, reflect.Float64:
		return label + " phải là số"
	case reflect.Bool:
		return label + " phải là true hoặc false"
	case reflect.String:
		return label + " phải là chuỗi ký tự"
	case reflect.Slice, reflect.Array:
		return label + " phải là danh sách"
	case reflect.Struct, reflect.Map:
		return label + " không đúng cấu trúc"
	default:
		return label + " không đúng kiểu dữ liệu"
	}
}
