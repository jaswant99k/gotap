package goTap

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Test binding Name methods through exported variables
func TestBindingNames(t *testing.T) {
	tests := []struct {
		binding  Binding
		expected string
	}{
		{JSON, "json"},
		{XML, "xml"},
		{Form, "form"},
		{Query, "query"},
		{FormPost, "form-post"},
		{FormMultipart, "multipart/form-data"},
		{Header, "header"},
	}

	for _, tt := range tests {
		if name := tt.binding.Name(); name != tt.expected {
			t.Errorf("Expected binding name '%s', got '%s'", tt.expected, name)
		}
	}
	
	// Test URI binding separately
	if name := Uri.Name(); name != "uri" {
		t.Errorf("Expected URI binding name 'uri', got '%s'", name)
	}
}

// Test setSliceField with various types
func TestSetSliceField(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		var data struct {
			Tags []string `form:"tags"`
			IDs  []int    `form:"ids"`
		}
		
		c.Request.URL.RawQuery = "tags=go&tags=web&tags=framework&ids=1&ids=2&ids=3"
		
		if err := c.ShouldBindQuery(&data); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if len(data.Tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(data.Tags))
		}
		if data.Tags[0] != "go" || data.Tags[1] != "web" || data.Tags[2] != "framework" {
			t.Errorf("Tags mismatch: %v", data.Tags)
		}
		
		if len(data.IDs) != 3 {
			t.Errorf("Expected 3 IDs, got %d", len(data.IDs))
		}
		if data.IDs[0] != 1 || data.IDs[1] != 2 || data.IDs[2] != 3 {
			t.Errorf("IDs mismatch: %v", data.IDs)
		}
		
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?tags=go&tags=web&tags=framework&ids=1&ids=2&ids=3", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test binding with complex nested structures
func TestBindingNestedStructs(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		type Address struct {
			City    string `json:"city"`
			Country string `json:"country"`
		}
		type User struct {
			Name    string  `json:"name"`
			Age     int     `json:"age"`
			Address Address `json:"address"`
		}
		
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if user.Name != "John" {
			t.Errorf("Expected 'John', got '%s'", user.Name)
		}
		if user.Age != 30 {
			t.Errorf("Expected 30, got %d", user.Age)
		}
		if user.Address.City != "NYC" {
			t.Errorf("Expected 'NYC', got '%s'", user.Address.City)
		}
		if user.Address.Country != "USA" {
			t.Errorf("Expected 'USA', got '%s'", user.Address.Country)
		}
		
		c.String(200, "ok")
	})

	jsonData := `{"name":"John","age":30,"address":{"city":"NYC","country":"USA"}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(jsonData))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test form binding with multiple values
func TestFormBindingMultipleValues(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		var data struct {
			Colors []string `form:"color"`
			Count  int      `form:"count"`
		}
		
		if err := c.ShouldBind(&data); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if len(data.Colors) != 3 {
			t.Errorf("Expected 3 colors, got %d", len(data.Colors))
		}
		if data.Count != 5 {
			t.Errorf("Expected count 5, got %d", data.Count)
		}
		
		c.String(200, "ok")
	})

	form := url.Values{}
	form.Add("color", "red")
	form.Add("color", "green")
	form.Add("color", "blue")
	form.Add("count", "5")
	
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

// Test multipart form with file upload
func TestMultipartFormBinding(t *testing.T) {
	engine := New()
	engine.POST("/upload", func(c *Context) {
		var data struct {
			Title       string `form:"title"`
			Description string `form:"description"`
		}
		
		if err := c.ShouldBind(&data); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		// Also test file retrieval
		file, err := c.FormFile("document")
		if err != nil {
			t.Errorf("FormFile failed: %v", err)
		} else if file.Filename != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got '%s'", file.Filename)
		}
		
		if data.Title != "Test Document" {
			t.Errorf("Expected 'Test Document', got '%s'", data.Title)
		}
		
		c.String(200, "ok")
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("title", "Test Document")
	writer.WriteField("description", "A test file")
	
	part, _ := writer.CreateFormFile("document", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test binding with pointer fields
func TestBindingPointerFields(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		var data struct {
			Name  *string `form:"name"`
			Age   *int    `form:"age"`
			Active *bool  `form:"active"`
		}
		
		if err := c.ShouldBindQuery(&data); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if data.Name == nil || *data.Name != "Alice" {
			t.Error("Name pointer binding failed")
		}
		if data.Age == nil || *data.Age != 25 {
			t.Error("Age pointer binding failed")
		}
		if data.Active == nil || *data.Active != true {
			t.Error("Active pointer binding failed")
		}
		
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test?name=Alice&age=25&active=true", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test header binding with custom header names
func TestHeaderBindingCustomNames(t *testing.T) {
	engine := New()
	engine.GET("/test", func(c *Context) {
		var headers struct {
			Token     string `header:"X-Api-Token"`
			UserAgent string `header:"User-Agent"`
			Accept    string `header:"Accept"`
		}
		
		if err := c.ShouldBindHeader(&headers); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if headers.Token != "secret123" {
			t.Errorf("Expected 'secret123', got '%s'", headers.Token)
		}
		if headers.UserAgent != "TestAgent/1.0" {
			t.Errorf("Expected 'TestAgent/1.0', got '%s'", headers.UserAgent)
		}
		if headers.Accept != "application/json" {
			t.Errorf("Expected 'application/json', got '%s'", headers.Accept)
		}
		
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Api-Token", "secret123")
	req.Header.Set("User-Agent", "TestAgent/1.0")
	req.Header.Set("Accept", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test URI binding with multiple parameters
func TestURIBindingMultipleParams(t *testing.T) {
	engine := New()
	engine.GET("/user/:userId/post/:postId", func(c *Context) {
		var params struct {
			UserID string `uri:"userId"`
			PostID string `uri:"postId"`
		}
		
		if err := c.ShouldBindUri(&params); err != nil {
			t.Errorf("Bind failed: %v", err)
			c.String(500, "error")
			return
		}
		
		if params.UserID != "123" {
			t.Errorf("Expected '123', got '%s'", params.UserID)
		}
		if params.PostID != "456" {
			t.Errorf("Expected '456', got '%s'", params.PostID)
		}
		
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/user/123/post/456", nil)
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test binding errors with invalid data
func TestBindingInvalidData(t *testing.T) {
	engine := New()
	engine.POST("/test", func(c *Context) {
		var data struct {
			Name string `json:"name" binding:"required"`
			Age  int    `json:"age" binding:"required,min=1"`
		}
		
		err := c.ShouldBindJSON(&data)
		if err == nil {
			t.Error("Expected binding error for invalid JSON")
		}
		c.String(400, "invalid")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/test", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
