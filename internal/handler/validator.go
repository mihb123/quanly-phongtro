package handler

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()
var houseCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// init registers project-specific validation tags.
func init() {
	if err := validate.RegisterValidation("housecode", func(fl validator.FieldLevel) bool {
		return houseCodePattern.MatchString(fl.Field().String())
	}); err != nil {
		panic(fmt.Sprintf("register housecode validation: %v", err))
	}
}

func validateStruct(v any) error {
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
