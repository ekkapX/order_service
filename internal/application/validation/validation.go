package validation

import (
	"errors"
	"fmt"
	"l0/internal/domain/model"

	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

func (v *Validator) ValidateOrder(order model.Order) error {
	err := v.validate.Struct(order)
	if err != nil {
		var invalidErr *validator.InvalidValidationError
		if errors.As(err, &invalidErr) {
			return err
		}
		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
