# goTap Web Framework

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

goTap is a high-performance HTTP web framework for Go, inspired by Gin. Designed specifically for **highly available and low-latency POS (Point-of-Sale) systems**, REST APIs, web applications, and microservices.

## 🚀 Features

### Core Framework
- **High Performance** - Fast routing with radix tree, context pooling
- **Middleware Chain** - Extensible middleware system with Logger, Recovery
- **Route Grouping** - Organize routes with common prefixes and middleware
- **Parameter Binding** - URL params, query strings, JSON, XML, Form, Headers
- **Data Validation** - 11 built-in validators (required, email, min/max, etc.)
- **Error Management** - Panic recovery and centralized error handling

### Security Middleware
- **JWT Authentication** - HMAC-SHA256 token generation with role-based access
- **BasicAuth** - HTTP Basic Authentication with timing attack prevention
- **CORS** - Cross-Origin Resource Sharing with wildcard support
- **Rate Limiting** - Token bucket algorithm with per-user/path controls
- **IP Filtering** - Whitelist/blacklist with CIDR support

### Rendering & Responses
- **JSON Variants** - Standard, Indented, Secure, JSONP, ASCII, Pure JSON
- **XML/YAML** - Full XML and YAML rendering support
- **HTML Templates** - Go template engine with glob/file loading
- **Content Negotiation** - Accept header-based format selection
- **Gzip Compression** - Automatic response compression with smart thresholds
- **Server-Sent Events** - Streaming real-time updates

### API Documentation
- **Swagger/OpenAPI** - Interactive API documentation with Swagger UI
- **Auto-Generation** - Generate docs from code annotations
- **Interactive Testing** - Test APIs directly in browser
- **Authentication Support** - JWT, BasicAuth in Swagger UI
- **OpenAPI 3.0** - Industry-standard API specification

### POS-Optimized Features
- **Shadow Database** - Dual-DB with automatic failover and health monitoring
- **GORM Integration** - Type-safe ORM with MySQL, PostgreSQL, SQLite support
- **WebSocket Support** - Real-time bidirectional communication
- **Transaction Tracking** - Audit trail with UUID/POS ID generation
- **High Availability** - Built for 99.9%+ uptime retail systems

## 📦 Installation

```bash
go get -u github.com/jaswant99k/gotap
```

**Requirements:** Go 1.21+

## 🎯 Quick Start

```go
package main

import "github.com/jaswant99k/gotap"

func main() {
    // Create router with Logger and Recovery middleware
    r := goTap.Default()

    // Define routes
    r.GET("/ping", func(c *goTap.Context) {
        c.JSON(200, goTap.H{
            "message": "pong",
        })
    })

    // URL parameters
    r.GET("/hello/:name", func(c *goTap.Context) {
        name := c.Param("name")
        c.JSON(200, goTap.H{
            "message": "Hello " + name,
        })
    })

    // Start server on :8080
    r.Run()
}
```

## 📖 Examples

### Route Parameters

```go
r.GET("/user/:id", func(c *goTap.Context) {
    id := c.Param("id")
    c.String(200, "User ID: %s", id)
})
```

### Query Strings

```go
r.GET("/search", func(c *goTap.Context) {
    query := c.Query("q")
    c.JSON(200, goTap.H{"query": query})
})
```

### Route Groups

```go
api := r.Group("/api/v1")
{
    api.GET("/users", listUsers)
    api.POST("/users", createUser)
    api.GET("/users/:id", getUser)
}
```

### POST Form Data

```go
r.POST("/form", func(c *goTap.Context) {
    name := c.PostForm("name")
    email := c.PostForm("email")
    c.JSON(200, goTap.H{
        "name": name,
        "email": email,
    })
})
```

### Custom Middleware

```go
func AuthMiddleware() goTap.HandlerFunc {
    return func(c *goTap.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.AbortWithStatusJSON(401, goTap.H{"error": "unauthorized"})
            return
        }
        c.Next()
    }
}

// Use middleware
r.Use(AuthMiddleware())
```

### CORS Middleware

```go
// Allow all origins (development)
r.Use(goTap.CORS())

// Production: whitelist specific origins
r.Use(goTap.CORSWithConfig(goTap.CORSConfig{
    AllowOrigins: []string{"https://pos.retailer.com"},
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowCredentials: true,
    MaxAge: 3600,
}))
```

