package handler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// echoValidator wraps go-playground/validator so Echo can call c.Validate(req).
type echoValidator struct {
	v *validator.Validate
}

// NewValidator returns an echoValidator ready to be assigned to echo.Echo.Validator.
func NewValidator() *echoValidator {
	return &echoValidator{v: validator.New()}
}

// Validate satisfies the echo.Validator interface.
func (ev *echoValidator) Validate(i any) error {
	if err := ev.v.Struct(i); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, fieldError(fe))
			}
			return fmt.Errorf("%s", strings.Join(msgs, "; "))
		}
		return err
	}
	return nil
}

// fieldError converts a single ValidationError into a human-readable message.
func fieldError(fe validator.FieldError) string {
	field := strings.ToLower(fe.Field())
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "email":
		return field + " must be a valid email"
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, fe.Param())
	default:
		return fmt.Sprintf("%s failed validation (%s)", field, fe.Tag())
	}
}
