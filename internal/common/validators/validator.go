package validators

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

type Validator struct {
	validate *validator.Validate
	logger   zerolog.Logger
}

func (v Validator) Validate(s interface{}) error {
	err := v.validate.Struct(s)
	if err != nil {
		var invalidValidationError validator.InvalidValidationError
		if errors.Is(err, &invalidValidationError) {
			return err
		}
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, vErr := range validationErrors {
				v.logger.Error().Msgf("validator: failed on field <%s>, condition: %s", vErr.Field(), vErr.Tag())
			}
		}
		return fmt.Errorf("validator: %w", err)
	}
	return nil
}

func NewValidator(logger zerolog.Logger) Validator {
	return Validator{
		validate: validator.New(),
		logger:   logger,
	}
}

func NewValidatorWithTagFieldName(tagFieldName string, logger zerolog.Logger) Validator {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get(tagFieldName), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return Validator{
		validate: validate,
		logger:   logger,
	}
}
