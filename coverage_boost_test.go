package goTap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test DefaultPostForm edge cases
func TestDefaultPostFormEdgeCases(t *testing.T) {
	engine := New()
	
	engine.POST("/form", func(c *Context) {
		// Test with default when key doesn't exist
		value := c.DefaultPostForm("nonexistent", "default-value")
		if value != "default-value" {
			t.Errorf("Expected 'default-value', got '%s'", value)
		}
		
		// Test with empty value
		empty := c.DefaultPostForm("empty", "default")
		c.String(200, "%s,%s", value, empty)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/form", strings.NewReader("empty="))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(w, req)
	
	if !strings.Contains(w.Body.String(), "default-value") {
		t.Errorf("Expected default value in response")
	}
}

// Test Header method with multiple calls
func TestHeaderMultipleCalls(t *testing.T) {
	engine := New()
	
	engine.GET("/headers", func(c *Context) {
		// Set multiple headers
		c.Header("X-Custom-1", "value1")
		c.Header("X-Custom-2", "value2")
		c.Header("X-Custom-1", "updated") // Overwrite
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/headers", nil)
	engine.ServeHTTP(w, req)
	
	if w.Header().Get("X-Custom-1") != "updated" {
		t.Errorf("Expected updated header value")
	}
	if w.Header().Get("X-Custom-2") != "value2" {
		t.Errorf("Expected X-Custom-2 header")
	}
}

// Test Cookie with domain and path
func TestCookieWithOptions(t *testing.T) {
	engine := New()
	
	engine.GET("/cookie", func(c *Context) {
		c.SetCookie("test-cookie", "test-value", 3600, "/path", "example.com", false, true)
		c.String(200, "cookie set")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/cookie", nil)
	engine.ServeHTTP(w, req)
	
	cookies := w.Header().Get("Set-Cookie")
	if !strings.Contains(cookies, "test-cookie=test-value") {
		t.Errorf("Expected cookie in response")
	}
	if !strings.Contains(cookies, "Path=/path") {
		t.Errorf("Expected path in cookie")
	}
	if !strings.Contains(cookies, "Domain=example.com") {
		t.Errorf("Expected domain in cookie")
	}
}

// Test reading cookie from request
func TestReadCookie(t *testing.T) {
	engine := New()
	
	engine.GET("/read-cookie", func(c *Context) {
		value, err := c.Cookie("session-id")
		if err != nil {
			c.String(400, "no cookie")
			return
		}
		c.String(200, "cookie: %s", value)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/read-cookie", nil)
	req.AddCookie(&http.Cookie{Name: "session-id", Value: "abc123"})
	engine.ServeHTTP(w, req)
	
	if !strings.Contains(w.Body.String(), "abc123") {
		t.Errorf("Expected cookie value in response")
	}
}

// Test BindUri with different data types
func TestBindUriTypes(t *testing.T) {
	engine := New()
	
	// Test with integer param
	engine.GET("/user/:id/:name", func(c *Context) {
		var uri struct {
			ID   int    `uri:"id" binding:"required"`
			Name string `uri:"name" binding:"required"`
		}
		
		if err := c.BindUri(&uri); err != nil {
			c.String(400, "bind error: %v", err)
			return
		}
		
		c.JSON(200, uri)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/123/john", nil)
	engine.ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
	
	if !strings.Contains(w.Body.String(), "123") || !strings.Contains(w.Body.String(), "john") {
		t.Errorf("Expected ID and name in response")
	}
}

// Test Last method for getting last error
func TestLastError(t *testing.T) {
	engine := New()
	
	engine.GET("/errors", func(c *Context) {
		// Add multiple errors
		c.Error(&Error{Err: http.ErrNotSupported, Type: ErrorTypePrivate})
		c.Error(&Error{Err: http.ErrAbortHandler, Type: ErrorTypePublic})
		c.Error(&Error{Err: http.ErrServerClosed, Type: ErrorTypeAny})
		
		// Get last error
		last := c.Errors.Last()
		if last == nil {
			c.String(500, "no errors")
			return
		}
		
		c.String(200, "last: %v", last.Err)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/errors", nil)
	engine.ServeHTTP(w, req)
	
	if !strings.Contains(w.Body.String(), "closed") {
		t.Errorf("Expected last error in response")
	}
}
