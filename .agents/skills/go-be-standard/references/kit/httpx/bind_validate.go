package httpx

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	bindValidator = newBindValidator()
	ruleMessages  = map[string]string{}
	layoutHint    = strings.NewReplacer("2006", "YYYY", "01", "MM", "02", "DD", "15", "HH", "04", "mm", "05", "ss")
)

func newBindValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(fieldKey)
	return v
}

func init() {
	RegisterRule("housecode", ":field chỉ được gồm chữ, số, gạch ngang và gạch dưới", func(fl validator.FieldLevel) bool {
		return houseCodePattern.MatchString(fl.Field().String())
	})
	RegisterRule("monthperiod", ":field phải có dạng YYYY-MM", func(fl validator.FieldLevel) bool {
		return monthPeriodPattern.MatchString(fl.Field().String())
	})
}

func RegisterRule(tag, message string, fn validator.Func) {
	if err := bindValidator.RegisterValidation(tag, fn); err != nil {
		panic(fmt.Sprintf("httpx: register rule %q: %v", tag, err))
	}
	ruleMessages[tag] = message
}

func validateTags(dst any, errs *FieldErrors) {
	err := bindValidator.Struct(dst)
	if err == nil {
		return
	}
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		panic(fmt.Sprintf("httpx: validate %T: %v", dst, err))
	}
	root := reflect.TypeOf(dst)
	for _, fe := range verrs {
		key := keyOf(fe.Namespace())
		errs.Add(key, ruleMessage(fe, labelFor(root, key)))
	}
}

func keyOf(namespace string) string {
	_, key, _ := strings.Cut(namespace, ".")
	return strings.NewReplacer("[", ".", "]", "").Replace(key)
}

func ruleMessage(fe validator.FieldError, label string) string {
	param := fe.Param()
	if tmpl, ok := ruleMessages[fe.Tag()]; ok {
		return strings.NewReplacer(":field", label, ":param", param).Replace(tmpl)
	}
	switch fe.Tag() {
	case "required", "required_if", "required_unless", "required_with", "required_with_all",
		"required_without", "required_without_all":
		return label + " không được để trống"
	case "email":
		return label + " không đúng định dạng email"
	case "url", "http_url":
		return label + " không đúng định dạng đường dẫn"
	case "e164":
		return label + " không đúng định dạng số điện thoại"
	case "numeric", "number":
		return label + " phải là số"
	case "boolean":
		return label + " phải là true hoặc false"
	case "alphanum":
		return label + " chỉ được gồm chữ và số"
	case "oneof":
		return label + " phải là một trong: " + strings.Join(strings.Fields(param), ", ")
	case "datetime":
		return label + " phải có định dạng " + layoutHint.Replace(param)
	case "unique":
		return label + " không được trùng nhau"
	case "eqfield":
		return label + " không khớp"
	case "len":
		return sized(fe.Kind(), label, "phải có đúng "+param+" ký tự", "phải có đúng "+param+" phần tử", "phải bằng "+param)
	case "min", "gte":
		return sized(fe.Kind(), label, "phải có ít nhất "+param+" ký tự", "phải có ít nhất "+param+" phần tử", "phải lớn hơn hoặc bằng "+param)
	case "max", "lte":
		return sized(fe.Kind(), label, "không được quá "+param+" ký tự", "không được quá "+param+" phần tử", "phải nhỏ hơn hoặc bằng "+param)
	case "gt":
		return sized(fe.Kind(), label, "phải dài hơn "+param+" ký tự", "phải có nhiều hơn "+param+" phần tử", "phải lớn hơn "+param)
	case "lt":
		return sized(fe.Kind(), label, "phải ngắn hơn "+param+" ký tự", "phải có ít hơn "+param+" phần tử", "phải nhỏ hơn "+param)
	default:
		return label + " không hợp lệ"
	}
}

func sized(kind reflect.Kind, label, text, list, number string) string {
	switch kind {
	case reflect.String:
		return label + " " + text
	case reflect.Slice, reflect.Array, reflect.Map:
		return label + " " + list
	default:
		return label + " " + number
	}
}
