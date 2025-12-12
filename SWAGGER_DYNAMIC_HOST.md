# Dynamic Swagger Host Configuration

By default, Swagger annotations use `@host localhost:8080`. However, your application might run on different ports. goTap provides utilities to dynamically update the Swagger host at runtime.

## Problem

When you have this in your code:
```go
// @host localhost:8080
```

But run your server on port 3000 or 8055, the Swagger UI will still show curl examples with `:8080`, which won't work.

## Solution

Use `goTap.UpdateSwaggerHost()` to dynamically set the correct host:

```go
package main

import (
    "github.com/jaswant99k/gotap"
    "yourmodule/docs"  // Import docs package directly, not with _
)

// @host localhost:8080  // This will be overridden at runtime
func main() {
    r := goTap.Default()

    // Determine port (from env, config, or default)
    port := os.Getenv("PORT")
    if port == "" {
        port = "8055"
    }
    addr := ":" + port

    // Update Swagger host dynamically to match running port
    docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)

    // Setup Swagger
    goTap.SetupSwagger(r, "/swagger")

    // Your routes...

    // Start server
    r.Run(addr)
}
```

## How It Works

`UpdateSwaggerHost()` handles different address formats:

```go
goTap.UpdateSwaggerHost(":8080")          // Returns "localhost:8080"
goTap.UpdateSwaggerHost(":3000")          // Returns "localhost:3000"
goTap.UpdateSwaggerHost("0.0.0.0:8055")   // Returns "localhost:8055"
goTap.UpdateSwaggerHost("localhost:9000") // Returns "localhost:9000"
```

## Complete Example

```go
package main

import (
    "log"
    "os"

    "github.com/jaswant99k/gotap"
    "vervepos/docs"  // Note: Not using _ import
)

// @title           Verve POS API
// @version         1.0
// @host            localhost:8080  // Will be updated dynamically
// @BasePath        /api
func main() {
    r := goTap.Default()

    // Get port from environment or use default
    port := os.Getenv("PORT")
    if port == "" {
        port = "8055"
    }
    addr := ":" + port

    // IMPORTANT: Update Swagger host before setting up Swagger UI
    docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)

    // Setup Swagger
    goTap.SetupSwagger(r, "/swagger")

    // Your API routes
    r.GET("/health", func(c *goTap.Context) {
        c.JSON(200, goTap.H{"status": "ok"})
    })

    log.Printf("Server starting on http://localhost:%s", port)
    log.Printf("Swagger UI: http://localhost:%s/swagger/index.html", port)
    
    r.Run(addr)
}
```

## Import Note

⚠️ **Important**: Import the docs package **without the underscore** to access `SwaggerInfo`:

```go
// ✅ Correct - allows access to SwaggerInfo
import "yourmodule/docs"

// ❌ Wrong - blank import doesn't expose SwaggerInfo
import _ "yourmodule/docs"
```

## Using With Environment Variables

```go
port := os.Getenv("PORT")
if port == "" {
    port = "8080"
}

docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(":" + port)
r.Run(":" + port)
```

```bash
# Run on different ports
PORT=3000 go run cmd/server/main.go
PORT=8055 go run cmd/server/main.go
PORT=9000 go run cmd/server/main.go
```

Swagger UI will automatically show the correct port in all curl examples!

## Advanced: Custom Host

If you need complete control:

```go
// Override with custom host
docs.SwaggerInfo.Host = "api.mycompany.com"
docs.SwaggerInfo.BasePath = "/v1"
docs.SwaggerInfo.Schemes = []string{"https"}

goTap.SetupSwagger(r, "/swagger")
```

## Benefits

✅ Works with any port configuration  
✅ No need to regenerate Swagger docs when changing ports  
✅ One codebase works for development and production  
✅ Curl examples in Swagger UI always show correct URLs  
✅ Environment variable friendly  

## Common Patterns

### Pattern 1: Dev/Prod Ports
```go
port := "8080"
if os.Getenv("ENVIRONMENT") == "production" {
    port = "443"
}
addr := ":" + port
docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)
```

### Pattern 2: Random Port (Testing)
```go
addr := ":0"  // Random available port
docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)
```

### Pattern 3: Config File
```go
config := loadConfig()
addr := fmt.Sprintf(":%d", config.Port)
docs.SwaggerInfo.Host = goTap.UpdateSwaggerHost(addr)
```
