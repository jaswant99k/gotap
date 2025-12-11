package goTap

import (
	"bufio"
	"compress/gzip"
	"net"
	"net/http"
	"strings"
	"sync"
)

// GzipConfig defines configuration for Gzip middleware
type GzipConfig struct {
	// Level of compression (0-9, where 9 is best compression)
	// Default: gzip.DefaultCompression (-1)
	Level int

	// MinLength is minimum response size to compress (bytes)
	// Responses smaller than this won't be compressed
	// Default: 1024 (1KB)
	MinLength int

	// ExcludedExtensions is a list of file extensions that shouldn't be compressed
	// Example: []string{".png", ".jpg", ".jpeg", ".gif", ".zip"}
	ExcludedExtensions []string

	// ExcludedPaths is a list of paths that shouldn't be compressed
	// Example: []string{"/api/download", "/images"}
	ExcludedPaths []string

	// ExcludedPathsRegexs is a list of regex patterns for paths to exclude
	// More flexible than ExcludedPaths but slower
	ExcludedPathsRegexs []string
}

// DefaultGzipConfig returns a default Gzip configuration
func DefaultGzipConfig() GzipConfig {
	return GzipConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 1024, // 1KB
		ExcludedExtensions: []string{
			".png", ".jpg", ".jpeg", ".gif", ".ico", ".bmp", ".webp", // Images
			".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", // Archives
			".mp4", ".avi", ".mov", ".mkv", ".webm", // Videos
			".mp3", ".wav", ".ogg", ".flac", // Audio
			".pdf", // PDFs are already compressed
		},
		ExcludedPaths:       []string{},
		ExcludedPathsRegexs: []string{},
	}
}

// gzipWriter wraps the response writer and compresses the response
type gzipWriter struct {
	http.ResponseWriter
	writer       *gzip.Writer
	statusCode   int
	headerSent   bool
	minLength    int
	level        int
	bufferPool   *sync.Pool
	buffer       []byte
	bytesWritten int
}

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		return &gzipWriter{}
	},
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0, 32*1024) // 32KB initial buffer
	},
}

// Write implements http.ResponseWriter.Write
func (g *gzipWriter) Write(data []byte) (int, error) {
	if !g.headerSent {
		g.WriteHeader(http.StatusOK)
	}

	// Buffer small responses to check against MinLength
	if g.writer == nil && g.bytesWritten+len(data) < g.minLength {
		g.buffer = append(g.buffer, data...)
		g.bytesWritten += len(data)
		return len(data), nil
	}

	// If we need to start compressing and haven't created writer yet
	if g.writer == nil && g.bytesWritten+len(data) >= g.minLength {
		// Create gzip writer now
		var err error
		g.writer, err = gzip.NewWriterLevel(g.ResponseWriter, g.level)
		if err != nil {
			// Fall back to uncompressed if gzip creation fails
			if len(g.buffer) > 0 {
				g.ResponseWriter.Write(g.buffer)
				g.buffer = g.buffer[:0]
			}
			return g.ResponseWriter.Write(data)
		}

		// Set compression headers
		g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
		g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")
		g.ResponseWriter.Header().Del("Content-Length")

		// Write buffered data
		if len(g.buffer) > 0 {
			if _, err := g.writer.Write(g.buffer); err != nil {
				return 0, err
			}
			g.buffer = g.buffer[:0]
		}
	}

	// Write to gzip writer if it exists
	if g.writer != nil {
		g.bytesWritten += len(data)
		return g.writer.Write(data)
	}

	// Still buffering
	g.buffer = append(g.buffer, data...)
	g.bytesWritten += len(data)
	return len(data), nil
}

// WriteHeader implements http.ResponseWriter.WriteHeader
func (g *gzipWriter) WriteHeader(code int) {
	if g.headerSent {
		return
	}

	g.statusCode = code
	g.headerSent = true

	// Don't write headers yet if we haven't decided to compress
	// Headers will be written in Write() when we know if we're compressing
	if g.writer == nil {
		return
	}

	// If writer exists, we're compressing
	g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	g.ResponseWriter.Header().Set("Vary", "Accept-Encoding")
	g.ResponseWriter.Header().Del("Content-Length")
	g.ResponseWriter.WriteHeader(code)
}

