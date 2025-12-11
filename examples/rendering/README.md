# goTap Rendering Example

This example demonstrates all rendering capabilities in goTap.

## Features Demonstrated

### 1. **JSON Rendering**
- Standard JSON responses
- Structured data serialization

### 2. **XML Rendering**
- XML document generation
- Automatic formatting

### 3. **YAML Rendering**
- YAML configuration output
- Key-value pair formatting

### 4. **HTML Rendering**
- Template rendering
- Dynamic data injection
- Multiple templates

### 5. **Content Negotiation**
- Automatic format selection based on Accept header
- Support for JSON, XML, HTML

### 6. **Server-Sent Events (SSE)**
- Real-time event streaming
- Periodic updates
- Keep-alive connections

### 7. **Cookie Management**
- Set/Get cookies
- Cookie options (secure, httponly, maxage)

### 8. **Redirects**
- Permanent redirects (301)
- Temporary redirects (302)

### 9. **Raw Data**
- Binary data responses
- Custom content types

### 10. **File Downloads**
- File attachment headers
- Custom filename

## Running the Example

```bash
cd examples/rendering
go run main.go
```

The server will start on `http://localhost:5066`

## API Examples

### JSON Response
```bash
curl http://localhost:5066/api/products
```

Response:
```json
{
  "products": [
    {
      "id": "P001",
      "name": "Laptop",
      "price": 999.99,
      "description": "High-performance laptop",
      "category": "Electronics"
    }
  ],
  "total": 3
}
```

### XML Response
```bash
curl http://localhost:5066/api/products.xml
```

Response:
```xml
<products>
  <id>P001</id>
  <name>Laptop</name>
  <price>999.99</price>
  <description>High-performance laptop</description>
  <category>Electronics</category>
</products>
```

### YAML Response
```bash
curl http://localhost:5066/api/products.yaml
```

Response:
```yaml
name: Product Catalog
version: 1.0
products: 3
```

### HTML Response
Open in browser:
```
http://localhost:5066/
http://localhost:5066/products
http://localhost:5066/product/P001
```

### Content Negotiation
```bash
# Get JSON
curl -H "Accept: application/json" http://localhost:5066/api/data

# Get XML
curl -H "Accept: application/xml" http://localhost:5066/api/data

# Get HTML
curl -H "Accept: text/html" http://localhost:5066/api/data
```

### Server-Sent Events
```bash
# Single event
curl http://localhost:5066/events

# Continuous stream
curl http://localhost:5066/events/stream
```

### Cookies
```bash
# Set cookies
curl -c cookies.txt http://localhost:5066/cookie/set

# Get cookies
curl -b cookies.txt http://localhost:5066/cookie/get
```

### Redirects
```bash
# Permanent redirect
curl -L http://localhost:5066/redirect

# Temporary redirect
curl -L http://localhost:5066/redirect-temp
```

## Code Examples

### JSON Rendering
```go
app.GET("/api/data", func(c *goTap.Context) {
    c.JSON(http.StatusOK, goTap.H{
        "message": "Hello World",
        "status": "success",
    })
})
```

### XML Rendering
```go
type Product struct {
    ID    string  `xml:"id"`
    Name  string  `xml:"name"`
    Price float64 `xml:"price"`
}

app.GET("/api/product", func(c *goTap.Context) {
    product := Product{ID: "P001", Name: "Laptop", Price: 999.99}
    c.XML(http.StatusOK, product)
})
```

### HTML Template Rendering
```go
// Load templates
app.LoadHTMLGlob("templates/*")

app.GET("/", func(c *goTap.Context) {
    c.HTML(http.StatusOK, "index.html", goTap.H{
        "title": "Home",
        "user": "John",
    })
})
```

### Content Negotiation
```go
app.GET("/api/data", func(c *goTap.Context) {
    data := Product{ID: "P001", Name: "Laptop", Price: 999.99}
    
    c.Negotiate(http.StatusOK, goTap.Negotiate{
        Offered:  []string{"application/json", "application/xml"},
        JSONData: data,
        XMLData:  data,
    })
})
```

### Server-Sent Events
```go
app.GET("/events", func(c *goTap.Context) {
    c.SSE("message", goTap.H{
        "time": time.Now(),
        "data": "Event data",
    })
})
```

### Cookie Management
```go
app.GET("/cookie/set", func(c *goTap.Context) {
    c.SetCookie("session", "abc123", 3600, "/", "", false, true)
    c.JSON(http.StatusOK, goTap.H{"message": "Cookie set"})
})

app.GET("/cookie/get", func(c *goTap.Context) {
    session, err := c.Cookie("session")
    c.JSON(http.StatusOK, goTap.H{"session": session, "error": err})
})
```

## Template Syntax

goTap uses Go's `html/template` package. Example:

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{.title}}</title>
</head>
<body>
    <h1>{{.message}}</h1>
    
    {{range .products}}
    <div>
        <h2>{{.Name}}</h2>
        <p>${{.Price}}</p>
    </div>
    {{end}}
</body>
</html>
```

## Rendering Methods

| Method | Description | Content-Type |
|--------|-------------|--------------|
| `c.JSON(code, obj)` | Render JSON | `application/json` |
| `c.XML(code, obj)` | Render XML | `application/xml` |
| `c.YAML(code, obj)` | Render YAML | `application/x-yaml` |
| `c.HTML(code, name, data)` | Render HTML template | `text/html` |
| `c.String(code, format, ...)` | Render plain text | `text/plain` |
| `c.Data(code, type, data)` | Render raw bytes | Custom |
| `c.SSE(event, data)` | Server-Sent Event | `text/event-stream` |
| `c.File(filepath)` | Serve file | Auto-detected |
| `c.FileAttachment(file, name)` | Download file | Auto-detected |
| `c.Redirect(code, location)` | HTTP redirect | - |

## Best Practices

1. **Use appropriate rendering based on client**
   - APIs: JSON/XML
   - Web pages: HTML
   - Config files: YAML

2. **Set proper status codes**
   - 200 OK for success
   - 201 Created for new resources
   - 301/302 for redirects

3. **Load templates once at startup**
   ```go
   app.LoadHTMLGlob("templates/*")
   ```

4. **Use content negotiation for flexible APIs**
   - Support multiple formats
   - Let client choose via Accept header

5. **Secure cookies appropriately**
   - Use HttpOnly for session cookies
   - Use Secure in production (HTTPS)
   - Set appropriate MaxAge

6. **Handle SSE disconnections**
   - Check for client disconnect
   - Clean up resources

## Performance Tips

- Cache compiled templates
- Use streaming for large responses
- Minimize template logic
- Pre-allocate buffers for large JSON/XML
