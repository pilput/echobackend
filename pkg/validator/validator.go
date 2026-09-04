package validator

import (
	"errors"
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var messages = map[string]string{
	"required":   "%s is required",
	"email":      "%s must be a valid email address",
	"min":        "%s must be at least %s characters long",
	"max":        "%s must not exceed %s characters",
	"oneof":      "%s must be one of [%s]",
	"free_model": "%s must be a free OpenRouter model (use :free suffix or openrouter/free)",
	"default":    "%s failed validation for tag %s",
}

type CustomValidator struct {
	validator *validator.Validate
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

func (v ValidationErrors) Error() string {
	if len(v.Errors) == 0 {
		return "validation failed"
	}
	return v.Errors[0].Message
}

func NewValidator() *CustomValidator {
	v := validator.New()
	_ = v.RegisterValidation("free_model", validateFreeModel)
	return &CustomValidator{validator: v}
}

func validateFreeModel(fl validator.FieldLevel) bool {
	switch value := fl.Field().Interface().(type) {
	case string:
		return value == "" || IsFreeOpenRouterModel(value)
	case *string:
		if value == nil || *value == "" {
			return true
		}
		return IsFreeOpenRouterModel(*value)
	default:
		return false
	}
}

// IsFreeOpenRouterModel reports whether model ID is an OpenRouter free-tier model.
func IsFreeOpenRouterModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	if strings.EqualFold(model, "openrouter/free") {
		return true
	}
	return strings.HasSuffix(model, ":free")
}

func (cv *CustomValidator) Validate(i any) error {
	if err := cv.validator.Struct(i); err != nil {
		if validationErrors, ok := errors.AsType[validator.ValidationErrors](err); ok {
			result := ValidationErrors{
				Errors: make([]ValidationError, len(validationErrors)),
			}
			for i, e := range validationErrors {
				result.Errors[i] = ValidationError{
					Field:   e.Field(),
					Message: getErrorMessage(e),
					Value:   e.Value(),
					Tag:     e.Tag(),
				}
			}
			return result
		}
		return err
	}
	return nil
}

func getErrorMessage(e validator.FieldError) string {
	fieldName := toReadableFieldName(e.Field())
	switch e.Tag() {
	case "required":
		return fmt.Sprintf(messages["required"], fieldName)
	case "email":
		return fmt.Sprintf(messages["email"], fieldName)
	case "min":
		return fmt.Sprintf(messages["min"], fieldName, e.Param())
	case "max":
		return fmt.Sprintf(messages["max"], fieldName, e.Param())
	case "oneof":
		return fmt.Sprintf(messages["oneof"], fieldName, e.Param())
	case "free_model":
		return fmt.Sprintf(messages["free_model"], fieldName)
	default:
		return fmt.Sprintf(messages["default"], fieldName, e.Tag())
	}
}

func toReadableFieldName(field string) string {
	var result strings.Builder
	for i, r := range field {
		if i > 0 && unicode.IsUpper(r) {
			result.WriteRune(' ')
		}
		if i == 0 {
			result.WriteRune(unicode.ToUpper(r))
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}
	return result.String()
}

// IsValidUUID validates if a string is a valid UUID format (36 chars: 8-4-4-4-12 hex with hyphens).
// It uses a zero-allocation byte check that is ~100x faster than regexp matching.
func IsValidUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		return false
	}
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		b := uuid[i]
		if (b < '0' || b > '9') && (b < 'a' || b > 'f') {
			return false
		}
	}
	return true
}

// ValidatePagination validates pagination parameters
func ValidatePagination(limit, offset int) error {
	if limit <= 0 {
		return errors.New("limit must be greater than 0")
	}
	if limit > 100 {
		return errors.New("limit must not exceed 100")
	}
	if offset < 0 {
		return errors.New("offset must be non-negative")
	}
	return nil
}

// ValidatePostLikeInput validates input for post like operations
func ValidatePostLikeInput(postID, userID string) error {
	if !IsValidUUID(postID) {
		return errors.New("invalid post ID format")
	}
	if !IsValidUUID(userID) {
		return errors.New("invalid user ID format")
	}
	return nil
}

// ValidatePaginationWithDefaults validates pagination parameters and applies defaults
func ValidatePaginationWithDefaults(limitParam, offsetParam string) (int, int, error) {
	limit := 10 // default
	if limitParam != "" {
		parsedLimit, err := strconv.Atoi(limitParam)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid limit parameter: %w", err)
		}
		limit = parsedLimit
	}

	offset := 0 // default
	if offsetParam != "" {
		parsedOffset, err := strconv.Atoi(offsetParam)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid offset parameter: %w", err)
		}
		offset = parsedOffset
	}

	if err := ValidatePagination(limit, offset); err != nil {
		return 0, 0, err
	}

	return limit, offset, nil
}

// SanitizeString escapes HTML special characters to prevent XSS attacks.
// This converts <, >, &, ', and " to their HTML entity equivalents.
func SanitizeString(input string) string {
	escaped := html.EscapeString(input)
	return strings.TrimSpace(escaped)
}