// Flush flushes any buffered data
func (g *gzipWriter) Flush() {
	// Flush buffered data if below minLength
	if len(g.buffer) > 0 && g.writer == nil {
		g.ResponseWriter.Write(g.buffer)
		g.buffer = g.buffer[:0]
		return
	}

	if g.writer != nil {
		if len(g.buffer) > 0 {
			g.writer.Write(g.buffer)
			g.buffer = g.buffer[:0]
		}
		g.writer.Flush()
	}

	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Close closes the gzip writer
func (g *gzipWriter) Close() error {
	// If we have buffered data but didn't reach minLength, write uncompressed
	if len(g.buffer) > 0 && g.writer == nil {
		// Write status if not yet written
		if !g.headerSent {
			g.ResponseWriter.WriteHeader(g.statusCode)
		}
		g.ResponseWriter.Write(g.buffer)
		g.buffer = g.buffer[:0]
		return nil
	}

	if g.writer != nil {
		// Flush any remaining buffered data
		if len(g.buffer) > 0 {
			g.writer.Write(g.buffer)
			g.buffer = g.buffer[:0]
		}
		return g.writer.Close()
	}

	// No data written at all, just write headers
	if !g.headerSent && g.bytesWritten == 0 {
		g.ResponseWriter.WriteHeader(g.statusCode)
	}

	return nil
}

// Hijack implements http.Hijacker interface
func (g *gzipWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := g.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Status returns the HTTP status code
func (g *gzipWriter) Status() int {
	return g.statusCode
}

// Size returns the number of bytes written
func (g *gzipWriter) Size() int {
	return g.bytesWritten
}

// WriteString implements io.StringWriter
func (g *gzipWriter) WriteString(s string) (int, error) {
	return g.Write([]byte(s))
}

// Written returns true if the response has been written
func (g *gzipWriter) Written() bool {
	return g.headerSent
}

// WriteHeaderNow forces writing the headers
func (g *gzipWriter) WriteHeaderNow() {
	if !g.headerSent {
		g.WriteHeader(g.statusCode)
	}
}

// Gzip returns a middleware that compresses HTTP responses using Gzip compression
// It uses default configuration which compresses responses larger than 1KB
func Gzip() HandlerFunc {
	return GzipWithConfig(DefaultGzipConfig())
}

// GzipWithConfig returns a Gzip middleware with custom configuration
func GzipWithConfig(config GzipConfig) HandlerFunc {
	// Set defaults if not configured
	if config.Level < gzip.HuffmanOnly || config.Level > gzip.BestCompression {
		config.Level = gzip.DefaultCompression
	}
	if config.MinLength == 0 {
		config.MinLength = 1024 // 1KB default
	}

	return func(c *Context) {
		// Check if client accepts gzip
		if !strings.Contains(c.Request.Header.Get("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		// Check excluded paths
		path := c.Request.URL.Path
		for _, excluded := range config.ExcludedPaths {
			if strings.HasPrefix(path, excluded) {
				c.Next()
				return
			}
		}

		// Check excluded extensions
		for _, ext := range config.ExcludedExtensions {
			if strings.HasSuffix(path, ext) {
				c.Next()
				return
			}
		}

		// Get gzip writer from pool
		gw := gzipWriterPool.Get().(*gzipWriter)
		gw.ResponseWriter = c.Writer
		gw.statusCode = http.StatusOK
		gw.headerSent = false
		gw.minLength = config.MinLength
		gw.level = config.Level
		gw.bytesWritten = 0
		gw.buffer = bufferPool.Get().([]byte)[:0]
		gw.writer = nil // Don't create writer until we know we need it

		// Replace response writer
		c.Writer = gw

		// Process request
		c.Next()

		// Clean up
		gw.Close()

		// Return buffer to pool
		if gw.buffer != nil {
			bufferPool.Put(gw.buffer[:0])
		}

		// Reset and return gzip writer to pool
		gw.ResponseWriter = nil
		gw.writer = nil
		gw.buffer = nil
		gzipWriterPool.Put(gw)
	}
}
