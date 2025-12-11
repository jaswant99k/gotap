// Copyright 2025 goTap Authors. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package goTap

import (
	"context"
	"html/template"
	"net"
	"net/http"
	"sync"
	"time"
)

const Version = "0.1.0"

// HandlerFunc defines the handler used by goTap middleware as return value.
type HandlerFunc func(*Context)

// HandlersChain defines a HandlerFunc slice.
type HandlersChain []HandlerFunc

// Last returns the last handler in the chain. i.e. the last handler is the main one.
func (c HandlersChain) Last() HandlerFunc {
	if length := len(c); length > 0 {
		return c[length-1]
	}
	return nil
}

// RouteInfo represents a request route's specification which contains method and path and its handler.
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	HandlerFunc HandlerFunc
}

// RoutesInfo defines a RouteInfo slice.
type RoutesInfo []RouteInfo

// Engine is the framework's instance, it contains the muxer, middleware and configuration settings.
// Create an instance of Engine, by using New() or Default()
type Engine struct {
	RouterGroup

	// Router configuration
	RedirectTrailingSlash  bool
	RedirectFixedPath      bool
	HandleMethodNotAllowed bool
	ForwardedByClientIP    bool
	UseRawPath             bool
	UnescapePathValues     bool
	RemoveExtraSlash       bool

	// Template rendering
	delims             Delims
	FuncMap            template.FuncMap
	allNoRoute         HandlersChain
	allNoMethod        HandlersChain
	noRoute            HandlersChain
	noMethod           HandlersChain
	pool               sync.Pool
	trees              methodTrees
	maxParams          uint16
	maxSections        uint16
	trustedProxies     []string
	trustedCIDRs       []*net.IPNet
	MaxMultipartMemory int64

	// JSON rendering
	secureJSONPrefix string
}

// Delims represents template delimiters
type Delims struct {
	Left  string
	Right string
}

var _ IRouter = (*Engine)(nil)

const defaultMultipartMemory = 32 << 20 // 32 MB

// New returns a new blank Engine instance without any middleware attached.
// By default, the configuration is:
// - RedirectTrailingSlash:  true
// - RedirectFixedPath:      false
// - HandleMethodNotAllowed: false
// - ForwardedByClientIP:    true
// - UseRawPath:             false
// - UnescapePathValues:     true
func New() *Engine {
	debugPrint("goTap v%s - High-performance web framework\n", Version)

	engine := &Engine{
		RouterGroup: RouterGroup{
			Handlers: nil,
			basePath: "/",
			root:     true,
		},
		FuncMap:                template.FuncMap{},
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      false,
		HandleMethodNotAllowed: false,
		ForwardedByClientIP:    true,
		UseRawPath:             false,
		UnescapePathValues:     true,
		MaxMultipartMemory:     defaultMultipartMemory,
		trees:                  make(methodTrees, 0, 9),
		delims:                 Delims{Left: "{{", Right: "}}"},
		trustedProxies:         []string{"0.0.0.0/0", "::/0"},
	}
	engine.RouterGroup.engine = engine
	engine.pool.New = func() any {
		return engine.allocateContext(engine.maxParams)
	}
	return engine
}

// Default returns an Engine instance with the Logger and Recovery middleware already attached.
func Default() *Engine {
	debugPrintWARNINGDefault()
	engine := New()
	engine.Use(Logger(), Recovery())
	return engine
}

func (engine *Engine) allocateContext(maxParams uint16) *Context {
	v := make(Params, 0, maxParams)
	skippedNodes := make([]skippedNode, 0, engine.maxSections)
	return &Context{engine: engine, params: &v, skippedNodes: &skippedNodes}
}

// Use attaches a global middleware to the router. i.e. the middleware attached through Use() will be
// included in the handlers chain for every single request. Even 404, 405, static files...
// For example, this is the right place for a logger or error management middleware.
func (engine *Engine) Use(middleware ...HandlerFunc) IRoutes {
	engine.RouterGroup.Use(middleware...)
	engine.rebuild404Handlers()
	engine.rebuild405Handlers()
	return engine
}

func (engine *Engine) rebuild404Handlers() {
	engine.allNoRoute = engine.combineHandlers(engine.noRoute)
}

func (engine *Engine) rebuild405Handlers() {
	engine.allNoMethod = engine.combineHandlers(engine.noMethod)
}

// SecureJSONPrefix sets the prefix for SecureJSON rendering
// Default prefix is "while(1);"
func (engine *Engine) SecureJSONPrefix(prefix string) {
	engine.secureJSONPrefix = prefix
}

// NoRoute adds handlers for NoRoute. It returns a 404 code by default.
func (engine *Engine) NoRoute(handlers ...HandlerFunc) {
	engine.noRoute = handlers
	engine.rebuild404Handlers()
}

