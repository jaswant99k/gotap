package goTap

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestXMLRendering(t *testing.T) {
	router := New()

	type Person struct {
		Name string `xml:"name"`
		Age  int    `xml:"age"`
	}

	router.GET("/xml", func(c *Context) {
		c.XML(http.StatusOK, Person{Name: "John", Age: 30})
	})

	req := httptest.NewRequest("GET", "/xml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "application/xml") {
		t.Errorf("Expected Content-Type application/xml, got %s", w.Header().Get("Content-Type"))
	}

	if !strings.Contains(w.Body.String(), "<name>John</name>") {
		t.Errorf("Expected XML body to contain <name>John</name>, got %s", w.Body.String())
	}
}

func TestYAMLRendering(t *testing.T) {
	router := New()

	router.GET("/yaml", func(c *Context) {
		c.YAML(http.StatusOK, H{
			"name": "John",
			"age":  "30",
		})
	})

	req := httptest.NewRequest("GET", "/yaml", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "application/x-yaml") {
		t.Errorf("Expected Content-Type application/x-yaml, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "name:") || !strings.Contains(body, "John") {
		t.Errorf("Expected YAML body to contain 'name: John', got %s", body)
	}
}

func TestHTMLRendering(t *testing.T) {
	router := New()

	// Load HTML templates from string
	router.SetHTMLTemplate(mustParseTemplate(`
		{{define "index.html"}}
		<html><body>Hello {{.name}}</body></html>
		{{end}}
	`))

	router.GET("/html", func(c *Context) {
		c.HTML(http.StatusOK, "index.html", H{"name": "World"})
	})

	req := httptest.NewRequest("GET", "/html", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Header().Get("Content-Type"), "text/html") {
		t.Errorf("Expected Content-Type text/html, got %s", w.Header().Get("Content-Type"))
	}

	if !strings.Contains(w.Body.String(), "Hello World") {
		t.Errorf("Expected HTML body to contain 'Hello World', got %s", w.Body.String())
	}
}

func TestNegotiateJSON(t *testing.T) {
	router := New()

	router.GET("/data", func(c *Context) {
		c.Negotiate(http.StatusOK, Negotiate{
			Offered: []string{"application/json", "application/xml"},
			Data:    H{"message": "hello"},
		})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), `"message"`) {
		t.Errorf("Expected JSON response, got %s", w.Body.String())
	}
}

func TestNegotiateXML(t *testing.T) {
	router := New()

	type Data struct {
		Message string `xml:"message"`
	}

	router.GET("/data", func(c *Context) {
		c.Negotiate(http.StatusOK, Negotiate{
			Offered: []string{"application/json", "application/xml"},
			Data:    Data{Message: "hello"},
		})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("Accept", "application/xml")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "<message>") {
		t.Errorf("Expected XML response, got %s", w.Body.String())
	}
}

func TestSSE(t *testing.T) {
	router := New()

	router.GET("/events", func(c *Context) {
		c.SSE("message", "hello")
	})

	req := httptest.NewRequest("GET", "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Content-Type"), "text/event-stream") {
		t.Errorf("Expected Content-Type text/event-stream, got %s", w.Header().Get("Content-Type"))
	}

	body := w.Body.String()
	if !strings.Contains(body, "event: message") {
		t.Errorf("Expected SSE event, got %s", body)
	}
	if !strings.Contains(body, "data: hello") {
		t.Errorf("Expected SSE data, got %s", body)
	}
}

func TestNegotiateFormat(t *testing.T) {
	router := New()

	router.GET("/test", func(c *Context) {
		format := c.NegotiateFormat("application/json", "application/xml")
		c.String(http.StatusOK, format)
	})

	// Test JSON
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Body.String() != "application/json" {
		t.Errorf("Expected 'application/json', got %s", w.Body.String())
	}

	// Test XML
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept", "application/xml")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Body.String() != "application/xml" {
		t.Errorf("Expected 'application/xml', got %s", w.Body.String())
	}

	// Test default (no Accept header)
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Body.String() != "application/json" {
		t.Errorf("Expected default 'application/json', got %s", w.Body.String())
	}
}

// Helper function to parse template from string
func mustParseTemplate(tmpl string) *template.Template {
	return template.Must(template.New("").Parse(tmpl))
}

// Benchmarks
func BenchmarkXMLRendering(b *testing.B) {
	router := New()

	type Data struct {
		Name  string `xml:"name"`
		Value int    `xml:"value"`
	}

	router.GET("/xml", func(c *Context) {
		c.XML(http.StatusOK, Data{Name: "test", Value: 123})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/xml", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkYAMLRendering(b *testing.B) {
	router := New()

	router.GET("/yaml", func(c *Context) {
		c.YAML(http.StatusOK, H{"name": "test", "value": "123"})
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/yaml", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkNegotiation(b *testing.B) {
	router := New()

	router.GET("/data", func(c *Context) {
		c.Negotiate(http.StatusOK, Negotiate{
			Offered: []string{"application/json", "application/xml"},
			Data:    H{"message": "hello"},
		})
	})

	req := httptest.NewRequest("GET", "/data", nil)
	req.Header.Set("Accept", "application/json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
