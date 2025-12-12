# Graceful Shutdown Example

This example demonstrates how to implement graceful shutdown in goTap, ensuring all active requests complete before the server stops.

## Why Graceful Shutdown?

In production systems, especially **POS terminals**, you need to ensure:
- **Payment transactions complete** before shutdown
- **Database connections close properly**
- **Active requests finish** without errors
- **No data loss** during restarts

## Features

- ✅ **Signal handling** (SIGINT, SIGTERM)
- ✅ **Configurable timeout** for shutdown
- ✅ **Active connections** complete before exit
- ✅ **Proper resource cleanup**

## How It Works

1. **Server starts** and listens for requests
2. **Signal received** (Ctrl+C or kill command)
3. **Server stops accepting** new connections
4. **Existing requests complete** (up to timeout)
5. **Server shuts down** gracefully

## Running the Example

```bash
cd examples/graceful-shutdown
go run main.go
```

Server starts on `http://localhost:5066`

## Testing Graceful Shutdown

### Test 1: Quick Request (Completes During Shutdown)

```bash
# Terminal 1: Start server
go run main.go

# Terminal 2: Send quick request
curl http://localhost:5066/ping

# Terminal 1: Press Ctrl+C
# Server will shutdown immediately (no active requests)
```

### Test 2: Long Request (Waits for Completion)

```bash
# Terminal 1: Start server
go run main.go

# Terminal 2: Start a long-running request (simulates payment processing)
curl "http://localhost:5066/process?duration=15"

# Terminal 1: Press Ctrl+C immediately after starting the request
# Server will wait up to 10 seconds for the request to complete
# You should see: "Shutdown Server ..." but request still completes
```

**Output:**
```
Processing request for 15s...
Shutdown Server ...
(waits 10 seconds - timeout)
Server forced to shutdown: context deadline exceeded
Server exiting
```

### Test 3: Multiple Concurrent Requests

```bash
# Terminal 1: Start server
go run main.go

# Terminal 2: Start multiple requests
curl "http://localhost:5066/process?duration=5" &
curl "http://localhost:5066/process?duration=5" &
curl "http://localhost:5066/process?duration=5" &

# Terminal 1: Press Ctrl+C
# All 3 requests should complete (within 10s timeout)
```

## Code Explanation

### 1. Standard Run (Blocking)

```go
// Simple way - blocks until error
router.Run(":5066")
```

### 2. Graceful Shutdown (Recommended for Production)

```go
// Get server instance
srv := router.RunServer(":5066")

// Setup signal handling
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

// Shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

### 3. Convenience Method

```go
// Simplified shutdown with default 5s timeout
srv := router.RunServer(":5066")
// ... wait for signal ...
goTap.ShutdownWithTimeout(srv, 10*time.Second)
```

## Advanced Usage: POS Terminal

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
    
    "github.com/jaswant99k/gotap"
)

func main() {
    router := goTap.Default()
    
    // Payment processing endpoint
    router.POST("/payment", func(c *goTap.Context) {
        // Simulate payment processing
        time.Sleep(3 * time.Second)
        
        c.JSON(200, goTap.H{
            "status": "success",
            "transaction_id": "TXN-001",
        })
    })
    
    // Database cleanup on shutdown
    db := initDatabase() // Your database connection
    
    srv := router.RunServer(":5066")
    
    // Wait for interrupt
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down POS terminal...")
    
    // Shutdown server (wait for active payments to complete)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal("Server shutdown failed:", err)
    }
    
    // Clean up database connections
    db.Close()
    
    log.Println("POS terminal stopped successfully")
}
```

## Configuration Options

### Timeout Values

| Timeout | Use Case |
|---------|----------|
| **5s** | Simple APIs, health checks |
| **10s** | Standard web applications |
| **30s** | Payment processing, transactions |
| **60s** | Long-running reports, exports |

### Production Example

```go
// Production server configuration
srv := &http.Server{
    Addr:           ":5066",
    Handler:        router,
    ReadTimeout:    10 * time.Second,
    WriteTimeout:   10 * time.Second,
    MaxHeaderBytes: 1 << 20,
}

// Start server in goroutine
go func() {
    if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("listen: %s\n", err)
    }
}()

// Graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
srv.Shutdown(ctx)
```

## Signals Supported

| Signal | Command | Behavior |
|--------|---------|----------|
| **SIGINT** | `Ctrl+C` | Graceful shutdown |
| **SIGTERM** | `kill <pid>` | Graceful shutdown |
| **SIGKILL** | `kill -9 <pid>` | Force kill (no cleanup) |

## Testing Shutdown Behavior

### Test Script

```bash
#!/bin/bash

# Start server in background
go run main.go &
SERVER_PID=$!

echo "Server started with PID: $SERVER_PID"
sleep 2

# Start a long request
curl "http://localhost:5066/process?duration=8" &
REQUEST_PID=$!

echo "Long request started"
sleep 1

# Send shutdown signal
kill -SIGTERM $SERVER_PID

# Wait for request to complete
wait $REQUEST_PID

echo "Request completed after shutdown signal"
```

## Comparison with Gin

goTap graceful shutdown is **compatible** with Gin's pattern:

**Gin:**
```go
srv := &http.Server{
    Addr:    ":8080",
    Handler: router,
}
// ... shutdown code ...
```

**goTap:**
```go
srv := router.RunServer(":5066")
// ... shutdown code ...
```

Both support the same `srv.Shutdown(ctx)` method.

## Best Practices

1. ✅ **Always set a timeout** - prevents hanging forever
2. ✅ **Log shutdown events** - helps debugging
3. ✅ **Clean up resources** - close DB connections, files
4. ✅ **Test with long requests** - ensure they complete
5. ✅ **Monitor shutdown metrics** - track graceful vs forced shutdowns
6. ✅ **Use appropriate timeouts** - based on your longest operation

## Deployment

### Docker Container

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o server
CMD ["./server"]

# Docker will send SIGTERM on stop
# Server will gracefully shutdown
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: gotap-server
        # Kubernetes sends SIGTERM before pod deletion
        lifecycle:
          preStop:
            exec:
              command: ["/bin/sh", "-c", "sleep 15"]
        # terminationGracePeriodSeconds: 30 (default)
```

### systemd Service

```ini
[Unit]
Description=goTap POS Server
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/gotap-server
# systemd sends SIGTERM on stop
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
```

## Monitoring

```go
// Add metrics endpoint
router.GET("/metrics", func(c *goTap.Context) {
    c.JSON(200, goTap.H{
        "uptime": time.Since(startTime).String(),
        "requests_total": requestCounter,
        "active_connections": activeConnections,
    })
})
```

## Error Handling

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := srv.Shutdown(ctx); err != nil {
    switch err {
    case context.DeadlineExceeded:
        log.Println("Shutdown timeout exceeded, forcing shutdown")
    default:
        log.Printf("Shutdown error: %v", err)
    }
}
```
