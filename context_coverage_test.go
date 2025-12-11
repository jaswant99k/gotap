package goTap

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test BindBody method
func TestBindBody(t *testing.T) {
	engine := New()
	
	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	
	engine.POST("/test", func(c *Context) {
		var data TestData
		if err := JSON.Bind(c.Request, &data); err != nil {
			c.JSON(400, H{"error": err.Error()})
			return
		}
		c.JSON(200, data)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"John","age":30}`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// Test FormFile error handling
func TestFormFileErrors(t *testing.T) {
	engine := New()
	
	engine.POST("/upload", func(c *Context) {
		// Try to get file that doesn't exist
		_, err := c.FormFile("nonexistent")
		if err == nil {
			t.Error("Expected error for missing file")
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/upload", nil)
	engine.ServeHTTP(w, req)
}

// Test SaveUploadedFile error paths
func TestSaveUploadedFileErrors(t *testing.T) {
	engine := New()
	
	engine.POST("/upload", func(c *Context) {
		// Try to save to invalid path
		file, _ := c.FormFile("file")
		if file != nil {
			err := c.SaveUploadedFile(file, "/invalid/path/that/does/not/exist/file.txt")
			if err != nil {
				c.String(500, "error")
				return
			}
		}
		c.String(200, "tested")
	})

	// Create multipart form
	body := "--boundary\r\n"
	body += "Content-Disposition: form-data; name=\"file\"; filename=\"test.txt\"\r\n"
	body += "Content-Type: text/plain\r\n\r\n"
	body += "test content\r\n"
	body += "--boundary--\r\n"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/upload", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	engine.ServeHTTP(w, req)
}

// Test DefaultBinding with different content types
func TestDefaultBindingContentTypes(t *testing.T) {
	type TestData struct {
		Name string `json:"name" xml:"name" form:"name"`
	}
	
	tests := []struct {
		contentType string
		body        string
		expected    string
	}{
		{"application/json", `{"name":"json"}`, "json"},
		{"application/xml", `<TestData><name>xml</name></TestData>`, "xml"},
		{"application/x-www-form-urlencoded", "name=form", "form"},
	}

	for _, tt := range tests {
		eng := New()
		eng.POST("/test", func(c *Context) {
			var data TestData
			if err := c.Bind(&data); err != nil {
				c.JSON(400, H{"error": err.Error()})
				return
			}
			c.JSON(200, data)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/test", strings.NewReader(tt.body))
		req.Header.Set("Content-Type", tt.contentType)
		eng.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("ContentType %s: Expected 200, got %d - body: %s", tt.contentType, w.Code, w.Body.String())
		}
	}
}

// Test Cookie error handling
func TestCookieErrors(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Try to get cookie that doesn't exist
		_, err := c.Cookie("nonexistent")
		if err == nil {
			t.Error("Expected error for missing cookie")
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
}

// Test SetCookie  with various options
func TestSetCookieOptions(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		c.SetCookie("test", "value", 3600, "/", "localhost", true, true)
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	cookies := w.Result().Cookies()
	if len(cookies) == 0 {
		t.Error("Expected cookie to be set")
	}
}

// Test Header method for reading
func TestHeaderGet(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		userAgent := c.Request.Header.Get("User-Agent")
		c.String(200, userAgent)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	engine.ServeHTTP(w, req)

	if w.Body.String() != "TestAgent/1.0" {
		t.Errorf("Expected TestAgent/1.0, got %s", w.Body.String())
	}
}

// Test MustGet panic
func TestMustGetPanic(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic from MustGet for missing key")
			}
		}()
		
		c.MustGet("nonexistent")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
}

// Test initQueryCache edge cases
func TestQueryCacheEdgeCases(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Access query multiple times to test caching
		q1 := c.Query("key")
		q2 := c.Query("key")
		
		if q1 != q2 {
			t.Error("Query cache inconsistent")
		}
		
		c.String(200, q1)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?key=value&key=value2", nil)
	engine.ServeHTTP(w, req)
}

// Test DefaultPostForm with missing value
func TestDefaultPostFormMissing(t *testing.T) {
	engine := New()
	
	engine.POST("/test", func(c *Context) {
		value := c.DefaultPostForm("missing", "default")
		if value != "default" {
			t.Errorf("Expected 'default', got '%s'", value)
		}
		c.String(200, value)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader("other=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(w, req)

	if w.Body.String() != "default" {
		t.Errorf("Expected 'default', got '%s'", w.Body.String())
	}
}

// Test FileAttachment error
func TestFileAttachmentError(t *testing.T) {
	engine := New()
	
	engine.GET("/download", func(c *Context) {
		// Try to send non-existent file
		c.FileAttachment("/nonexistent/file.txt", "download.txt")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/download", nil)
	engine.ServeHTTP(w, req)

	// Should handle error gracefully
	if w.Code == 200 {
		t.Error("Expected error for non-existent file")
	}
}

// Test Redirect with different codes
func TestRedirectCodes(t *testing.T) {
	codes := []int{301, 302, 303, 307, 308}
	
	for _, code := range codes {
		engine := New()
		engine.GET("/test", func(c *Context) {
			c.Redirect(code, "/redirect")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		engine.ServeHTTP(w, req)

		if w.Code != code {
			t.Errorf("Expected %d, got %d", code, w.Code)
		}

		if w.Header().Get("Location") != "/redirect" {
			t.Errorf("Expected Location header")
		}
	}
}

// Test Context Value method
func TestContextValueMethod(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Test Value method (context.Context interface)
		val := c.Value("key")
		if val != nil {
			t.Error("Expected nil for undefined key")
		}
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
}

// Test JSON rendering error path
func TestJSONRenderError(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Try to JSON encode invalid data
		c.JSON(200, make(chan int)) // channels can't be JSON encoded
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	// Should handle error (likely 500 or panic recovery)
}

// Test Data method
func TestDataMethod(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Use Data with byte slice
		c.Data(200, "text/plain", []byte("test"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
	
	if w.Body.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", w.Body.String())
	}
}

// errorReader implements io.Reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

// Test String rendering error
func TestStringRenderError(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// String with format that might cause issues
		c.String(200, "%s", "test")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)

	if w.Body.String() != "test" {
		t.Errorf("Expected 'test', got '%s'", w.Body.String())
	}
}

// Test initFormCache with multipart
func TestFormCacheMultipart(t *testing.T) {
	engine := New()
	
	engine.POST("/test", func(c *Context) {
		// Access form multiple times
		v1 := c.PostForm("field")
		v2 := c.PostForm("field")
		
		if v1 != v2 {
			t.Error("Form cache inconsistent")
		}
		
		c.String(200, v1)
	})

	body := "--boundary\r\n"
	body += "Content-Disposition: form-data; name=\"field\"\r\n\r\n"
	body += "value\r\n"
	body += "--boundary--\r\n"

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=boundary")
	engine.ServeHTTP(w, req)
}

// Test ClientIP with various headers
func TestClientIPHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"X-Forwarded-For", "X-Forwarded-For", "192.168.1.1"},
		{"X-Real-IP", "X-Real-IP", "192.168.1.2"},
		{"RemoteAddr", "", "192.168.1.3:1234"},
	}

	for _, tt := range tests {
		engine := New()
		engine.GET("/test", func(c *Context) {
			ip := c.ClientIP()
			c.String(200, ip)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/test", nil)
		if tt.header != "" {
			req.Header.Set(tt.header, tt.value)
		} else {
			req.RemoteAddr = tt.value
		}
		engine.ServeHTTP(w, req)

		t.Logf("%s: IP = %s", tt.name, w.Body.String())
	}
}

// Test setField error paths
func TestSetFieldErrors(t *testing.T) {
	engine := New()
	
	type ComplexStruct struct {
		InvalidType chan int `form:"invalid"`
	}
	
	engine.POST("/test", func(c *Context) {
		var data ComplexStruct
		// This should handle unsupported types gracefully
		c.ShouldBind(&data)
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader("invalid=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(w, req)
}

// Test Error method variations
func TestErrorMethodVariations(t *testing.T) {
	engine := New()
	
	engine.GET("/test", func(c *Context) {
		// Add multiple errors
		c.Error(fmt.Errorf("error 1"))
		c.Error(fmt.Errorf("error 2"))
		
		errors := c.Errors
		if len(errors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errors))
		}
		
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	engine.ServeHTTP(w, req)
}

// Test ShouldBindBodyWith caching
func TestShouldBindBodyWithCaching(t *testing.T) {
	engine := New()
	
	type TestData struct {
		Name string `json:"name"`
	}
	
	engine.POST("/test", func(c *Context) {
		var data1 TestData
		var data2 TestData
		
		// Bind twice to test caching
		c.ShouldBindBodyWith(&data1, JSON)
		c.ShouldBindBodyWith(&data2, JSON)
		
		if data1.Name != data2.Name {
			t.Error("Body cache inconsistent")
		}
		
		c.JSON(200, data1)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(`{"name":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)
}
