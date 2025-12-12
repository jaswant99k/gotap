package goTap

import (
	"errors"
	"testing"
)

func TestErrorSetType(t *testing.T) {
	err := &Error{
		Err:  errors.New("test error"),
		Type: ErrorTypePrivate,
	}

	err.SetType(ErrorTypePublic)
	if err.Type != ErrorTypePublic {
		t.Errorf("Expected type %d, got %d", ErrorTypePublic, err.Type)
	}
}

func TestErrorSetMeta(t *testing.T) {
	err := &Error{
		Err: errors.New("test error"),
	}

	meta := H{"key": "value"}
	err.SetMeta(meta)

	if err.Meta == nil {
		t.Error("Expected meta to be set")
	}
	if m, ok := err.Meta.(H); !ok || m["key"] != "value" {
		t.Errorf("Expected meta to be %v, got %v", meta, err.Meta)
	}
}

func TestErrorJSON(t *testing.T) {
	err := &Error{
		Err:  errors.New("test error"),
		Type: ErrorTypePublic,
		Meta: H{"code": "E001"},
	}

	jsonResult := err.JSON()

	// JSON returns interface{}, should be a map
	if result, ok := jsonResult.(H); ok {
		if result["error"] != "test error" {
			t.Errorf("Expected error message 'test error', got %v", result["error"])
		}
		if result["code"] != "E001" {
			t.Errorf("Expected code 'E001', got %v", result["code"])
		}
	} else {
		t.Errorf("Expected JSON result to be H type, got %T", jsonResult)
	}
}

func TestErrorError(t *testing.T) {
	err := &Error{
		Err:  errors.New("test error message"),
		Type: ErrorTypePublic,
	}

	if err.Error() != "test error message" {
		t.Errorf("Expected 'test error message', got '%s'", err.Error())
	}
}

func TestErrorIsType(t *testing.T) {
	err := &Error{
		Err:  errors.New("test"),
		Type: ErrorTypeBind,
	}

	if !err.IsType(ErrorTypeBind) {
		t.Error("Expected IsType(ErrorTypeBind) to be true")
	}
	if err.IsType(ErrorTypePublic) {
		t.Error("Expected IsType(ErrorTypePublic) to be false")
	}
}

func TestErrorUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := &Error{
		Err:  originalErr,
		Type: ErrorTypePrivate,
	}

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Expected unwrapped error to be %v, got %v", originalErr, unwrapped)
	}
}

func TestErrorMsgsByType(t *testing.T) {
	errs := errorMsgs{
		{Err: errors.New("bind error"), Type: ErrorTypeBind},
		{Err: errors.New("render error"), Type: ErrorTypeRender},
		{Err: errors.New("another bind"), Type: ErrorTypeBind},
		{Err: errors.New("public error"), Type: ErrorTypePublic},
	}

	bindErrors := errs.ByType(ErrorTypeBind)
	if len(bindErrors) != 2 {
		t.Errorf("Expected 2 bind errors, got %d", len(bindErrors))
	}

	publicErrors := errs.ByType(ErrorTypePublic)
	if len(publicErrors) != 1 {
		t.Errorf("Expected 1 public error, got %d", len(publicErrors))
	}

	privateErrors := errs.ByType(ErrorTypePrivate)
	if len(privateErrors) != 0 {
		t.Errorf("Expected 0 private errors, got %d", len(privateErrors))
	}
}

func TestErrorMsgsLast(t *testing.T) {
	errs := errorMsgs{
		{Err: errors.New("first"), Type: ErrorTypeBind},
		{Err: errors.New("second"), Type: ErrorTypeRender},
		{Err: errors.New("third"), Type: ErrorTypePublic},
	}

	last := errs.Last()
	if last == nil {
		t.Fatal("Expected last error to exist")
	}
	if last.Error() != "third" {
		t.Errorf("Expected last error to be 'third', got '%s'", last.Error())
	}

	// Test empty list
	var emptyErrs errorMsgs
	if emptyErrs.Last() != nil {
		t.Error("Expected Last() on empty list to return nil")
	}
}

func TestErrorMsgsErrors(t *testing.T) {
	errs := errorMsgs{
		{Err: errors.New("error 1"), Type: ErrorTypeBind},
		{Err: errors.New("error 2"), Type: ErrorTypeRender},
	}

	errStrings := errs.Errors()
	if len(errStrings) != 2 {
		t.Errorf("Expected 2 error strings, got %d", len(errStrings))
	}
	if errStrings[0] != "error 1" {
		t.Errorf("Expected 'error 1', got '%s'", errStrings[0])
	}
	if errStrings[1] != "error 2" {
		t.Errorf("Expected 'error 2', got '%s'", errStrings[1])
	}
}

func TestErrorMsgsJSON(t *testing.T) {
	// Test with multiple errors
	errs := errorMsgs{
		{Err: errors.New("error 1"), Type: ErrorTypePublic},
		{Err: errors.New("error 2"), Type: ErrorTypePublic},
	}

	jsonResult := errs.JSON()

	// JSON returns interface{}, should be a slice for multiple errors
	if result, ok := jsonResult.([]interface{}); ok {
		if len(result) != 2 {
			t.Errorf("Expected 2 errors in JSON, got %d", len(result))
		}
		if firstErr, ok := result[0].(H); ok {
			if firstErr["error"] != "error 1" {
				t.Errorf("Expected first error to be 'error 1', got %v", firstErr["error"])
			}
		} else {
			t.Errorf("Expected first error to be H type, got %T", result[0])
		}
	} else {
		t.Errorf("Expected JSON result to be a slice, got %T", jsonResult)
	}

	// Test single error case
	singleErr := errorMsgs{
		{Err: errors.New("single error"), Type: ErrorTypePublic},
	}
	singleResult := singleErr.JSON()
	if _, ok := singleResult.(H); !ok {
		t.Errorf("Expected single error JSON to be H type, got %T", singleResult)
	}
}

func TestErrorMsgsString(t *testing.T) {
	errs := errorMsgs{
		{Err: errors.New("error one"), Type: ErrorTypePublic},
		{Err: errors.New("error two"), Type: ErrorTypeBind},
	}

	str := errs.String()
	expected := "Error #01: error one\nError #02: error two\n"
	if str != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, str)
	}

	// Test empty list
	var emptyErrs errorMsgs
	if emptyErrs.String() != "" {
		t.Error("Expected empty string for empty error list")
	}
}
