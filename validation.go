package goTap

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// DefaultValidator is a simple built-in validator using struct tags
type DefaultValidator struct{}

// ValidateStruct validates a struct based on "validate" tags
func (v *DefaultValidator) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := value.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		validateTag := field.Tag.Get("validate")
		if validateTag == "" || validateTag == "-" {
			continue
		}

		// Parse validation rules
		rules := strings.Split(validateTag, ",")
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if err := validateRule(field.Name, fieldValue, rule); err != nil {
				return err
			}
		}
	}

	return nil
}

// Engine returns the underlying validation engine
func (v *DefaultValidator) Engine() interface{} {
	return v
}

// validateRule validates a single rule
func validateRule(fieldName string, value reflect.Value, rule string) error {
	// Handle parameterized rules (e.g., "min=5", "max=100")
	parts := strings.SplitN(rule, "=", 2)
	ruleName := parts[0]
	var ruleParam string
	if len(parts) > 1 {
		ruleParam = parts[1]
	}

	switch ruleName {
	case "required":
		return validateRequired(fieldName, value)
	case "email":
		return validateEmail(fieldName, value)
	case "min":
		return validateMin(fieldName, value, ruleParam)
	case "max":
		return validateMax(fieldName, value, ruleParam)
	case "len":
		return validateLen(fieldName, value, ruleParam)
	case "numeric":
		return validateNumeric(fieldName, value)
	case "alpha":
		return validateAlpha(fieldName, value)
	case "alphanum":
		return validateAlphaNum(fieldName, value)
	case "url":
		return validateURL(fieldName, value)
	case "oneof":
		return validateOneOf(fieldName, value, ruleParam)
	default:
		// Unknown rules are ignored
		return nil
	}
}

func validateRequired(fieldName string, value reflect.Value) error {
	if isZero(value) {
		return fmt.Errorf("field '%s' is required", fieldName)
	}
	return nil
}

func validateEmail(fieldName string, value reflect.Value) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("field '%s' must be a valid email", fieldName)
	}
	return nil
}

func validateMin(fieldName string, value reflect.Value, param string) error {
	min, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return fmt.Errorf("invalid min parameter: %s", param)
	}

	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		if float64(value.Len()) < min {
			return fmt.Errorf("field '%s' must have at least %v items", fieldName, min)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(value.Int()) < min {
			return fmt.Errorf("field '%s' must be at least %v", fieldName, min)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(value.Uint()) < min {
			return fmt.Errorf("field '%s' must be at least %v", fieldName, min)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() < min {
			return fmt.Errorf("field '%s' must be at least %v", fieldName, min)
		}
	}
	return nil
}

func validateMax(fieldName string, value reflect.Value, param string) error {
	max, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return fmt.Errorf("invalid max parameter: %s", param)
	}

	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		if float64(value.Len()) > max {
			return fmt.Errorf("field '%s' must have at most %v items", fieldName, max)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if float64(value.Int()) > max {
			return fmt.Errorf("field '%s' must be at most %v", fieldName, max)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if float64(value.Uint()) > max {
			return fmt.Errorf("field '%s' must be at most %v", fieldName, max)
		}
	case reflect.Float32, reflect.Float64:
		if value.Float() > max {
			return fmt.Errorf("field '%s' must be at most %v", fieldName, max)
		}
	}
	return nil
}

func validateLen(fieldName string, value reflect.Value, param string) error {
	length, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid len parameter: %s", param)
	}

	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		if value.Len() != length {
			return fmt.Errorf("field '%s' must have exactly %d items", fieldName, length)
		}
	}
	return nil
}

func validateNumeric(fieldName string, value reflect.Value) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}
	numericRegex := regexp.MustCompile(`^[0-9]+$`)
	if !numericRegex.MatchString(str) {
		return fmt.Errorf("field '%s' must contain only numbers", fieldName)
	}
	return nil
}

func validateAlpha(fieldName string, value reflect.Value) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}
	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("field '%s' must contain only letters", fieldName)
	}
	return nil
}

func validateAlphaNum(fieldName string, value reflect.Value) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}
	alphaNumRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphaNumRegex.MatchString(str) {
		return fmt.Errorf("field '%s' must contain only letters and numbers", fieldName)
	}
	return nil
}

func validateURL(fieldName string, value reflect.Value) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("field '%s' must be a valid URL", fieldName)
	}
	return nil
}

func validateOneOf(fieldName string, value reflect.Value, param string) error {
	if value.Kind() != reflect.String {
		return nil
	}
	str := value.String()
	if str == "" {
		return nil
	}

	allowed := strings.Split(param, " ")
	for _, a := range allowed {
		if str == a {
			return nil
		}
	}

	return fmt.Errorf("field '%s' must be one of: %s", fieldName, param)
}

func isZero(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String, reflect.Array:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return value.IsNil()
	case reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return value.IsNil() || value.Len() == 0
	}
	return false
}

func init() {
	// Set default validator
	SetValidator(&DefaultValidator{})
}
