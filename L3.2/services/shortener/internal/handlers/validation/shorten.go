package validation

import (
	"shortener/internal/handlers/dto"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New()

func ValidateShorten(req dto.ShortenRequest) error {
	return Validate.Struct(req)
}
