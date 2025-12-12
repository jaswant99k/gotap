package goTap

import (
	"bufio"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerConfig holds Swagger UI configuration
type SwaggerConfig struct {
	// URL to swagger.json or swagger.yaml
	URL string
	// DocExpansion list, full, none
	DocExpansion string
	// DeepLinking enables deep linking for tags and operations
	DeepLinking bool
	// PersistAuthorization persists authorization data
	PersistAuthorization bool
	// DefaultModelsExpandDepth sets the default expansion depth for models
	DefaultModelsExpandDepth int
}

// DefaultSwaggerConfig returns default Swagger configuration
func DefaultSwaggerConfig() *SwaggerConfig {
	return &SwaggerConfig{
		URL:                      "doc.json",
		DocExpansion:             "list",
		DeepLinking:              true,
		PersistAuthorization:     true,
		DefaultModelsExpandDepth: 1,
	}
}

// SwaggerHandler returns a handler that serves Swagger UI
// It wraps gin-swagger to work with goTap's Context
func SwaggerHandler(config *SwaggerConfig) HandlerFunc {
	if config == nil {
		config = DefaultSwaggerConfig()
	}

	// Create gin-swagger handler with configuration
	ginHandler := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.URL(config.URL),
		ginSwagger.DocExpansion(config.DocExpansion),
		ginSwagger.DeepLinking(config.DeepLinking),
		ginSwagger.PersistAuthorization(config.PersistAuthorization),
		ginSwagger.DefaultModelsExpandDepth(config.DefaultModelsExpandDepth),
	)

	return func(c *Context) {
		// Call the gin-swagger handler directly with our request/response
		ginHandler(&gin.Context{
			Request: c.Request,
			Writer:  &ginResponseWriter{c.Writer},
		})
	}
}

// ginResponseWriter wraps goTap's ResponseWriter to work with gin
type ginResponseWriter struct {
	http.ResponseWriter
}

func (w *ginResponseWriter) Status() int { return 200 }
func (w *ginResponseWriter) Size() int   { return -1 }
func (w *ginResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}
func (w *ginResponseWriter) Written() bool       { return true }
func (w *ginResponseWriter) WriteHeaderNow()     {}
func (w *ginResponseWriter) Pusher() http.Pusher { return nil }
func (w *ginResponseWriter) CloseNotify() <-chan bool {
	return nil
}
func (w *ginResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
func (w *ginResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// SwaggerJSON serves the swagger.json file
func SwaggerJSON(jsonData []byte) HandlerFunc {
	return func(c *Context) {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Status(http.StatusOK)
		c.Writer.Write(jsonData)
	}
}

// SwaggerYAML serves the swagger.yaml file
func SwaggerYAML(yamlData []byte) HandlerFunc {
	return func(c *Context) {
		c.Header("Content-Type", "text/yaml; charset=utf-8")
		c.Status(http.StatusOK)
		c.Writer.Write(yamlData)
	}
}

// SetupSwagger registers Swagger UI routes
// Usage:
//
//	import _ "yourmodule/docs" // swagger docs
//	goTap.SetupSwagger(r, "/swagger")
func SetupSwagger(r *Engine, basePath string) {
	if basePath == "" {
		basePath = "/swagger"
	}

	// Swagger UI
	r.GET(basePath+"/*any", SwaggerHandler(nil))
}

// SetupSwaggerWithAuth registers Swagger UI routes with authentication
// Usage:
//
//	goTap.SetupSwaggerWithAuth(r, "/swagger", goTap.JWTAuth(jwtSecret))
func SetupSwaggerWithAuth(r *Engine, basePath string, authMiddleware ...HandlerFunc) {
	if basePath == "" {
		basePath = "/swagger"
	}

	group := r.Group(basePath)
	group.Use(authMiddleware...)
	group.GET("/*any", SwaggerHandler(nil))
}
