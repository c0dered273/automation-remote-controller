package validators

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"
)

// Validator адаптер валидатора для echo
type Validator struct {
	validate *validator.Validate
	logger   zerolog.Logger
}

// Validate проверяет структуру на соответствие ограничениям из тегов `validate`
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

// NewValidatorWithTagFieldName конструктор валидатор
// в параметре tagFieldName указывается структурный тег из которого валидатор будет брать имя проверяемого поля
// нужно для более понятного логирования ошибок, если название полей, например json, отличается от имен полей структуры
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
