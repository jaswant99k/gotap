package goTap

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// Binding describes the interface which needs to be implemented for binding request data
type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}

// BindingBody adds BindBody method to Binding. BindBody is similar to Bind,
// but it reads the body from supplied io.Reader instead of req.Body
type BindingBody interface {
	Binding
	BindBody(io.Reader, interface{}) error
}

// BindingUri binds from request URI (path and query params)
type BindingUri interface {
	Name() string
	BindUri(map[string][]string, interface{}) error
}

// Validator is the interface which needs to be implemented for validating request data
type Validator interface {
	ValidateStruct(interface{}) error
	Engine() interface{}
}

var (
	JSON          = jsonBinding{}
	XML           = xmlBinding{}
	Form          = formBinding{}
	Query         = queryBinding{}
	FormPost      = formPostBinding{}
	FormMultipart = formMultipartBinding{}
	Header        = headerBinding{}
	Uri           = uriBinding{}
)

var defaultValidator Validator

// SetValidator sets the default validator
func SetValidator(v Validator) {
	defaultValidator = v
}

// GetValidator returns the default validator
func GetValidator() Validator {
	return defaultValidator
}

// validate validates the struct using the default validator
func validate(obj interface{}) error {
	if defaultValidator == nil {
		return nil
	}
	return defaultValidator.ValidateStruct(obj)
}

// ========== JSON Binding ==========

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (jsonBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func (jsonBinding) BindBody(body io.Reader, obj interface{}) error {
	return decodeJSON(body, obj)
}

func decodeJSON(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

// ========== XML Binding ==========

type xmlBinding struct{}

func (xmlBinding) Name() string {
	return "xml"
}

func (xmlBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}
	return decodeXML(req.Body, obj)
}

func (xmlBinding) BindBody(body io.Reader, obj interface{}) error {
	return decodeXML(body, obj)
}

func decodeXML(r io.Reader, obj interface{}) error {
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Form Binding ==========

type formBinding struct{}

func (formBinding) Name() string {
	return "form"
}

func (formBinding) Bind(req *http.Request, obj interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Query Binding ==========

type queryBinding struct{}

func (queryBinding) Name() string {
	return "query"
}

func (queryBinding) Bind(req *http.Request, obj interface{}) error {
	values := req.URL.Query()
	if err := mapForm(obj, values); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Form-Post Binding ==========

type formPostBinding struct{}

func (formPostBinding) Name() string {
	return "form-post"
}

func (formPostBinding) Bind(req *http.Request, obj interface{}) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := mapForm(obj, req.PostForm); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Multipart Form Binding ==========

type formMultipartBinding struct{}

func (formMultipartBinding) Name() string {
	return "multipart/form-data"
}

func (formMultipartBinding) Bind(req *http.Request, obj interface{}) error {
	if err := req.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
		return err
	}
	if err := mapForm(obj, req.MultipartForm.Value); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Header Binding ==========

type headerBinding struct{}

func (headerBinding) Name() string {
	return "header"
}

func (headerBinding) Bind(req *http.Request, obj interface{}) error {
	if err := mapHeader(obj, req.Header); err != nil {
		return err
	}
	return validate(obj)
}

// ========== URI Binding ==========

type uriBinding struct{}

func (uriBinding) Name() string {
	return "uri"
}

func (uriBinding) BindUri(m map[string][]string, obj interface{}) error {
	if err := mapUri(obj, m); err != nil {
		return err
	}
	return validate(obj)
}

// ========== Helper Functions ==========

// mapForm maps url.Values to struct fields based on "form" tag
func mapForm(ptr interface{}, form map[string][]string) error {
	return mappingByPtr(ptr, formSource(form), "form")
}

// mapHeader maps http.Header to struct fields based on "header" tag
func mapHeader(ptr interface{}, header map[string][]string) error {
	return mappingByPtr(ptr, formSource(header), "header")
}

// mapUri maps URI parameters to struct fields based on "uri" tag
func mapUri(ptr interface{}, m map[string][]string) error {
	return mappingByPtr(ptr, formSource(m), "uri")
}

type formSource map[string][]string

func (f formSource) TryGet(key string) ([]string, bool) {
	v, ok := f[key]
	return v, ok
}

func mappingByPtr(ptr interface{}, source formSource, tag string) error {
	return mapping(reflect.ValueOf(ptr), source, tag)
}

func mapping(value reflect.Value, source formSource, tag string) error {
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return fmt.Errorf("binding element must be a struct")
	}

	typ := value.Type()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := value.Field(i)

		if !structField.CanSet() {
			continue
		}

		fieldTag := typeField.Tag.Get(tag)
		if fieldTag == "" || fieldTag == "-" {
			continue
		}

		// Parse tag options (e.g., "name,required")
		tagParts := strings.Split(fieldTag, ",")
		fieldName := tagParts[0]

		// Get values from source
		values, ok := source.TryGet(fieldName)
		if !ok || len(values) == 0 {
			// Check if field is required
			for _, opt := range tagParts[1:] {
				if opt == "required" {
					return fmt.Errorf("field '%s' is required", fieldName)
				}
			}
			continue
		}

		// Set the field value
		if err := setField(structField, values); err != nil {
			return fmt.Errorf("error setting field '%s': %v", fieldName, err)
		}
	}
	return nil
}

func setField(field reflect.Value, values []string) error {
	if !field.CanSet() {
		return fmt.Errorf("cannot set field")
	}

	kind := field.Kind()
	val := values[0]

	switch kind {
	case reflect.String:
		field.SetString(val)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intVal)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintVal)

	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatVal)

	case reflect.Bool:
		boolVal, err := strconv.ParseBool(val)
		if err != nil {
			return err
		}
		field.SetBool(boolVal)

	case reflect.Slice:
		return setSliceField(field, values)

	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setField(field.Elem(), values)

	default:
		return fmt.Errorf("unsupported type: %s", kind)
	}

	return nil
}

