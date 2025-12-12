# How to Fix Swagger Host in Your Project

## The Problem
Your project runs on port 8055, but Swagger shows `localhost:8080` in the API examples.

## Quick Fix

Update your `cmd/server/main.go`:

### Step 1: Change the import
```go
// OLD (with underscore):
import _ "vervepos/docs"

// NEW (without underscore):
import "vervepos/docs"
```

### Step 2: Add dynamic host configuration
```go
func main() {
    r := gotap.Default()

    // Get port from environment
    port := os.Getenv("PORT")
    if port == "" {
        port = "8055"  // Your default port
    }
    addr := ":" + port

    // 🔥 ADD THIS LINE - Updates Swagger host dynamically
    docs.SwaggerInfo.Host = gotap.UpdateSwaggerHost(addr)

    // Setup Swagger (after updating host)
    gotap.SetupSwagger(r, "/swagger")

    // ... rest of your code ...

    // Use addr variable consistently
    r.Run(addr)
}
```

### Complete Example

```go
package main

import (
    "log"
    "os"

    "github.com/jaswant99k/gotap"
    "vervepos/docs"  // ✅ No underscore!
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    
    // Your modules...
)

// @title Verve POS API
// @version 1.0
// @host localhost:8080  // Will be updated to 8055 at runtime
// @BasePath /api
func main() {
    r := gotap.Default()

    // Database setup
    dsn := os.Getenv("DB_DSN")
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Database connection failed:", err)
    }
    gotap.GormInject(db)

    // Port configuration
    port := os.Getenv("PORT")
    if port == "" {
        port = "8055"
    }
    addr := ":" + port

    // 🔥 Update Swagger host to match running port
    docs.SwaggerInfo.Host = gotap.UpdateSwaggerHost(addr)

    // Swagger UI
    gotap.SetupSwagger(r, "/swagger")

    // Health endpoint
    r.GET("/health", func(c *gotap.Context) {
        c.JSON(200, gotap.H{"status": "healthy"})
    })

    // Your API routes
    api := r.Group("/api")
    {
        // auth routes
        // product routes
        // admin routes
    }

    log.Printf("🚀 Server: http://localhost:%s", port)
    log.Printf("📚 Swagger: http://localhost:%s/swagger/index.html", port)
    log.Printf("💾 Database: Connected")
    
    r.Run(addr)
}
```

## Test It

1. Make the changes above
2. Run: `go run cmd/server/main.go`
3. Open: http://localhost:8055/swagger/index.html
4. Check any endpoint - the curl examples should now show `:8055` ✅

## Run on Different Ports

```bash
# Default (8055)
go run cmd/server/main.go

# Custom port
PORT=3000 go run cmd/server/main.go
PORT=9000 go run cmd/server/main.go
```

Swagger will automatically adjust to the correct port!

## Why This Works

- `docs.SwaggerInfo.Host` is the runtime host configuration
- `UpdateSwaggerHost()` converts `:8055` to `localhost:8055`
- Swagger UI reads this value when generating API examples
- No need to regenerate docs with `swag init`

## Troubleshooting

### Error: "docs.SwaggerInfo undefined"
**Cause**: You're using `import _ "vervepos/docs"`  
**Fix**: Remove the underscore: `import "vervepos/docs"`

### Still shows wrong port
**Cause**: Line order - must update before `SetupSwagger()`  
**Fix**: Make sure this comes first:
```go
docs.SwaggerInfo.Host = gotap.UpdateSwaggerHost(addr)
gotap.SetupSwagger(r, "/swagger")  // After!
```

### Swagger docs not found
**Cause**: Need to regenerate docs  
**Fix**: Run `swag init -g cmd/server/main.go --output docs`
