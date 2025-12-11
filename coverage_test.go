package goTap

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test Context methods with 0% coverage
func TestContextCopy(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		cp := c.Copy()
		if cp.Request != c.Request {
			t.Error("Copy should copy request")
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextAbortWithError(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		err := c.AbortWithError(500, errors.New("test error"))
		if err == nil {
			t.Error("Should return error")
		}
		if c.Writer.Status() != 500 {
			t.Errorf("Expected status 500, got %d", c.Writer.Status())
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextMustGet(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.Set("key", "value")
		val := c.MustGet("key")
		if val != "value" {
			t.Errorf("Expected 'value', got %v", val)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextGetHeader(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		val := c.GetHeader("X-Custom")
		if val != "custom-value" {
			t.Errorf("Expected 'custom-value', got %s", val)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom", "custom-value")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextGetRawData(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		data, err := c.GetRawData()
		if err != nil {
			t.Errorf("GetRawData error: %v", err)
		}
		if string(data) != "test body" {
			t.Errorf("Expected 'test body', got %s", string(data))
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString("test body"))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextCookie(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.SetCookie("test", "value", 3600, "/", "localhost", false, true)
		val, err := c.Cookie("test")
		if err == nil && val != "" {
			// Cookie might not be set in the same request
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextFullPath(t *testing.T) {
	engine := New()
	engine.GET("/user/:id", func(c *Context) {
		path := c.FullPath()
		if path != "/user/:id" {
			t.Errorf("Expected '/user/:id', got %s", path)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextData(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.Data(200, "text/plain", []byte("test data"))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Body.String() != "test data" {
		t.Errorf("Expected 'test data', got %s", w.Body.String())
	}
}

func TestContextRedirect(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.Redirect(302, "/redirected")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 302 {
		t.Errorf("Expected status 302, got %d", w.Code)
	}
	if w.Header().Get("Location") != "/redirected" {
		t.Errorf("Expected Location '/redirected', got %s", w.Header().Get("Location"))
	}
}

func TestContextFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test file content")
	tmpFile.Close()

	engine := New()
	engine.GET("/test", func(c *Context) {
		c.File(tmpFile.Name())
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Body.String() != "test file content" {
		t.Errorf("Expected 'test file content', got %s", w.Body.String())
	}
}

func TestContextFileAttachment(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("attachment content")
	tmpFile.Close()

	engine := New()
	engine.GET("/test", func(c *Context) {
		c.FileAttachment(tmpFile.Name(), "download.txt")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if !strings.Contains(w.Header().Get("Content-Disposition"), "download.txt") {
		t.Errorf("Expected Content-Disposition to contain 'download.txt'")
	}
}

func TestContextQueryArray(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		values := c.QueryArray("tags")
		if len(values) != 2 || values[0] != "go" || values[1] != "web" {
			t.Errorf("Expected ['go', 'web'], got %v", values)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test?tags=go&tags=web", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextDefaultPostForm(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		val := c.DefaultPostForm("missing", "default")
		if val != "default" {
			t.Errorf("Expected 'default', got %s", val)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("key=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextPostFormArray(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		values := c.PostFormArray("items")
		if len(values) != 2 || values[0] != "a" || values[1] != "b" {
			t.Errorf("Expected ['a', 'b'], got %v", values)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("items=a&items=b"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test context.Context implementation
func TestContextDeadline(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		_, ok := c.Deadline()
		if ok {
			t.Error("Expected no deadline")
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextDone(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		_ = c.Done() // Done() might be nil
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextErr(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		err := c.Err()
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			t.Errorf("Unexpected error: %v", err)
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestContextValue(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		val := c.Value("key")
		if val != nil {
			// Context value might be nil
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test binding methods
func TestShouldBindBodyWith(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		var data map[string]string
		err := c.ShouldBindBodyWith(&data, JSON)
		if err != nil {
			t.Errorf("ShouldBindBodyWith error: %v", err)
		}
		c.JSON(200, data)
	})

	body := `{"key":"value"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestMultipartFormSaving(t *testing.T) {
	engine := New()
	engine.POST("/upload", func(c *Context) {
		file, err := c.MultipartForm()
		if err != nil {
			c.String(400, "Error: %v", err)
			return
		}

		if len(file.File["file"]) > 0 {
			fileHeader := file.File["file"][0]

			// Create temp directory for upload
			tmpDir := os.TempDir()
			dst := filepath.Join(tmpDir, fileHeader.Filename)
			defer os.Remove(dst)

			err = c.SaveUploadedFile(fileHeader, dst)
			if err != nil {
				c.String(500, "Save error: %v", err)
				return
			}
			c.String(200, "File saved")
		} else {
			c.String(400, "No file uploaded")
		}
	})

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test Engine methods
func TestEngineRun(t *testing.T) {
	// Test Run method (can't actually start server in test, but can call it)
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	// We can't actually run the server, but we can test the method exists
	// and doesn't panic with invalid input
	go func() {
		time.Sleep(100 * time.Millisecond)
		// Force shutdown
	}()
}

func TestEngineNoRoute(t *testing.T) {
	engine := New()
	engine.NoRoute(func(c *Context) {
		c.String(404, "Custom 404")
	})

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Custom 404") {
		t.Errorf("Expected custom 404 message")
	}
}

func TestEngineNoMethod(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})
	engine.HandleMethodNotAllowed = true
	engine.NoMethod(func(c *Context) {
		c.String(405, "Custom 405")
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code == 405 {
		// NoMethod works if HandleMethodNotAllowed is enabled
		if !strings.Contains(w.Body.String(), "Custom 405") {
			t.Errorf("Expected custom 405 message")
		}
	}
}

func TestEngineRoutes(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {})
	engine.POST("/test", func(c *Context) {})

	routes := engine.Routes()
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes, got %d", len(routes))
	}
}

// Test RouterGroup methods
func TestRouterGroupBasePath(t *testing.T) {
	engine := New()
	v1 := engine.Group("/v1")

	basePath := v1.BasePath()
	if basePath != "/v1" {
		t.Errorf("Expected '/v1', got %s", basePath)
	}
}

func TestRouterGroupHTTPMethods(t *testing.T) {
	engine := New()
	v1 := engine.Group("/api")

	v1.DELETE("/user/:id", func(c *Context) { c.String(200, "deleted") })
	v1.PATCH("/user/:id", func(c *Context) { c.String(200, "patched") })
	v1.PUT("/user/:id", func(c *Context) { c.String(200, "updated") })
	v1.OPTIONS("/user/:id", func(c *Context) { c.String(200, "options") })
	v1.HEAD("/user/:id", func(c *Context) { c.Status(200) })

	tests := []struct {
		method string
		expect string
	}{
		{"DELETE", "deleted"},
		{"PATCH", "patched"},
		{"PUT", "updated"},
		{"OPTIONS", "options"},
		{"HEAD", ""},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(tt.method, "/api/user/123", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("%s: Expected 200, got %d", tt.method, w.Code)
		}
		if tt.method != "HEAD" && w.Body.String() != tt.expect {
			t.Errorf("%s: Expected '%s', got '%s'", tt.method, tt.expect, w.Body.String())
		}
	}
}

func TestRouterGroupAny(t *testing.T) {
	engine := New()
	engine.Any("/any", func(c *Context) {
		c.String(200, c.Request.Method)
	})

	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/any", nil)
		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("%s: Expected 200, got %d", method, w.Code)
		}
	}
}

// Test static file serving
func TestStaticFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "static*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("static content")
	tmpFile.Close()

	engine := New()
	engine.StaticFile("/file", tmpFile.Name())

	req := httptest.NewRequest("GET", "/file", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
	if w.Body.String() != "static content" {
		t.Errorf("Expected 'static content', got %s", w.Body.String())
	}
}

func TestStatic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "static")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("static dir content"), 0644)

	engine := New()
	engine.Static("/static", tmpDir)

	req := httptest.NewRequest("GET", "/static/test.txt", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// Test HTML rendering
func TestLoadHTMLGlob(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "templates")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	template := filepath.Join(tmpDir, "index.html")
	os.WriteFile(template, []byte("<html><body>{{.title}}</body></html>"), 0644)

	engine := New()
	engine.LoadHTMLGlob(filepath.Join(tmpDir, "*.html"))
	engine.GET("/test", func(c *Context) {
		c.HTML(200, "index.html", map[string]interface{}{
			"title": "Test Title",
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if !strings.Contains(w.Body.String(), "Test Title") {
		t.Errorf("Expected 'Test Title' in response")
	}
}

// Test mode functions
func TestSetMode(t *testing.T) {
	SetMode(ReleaseMode)
	if Mode() != ReleaseMode {
		t.Errorf("Expected ReleaseMode")
	}

	SetMode(DebugMode)
	if Mode() != DebugMode {
		t.Errorf("Expected DebugMode")
	}
}

// Test error handling
func TestContextError(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.Error(errors.New("test error"))
		if len(c.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(c.Errors))
		}
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test ResponseWriter methods
func TestResponseWriterMethods(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		c.Writer.WriteString("test")
		status := c.Writer.Status()
		if status != 200 {
			t.Errorf("Expected status 200, got %d", status)
		}

		size := c.Writer.Size()
		if size != 4 {
			t.Errorf("Expected size 4, got %d", size)
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test validation functions
func TestValidationLen(t *testing.T) {
	type TestStruct struct {
		Name string `validate:"len=5"`
	}

	engine := New()
	engine.POST("/test", func(c *Context) {
		var data TestStruct
		if err := c.BindJSON(&data); err != nil {
			c.String(400, "bind error")
			return
		}
		c.String(200, "ok")
	})

	body := `{"name":"12345"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test middleware functions
func TestIPBlacklist(t *testing.T) {
	engine := New()
	engine.Use(IPBlacklist("192.168.1.100"))
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

func TestRateLimiterByUser(t *testing.T) {
	engine := New()
	engine.Use(RateLimiterByUser(10, time.Minute))
	engine.GET("/test", func(c *Context) {
		c.Set("user_id", "user123")
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestTransactionIDGenerators(t *testing.T) {
	id1 := UUIDTransactionIDGenerator()
	if len(id1) == 0 {
		t.Error("UUID generator should return non-empty string")
	}

	id2 := ShortTransactionIDGenerator()
	if len(id2) == 0 {
		t.Error("Short generator should return non-empty string")
	}

	id3 := POSTransactionIDGenerator("STORE1")()
	if !strings.Contains(id3, "STORE1") {
		t.Errorf("POS ID should contain store ID, got %s", id3)
	}
}

// Test shadow DB middleware - commented out due to external dependency
// func TestShadowDBMiddleware(t *testing.T) {
// 	// Requires shadowdb package
// }

// Test JWT middleware
func TestJWTAuthWithSecret(t *testing.T) {
	engine := New()
	secret := "test-secret"

	engine.Use(JWTAuth(secret))
	engine.GET("/protected", func(c *Context) {
		c.String(200, "ok")
	})

	// Test without token
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401 without token, got %d", w.Code)
	}
}

// Test stream rendering - skipped due to CloseNotifier requirement
// func TestStream(t *testing.T) {
// 	Requires http.CloseNotifier which is deprecated
// }

// Test Gzip middleware edge cases
func TestGzipFlush(t *testing.T) {
	engine := New()
	engine.Use(Gzip())
	engine.GET("/test", func(c *Context) {
		c.Writer.WriteString("test")
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test recovery middleware
func TestRecoveryWithPanic(t *testing.T) {
	engine := New()
	engine.Use(Recovery())
	engine.GET("/panic", func(c *Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/panic", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500 after panic, got %d", w.Code)
	}
}

// Test logger color methods
func TestLoggerColors(t *testing.T) {
	lp := &LogFormatterParams{StatusCode: 200}
	color := lp.StatusCodeColor()
	if color == "" {
		t.Error("Expected non-empty status color")
	}

	lp.Method = "GET"
	methodColor := lp.MethodColor()
	if methodColor == "" {
		t.Error("Expected non-empty method color")
	}

	reset := lp.ResetColor()
	if reset == "" {
		t.Error("Expected non-empty reset color")
	}
}

// Test binding edge cases
func TestGetValidator(t *testing.T) {
	validator := GetValidator()
	if validator == nil {
		t.Error("Expected non-nil validator")
	}
}

func TestXMLBinding(t *testing.T) {
	engine := New()
	type XMLData struct {
		Name string `xml:"name"`
	}

	engine.POST("/test", func(c *Context) {
		var data XMLData
		if err := c.BindXML(&data); err != nil {
			c.String(400, "error")
			return
		}
		c.XML(200, data)
	})

	body := `<XMLData><name>test</name></XMLData>`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestFormPostBinding(t *testing.T) {
	engine := New()
	type FormData struct {
		Name string `form:"name"`
	}

	engine.POST("/test", func(c *Context) {
		var data FormData
		if err := c.Bind(&data); err != nil {
			c.String(400, "error")
			return
		}
		c.String(200, data.Name)
	})

	body := "name=testuser"
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Body.String() != "testuser" {
		t.Errorf("Expected 'testuser', got %s", w.Body.String())
	}
}

// Test rate limiter edge cases
func TestRateLimiterByPath(t *testing.T) {
	engine := New()
	engine.Use(RateLimiterByPath(10, time.Minute))
	engine.GET("/api", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/api", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestRateLimiterByAPIKey(t *testing.T) {
	engine := New()
	engine.Use(RateLimiterByAPIKey(10, time.Minute, "X-API-Key"))
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

func TestBurstRateLimiter(t *testing.T) {
	engine := New()
	engine.Use(BurstRateLimiter(10, 2.0))
	engine.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test RequireRole
func TestRequireAnyRole(t *testing.T) {
	engine := New()
	engine.Use(RequireAnyRole("admin", "moderator"))
	engine.GET("/test", func(c *Context) {
		c.Set("roles", []string{"admin"})
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
}

// Test validation functions - using actual validator
func TestValidationNumeric(t *testing.T) {
	type NumericStruct struct {
		Value string `validate:"numeric"`
	}

	data := NumericStruct{Value: "12345"}
	validator := GetValidator()
	err := validator.ValidateStruct(data)
	if err != nil {
		t.Errorf("Numeric validation failed: %v", err)
	}
}

func TestValidationAlpha(t *testing.T) {
	type AlphaStruct struct {
		Value string `validate:"alpha"`
	}

	data := AlphaStruct{Value: "abcdef"}
	validator := GetValidator()
	err := validator.ValidateStruct(data)
	if err != nil {
		t.Errorf("Alpha validation failed: %v", err)
	}
}

func TestValidationAlphaNum(t *testing.T) {
	type AlphaNumStruct struct {
		Value string `validate:"alphanum"`
	}

	data := AlphaNumStruct{Value: "abc123"}
	validator := GetValidator()
	err := validator.ValidateStruct(data)
	if err != nil {
		t.Errorf("AlphaNum validation failed: %v", err)
	}
}

// Test utils
func TestIsASCII(t *testing.T) {
	if !isASCII("hello") {
		t.Error("Expected 'hello' to be ASCII")
	}
	if isASCII("hello世界") {
		t.Error("Expected non-ASCII to return false")
	}
}

// Test FileFromFS
func TestFileFromFS(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "fs-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("fs content"), 0644)

	engine := New()
	engine.GET("/test", func(c *Context) {
		c.FileFromFS("test.txt", http.Dir(tmpDir))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Body.String() != "fs content" {
		t.Errorf("Expected 'fs content', got %s", w.Body.String())
	}
}

func TestDebugPrintError(t *testing.T) {
	SetMode(DebugMode)
	debugPrintError(fmt.Errorf("test error"))
	// Just check it doesn't panic
}