func setSliceField(field reflect.Value, values []string) error {
	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, val := range values {
		elem := slice.Index(i)
		if err := setField(elem, []string{val}); err != nil {
			return err
		}
	}

	field.Set(slice)
	return nil
}

// ========== Context Binding Methods ==========

// Bind checks the Content-Type to select a binding engine automatically
func (c *Context) Bind(obj interface{}) error {
	b := DefaultBinding(c.Request.Method, c.ContentType())
	return c.MustBindWith(obj, b)
}

// BindJSON is a shortcut for c.MustBindWith(obj, binding.JSON)
func (c *Context) BindJSON(obj interface{}) error {
	return c.MustBindWith(obj, JSON)
}

// BindXML is a shortcut for c.MustBindWith(obj, binding.XML)
func (c *Context) BindXML(obj interface{}) error {
	return c.MustBindWith(obj, XML)
}

// BindQuery is a shortcut for c.MustBindWith(obj, binding.Query)
func (c *Context) BindQuery(obj interface{}) error {
	return c.MustBindWith(obj, Query)
}

// BindHeader is a shortcut for c.MustBindWith(obj, binding.Header)
func (c *Context) BindHeader(obj interface{}) error {
	return c.MustBindWith(obj, Header)
}

// BindUri binds the passed struct pointer using the URI parameters
func (c *Context) BindUri(obj interface{}) error {
	m := make(map[string][]string)
	for _, v := range c.Params {
		m[v.Key] = []string{v.Value}
	}
	return Uri.BindUri(m, obj)
}

// MustBindWith binds the request body into obj using the specified binding engine
func (c *Context) MustBindWith(obj interface{}, b Binding) error {
	if err := c.ShouldBindWith(obj, b); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, H{"error": err.Error()})
		return err
	}
	return nil
}

// ShouldBind checks the Content-Type to select a binding engine automatically
func (c *Context) ShouldBind(obj interface{}) error {
	b := DefaultBinding(c.Request.Method, c.ContentType())
	return c.ShouldBindWith(obj, b)
}

// ShouldBindJSON is a shortcut for c.ShouldBindWith(obj, binding.JSON)
func (c *Context) ShouldBindJSON(obj interface{}) error {
	return c.ShouldBindWith(obj, JSON)
}

// ShouldBindXML is a shortcut for c.ShouldBindWith(obj, binding.XML)
func (c *Context) ShouldBindXML(obj interface{}) error {
	return c.ShouldBindWith(obj, XML)
}

// ShouldBindQuery is a shortcut for c.ShouldBindWith(obj, binding.Query)
func (c *Context) ShouldBindQuery(obj interface{}) error {
	return c.ShouldBindWith(obj, Query)
}

// ShouldBindHeader is a shortcut for c.ShouldBindWith(obj, binding.Header)
func (c *Context) ShouldBindHeader(obj interface{}) error {
	return c.ShouldBindWith(obj, Header)
}

// ShouldBindUri binds the passed struct pointer using the URI parameters
func (c *Context) ShouldBindUri(obj interface{}) error {
	m := make(map[string][]string)
	for _, v := range c.Params {
		m[v.Key] = []string{v.Value}
	}
	return Uri.BindUri(m, obj)
}

// ShouldBindWith binds the request body into obj using the specified binding engine
func (c *Context) ShouldBindWith(obj interface{}, b Binding) error {
	return b.Bind(c.Request, obj)
}

// ShouldBindBodyWith is similar to ShouldBindWith but it stores the request
// body into the context and reuses when called again
func (c *Context) ShouldBindBodyWith(obj interface{}, bb BindingBody) (err error) {
	var body []byte
	if cb, ok := c.Get("gotap.request.body"); ok {
		body = cb.([]byte)
	} else {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		c.Set("gotap.request.body", body)
	}
	return bb.BindBody(io.NopCloser(strings.NewReader(string(body))), obj)
}

// DefaultBinding returns the appropriate Binding instance based on the HTTP method
// and Content-Type
func DefaultBinding(method, contentType string) Binding {
	if method == http.MethodGet {
		return Query
	}

	switch contentType {
	case "application/json":
		return JSON
	case "application/xml", "text/xml":
		return XML
	case "application/x-www-form-urlencoded":
		return FormPost
	case "multipart/form-data":
		return FormMultipart
	default:
		return Form
	}
}

// MultipartForm is a helper to access multipart form data
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
	return c.Request.MultipartForm, err
}

// FormFile returns the first file for the provided form key
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	if c.Request.MultipartForm == nil {
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
	}
	f, fh, err := c.Request.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, nil
}

// SaveUploadedFile uploads the form file to specific dst
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := createFile(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
}

// createFile creates a file with all parent directories
func createFile(filename string) (*os.File, error) {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Create the file
	return os.Create(filename)
}
