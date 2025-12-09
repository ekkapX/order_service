package validation

import (
	"fmt"
	"l0/internal/domain"

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

func (v *Validator) ValidateOrder(order domain.Order) error {
	err := v.validate.Struct(order)
	if err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			return err
		}

		return fmt.Errorf("validation failed: %w", err)
	}
	return nil
}
