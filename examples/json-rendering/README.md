# JSON Rendering Example

This example demonstrates all JSON rendering methods available in goTap, matching Gin's feature set.

## Features Demonstrated

### 1. **Regular JSON** (`c.JSON()`)
- Escapes HTML characters for security
- `<` → `\u003c`, `>` → `\u003e`, `&` → `\u0026`
- Default choice for most APIs

### 2. **IndentedJSON** (`c.IndentedJSON()`)
- Pretty-printed with 4-space indentation
- **Use only for debugging** - higher CPU/bandwidth cost
- Makes JSON human-readable

### 3. **SecureJSON** (`c.SecureJSON()`)
- **Prevents JSON hijacking attacks**
- Prepends `while(1);` (or custom prefix) to array responses
- Essential for sensitive array data in production
- No prefix for non-array responses

### 4. **JSONP** (`c.JSONP()`)
- Cross-domain request support
- Wraps response in callback: `callback({...});`
- Reads callback from `?callback=` query parameter
- **XSS protection**: Escapes callback name

### 5. **AsciiJSON** (`c.AsciiJSON()`)
- Converts all Unicode to ASCII escape sequences
- `GO语言` → `GO\u8bed\u8a00`
- Perfect for **international POS systems**
- Ensures compatibility with ASCII-only systems

### 6. **PureJSON** (`c.PureJSON()`)
- **No HTML character escaping**
- `<b>` stays as `<b>` instead of `\u003cb\u003e`
- Use when HTML chars are data, not code
- Great for display systems

### 7. **AbortWithStatusJSON** / **AbortWithStatusPureJSON**
- Stop execution and return JSON error
- Middleware-friendly error handling

## Running the Example

```bash
cd examples/json-rendering
go run main.go
```

Server starts on `http://localhost:5066`

## Testing Endpoints

### 1. Regular JSON (HTML Escaped)
```bash
curl http://localhost:5066/json
```
**Output:**
```json
{"html":"\u003cb\u003eThis will be escaped\u003c/b\u003e","message":"Regular JSON","url":"http://example.com?foo=bar\u0026baz=qux"}
```

### 2. Indented JSON (Pretty-Printed)
```bash
curl http://localhost:5066/json/indented
```
**Output:**
```json
{
    "message": "Indented JSON for debugging",
    "products": [
        {
            "id": 1,
            "name": "Product A",
            "price": 99.99
        }
    ],
    "user": {
        "email": "john@example.com",
        "id": 1,
        "name": "John Doe"
    }
}
```

### 3. Secure JSON (Anti-Hijacking)
```bash
curl http://localhost:5066/json/secure
```
**Output:**
```
)]}',
["sensitive","data","array"]
```
Note the prefix `)]}',\n` prevents JSON hijacking.

### 4. JSONP (Cross-Domain)
```bash
curl "http://localhost:5066/json/jsonp?callback=myCallback"
```
**Output:**
```javascript
myCallback({"data":"This supports cross-domain requests","message":"JSONP response"});
```

### 5. ASCII JSON (Unicode → ASCII)
```bash
curl http://localhost:5066/json/ascii
```
**Output:**
```json
{"chinese":"GO\u8bed\u8a00","emoji":"\ud83d\ude80\ud83d\udcbb","html":"\u003cscript\u003ealert('test')\u003c/script\u003e","japanese":"\u65e5\u672c\u8a9e","message":"ASCII JSON"}
```

### 6. Pure JSON (No Escaping)
```bash
curl http://localhost:5066/json/pure
```
**Output:**
```json
{"description":"HTML chars are NOT escaped: < > & ' \"","html":"<b>Not escaped</b>","message":"Pure JSON","script":"<script>console.log('literal')</script>","url":"http://example.com?foo=bar&baz=qux"}
```

### 7. Compare Formats
```bash
# Regular JSON
curl "http://localhost:5066/json/compare?format=json"

# Pure JSON
curl "http://localhost:5066/json/compare?format=pure"

# ASCII JSON
curl "http://localhost:5066/json/compare?format=ascii"

# Secure JSON
curl "http://localhost:5066/json/compare?format=secure"

# Indented JSON
curl "http://localhost:5066/json/compare?format=indented"
```

### 8. Error Handling
```bash
# Unauthorized error
curl "http://localhost:5066/json/abort?auth=false"

# Pure JSON error
curl http://localhost:5066/json/abort-pure
```

### 9. International POS Transaction
```bash
# Regular JSON (escaped)
curl "http://localhost:5066/pos/transaction?format=json"

# ASCII JSON (for legacy systems)
curl "http://localhost:5066/pos/transaction?format=ascii"

# Pure JSON (for display)
curl "http://localhost:5066/pos/transaction?format=pure"
```

## Use Cases

### When to Use Each Type

| Type | Use Case | Example |
|------|----------|---------|
| **JSON** | Default for APIs | Product catalogs, user data |
| **IndentedJSON** | Development/debugging | API documentation, dev tools |
| **SecureJSON** | Sensitive array data | User lists, transaction arrays |
| **JSONP** | Legacy cross-domain | Old browsers, cross-origin APIs |
| **AsciiJSON** | International data | Chinese/Japanese POS systems |
| **PureJSON** | Display systems | HTML content, rich text |

### Security Considerations

1. **Regular JSON**: Safe for most APIs (escapes HTML)
2. **SecureJSON**: Use for arrays with sensitive data
3. **JSONP**: Ensure callback XSS protection (goTap handles this)
4. **PureJSON**: ⚠️ **Careful with user input** - no escaping
5. **AsciiJSON**: Safe - converts all non-ASCII to escapes

## Performance

From fastest to slowest:
1. JSON (standard)
2. PureJSON (no escaping)
3. AsciiJSON (Unicode conversion)
4. SecureJSON (prefix check)
5. IndentedJSON (formatting)
6. JSONP (wrapping)

**Recommendation**: Use regular `JSON()` unless you have a specific need.

## Comparison with Gin

goTap now has **feature parity** with Gin for JSON rendering:

- ✅ Regular JSON
- ✅ IndentedJSON
- ✅ SecureJSON with custom prefix
- ✅ JSONP with XSS protection
- ✅ AsciiJSON
- ✅ PureJSON
- ✅ AbortWithStatusJSON
- ✅ AbortWithStatusPureJSON

## Advanced Example: Custom SecureJSON Prefix

```go
router := goTap.Default()

// Use custom prefix instead of "while(1);"
router.SecureJSONPrefix(")]}',\n")

router.GET("/api/users", func(c *goTap.Context) {
    users := []User{...}
    c.SecureJSON(200, users) // Will use ")]}',\n" prefix
})
```

## Integration with POS Systems

For Point-of-Sale systems handling international transactions:

```go
// Use AsciiJSON for terminals with ASCII-only displays
c.AsciiJSON(200, goTap.H{
    "receipt": "收据",  // → "\u6536\u636e"
    "total": "¥1,234.56", // → "\u00a51,234.56"
})

// Use PureJSON for modern display systems
c.PureJSON(200, goTap.H{
    "receipt": "收据",  // Stays as 收据
    "html": "<b>Total</b>", // Stays literal
})
```

## API Documentation

Visit the root endpoint for full API documentation:
```bash
curl http://localhost:5066/
```
