# goTap Middleware Documentation

## CORS Middleware

Cross-Origin Resource Sharing (CORS) middleware for handling cross-origin requests.

### Basic Usage

```go
router := goTap.New()

// Default CORS - allows all origins
router.Use(goTap.CORS())
```

### Advanced Configuration

```go
router.Use(goTap.CORSWithConfig(goTap.CORSConfig{
    AllowOrigins: []string{
        "https://pos.retailer.com",
        "https://terminal.retailer.com",
    },
    AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders: []string{"Authorization", "Content-Type", "X-Transaction-ID"},
    ExposeHeaders: []string{"X-Receipt-Number"},
    AllowCredentials: true,
    MaxAge: 3600, // 1 hour
}))
```

### Wildcard Subdomain Matching

```go
router.Use(goTap.CORSWithConfig(goTap.CORSConfig{
    AllowOrigins: []string{"https://*.example.com"},
    AllowWildcard: true,
}))
```

### Custom Origin Validation

```go
router.Use(goTap.CORSWithConfig(goTap.CORSConfig{
    AllowOriginFunc: func(origin string) bool {
        // Custom logic - e.g., check database
        return isValidOrigin(origin)
    },
}))
```

### Configuration Options

- **AllowOrigins**: List of allowed origins (use `"*"` for all)
- **AllowMethods**: HTTP methods allowed (default: GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS)
- **AllowHeaders**: Request headers allowed
- **ExposeHeaders**: Response headers to expose
- **AllowCredentials**: Enable credentials (cookies, auth headers)
- **MaxAge**: Preflight cache duration in seconds
- **AllowWildcard**: Enable wildcard subdomain matching
- **AllowOriginFunc**: Custom validation function

---

## Gzip Middleware

Response compression middleware using Gzip algorithm.

### Basic Usage

```go
router := goTap.New()

// Default Gzip - compresses responses >1KB
router.Use(goTap.Gzip())
```

### Advanced Configuration

```go
router.Use(goTap.GzipWithConfig(goTap.GzipConfig{
    Level: gzip.BestSpeed,      // Compression level (0-9)
    MinLength: 512,              // Minimum size to compress (bytes)
    ExcludedExtensions: []string{
        ".jpg", ".png", ".pdf",  // Don't compress images/PDFs
    },
    ExcludedPaths: []string{
        "/api/download",         // Don't compress downloads
    },
}))
```

### POS System Example

```go
// Fast compression for real-time POS responses
router.Use(goTap.GzipWithConfig(goTap.GzipConfig{
    Level: gzip.BestSpeed,       // Prioritize speed
    MinLength: 512,              // Compress API responses
    ExcludedExtensions: []string{
        ".jpg", ".png", ".pdf",  // Receipt images
    },
    ExcludedPaths: []string{
        "/api/receipt/download", // Binary receipts
    },
}))
```

### Configuration Options

- **Level**: Compression level
  - `gzip.BestSpeed` (1): Fastest, lower compression
  - `gzip.DefaultCompression` (-1): Balanced (default)
  - `gzip.BestCompression` (9): Slowest, best compression
- **MinLength**: Minimum response size to compress (default: 1024 bytes)
- **ExcludedExtensions**: File extensions to skip (images, archives, etc.)
- **ExcludedPaths**: URL paths to skip compression

### Default Excluded Extensions

The default configuration automatically excludes:
- **Images**: .png, .jpg, .jpeg, .gif, .ico, .bmp, .webp
- **Archives**: .zip, .rar, .7z, .tar, .gz, .bz2
- **Videos**: .mp4, .avi, .mov, .mkv, .webm
- **Audio**: .mp3, .wav, .ogg, .flac
- **Documents**: .pdf

### Performance Notes

- Uses memory pooling for efficiency
- Only compresses when client supports gzip (Accept-Encoding header)
- Automatically handles Content-Encoding and Vary headers
- Small responses below MinLength are not compressed
- Compression ratio typically 2x-10x for text/JSON

---

## Combining Middleware

```go
router := goTap.New()

// Add middleware in order
router.Use(goTap.Logger())
router.Use(goTap.Recovery())
router.Use(goTap.CORS())           // Handle CORS first
router.Use(goTap.Gzip())           // Then compress responses
router.Use(goTap.TransactionID())  // Track requests

// Your routes
router.GET("/api/data", handleData)
```

---

## Testing

Both middleware have comprehensive test suites:

```bash
# Test CORS
go test -v -run TestCORS

# Test Gzip
go test -v -run TestGzip

# All tests
go test -v
```

---

## Examples

See the examples directory for complete working examples:
- `examples/security/` - CORS with JWT authentication
- `examples/json-rendering/` - Gzip with JSON responses

---

## Performance

Both middleware are optimized for production use:
- **Memory pooling**: Reuses buffers and writers
- **Lazy initialization**: Only allocates when needed
- **Zero-copy**: Where possible
- **Benchmarked**: Performance tests included

Run benchmarks:
```bash
go test -bench=BenchmarkCORS -benchmem
go test -bench=BenchmarkGzip -benchmem
```

---

## Security Considerations

### CORS
- Use whitelist instead of `"*"` in production
- Enable `AllowCredentials` only when needed
- Set specific `AllowHeaders` instead of allowing all
- Use `MaxAge` to reduce preflight requests

### Gzip
- Don't compress already-compressed formats (images, PDFs)
- Set appropriate `MinLength` to avoid overhead on small responses
- Use `BestSpeed` for real-time applications
- Monitor CPU usage in high-traffic scenarios

---

## License

MIT License - See LICENSE file for details
