package handler

import (
	"strings"
	"testing"
)

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required,email"`
	}

	t.Run("Valid struct", func(t *testing.T) {
		valid := TestStruct{
			Name:  "Test",
			Email: "test@example.com",
		}
		if err := validateStruct(valid); err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Missing required field", func(t *testing.T) {
		invalid := TestStruct{
			Email: "test@example.com",
		}
		err := validateStruct(invalid)
		if err == nil {
			t.Errorf("expected error for missing required field")
		}
		if !strings.Contains(err.Error(), "invalid field \"Name\"") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("Invalid email format", func(t *testing.T) {
		invalid := TestStruct{
			Name:  "Test",
			Email: "not-an-email",
		}
		err := validateStruct(invalid)
		if err == nil {
			t.Errorf("expected error for invalid email format")
		}
		if !strings.Contains(err.Error(), "invalid field \"Email\"") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
