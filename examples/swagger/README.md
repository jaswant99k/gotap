# goTap Swagger/OpenAPI Integration

This guide shows you how to add Swagger/OpenAPI documentation to your goTap applications for interactive API testing.

## Features

✅ **Automatic API Documentation** - Generate docs from code annotations  
✅ **Interactive UI** - Test APIs directly in the browser  
✅ **OpenAPI 3.0 Support** - Industry-standard API specification  
✅ **Authentication Support** - JWT, BasicAuth in Swagger UI  
✅ **Request/Response Examples** - Show example payloads  
✅ **Model Schemas** - Auto-generate from structs  

## Installation

### 1. Install Swagger CLI Tool

```powershell
go install github.com/swaggo/swag/cmd/swag@latest
```

### 2. Add Dependencies to Your Project

```powershell
go get -u github.com/swaggo/files
go get -u github.com/swaggo/gin-swagger
```

## Quick Start

### Step 1: Add Swagger Annotations to main.go

```go
package main

import (
	"github.com/yourusername/goTap"
	_ "yourproject/docs" // Import generated docs
)

// @title           VervePOS API
// @version         1.0
// @description     Point of Sale API with authentication and product management
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8055
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	r := goTap.Default()

	// Setup Swagger UI
	goTap.SetupSwagger(r, "/swagger")

	// Your routes...
	r.Run(":8055")
}
```

### Step 2: Add Annotations to Handlers

```go
// Register godoc
// @Summary      Register new user
// @Description  Create a new user account
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration data"
// @Success      201  {object}  map[string]interface{}  "User created successfully"
// @Failure      400  {object}  map[string]interface{}  "Bad request"
// @Router       /auth/register [post]
func (h *Handler) Register(c *goTap.Context) {
	// Implementation...
}

// Login godoc
// @Summary      User login
// @Description  Authenticate user and return JWT token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login credentials"
// @Success      200  {object}  map[string]interface{}  "Login successful"
// @Failure      401  {object}  map[string]interface{}  "Invalid credentials"
// @Router       /auth/login [post]
func (h *Handler) Login(c *goTap.Context) {
	// Implementation...
}

// GetProfile godoc
// @Summary      Get user profile
// @Description  Get current authenticated user's profile
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}  "User profile"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Router       /auth/profile [get]
func (h *Handler) GetProfile(c *goTap.Context) {
	// Implementation...
}

// GetProducts godoc
// @Summary      List products
// @Description  Get paginated list of products
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        page     query  int     false  "Page number"  default(1)
// @Param        per_page query  int     false  "Items per page"  default(10)
// @Param        category query  string  false  "Filter by category"
// @Success      200  {object}  map[string]interface{}  "Products list"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Router       /products [get]
func (h *Handler) GetProducts(c *goTap.Context) {
	// Implementation...
}

// CreateProduct godoc
// @Summary      Create product
// @Description  Create a new product (admin only)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateProductRequest true "Product data"
// @Success      201  {object}  map[string]interface{}  "Product created"
// @Failure      400  {object}  map[string]interface{}  "Bad request"
// @Failure      401  {object}  map[string]interface{}  "Unauthorized"
// @Failure      403  {object}  map[string]interface{}  "Forbidden"
// @Router       /products [post]
func (h *Handler) CreateProduct(c *goTap.Context) {
	// Implementation...
}
```

### Step 3: Generate Swagger Documentation

```powershell
# Navigate to your project
cd C:\projects\vervepos_api

# Generate docs (creates docs/docs.go, docs/swagger.json, docs/swagger.yaml)
swag init -g cmd/server/main.go --output docs
```

### Step 4: Run and Access Swagger UI

```powershell
go run cmd/server/main.go
```

Open browser: **http://localhost:8055/swagger/index.html**

## Advanced Usage

### Protected Swagger UI (Admin Only)

```go
// Setup Swagger with JWT authentication
goTap.SetupSwaggerWithAuth(r, "/swagger", 
	goTap.JWTAuth(jwtSecret),
	goTap.RequireRole("admin"),
)
```

### Custom Swagger Configuration

```go
import "github.com/yourusername/goTap"

swaggerConfig := &goTap.SwaggerConfig{
	URL:                      "doc.json",
	DocExpansion:             "list",  // list, full, none
	DeepLinking:              true,
	PersistAuthorization:     true,
	DefaultModelsExpandDepth: 1,
}

r.GET("/swagger/*any", goTap.SwaggerHandler(swaggerConfig))
```

### Model Documentation

