// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestContextQuery(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		name := c.Query("name")
		c.String(200, "Hello %s", name)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?name=John", nil)
	r.ServeHTTP(w, req)

	if w.Body.String() != "Hello John" {
		t.Errorf("Expected 'Hello John', got '%s'", w.Body.String())
	}
}

func TestContextDefaultQuery(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		name := c.DefaultQuery("name", "Guest")
		c.String(200, "Hello %s", name)
	})

	// Test with query parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?name=John", nil)
	r.ServeHTTP(w, req)

	if w.Body.String() != "Hello John" {
		t.Errorf("Expected 'Hello John', got '%s'", w.Body.String())
	}

	// Test without query parameter (should use default)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Body.String() != "Hello Guest" {
		t.Errorf("Expected 'Hello Guest', got '%s'", w.Body.String())
	}
}

func TestContextPostForm(t *testing.T) {
	r := New()
	r.POST("/test", func(c *Context) {
		name := c.PostForm("name")
		c.String(200, "Hello %s", name)
	})

	w := httptest.NewRecorder()
	form := url.Values{}
	form.Add("name", "John")
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Body.String() != "Hello John" {
		t.Errorf("Expected 'Hello John', got '%s'", w.Body.String())
	}
}

func TestContextSetGet(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.Set("user", "John")
		c.Set("age", 25)

		user, exists := c.Get("user")
		if !exists {
			t.Error("Expected user to exist")
		}
		if user != "John" {
			t.Errorf("Expected 'John', got '%v'", user)
		}

		age, exists := c.Get("age")
		if !exists {
			t.Error("Expected age to exist")
		}
		if age != 25 {
			t.Errorf("Expected 25, got %v", age)
		}

		_, exists = c.Get("nonexistent")
		if exists {
			t.Error("Expected nonexistent key to not exist")
		}

		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestContextJSON(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.JSON(200, H{
			"status":  "ok",
			"message": "test",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != MIMEJSON {
		t.Errorf("Expected Content-Type '%s', got '%s'", MIMEJSON, contentType)
	}

	expected := `{"message":"test","status":"ok"}`
	body := strings.TrimSpace(w.Body.String())
	if body != expected {
		t.Errorf("Expected '%s', got '%s'", expected, body)
	}
}

func TestContextClientIP(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		ip := c.ClientIP()
		c.String(200, ip)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	r.ServeHTTP(w, req)

	if w.Body.String() != "192.168.1.1" {
		t.Errorf("Expected '192.168.1.1', got '%s'", w.Body.String())
	}
}

func TestContextParam(t *testing.T) {
	r := New()
	r.GET("/user/:id/profile/:section", func(c *Context) {
		id := c.Param("id")
		section := c.Param("section")
		c.String(200, "User %s - Section %s", id, section)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/user/123/profile/settings", nil)
	r.ServeHTTP(w, req)

	expected := "User 123 - Section settings"
	if w.Body.String() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, w.Body.String())
	}
}

func TestContextHeader(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.Header("X-Custom-Header", "test-value")
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	header := w.Header().Get("X-Custom-Header")
	if header != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", header)
	}
}

func TestContextStatus(t *testing.T) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.Status(201)
		c.String(0, "created")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 201 {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestContextNext(t *testing.T) {
	r := New()

	order := []string{}

	r.Use(func(c *Context) {
		order = append(order, "middleware1-before")
		c.Next()
		order = append(order, "middleware1-after")
	})

	r.Use(func(c *Context) {
		order = append(order, "middleware2-before")
		c.Next()
		order = append(order, "middleware2-after")
	})

	r.GET("/test", func(c *Context) {
		order = append(order, "handler")
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(order) != len(expected) {
		t.Errorf("Expected %d items, got %d", len(expected), len(order))
		return
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("Expected order[%d] = '%s', got '%s'", i, v, order[i])
		}
	}
}

func TestContextAbort(t *testing.T) {
	r := New()

	executed := false

	r.Use(func(c *Context) {
		c.Abort()
	})

	r.GET("/test", func(c *Context) {
		executed = true
		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if executed {
		t.Error("Handler should not execute after Abort()")
	}
}

func TestContextIsAborted(t *testing.T) {
	r := New()

	r.GET("/test", func(c *Context) {
		if c.IsAborted() {
			t.Error("Should not be aborted initially")
		}

		c.Abort()

		if !c.IsAborted() {
			t.Error("Should be aborted after Abort()")
		}

		c.String(200, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}
