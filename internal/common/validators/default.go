package validators

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

func New() *validator.Validate {
	return validator.New()
}

func NewValidatorTagName(tagName string) *validator.Validate {
	validate := New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(tagName), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return validate
}

func ValidateStructWithLogger(str any, logger zerolog.Logger, validate *validator.Validate) error {
	err := validate.Struct(str)
	if err != nil {
		var invalidValidationError validator.InvalidValidationError
		if errors.Is(err, &invalidValidationError) {
			return err
		}
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, vErr := range validationErrors {
				logger.Error().Msgf("validator: failed on field <%s>, condition: %s", vErr.Field(), vErr.Tag())
			}
		}
		return fmt.Errorf("validator: %w", err)
	}
	return nil
}
