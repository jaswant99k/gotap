package goTap

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// Test structs
type LoginForm struct {
	Username string `form:"username" json:"username" xml:"username" validate:"required,min=3"`
	Password string `form:"password" json:"password" xml:"password" validate:"required,min=6"`
}

type UserQuery struct {
	Page  int    `form:"page" validate:"min=1"`
	Limit int    `form:"limit" validate:"min=1,max=100"`
	Sort  string `form:"sort" validate:"oneof=asc desc"`
}

type UserURI struct {
	ID   string `uri:"id" validate:"required"`
	Name string `uri:"name"`
}

type UserHeader struct {
	Authorization string `header:"Authorization" validate:"required"`
	UserAgent     string `header:"User-Agent"`
	ContentType   string `header:"Content-Type"`
}

type ProductInput struct {
	Name        string   `json:"name" validate:"required,min=3,max=100"`
	Price       float64  `json:"price" validate:"required,min=0.01"`
	Quantity    int      `json:"quantity" validate:"min=0"`
	Category    string   `json:"category" validate:"required,oneof=electronics clothing food"`
	Tags        []string `json:"tags"`
	Email       string   `json:"email" validate:"email"`
	Website     string   `json:"website" validate:"url"`
	Description string   `json:"description" validate:"max=500"`
}

// ========== JSON Binding Tests ==========