// NoMethod sets the handlers called when Engine.HandleMethodNotAllowed = true.
func (engine *Engine) NoMethod(handlers ...HandlerFunc) {
	engine.noMethod = handlers
	engine.rebuild405Handlers()
}

// ServeHTTP conforms to the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	c := engine.pool.Get().(*Context)
	c.writermem.reset(w)
	c.Request = req
	c.reset()

	engine.handleHTTPRequest(c)

	engine.pool.Put(c)
}

func (engine *Engine) handleHTTPRequest(c *Context) {
	httpMethod := c.Request.Method
	rPath := c.Request.URL.Path

	// Find root of the tree for the given HTTP method
	t := engine.trees
	for i, tl := 0, len(t); i < tl; i++ {
		if t[i].method != httpMethod {
			continue
		}
		root := t[i].root
		// Find route in tree
		value := root.getValue(rPath, c.params, c.skippedNodes, engine.UnescapePathValues)
		if value.params != nil {
			c.Params = *value.params
		}
		if value.handlers != nil {
			c.handlers = value.handlers
			c.fullPath = value.fullPath
			c.Next()
			c.writermem.WriteHeaderNow()
			return
		}
		break
	}

	// Handle 404
	c.handlers = engine.allNoRoute
	serveError(c, http.StatusNotFound, []byte("404 page not found"))
}

func serveError(c *Context, code int, defaultMessage []byte) {
	c.writermem.status = code
	c.Next()
	if c.writermem.Written() {
		return
	}
	if c.writermem.Status() == code {
		c.writermem.Header()["Content-Type"] = []string{"text/plain"}
		_, err := c.Writer.Write(defaultMessage)
		if err != nil {
			debugPrint("cannot write message to writer during serve error: %v", err)
		}
		return
	}
	c.writermem.WriteHeaderNow()
}

// Run attaches the router to a http.Server and starts listening and serving HTTP requests.
// It is a shortcut for http.ListenAndServe(addr, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) Run(addr ...string) (err error) {
	defer func() { debugPrintError(err) }()

	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)
	err = http.ListenAndServe(address, engine)
	return
}

// RunServer attaches the router to a http.Server and starts listening and serving HTTP requests.
// This method returns the http.Server instance for advanced configuration and graceful shutdown.
// Example:
//
//	srv := router.RunServer(":5066")
//	// Wait for interrupt signal
//	quit := make(chan os.Signal, 1)
//	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
//	<-quit
//	// Shutdown gracefully
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	srv.Shutdown(ctx)
func (engine *Engine) RunServer(addr ...string) *http.Server {
	address := resolveAddress(addr)
	debugPrint("Listening and serving HTTP on %s\n", address)

	srv := &http.Server{
		Addr:    address,
		Handler: engine,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			debugPrintError(err)
		}
	}()

	return srv
}

// Shutdown gracefully shuts down the server without interrupting active connections.
// This is a convenience wrapper around http.Server.Shutdown.
// It waits for all active requests to complete or until the context is canceled.
func Shutdown(srv *http.Server, ctx context.Context) error {
	debugPrint("Shutting down server gracefully...\n")
	return srv.Shutdown(ctx)
}

// ShutdownWithTimeout is a convenience method that creates a context with timeout
// and calls Shutdown. Default timeout is 5 seconds.
func ShutdownWithTimeout(srv *http.Server, timeout ...time.Duration) error {
	t := 5 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), t)
	defer cancel()

	return Shutdown(srv, ctx)
}

// RunTLS attaches the router to a http.Server and starts listening and serving HTTPS requests.
// It is a shortcut for http.ListenAndServeTLS(addr, certFile, keyFile, router)
// Note: this method will block the calling goroutine indefinitely unless an error happens.
func (engine *Engine) RunTLS(addr, certFile, keyFile string) (err error) {
	debugPrint("Listening and serving HTTPS on %s\n", addr)
	defer func() { debugPrintError(err) }()

	err = http.ListenAndServeTLS(addr, certFile, keyFile, engine)
	return
}

// Routes returns a slice of registered routes, including some useful information, such as:
// the http method, path, and the handler name.
func (engine *Engine) Routes() (routes RoutesInfo) {
	for _, tree := range engine.trees {
		routes = iterate("", tree.method, routes, tree.root)
	}
	return routes
}

func iterate(path, method string, routes RoutesInfo, root *node) RoutesInfo {
	path += root.path
	if len(root.handlers) > 0 {
		handlerFunc := root.handlers.Last()
		routes = append(routes, RouteInfo{
			Method:      method,
			Path:        path,
			Handler:     nameOfFunction(handlerFunc),
			HandlerFunc: handlerFunc,
		})
	}
	for _, child := range root.children {
		routes = iterate(path, method, routes, child)
	}
	return routes
}

func resolveAddress(addr []string) string {
	switch len(addr) {
	case 0:
		return ":5066"
	case 1:
		return addr[0]
	default:
		panic("too many parameters")
	}
}
