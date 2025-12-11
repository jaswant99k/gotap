package goTap

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"unicode"
)

// ========== JSON Rendering ==========

// IndentedJSON serializes the given struct as pretty JSON (indented + endlines) into the response body
// It also sets the Content-Type as "application/json"
// WARNING: use this only for development purposes since printing pretty JSON is
// more CPU and bandwidth consuming. Use Context.JSON() instead.
func (c *Context) IndentedJSON(code int, obj interface{}) {
	c.Status(code)
	c.setContentType("application/json; charset=utf-8")

	jsonBytes, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	c.Writer.Write(jsonBytes)
}

// SecureJSON serializes the given struct as Secure JSON into the response body
// Default prepends "while(1)," to response body if the given struct is array values
// It also sets the Content-Type as "application/json"
func (c *Context) SecureJSON(code int, obj interface{}) {
	prefix := c.engine.secureJSONPrefix
	if prefix == "" {
		prefix = "while(1);"
	}
	c.SecureJSONWithPrefix(code, prefix, obj)
}

// SecureJSONWithPrefix serializes the given struct as Secure JSON with custom prefix
func (c *Context) SecureJSONWithPrefix(code int, prefix string, obj interface{}) {
	c.Status(code)
	c.setContentType("application/json; charset=utf-8")

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// If the jsonBytes is array values, prepend the prefix
	if len(jsonBytes) > 0 && jsonBytes[0] == '[' {
		c.Writer.Write([]byte(prefix))
	}
	c.Writer.Write(jsonBytes)
}

// JSONP serializes the given struct as JSON into the response body
// It adds padding to response body to request data from a server residing in a different domain than the client
// It also sets the Content-Type as "application/javascript"
func (c *Context) JSONP(code int, obj interface{}) {
	callback := c.DefaultQuery("callback", "")
	if callback == "" {
		c.JSON(code, obj)
		return
	}

	c.Status(code)
	c.setContentType("application/javascript; charset=utf-8")

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// Escape callback to prevent XSS
	callback = template.JSEscapeString(callback)

	c.Writer.Write([]byte(callback))
	c.Writer.Write([]byte("("))
	c.Writer.Write(jsonBytes)
	c.Writer.Write([]byte(");"))
}

// AsciiJSON serializes the given struct as JSON into the response body with unicode to ASCII string
// It also sets the Content-Type as "application/json"
func (c *Context) AsciiJSON(code int, obj interface{}) {
	c.Status(code)
	c.setContentType("application/json")

	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}

	var buffer bytes.Buffer
	for _, r := range string(jsonBytes) {
		if r > unicode.MaxASCII {
			// Convert to Unicode escape sequence
			fmt.Fprintf(&buffer, "\\u%04x", r)
		} else {
			buffer.WriteRune(r)
		}
	}

	c.Writer.Write(buffer.Bytes())
}

