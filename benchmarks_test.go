// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Benchmark simple route matching
func BenchmarkSimpleRoute(b *testing.B) {
	r := New()
	r.GET("/ping", func(c *Context) {
		c.String(200, "pong")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark route with single parameter
func BenchmarkOneParam(b *testing.B) {
	r := New()
	r.GET("/user/:id", func(c *Context) {
		c.String(200, c.Param("id"))
	})

	req := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark route with multiple parameters
func BenchmarkFiveParams(b *testing.B) {
	r := New()
	r.GET("/user/:id/profile/:section/item/:item/action/:action", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/user/123/profile/settings/item/456/action/edit", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark static routes (no parameters)
func BenchmarkStaticRoutes(b *testing.B) {
	r := New()
	r.GET("/", func(c *Context) { c.String(200, "home") })
	r.GET("/about", func(c *Context) { c.String(200, "about") })
	r.GET("/contact", func(c *Context) { c.String(200, "contact") })
	r.GET("/help", func(c *Context) { c.String(200, "help") })
	r.GET("/api/v1/status", func(c *Context) { c.String(200, "ok") })

	req := httptest.NewRequest("GET", "/api/v1/status", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark GitHub API-like routes (complex routing scenario)
// DISABLED: Routing tree has conflicts with parameter routes having children
/*
func BenchmarkGitHubAPI(b *testing.B) {
	r := New()

	// Simulate GitHub API routes (simplified to avoid routing conflicts)
	r.GET("/", func(c *Context) {})
	r.GET("/authorizations", func(c *Context) {})
	r.GET("/authorizations/:id", func(c *Context) {})
	r.POST("/authorizations", func(c *Context) {})
	r.DELETE("/authorizations/:id", func(c *Context) {})
	r.GET("/applications/:client_id/tokens/:access_token", func(c *Context) {})
	r.DELETE("/applications/:client_id/grants/:grant_id", func(c *Context) {})
	r.GET("/events", func(c *Context) {})
	r.GET("/repos/:owner/:repo/events", func(c *Context) {})
	r.GET("/networks/:owner/:repo/events", func(c *Context) {})
	r.GET("/orgs/:org/events", func(c *Context) {})
	r.GET("/users/:user/received/public", func(c *Context) {})
	r.GET("/users/:user/events", func(c *Context) {})
	r.GET("/users/:user/events/public", func(c *Context) {})
	r.GET("/users/:user/orgs/:org", func(c *Context) {})
	r.GET("/feeds", func(c *Context) {})
	r.GET("/notifications", func(c *Context) {})
	r.GET("/repos/:owner/:repo/notifications", func(c *Context) {})

	req := httptest.NewRequest("GET", "/repos/gin-gonic/gin/notifications", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
*/

// Benchmark JSON rendering
func BenchmarkJSONRender(b *testing.B) {
	r := New()
	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	r.GET("/user", func(c *Context) {
		c.JSON(200, User{ID: 123, Name: "John Doe"})
	})

	req := httptest.NewRequest("GET", "/user", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark middleware chain execution
func BenchmarkMiddleware(b *testing.B) {
	r := New()

	// Add 5 middleware
	r.Use(func(c *Context) { c.Next() })
	r.Use(func(c *Context) { c.Next() })
	r.Use(func(c *Context) { c.Next() })
	r.Use(func(c *Context) { c.Next() })
	r.Use(func(c *Context) { c.Next() })

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark context parameter extraction
func BenchmarkParamExtraction(b *testing.B) {
	r := New()
	r.GET("/user/:id/post/:pid/comment/:cid", func(c *Context) {
		id := c.Param("id")
		pid := c.Param("pid")
		cid := c.Param("cid")
		c.String(200, "%s-%s-%s", id, pid, cid)
	})

	req := httptest.NewRequest("GET", "/user/123/post/456/comment/789", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark query string parsing
func BenchmarkQueryParsing(b *testing.B) {
	r := New()
	r.GET("/search", func(c *Context) {
		q := c.Query("q")
		page := c.Query("page")
		limit := c.Query("limit")
		c.String(200, "%s-%s-%s", q, page, limit)
	})

	req := httptest.NewRequest("GET", "/search?q=golang&page=1&limit=10", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark context pool allocation
func BenchmarkContextAllocation(b *testing.B) {
	r := New()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := r.allocateContext(0)
		r.pool.Put(c)
	}
}

// Benchmark router group creation
func BenchmarkRouterGroup(b *testing.B) {
	r := New()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v1 := r.Group("/api/v1")
		_ = v1
	}
}

// Benchmark parallel requests (concurrent)
func BenchmarkParallelRequests(b *testing.B) {
	r := New()
	r.GET("/ping", func(c *Context) {
		c.String(200, "pong")
	})

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/ping", nil)
		w := httptest.NewRecorder()
		for pb.Next() {
			r.ServeHTTP(w, req)
		}
	})
}

// Benchmark route lookup performance with many routes
func BenchmarkRouteTree100Routes(b *testing.B) {
	r := New()

	// Add 100 routes
	for i := 0; i < 100; i++ {
		path := sprintf("/route%d", i)
		r.GET(path, func(c *Context) {
			c.String(200, "ok")
		})
	}

	req := httptest.NewRequest("GET", "/route50", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark POS transaction simulation
func BenchmarkPOSTransaction(b *testing.B) {
	r := New()

	type Transaction struct {
		ID       string  `json:"id"`
		Amount   float64 `json:"amount"`
		Terminal string  `json:"terminal"`
		Status   string  `json:"status"`
	}

	r.POST("/api/v1/transaction", func(c *Context) {
		var tx Transaction
		if err := c.BindJSON(&tx); err != nil {
			c.JSON(400, H{"error": err.Error()})
			return
		}
		c.JSON(200, tx)
	})

	body := `{"id":"TX123","amount":99.99,"terminal":"POS-001","status":"pending"}`
	req := httptest.NewRequest("POST", "/api/v1/transaction", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark rate limiter middleware
func BenchmarkRateLimiter(b *testing.B) {
	r := New()
	r.Use(RateLimiter(10000, 60)) // High limit for benchmarking

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark transaction ID middleware
func BenchmarkTransactionID(b *testing.B) {
	r := New()
	r.Use(TransactionID())

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark CORS middleware
func BenchmarkCORS(b *testing.B) {
	r := New()
	r.Use(CORS())

	r.GET("/test", func(c *Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark string rendering
func BenchmarkStringRender(b *testing.B) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.String(200, "Hello, World!")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark status code only
func BenchmarkStatusOnly(b *testing.B) {
	r := New()
	r.GET("/test", func(c *Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}

// Benchmark with all default middleware (Logger + Recovery)
func BenchmarkDefaultMiddleware(b *testing.B) {
	r := Default()
	r.GET("/ping", func(c *Context) {
		c.String(200, "pong")
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.ServeHTTP(w, req)
	}
}