func TestBindJSON(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.BindJSON(&form); err != nil {
			return // Error response already sent by BindJSON
		}
		c.JSON(http.StatusOK, H{
			"username": form.Username,
			"password": form.Password,
		})
	})

	body := `{"username":"testuser","password":"secret123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["username"] != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", result["username"])
	}
	if result["password"] != "secret123" {
		t.Errorf("Expected password 'secret123', got '%s'", result["password"])
	}
}

func TestShouldBindJSON(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, H{"username": form.Username})
	})

	// Test valid JSON
	body := `{"username":"testuser","password":"secret123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/login", strings.NewReader(`{invalid json}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

// ========== XML Binding Tests ==========

func TestBindXML(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.BindXML(&form); err != nil {
			return
		}
		c.JSON(http.StatusOK, H{"username": form.Username})
	})

	body := `<LoginForm><username>testuser</username><password>secret123</password></LoginForm>`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestShouldBindXML(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.ShouldBindXML(&form); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, H{"username": form.Username})
	})

	body := `<LoginForm><username>testuser</username><password>secret123</password></LoginForm>`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/xml")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Query Binding Tests ==========

func TestBindQuery(t *testing.T) {
	router := New()
	router.GET("/users", func(c *Context) {
		var query UserQuery
		if err := c.BindQuery(&query); err != nil {
			return
		}
		c.JSON(http.StatusOK, H{
			"page":  query.Page,
			"limit": query.Limit,
			"sort":  query.Sort,
		})
	})

	req := httptest.NewRequest("GET", "/users?page=2&limit=20&sort=asc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["page"] != float64(2) {
		t.Errorf("Expected page 2, got %v", result["page"])
	}
}

func TestShouldBindQuery(t *testing.T) {
	router := New()
	router.GET("/users", func(c *Context) {
		var query UserQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, query)
	})

	req := httptest.NewRequest("GET", "/users?page=1&limit=10&sort=desc", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Form Binding Tests ==========

func TestBindForm(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.ShouldBindWith(&form, Form); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, H{"username": form.Username})
	})

	formData := url.Values{}
	formData.Set("username", "testuser")
	formData.Set("password", "secret123")

	req := httptest.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Multipart Form Tests ==========

func TestMultipartForm(t *testing.T) {
	router := New()
	router.POST("/upload", func(c *Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, H{
			"filename": file.Filename,
			"size":     file.Size,
		})
	})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, _ := writer.CreateFormFile("file", "test.txt")
	part.Write([]byte("test content"))
	writer.Close()

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== URI Binding Tests ==========

func TestBindUri(t *testing.T) {
	router := New()
	router.GET("/user/:id/:name", func(c *Context) {
		var uri UserURI
		if err := c.BindUri(&uri); err != nil {
			return
		}
		c.JSON(http.StatusOK, H{
			"id":   uri.ID,
			"name": uri.Name,
		})
	})

	req := httptest.NewRequest("GET", "/user/123/john", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	json.Unmarshal(w.Body.Bytes(), &result)

	if result["id"] != "123" {
		t.Errorf("Expected id '123', got '%s'", result["id"])
	}
	if result["name"] != "john" {
		t.Errorf("Expected name 'john', got '%s'", result["name"])
	}
}

func TestShouldBindUri(t *testing.T) {
	router := New()
	router.GET("/user/:id", func(c *Context) {
		var uri UserURI
		if err := c.ShouldBindUri(&uri); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, uri)
	})

	req := httptest.NewRequest("GET", "/user/456", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Header Binding Tests ==========

func TestBindHeader(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		var header UserHeader
		if err := c.BindHeader(&header); err != nil {
			return
		}
		c.JSON(http.StatusOK, H{
			"auth":         header.Authorization,
			"user_agent":   header.UserAgent,
			"content_type": header.ContentType,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "goTap/1.0")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestShouldBindHeader(t *testing.T) {
	router := New()
	router.GET("/test", func(c *Context) {
		var header UserHeader
		if err := c.ShouldBindHeader(&header); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, header)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Validation Tests ==========

func TestValidationRequired(t *testing.T) {
	router := New()
	router.POST("/product", func(c *Context) {
		var product ProductInput
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, product)
	})

	// Missing required field
	body := `{"price": 99.99}`
	req := httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing required field, got %d", w.Code)
	}
}

func TestValidationMin(t *testing.T) {
	router := New()
	router.POST("/product", func(c *Context) {
		var product ProductInput
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, product)
	})

	// Name too short (min=3)
	body := `{"name":"AB","price":99.99,"category":"electronics"}`
	req := httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for min validation, got %d", w.Code)
	}
}

func TestValidationEmail(t *testing.T) {
	router := New()
	router.POST("/product", func(c *Context) {
		var product ProductInput
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, product)
	})

	// Invalid email
	body := `{"name":"Product","price":99.99,"category":"electronics","email":"invalid-email"}`
	req := httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid email, got %d", w.Code)
	}

	// Valid email
	body = `{"name":"Product","price":99.99,"category":"electronics","email":"test@example.com"}`
	req = httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid email, got %d", w.Code)
	}
}

func TestValidationOneOf(t *testing.T) {
	router := New()
	router.POST("/product", func(c *Context) {
		var product ProductInput
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, product)
	})

	// Invalid category (not in: electronics, clothing, food)
	body := `{"name":"Product","price":99.99,"category":"invalid"}`
	req := httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid oneof, got %d", w.Code)
	}
}

func TestValidationURL(t *testing.T) {
	router := New()
	router.POST("/product", func(c *Context) {
		var product ProductInput
		if err := c.ShouldBindJSON(&product); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, product)
	})

	// Invalid URL
	body := `{"name":"Product","price":99.99,"category":"electronics","website":"not-a-url"}`
	req := httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid URL, got %d", w.Code)
	}

	// Valid URL
	body = `{"name":"Product","price":99.99,"category":"electronics","website":"https://example.com"}`
	req = httptest.NewRequest("POST", "/product", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for valid URL, got %d", w.Code)
	}
}

// ========== Auto-Binding Tests ==========

func TestShouldBind(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.ShouldBind(&form); err != nil {
			c.JSON(http.StatusBadRequest, H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, form)
	})

	// Test JSON
	body := `{"username":"testuser","password":"secret123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for JSON, got %d", w.Code)
	}

	// Test Form
	formData := url.Values{}
	formData.Set("username", "testuser")
	formData.Set("password", "secret123")
	req = httptest.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for Form, got %d", w.Code)
	}
}

func TestBind(t *testing.T) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		if err := c.Bind(&form); err != nil {
			return // Error already sent
		}
		c.JSON(http.StatusOK, form)
	})

	body := `{"username":"testuser","password":"secret123"}`
	req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// ========== Benchmarks ==========

func BenchmarkBindJSON(b *testing.B) {
	router := New()
	router.POST("/login", func(c *Context) {
		var form LoginForm
		c.BindJSON(&form)
		c.JSON(http.StatusOK, form)
	})

	body := `{"username":"testuser","password":"secret123"}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkBindQuery(b *testing.B) {
	router := New()
	router.GET("/users", func(c *Context) {
		var query UserQuery
		c.BindQuery(&query)
		c.JSON(http.StatusOK, query)
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/users?page=2&limit=20&sort=asc", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkValidation(b *testing.B) {
	validator := &DefaultValidator{}
	product := ProductInput{
		Name:     "Test Product",
		Price:    99.99,
		Category: "electronics",
		Email:    "test@example.com",
		Website:  "https://example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateStruct(&product)
	}
}