// PureJSON serializes the given struct as JSON into the response body
// PureJSON, unlike JSON, does not replace special html characters with their unicode entities
func (c *Context) PureJSON(code int, obj interface{}) {
	c.Status(code)
	c.setContentType("application/json; charset=utf-8")

	encoder := json.NewEncoder(c.Writer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(obj); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

// ========== XML Rendering ==========

// XML serializes the given struct as XML into the response body
func (c *Context) XML(code int, obj interface{}) {
	c.Status(code)
	c.setContentType("application/xml; charset=utf-8")

	encoder := xml.NewEncoder(c.Writer)
	encoder.Indent("", "  ")

	if err := encoder.Encode(obj); err != nil {
		panic(err)
	}
}

// ========== YAML Rendering ==========

// YAML serializes the given struct as YAML into the response body
func (c *Context) YAML(code int, obj interface{}) {
	c.Status(code)
	c.setContentType("application/x-yaml; charset=utf-8")

	// Simple YAML serialization (basic implementation)
	// For production, use gopkg.in/yaml.v3
	yamlData := convertToYAML(obj)
	c.Writer.Write([]byte(yamlData))
}

// convertToYAML provides basic YAML conversion
// For full YAML support, integrate gopkg.in/yaml.v3
func convertToYAML(obj interface{}) string {
	// This is a simplified version
	// In production, use: yaml.Marshal(obj)
	switch v := obj.(type) {
	case H:
		result := ""
		for key, val := range v {
			result += key + ": "
			switch val := val.(type) {
			case string:
				result += val + "\n"
			case int, int64, float64, bool:
				result += toString(val) + "\n"
			default:
				result += toString(val) + "\n"
			}
		}
		return result
	default:
		return toString(obj)
	}
}

// ========== HTML Rendering ==========

var htmlTemplates *template.Template

// LoadHTMLGlob loads HTML templates from a glob pattern
func (engine *Engine) LoadHTMLGlob(pattern string) {
	htmlTemplates = template.Must(template.ParseGlob(pattern))
}

// LoadHTMLFiles loads HTML templates from specific files
func (engine *Engine) LoadHTMLFiles(files ...string) {
	htmlTemplates = template.Must(template.ParseFiles(files...))
}

// SetHTMLTemplate sets a custom HTML template
func (engine *Engine) SetHTMLTemplate(templ *template.Template) {
	htmlTemplates = templ
}

// HTML renders the HTTP template specified by its file name
func (c *Context) HTML(code int, name string, obj interface{}) {
	c.Status(code)
	c.setContentType("text/html; charset=utf-8")

	if htmlTemplates == nil {
		panic("HTML templates not loaded. Use LoadHTMLGlob() or LoadHTMLFiles()")
	}

	if err := htmlTemplates.ExecuteTemplate(c.Writer, name, obj); err != nil {
		panic(err)
	}
}

// ========== Redirect ==================

// Stream sends a streaming response and returns a boolean indicating "Is client disconnected?"
func (c *Context) Stream(step func(w http.ResponseWriter) bool) bool {
	w := c.Writer
	clientGone := w.(http.CloseNotifier).CloseNotify()
	for {
		select {
		case <-clientGone:
			return true
		default:
			keepOpen := step(w)
			w.(http.Flusher).Flush()
			if !keepOpen {
				return false
			}
		}
	}
}

// SSEvent is a Server-Sent Event
type SSEvent struct {
	Event string
	Data  interface{}
	ID    string
	Retry uint
}

// Render renders a Server-Sent Event
func (e SSEvent) Render(w http.ResponseWriter) error {
	if e.Event != "" {
		if _, err := w.Write([]byte("event: " + e.Event + "\n")); err != nil {
			return err
		}
	}

	if e.ID != "" {
		if _, err := w.Write([]byte("id: " + e.ID + "\n")); err != nil {
			return err
		}
	}

	if e.Retry > 0 {
		if _, err := w.Write([]byte("retry: " + toString(e.Retry) + "\n")); err != nil {
			return err
		}
	}

	dataStr := toString(e.Data)
	if _, err := w.Write([]byte("data: " + dataStr + "\n\n")); err != nil {
		return err
	}

	return nil
}

// SSE writes Server-Sent Events into the response stream
func (c *Context) SSE(event string, data interface{}) {
	c.Render(-1, SSEvent{
		Event: event,
		Data:  data,
	})
}

// Render writes a response using the provided renderer
func (c *Context) Render(code int, r interface{}) {
	if code > 0 {
		c.Status(code)
	}

	switch v := r.(type) {
	case SSEvent:
		c.setContentType("text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		v.Render(c.Writer)
	default:
		panic("Unknown render type")
	}
}

// ========== Negotiation ==========

// Negotiate contains all negotiations data
type Negotiate struct {
	Offered  []string
	HTMLName string
	HTMLData interface{}
	JSONData interface{}
	XMLData  interface{}
	YAMLData interface{}
	Data     interface{}
}

// Negotiate chooses the best format to render based on Accept header
func (c *Context) Negotiate(code int, config Negotiate) {
	switch c.NegotiateFormat(config.Offered...) {
	case "application/json":
		data := chooseData(config.JSONData, config.Data)
		c.JSON(code, data)

	case "application/xml", "text/xml":
		data := chooseData(config.XMLData, config.Data)
		c.XML(code, data)

	case "application/x-yaml":
		data := chooseData(config.YAMLData, config.Data)
		c.YAML(code, data)

	case "text/html":
		data := chooseData(config.HTMLData, config.Data)
		if config.HTMLName != "" {
			c.HTML(code, config.HTMLName, data)
		} else {
			c.String(code, toString(data))
		}

	default:
		c.String(code, toString(config.Data))
	}
}

// NegotiateFormat returns an acceptable format from the Accept header
func (c *Context) NegotiateFormat(offered ...string) string {
	if len(offered) == 0 {
		return ""
	}

	accept := c.Request.Header.Get("Accept")
	if accept == "" {
		return offered[0]
	}

	// Simple content negotiation
	for _, offer := range offered {
		if contains(accept, offer) {
			return offer
		}
	}

	return offered[0]
}

func chooseData(custom, wildcard interface{}) interface{} {
	if custom != nil {
		return custom
	}
	return wildcard
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && s[:len(substr)] == substr)
}

// toString converts any value to string
func toString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
