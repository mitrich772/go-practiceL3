// Package validation содержит функции валидации входящих DTO.
package validation

import (
	"shortener/internal/handlers/dto"

	"github.com/go-playground/validator/v10"
)

// Validate — экземпляр валидатора, используемый в пакете validation.
var Validate = validator.New()

// ValidateShorten валидирует запрос на создание короткой ссылки.
func ValidateShorten(req dto.ShortenRequest) error {
	return Validate.Struct(req)
}
