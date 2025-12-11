package main

import (
	"net/http"

	"github.com/yourusername/goTap"
)

func main() {
	router := goTap.Default()

	// Custom SecureJSON prefix (for enhanced security)
	router.SecureJSONPrefix(")]}',\n")

	// 1. Regular JSON - Escapes HTML characters
	router.GET("/json", func(c *goTap.Context) {
		c.JSON(http.StatusOK, goTap.H{
			"message": "Regular JSON",
			"html":    "<b>This will be escaped</b>",
			"url":     "http://example.com?foo=bar&baz=qux",
		})
	})

	// 2. IndentedJSON - Pretty-printed JSON for debugging
	router.GET("/json/indented", func(c *goTap.Context) {
		c.IndentedJSON(http.StatusOK, goTap.H{
			"message": "Indented JSON for debugging",
			"user": goTap.H{
				"id":    1,
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"products": []goTap.H{
				{"id": 1, "name": "Product A", "price": 99.99},
				{"id": 2, "name": "Product B", "price": 149.99},
			},
		})
	})

	// 3. SecureJSON - Prevents JSON hijacking for arrays
	router.GET("/json/secure", func(c *goTap.Context) {
		// This will prepend ")]}',\n" for arrays
		c.SecureJSON(http.StatusOK, []string{
			"sensitive",
			"data",
			"array",
		})
	})

	// 4. SecureJSON with object (no prefix)
	router.GET("/json/secure-object", func(c *goTap.Context) {
		// No prefix for objects
		c.SecureJSON(http.StatusOK, goTap.H{
			"status": "ok",
			"data":   "This is an object, no prefix needed",
		})
	})

	// 5. JSONP - For cross-domain requests
	router.GET("/json/jsonp", func(c *goTap.Context) {
		// Callback parameter: ?callback=myCallback
		c.JSONP(http.StatusOK, goTap.H{
			"message": "JSONP response",
			"data":    "This supports cross-domain requests",
		})
	})

	// 6. AsciiJSON - Unicode characters escaped to ASCII
	router.GET("/json/ascii", func(c *goTap.Context) {
		c.AsciiJSON(http.StatusOK, goTap.H{
			"message":  "ASCII JSON",
			"chinese":  "GO语言",
			"japanese": "日本語",
			"emoji":    "🚀💻",
			"html":     "<script>alert('test')</script>",
		})
	})

	// 7. PureJSON - No HTML escaping
	router.GET("/json/pure", func(c *goTap.Context) {
		c.PureJSON(http.StatusOK, goTap.H{
			"message":     "Pure JSON",
			"html":        "<b>Not escaped</b>",
			"script":      "<script>console.log('literal')</script>",
			"url":         "http://example.com?foo=bar&baz=qux",
			"description": "HTML chars are NOT escaped: < > & ' \"",
		})
	})

	// 8. Comparison endpoint - Shows difference between JSON and PureJSON
	router.GET("/json/compare", func(c *goTap.Context) {
		format := c.DefaultQuery("format", "json")

		data := goTap.H{
			"html":   "<b>Bold text</b>",
			"script": "<script>alert('xss')</script>",
			"url":    "http://example.com?a=1&b=2",
		}

		switch format {
		case "pure":
			c.PureJSON(http.StatusOK, data)
		case "ascii":
			c.AsciiJSON(http.StatusOK, data)
		case "secure":
			c.SecureJSON(http.StatusOK, data)
		case "indented":
			c.IndentedJSON(http.StatusOK, data)
		default:
			c.JSON(http.StatusOK, data)
		}
	})

	// 9. Abort with JSON - Error handling
	router.GET("/json/abort", func(c *goTap.Context) {
		authenticated := c.DefaultQuery("auth", "false")

		if authenticated != "true" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, goTap.H{
				"error":   "Unauthorized",
				"message": "Authentication required",
				"code":    401,
			})
			return
		}

		c.JSON(http.StatusOK, goTap.H{
			"message": "Authenticated successfully",
		})
	})

	// 10. Abort with PureJSON - No HTML escaping in errors
	router.GET("/json/abort-pure", func(c *goTap.Context) {
		c.AbortWithStatusPureJSON(http.StatusBadRequest, goTap.H{
			"error":   "Invalid request",
			"details": "<error>Malformed input</error>",
			"hint":    "Check your <parameters>",
		})
	})

	// 11. International POS Transaction Example
	router.GET("/pos/transaction", func(c *goTap.Context) {
		format := c.DefaultQuery("format", "json")

		transaction := goTap.H{
			"id":       "TXN-2025-001",
			"merchant": "商店名称",
			"amount":   1234.56,
			"currency": "¥",
			"items": []goTap.H{
				{"name": "产品A", "price": 99.99},
				{"name": "製品B", "price": 149.99},
			},
			"note": "<Important: 重要な取引>",
		}

		switch format {
		case "ascii":
			// For international characters - convert to ASCII
			c.AsciiJSON(http.StatusOK, transaction)
		case "pure":
			// Keep HTML/special chars literal
			c.PureJSON(http.StatusOK, transaction)
		default:
			c.JSON(http.StatusOK, transaction)
		}
	})

	// 12. API Documentation endpoint
	router.GET("/", func(c *goTap.Context) {
		c.IndentedJSON(http.StatusOK, goTap.H{
			"title":   "goTap JSON Rendering Demo",
			"version": "1.0.0",
			"endpoints": []goTap.H{
				{
					"path":        "/json",
					"method":      "GET",
					"description": "Regular JSON with HTML escaping",
				},
				{
					"path":        "/json/indented",
					"method":      "GET",
					"description": "Pretty-printed JSON (use for debugging)",
				},
				{
					"path":        "/json/secure",
					"method":      "GET",
					"description": "Secure JSON with anti-hijacking prefix for arrays",
				},
				{
					"path":        "/json/secure-object",
					"method":      "GET",
					"description": "Secure JSON with object (no prefix)",
				},
				{
					"path":        "/json/jsonp?callback=myFunc",
					"method":      "GET",
					"description": "JSONP for cross-domain requests",
				},
				{
					"path":        "/json/ascii",
					"method":      "GET",
					"description": "ASCII-only JSON with Unicode escapes",
				},
				{
					"path":        "/json/pure",
					"method":      "GET",
					"description": "Pure JSON without HTML escaping",
				},
				{
					"path":        "/json/compare?format=json|pure|ascii|secure|indented",
					"method":      "GET",
					"description": "Compare different JSON formats",
				},
				{
					"path":        "/json/abort?auth=true|false",
					"method":      "GET",
					"description": "Abort with JSON response",
				},
				{
					"path":        "/json/abort-pure",
					"method":      "GET",
					"description": "Abort with PureJSON response",
				},
				{
					"path":        "/pos/transaction?format=json|ascii|pure",
					"method":      "GET",
					"description": "International POS transaction example",
				},
			},
			"usage": goTap.H{
				"json":     "Default - Escapes HTML: < becomes \\u003c",
				"indented": "Pretty-printed with 4-space indentation",
				"secure":   "Adds 'while(1);' or custom prefix to arrays",
				"jsonp":    "Wraps response: callback({...});",
				"ascii":    "Unicode → \\uXXXX (GO语言 → GO\\u8bed\\u8a00)",
				"pure":     "Literal HTML chars (no escaping)",
			},
		})
	})

	router.Run(":5066")
}
