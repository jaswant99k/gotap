# goTap Binding & Validation Example

This example demonstrates goTap's comprehensive data binding and validation system.

## Features Demonstrated

### 1. **JSON Binding**
- Automatic JSON parsing
- Built-in validation
- Nested struct support

### 2. **Query Parameter Binding**
- URL query string parsing
- Type conversion
- Default values

### 3. **URI Parameter Binding**
- Path parameter extraction
- Validation support

### 4. **Header Binding**
- HTTP header extraction
- Authentication headers
- Custom headers

### 5. **Form Binding**
- Form-urlencoded data
- Multipart form data
- File uploads

### 6. **XML Binding**
- XML payload parsing
- Struct mapping

### 7. **Validation**
- Required fields
- Min/Max constraints
- Email validation
- URL validation
- OneOf constraints
- Custom rules

## Running the Example

```bash
cd examples/binding
go run main.go
```

The server will start on `http://localhost:5066`

## API Examples

### Create Transaction (JSON)
```bash
curl -X POST http://localhost:5066/api/transactions \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 299.99,
    "currency": "USD",
    "description": "Office supplies",
    "customer_id": "C12345",
    "items": [
      {"product_id": "P001", "name": "Laptop", "quantity": 1, "price": 299.99}
    ]
  }'
```

### Search Users (Query Parameters)
```bash
curl "http://localhost:5066/api/users?page=1&page_size=20&search=john&status=active"
```

### Get Transaction (URI Parameter)
```bash
curl http://localhost:5066/api/transactions/TXN123
```

### Protected Route (Headers)
```bash
curl http://localhost:5066/api/protected \
  -H "Authorization: Bearer token123" \
  -H "X-API-Key: key456" \
  -H "X-Device-ID: device789"
```

### Create Product (Form)
```bash
curl -X POST http://localhost:5066/api/products \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "name=Laptop&price=999.99&category=Electronics&in_stock=true&tags=computer&tags=office"
```

### Upload Product Image (Multipart)
```bash
curl -X POST http://localhost:5066/api/products/P001/image \
  -F "image=@product.jpg"
```

### Register Customer (Validation)
```bash
curl -X POST http://localhost:5066/api/customers/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "phone": "1234567890",
    "website": "https://example.com",
    "age": 25,
    "accept_terms": true
  }'
```

### Update Transaction (Mixed Binding)
```bash
curl -X PUT http://localhost:5066/api/transactions/TXN123 \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer token123" \
  -d '{"status": "completed", "description": "Payment received"}'
```

## Validation Tags

goTap supports the following validation tags:

| Tag | Description | Example |
|-----|-------------|---------|
| `required` | Field must not be empty | `validate:"required"` |
| `min` | Minimum value/length | `validate:"min=3"` |
| `max` | Maximum value/length | `validate:"max=100"` |
| `len` | Exact length | `validate:"len=10"` |
| `email` | Valid email format | `validate:"email"` |
| `url` | Valid URL format | `validate:"url"` |
| `numeric` | Only numbers | `validate:"numeric"` |
| `alpha` | Only letters | `validate:"alpha"` |
| `alphanum` | Letters and numbers | `validate:"alphanum"` |
| `oneof` | One of allowed values | `validate:"oneof=USD EUR GBP"` |

## Struct Tags

### Binding Tags
- `json:"field_name"` - JSON field mapping
- `form:"field_name"` - Form/Query field mapping
- `uri:"field_name"` - URI parameter mapping
- `header:"Header-Name"` - HTTP header mapping
- `xml:"field_name"` - XML field mapping

### Example Struct
```go
type Product struct {
    ID          string  `uri:"id" validate:"required"`
    Name        string  `json:"name" form:"name" validate:"required,min=3,max=100"`
    Price       float64 `json:"price" form:"price" validate:"required,min=0.01"`
    Category    string  `json:"category" validate:"oneof=electronics clothing food"`
    Description string  `json:"description" validate:"max=500"`
}
```

## Binding Methods

### Must Bind (Auto-aborts on error)
- `c.Bind(obj)` - Auto-detect binding type
- `c.BindJSON(obj)` - JSON only
- `c.BindXML(obj)` - XML only
- `c.BindQuery(obj)` - Query parameters
- `c.BindHeader(obj)` - HTTP headers
- `c.BindUri(obj)` - URI parameters

### Should Bind (Returns error)
- `c.ShouldBind(obj)` - Auto-detect binding type
- `c.ShouldBindJSON(obj)` - JSON only
- `c.ShouldBindXML(obj)` - XML only
- `c.ShouldBindQuery(obj)` - Query parameters
- `c.ShouldBindHeader(obj)` - HTTP headers
- `c.ShouldBindUri(obj)` - URI parameters
- `c.ShouldBindWith(obj, binding)` - Custom binding
- `c.ShouldBindBodyWith(obj, binding)` - Reusable body

## File Upload

```go
// Get single file
file, err := c.FormFile("image")
if err != nil {
    return err
}

// Save file
c.SaveUploadedFile(file, "/path/to/save/file.jpg")

// Access multipart form
form, err := c.MultipartForm()
files := form.File["images"] // Multiple files
```

## Custom Validation

You can create a custom validator:

```go
type MyValidator struct{}

func (v *MyValidator) ValidateStruct(obj interface{}) error {
    // Custom validation logic
    return nil
}

func (v *MyValidator) Engine() interface{} {
    return v
}

// Set custom validator
goTap.SetValidator(&MyValidator{})
```

## Error Handling

```go
if err := c.ShouldBindJSON(&obj); err != nil {
    c.JSON(http.StatusBadRequest, goTap.H{
        "error": "Invalid request",
        "details": err.Error(),
    })
    return
}
```

## Best Practices

1. **Use ShouldBind* for custom error handling**
   - More control over error responses
   - Better for APIs

2. **Use Bind* for simple cases**
   - Automatic 400 Bad Request on error
   - Less code

3. **Always validate sensitive data**
   - Add validation tags
   - Check business rules

4. **Set reasonable limits**
   - Use `min`/`max` for numbers
   - Limit string lengths
   - Control file sizes

5. **Combine multiple bindings**
   - URI for IDs
   - Headers for auth
   - Body for data

## POS-Specific Use Cases

### Transaction Processing
```go
type POSTransaction struct {
    TerminalID  string  `header:"X-Terminal-ID" validate:"required"`
    TxnID       string  `uri:"id"`
    Amount      float64 `json:"amount" validate:"required,min=0.01"`
    Currency    string  `json:"currency" validate:"required,oneof=USD EUR GBP"`
}
```

### Inventory Update
```go
type InventoryUpdate struct {
    ProductID string `uri:"product_id" validate:"required"`
    Quantity  int    `json:"quantity" validate:"required"`
    Location  string `json:"location" validate:"required"`
}
```

### Customer Lookup
```go
type CustomerSearch struct {
    Query  string `form:"q" validate:"required,min=2"`
    Limit  int    `form:"limit" validate:"min=1,max=50"`
    StoreID string `header:"X-Store-ID" validate:"required"`
}
```
