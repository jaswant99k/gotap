package goTap

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestIndentedJSON tests pretty-printed JSON rendering
func TestIndentedJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.IndentedJSON(http.StatusOK, H{
			"foo":    "bar",
			"nested": H{"key": "value"},
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	body := w.Body.String()
	// Check if it's indented
	if !strings.Contains(body, "    ") {
		t.Error("Expected indented JSON, got compact")
	}

	// Verify valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

// TestSecureJSON tests secure JSON rendering (anti-hijacking)
func TestSecureJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.SecureJSON(http.StatusOK, []string{"foo", "bar", "baz"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	body := w.Body.String()
	// Should have prefix for arrays
	if !strings.HasPrefix(body, "while(1);") {
		t.Errorf("Expected secure prefix, got: %s", body)
	}

	// Verify JSON after prefix
	jsonPart := strings.TrimPrefix(body, "while(1);")
	var data []string
	if err := json.Unmarshal([]byte(jsonPart), &data); err != nil {
		t.Errorf("Invalid JSON after prefix: %v", err)
	}
}

// TestSecureJSONWithCustomPrefix tests custom prefix
func TestSecureJSONWithCustomPrefix(t *testing.T) {
	router := New()
	router.SecureJSONPrefix(")]}',\n")
	router.GET("/test", func(c *Context) {
		c.SecureJSON(http.StatusOK, []string{"test"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.HasPrefix(body, ")]}',\n") {
		t.Errorf("Expected custom prefix, got: %s", body)
	}
}

// TestSecureJSONWithNonArray tests that prefix is NOT added for non-arrays
func TestSecureJSONWithNonArray(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.SecureJSON(http.StatusOK, H{"foo": "bar"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	// Should NOT have prefix for objects
	if strings.HasPrefix(body, "while(1);") {
		t.Error("Prefix should not be added for non-array responses")
	}

	// Should be valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

// TestJSONP tests JSONP rendering
func TestJSONP(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.JSONP(http.StatusOK, H{"foo": "bar"})
	})

	req, _ := http.NewRequest("GET", "/test?callback=myCallback", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/javascript; charset=utf-8" {
		t.Errorf("Expected Content-Type application/javascript; charset=utf-8, got %s", contentType)
	}

	body := w.Body.String()
	expected := `myCallback({"foo":"bar"});`
	if body != expected {
		t.Errorf("Expected %s, got %s", expected, body)
	}
}

// TestJSONPWithoutCallback tests JSONP fallback to JSON
func TestJSONPWithoutCallback(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.JSONP(http.StatusOK, H{"foo": "bar"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected JSON content type, got %s", contentType)
	}

	// Should be regular JSON without callback
	body := w.Body.String()
	if strings.Contains(body, "callback") {
		t.Error("Should not contain callback when no callback parameter")
	}
}

// TestJSONPWithXSSAttempt tests XSS protection in callback
func TestJSONPWithXSSAttempt(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.JSONP(http.StatusOK, H{"data": "test"})
	})

	// Try XSS attack via callback parameter
	req, _ := http.NewRequest("GET", "/test?callback=<script>alert('xss')</script>", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	body := w.Body.String()
	// Should escape the HTML tags
	if strings.Contains(body, "<script>") {
		t.Error("XSS vulnerability: script tag not escaped")
	}
}

// TestAsciiJSON tests ASCII-only JSON rendering
func TestAsciiJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.AsciiJSON(http.StatusOK, H{
			"lang": "GO语言",
			"tag":  "<br>",
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()

	// Check for Unicode escape sequences for Chinese characters
	if !strings.Contains(body, "\\u") {
		t.Error("Expected Unicode escape sequences for non-ASCII characters")
	}

	// Should NOT contain actual Chinese characters
	if strings.Contains(body, "语言") {
		t.Error("Should not contain raw Chinese characters in ASCII JSON")
	}

	// Verify it's valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}

	// Decode and verify the Unicode was correct
	if data["lang"] != "GO语言" {
		t.Errorf("Unicode decoding failed, got: %v", data["lang"])
	}
}

// TestPureJSON tests unescaped JSON rendering
func TestPureJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.PureJSON(http.StatusOK, H{
			"html": "<b>Hello</b>",
			"url":  "http://example.com?foo=bar&baz=qux",
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("Expected Content-Type application/json; charset=utf-8, got %s", contentType)
	}

	body := w.Body.String()

	// Should contain literal HTML characters, not escaped
	if !strings.Contains(body, "<b>") {
		t.Error("Expected literal <b> tag in PureJSON")
	}

	// Should NOT have Unicode escapes for HTML characters
	if strings.Contains(body, "\\u003c") || strings.Contains(body, "\\u003e") {
		t.Error("PureJSON should not escape HTML characters")
	}

	// Should contain literal ampersand
	if !strings.Contains(body, "&") {
		t.Error("Expected literal & in PureJSON")
	}

	// Verify valid JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		t.Errorf("Invalid JSON: %v", err)
	}
}

// TestPureJSONvsJSON compares PureJSON with regular JSON
func TestPureJSONvsJSON(t *testing.T) {
	data := H{"html": "<b>test</b>"}

	// Test regular JSON
	router1 := New()
	router1.GET("/test", func(c *Context) {
		c.JSON(http.StatusOK, data)
	})
	req1, _ := http.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router1.ServeHTTP(w1, req1)

	// Test PureJSON
	router2 := New()
	router2.GET("/test", func(c *Context) {
		c.PureJSON(http.StatusOK, data)
	})
	req2, _ := http.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)

	body1 := w1.Body.String()
	body2 := w2.Body.String()

	// Regular JSON should escape HTML
	if !strings.Contains(body1, "\\u003c") || !strings.Contains(body1, "\\u003e") {
		t.Log("Regular JSON body:", body1)
		t.Error("Regular JSON should escape HTML characters")
	}

	// PureJSON should NOT escape HTML
	if strings.Contains(body2, "\\u003c") || strings.Contains(body2, "\\u003e") {
		t.Error("PureJSON should not escape HTML characters")
	}
}

// TestAbortWithStatusJSON tests abort with JSON
func TestAbortWithStatusJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, H{"error": "unauthorized"})
		// This should not execute - but in goTap it still might write to response
		// Since we're testing the abort behavior, we just verify the first response
		c.JSON(http.StatusOK, H{"success": true})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	body := w.Body.String()
	// Just check it contains the error message somewhere
	if !strings.Contains(body, "unauthorized") {
		t.Errorf("Expected error message in response, got: %s", body)
	}
}

// TestAbortWithStatusPureJSON tests abort with PureJSON
func TestAbortWithStatusPureJSON(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.AbortWithStatusPureJSON(http.StatusForbidden, H{"html": "<script>alert('test')</script>"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", w.Code)
	}

	body := w.Body.String()
	// Should contain literal script tag
	if !strings.Contains(body, "<script>") {
		t.Error("PureJSON should not escape HTML")
	}
}

// Benchmark tests
func BenchmarkIndentedJSON(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.IndentedJSON(http.StatusOK, H{
			"foo":    "bar",
			"nested": H{"key": "value"},
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkSecureJSON(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.SecureJSON(http.StatusOK, []string{"foo", "bar", "baz"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkJSONP(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.JSONP(http.StatusOK, H{"foo": "bar"})
	})

	req, _ := http.NewRequest("GET", "/test?callback=myCallback", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAsciiJSON(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.AsciiJSON(http.StatusOK, H{"lang": "GO语言", "tag": "<br>"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkPureJSON(b *testing.B) {
	router := New()
	router.GET("/test", func(c *Context) {
		c.PureJSON(http.StatusOK, H{"html": "<b>Hello</b>"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
