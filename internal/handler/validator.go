package handler

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

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
