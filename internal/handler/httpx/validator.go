package httpx

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()
var houseCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var monthPeriodPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// init registers project-specific validation tags.
func init() {
	if err := validate.RegisterValidation("housecode", func(fl validator.FieldLevel) bool {
		return houseCodePattern.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("register housecode validation: %v", err))
	}
	if err := validate.RegisterValidation("monthperiod", func(fl validator.FieldLevel) bool {
		return monthPeriodPattern.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("register monthperiod validation: %v", err))
	}
}

func ValidateStruct(v any) error {
	if err := validate.Struct(v); err != nil {
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) {
			firstErr := validationErrs[0]
			return fmt.Errorf("invalid field %q (%s)", firstErr.Field(), firstErr.Tag())
		}

		return err
	}

	return nil
}

// validateMonthPeriodValue enforces the API period format used for monthly records.
func ValidateMonthPeriodValue(period string) error {
	if err := validate.Var(period, "required,monthperiod"); err != nil {
		return fmt.Errorf("invalid field %q (%s)", "Period", "monthperiod")
	}
	return nil
}