```go
// User represents a user account
type User struct {
	ID       uint   `json:"id" example:"1"`
	Username string `json:"username" example:"john_doe"`
	Email    string `json:"email" example:"john@example.com"`
	Role     string `json:"role" example:"user" enums:"admin,user,guest"`
	IsActive bool   `json:"is_active" example:"true"`
} // @name User

// RegisterRequest represents registration data
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3" example:"john_doe"`
	Email    string `json:"email" binding:"required,email" example:"john@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"password123"`
} // @name RegisterRequest
```

### Response Examples

```go
// @Success 200 {object} map[string]interface{} "Success" example({"message": "Success", "data": {"id": 1, "name": "Product"}})
// @Failure 400 {object} map[string]interface{} "Bad Request" example({"error": "Invalid input"})
```

## Swagger Annotation Reference

### General Information

```go
// @title           API Title
// @version         1.0
// @description     API Description
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api
```

### Security Definitions

```go
// JWT Bearer Token
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by JWT token

// Basic Authentication
// @securityDefinitions.basic BasicAuth

// OAuth2
// @securitydefinitions.oauth2.application OAuth2Application
// @tokenUrl https://example.com/oauth/token
// @scope.write Grants write access
// @scope.admin Grants admin access
```

### Endpoint Annotations

```go
// FunctionName godoc
// @Summary      Short summary
// @Description  Detailed description
// @Tags         TagName
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path      int     true  "Item ID"
// @Param        name    query     string  false "Name filter"
// @Param        request body      Model   true  "Request body"
// @Success      200     {object}  Model   "Success response"
// @Failure      400     {object}  Error   "Bad request"
// @Router       /items/{id} [get]
```

### Parameter Types

- `path` - URL path parameter (e.g., `/users/{id}`)
- `query` - URL query parameter (e.g., `?page=1`)
- `header` - HTTP header
- `body` - Request body
- `formData` - Form data

### Response Types

```go
// Object response
// @Success 200 {object} models.User

// Array response
// @Success 200 {array} models.User

// Primitive response
// @Success 200 {string} string "OK"

// No content
// @Success 204 "No Content"

// Multiple responses
// @Success 200 {object} models.User
// @Success 201 {object} models.User
```

## Complete Example Structure

```
yourproject/
├── cmd/
│   └── server/
│       └── main.go          # Swagger general info here
├── modules/
│   ├── auth/
│   │   ├── handlers.go      # Auth endpoint annotations
│   │   ├── models.go        # Request/Response models
│   │   └── routes.go
│   └── products/
│       ├── handlers.go      # Product endpoint annotations
│       ├── models.go        # Product models
│       └── routes.go
├── docs/                    # Generated by swag init
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
└── go.mod
```

## Testing in Swagger UI

1. **Open Swagger UI**: http://localhost:8055/swagger/index.html

2. **Authenticate**:
   - Click "Authorize" button
   - Enter: `Bearer YOUR_JWT_TOKEN`
   - Click "Authorize" then "Close"

3. **Test Endpoints**:
   - Click on endpoint to expand
   - Click "Try it out"
   - Fill in parameters
   - Click "Execute"
   - View response

## Swagger vs Postman

| Feature | Swagger UI | Postman |
|---------|-----------|---------|
| Auto-generated | ✅ Yes | ❌ Manual |
| In-browser | ✅ Yes | ❌ Desktop app |
| Documentation | ✅ Built-in | ➖ Separate |
| Code annotations | ✅ Yes | ❌ No |
| Collaboration | ✅ Share URL | ✅ Collections |
| Testing | ✅ Basic | ✅ Advanced |

## Troubleshooting

### Swagger UI Not Loading

```powershell
# Regenerate docs
swag init -g cmd/server/main.go --output docs

# Verify docs import
# Check main.go has: import _ "yourproject/docs"
```

### 404 on Swagger Routes

```go
// Make sure you registered Swagger routes
goTap.SetupSwagger(r, "/swagger")
```

### Authentication Not Working

```go
// In Swagger UI, enter token with "Bearer " prefix:
Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### Models Not Showing

```go
// Add example tags to struct fields
type User struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"John Doe"`
}
```

## Best Practices

1. **Add Examples**: Always include `example:""` tags in structs
2. **Descriptive Summaries**: Write clear @Summary and @Description
3. **Group Endpoints**: Use @Tags to organize endpoints
4. **Document Errors**: Include all possible @Failure codes
5. **Update Regularly**: Run `swag init` after API changes
6. **Version Control**: Add `docs/` to .gitignore if desired
7. **Security First**: Consider protecting Swagger UI in production

## Integration with CI/CD

```yaml
# GitHub Actions example
- name: Install swag
  run: go install github.com/swaggo/swag/cmd/swag@latest

- name: Generate Swagger docs
  run: swag init -g cmd/server/main.go --output docs

- name: Build
  run: go build -o app cmd/server/main.go
```

## Next Steps

- ✅ Add Swagger to your project
- ✅ Document all endpoints
- ✅ Test in Swagger UI
- ✅ Share with frontend team
- ✅ Automate doc generation in CI/CD

## Resources

- [Swaggo Documentation](https://github.com/swaggo/swag)
- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI Demo](https://petstore.swagger.io/)