### Gzip Compression

```go
// Default: compress responses >1KB
r.Use(goTap.Gzip())

// Custom: fast compression for real-time POS
r.Use(goTap.GzipWithConfig(goTap.GzipConfig{
    Level: gzip.BestSpeed,
    MinLength: 512,
    ExcludedExtensions: []string{".jpg", ".png", ".pdf"},
}))
```

### JWT Authentication

```go
secret := "your-secret-key-minimum-32-characters"

// Public routes
r.POST("/login", loginHandler)
r.POST("/register", registerHandler)

// Protected routes (require authentication)
auth := r.Group("/api")
auth.Use(goTap.JWTAuth(secret))
{
    auth.GET("/profile", getProfile)
    auth.PUT("/profile", updateProfile)
}

// Admin-only routes
admin := r.Group("/admin")
admin.Use(goTap.JWTAuth(secret))
admin.Use(goTap.RequireRole("admin"))
{
    admin.GET("/users", listUsers)
    admin.DELETE("/users/:id", deleteUser)
}

// Multiple roles allowed
manage := r.Group("/manage")
manage.Use(goTap.JWTAuth(secret))
manage.Use(goTap.RequireAnyRole("admin", "manager"))
{
    manage.GET("/orders", listOrders)
}
```

**See full authentication guide:** [`examples/auth/README.md`](examples/auth/README.md)

### GORM Database Integration

```go
import (
    "github.com/jaswant99k/gotap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

// Connect to database
db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

// Inject GORM into context
r.Use(goTap.GormInject(db))

// Use in handlers
r.GET("/users/:id", func(c *goTap.Context) {
    db := goTap.MustGetGorm(c)
    var user User
    db.First(&user, c.Param("id"))
    c.JSON(200, user)
})
```

**See full GORM guide:** [`examples/gorm/README.md`](examples/gorm/README.md)

## 🏗️ Project Structure

```
goTap/
├── gotap.go              # Main engine
├── context.go            # Request context
├── router.go             # Routing logic
├── routergroup.go        # Route grouping
├── tree.go               # Radix tree
├── middleware/           # Built-in middleware
├── examples/             # Usage examples
└── tests/                # Unit tests
```

## 🎯 Roadmap

### Phase 1: Core Foundation ✅
- [x] Basic engine and context
- [x] Routing with radix tree
- [x] HTTP method handlers
- [x] Logger and Recovery middleware

### Phase 2: Advanced Routing ✅
- [x] Static file serving
- [x] File uploads/downloads
- [x] Route groups

### Phase 3: Security Middleware ✅
- [x] JWT authentication middleware
- [x] BasicAuth middleware
- [x] CORS middleware
- [x] Gzip compression
- [x] Transaction ID tracking
- [x] IP whitelist middleware
- [x] Rate limiting

### Phase 4-6: Data & Rendering ✅
- [x] Data binding and validation
- [x] Multiple render formats (JSON variants, XML, YAML, HTML)
- [x] Content negotiation
- [x] Server-Sent Events
- [x] Shadow Database integration
- [x] WebSocket support

### Phase 7: Performance (In Progress)
- [ ] Comprehensive benchmarking suite
- [ ] Memory profiling and optimization
- [ ] Zero-allocation improvements
- [ ] Connection pooling

### Phase 8: Testing (85% Complete)
- [x] 113 comprehensive tests
- [x] Middleware tests
- [x] Binding and validation tests
- [ ] 90%+ code coverage target

### Phase 9: Advanced Features (Planned)
- [ ] Custom validators plugin system
- [ ] Hot reload support
- [ ] Metrics and tracing
- [ ] HTTP/2 Server Push

## 📊 Performance

goTap is designed for high performance:

- **Routing:** < 30,000 ns/op for complex routes
- **Throughput:** 100,000+ req/sec target
- **Latency:** Sub-millisecond response time
- **Memory:** Zero-allocation routing

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [Gin](https://github.com/gin-gonic/gin)
- Built for the POS industry with ❤️

## 📧 Contact

- Project Link: [https://github.com/jaswant99k/gotap](https://github.com/jaswant99k/gotap)
- Issues: [https://github.com/jaswant99k/gotap/issues](https://github.com/jaswant99k/gotap/issues)

---

**Made with ❤️ for high-performance POS systems**
