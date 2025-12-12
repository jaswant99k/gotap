# goTap Authentication & Authorization - Complete Guide

## ✅ YES, goTap Fully Supports Users, Roles & Permissions!

goTap has **built-in JWT authentication** with role-based access control (RBAC) and custom permissions support.

---

## Features

### 🔐 JWT Authentication
- HS256 token signing
- Customizable claims (UserID, Username, Email, Role, Custom fields)
- Token extraction from Header, Query, or Cookie
- Token expiration and validation
- Refresh token support
- Context integration

### 👥 Role-Based Access Control (RBAC)
- `RequireRole("admin")` - Single role requirement
- `RequireAnyRole("admin", "manager")` - Multiple role options
- Custom role hierarchy
- Role stored in JWT claims

### 🔑 Permissions System
- Many-to-Many user-permission relationships
- Fine-grained permission checking
- Custom permission middleware
- Permission preloading with GORM

### 🔒 Basic Authentication
- Simple username/password protection
- Good for internal tools and admin panels
- Constant-time comparison (timing attack prevention)

---

## Quick Example

```go
package main

import (
    "github.com/yourusername/goTap"
)

func main() {
    r := goTap.Default()
    secret := "your-secret-key-minimum-32-characters"
    
    // Public routes
    r.POST("/login", loginHandler)
    r.POST("/register", registerHandler)
    
    // Protected routes (require authentication)
    auth := r.Group("/api")
    auth.Use(goTap.JWTAuth(secret))
    {
        auth.GET("/profile", getProfile)
    }
    
    // Admin-only routes
    admin := r.Group("/admin")
    admin.Use(goTap.JWTAuth(secret))
    admin.Use(goTap.RequireRole("admin"))
    {
        admin.GET("/users", listUsers)
    }
    
    r.Run(":8080")
}
```

---

## How to Use in Your New Project

### Option 1: Manual Setup (5 minutes)

Follow the guide at: `c:\goTap\examples\auth\QUICK_START.md`

**Steps:**
1. Install bcrypt: `go get golang.org/x/crypto/bcrypt`
2. Add User and Permission models to `models/models.go`
3. Add Register, Login, GetProfile handlers to `handlers/handlers.go`
4. Update `main.go` with auth routes
5. Add `JWT_SECRET` to `config/.env`
6. Run and test!

### Option 2: Use Complete Example

Copy the complete working example:

```powershell
# Copy example to your project
Copy-Item "C:\goTap\examples\auth\complete_example.go" `
          -Destination "C:\Users\verve\Music\vervepos\main.go"

# Install dependencies
cd C:\Users\verve\Music\vervepos
go get golang.org/x/crypto/bcrypt
go mod tidy

# Configure database
# Edit config/.env with your PostgreSQL credentials

# Run
go run main.go
```

Default admin credentials:
- Email: `admin@example.com`
- Password: `admin123`

---

## Authentication Flow

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       │ POST /api/register
       │ {username, email, password}
       ▼
┌─────────────────────────┐
│  Register Handler       │
│  - Hash password        │
│  - Create user in DB    │
│  - Return success       │
└─────────────────────────┘

┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       │ POST /api/login
       │ {email, password}
       ▼
┌─────────────────────────┐
│  Login Handler          │
│  - Find user            │
│  - Verify password      │
│  - Generate JWT token   │
│  - Return token         │
└────────┬────────────────┘
         │
         │ Token: eyJhbGciOiJIUzI1NiIs...
         ▼
┌─────────────┐
│   Client    │ Stores token in memory/localStorage
└──────┬──────┘
       │
       │ GET /api/profile
       │ Header: Authorization: Bearer <token>
       ▼
┌─────────────────────────┐
│  JWT Middleware         │
│  - Extract token        │
│  - Validate signature   │
│  - Check expiration     │
│  - Store claims in ctx  │
└────────┬────────────────┘
         │
         │ Claims: {UserID, Role, ...}
         ▼
┌─────────────────────────┐
│  Role Middleware        │
│  - Check user role      │
│  - Allow/Deny access    │
└────────┬────────────────┘
         │
         │ Authorized
         ▼
┌─────────────────────────┐
│  Handler                │
│  - Get claims from ctx  │
│  - Process request      │
│  - Return response      │
└─────────────────────────┘
```

---

## Available Middleware

### 1. JWT Authentication

```go
// Basic
r.Use(goTap.JWTAuth(secret))

// With configuration
r.Use(goTap.JWTAuthWithConfig(goTap.JWTConfig{
    Secret:        secret,
    TokenLookup:   "header:Authorization",  // or "query:token" or "cookie:jwt"
    TokenHeadName: "Bearer",
    ErrorHandler:  customErrorHandler,
}))
```

### 2. Role Checking

```go
// Single role
r.Use(goTap.RequireRole("admin"))

// Multiple roles
r.Use(goTap.RequireAnyRole("admin", "manager", "employee"))
```

### 3. Basic Authentication

```go
accounts := goTap.Accounts{
    "admin":    "password123",
    "manager":  "secret456",
}
r.Use(goTap.BasicAuth(accounts))
```

---

## Working with JWT Claims

### Generate Token

```go
claims := goTap.JWTClaims{
    UserID:    "123",
    Username:  "john",
    Email:     "john@example.com",
    Role:      "user",
    ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    Custom: map[string]interface{}{
        "department": "sales",
        "region":     "US",
    },
}

token, err := goTap.GenerateJWT(secret, claims)
```

### Extract Claims in Handler

```go
func myHandler(c *goTap.Context) {
    // Get claims
    claims, exists := goTap.GetJWTClaims(c)
    if !exists {
        c.JSON(401, goTap.H{"error": "Unauthorized"})
        return
    }
    
    // Access user info
    userID := claims.UserID
    username := claims.Username
    role := claims.Role
    
    // Access custom fields
    department := claims.Custom["department"]
    
    c.JSON(200, goTap.H{
        "user_id":  userID,
        "username": username,
    })
}
```

### Refresh Token

```go
newToken, err := goTap.RefreshToken(oldToken, secret, 7*24*time.Hour)
```

---

## Common Use Cases

### 1. Public and Protected Routes

```go
// Public
r.POST("/login", handlers.Login)

// Protected (any authenticated user)
auth := r.Group("/api")
auth.Use(goTap.JWTAuth(secret))
{
    auth.GET("/profile", handlers.GetProfile)
    auth.GET("/products", handlers.GetProducts)
}
```

### 2. Admin Panel

```go
admin := r.Group("/admin")
admin.Use(goTap.JWTAuth(secret))
admin.Use(goTap.RequireRole("admin"))
{
    admin.GET("/users", handlers.ListUsers)
    admin.DELETE("/users/:id", handlers.DeleteUser)
    admin.POST("/permissions", handlers.CreatePermission)
}
```

### 3. Multi-Role Access

```go
manage := r.Group("/manage")
manage.Use(goTap.JWTAuth(secret))
manage.Use(goTap.RequireAnyRole("admin", "manager"))
{
    manage.GET("/orders", handlers.ListOrders)
    manage.PUT("/orders/:id", handlers.UpdateOrder)
}
```

### 4. Resource Ownership Check

```go
func UpdatePost(c *goTap.Context) {
    claims, _ := goTap.GetJWTClaims(c)
    postID := c.Param("id")
    
    db := goTap.MustGetGorm(c)
    var post Post
    
    // Check ownership
    if err := db.Where("id = ? AND user_id = ?", postID, claims.UserID).
              First(&post).Error; err != nil {
        c.JSON(403, goTap.H{"error": "You don't own this post"})
        return
    }
    
    // Update post...
}
```

### 5. Custom Permission Check

```go
func RequirePermission(perm string) goTap.HandlerFunc {
    return func(c *goTap.Context) {
        claims, _ := goTap.GetJWTClaims(c)
        
        db := goTap.MustGetGorm(c)
        var user User
        db.Preload("Permissions").First(&user, claims.UserID)
        
        if !user.HasPermission(perm) {
            c.JSON(403, goTap.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// Usage
r.DELETE("/products/:id", RequirePermission("delete_product"), handlers.DeleteProduct)
```

---

## Database Models

```go
type User struct {
    gorm.Model
    Username     string       `gorm:"uniqueIndex;not null" json:"username"`
    Email        string       `gorm:"uniqueIndex;not null" json:"email"`
    PasswordHash string       `gorm:"not null" json:"-"`
    Role         string       `gorm:"default:'user'" json:"role"`
    IsActive     bool         `gorm:"default:true" json:"is_active"`
    Permissions  []Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}

type Permission struct {
    gorm.Model
    Name        string `gorm:"uniqueIndex;not null" json:"name"`
    Description string `json:"description"`
    Users       []User `gorm:"many2many:user_permissions;" json:"-"`
}
```

---

## Testing with curl/PowerShell

### Register

```powershell
curl -X POST http://localhost:8080/api/register `
  -H "Content-Type: application/json" `
  -d '{\"username\":\"john\",\"email\":\"john@example.com\",\"password\":\"password123\"}'
```

### Login

```powershell
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/login" `
    -Method POST `
    -ContentType "application/json" `
    -Body '{"email":"john@example.com","password":"password123"}'

$token = $response.token
```

### Access Protected Route

```powershell
Invoke-RestMethod -Uri "http://localhost:8080/api/profile" `
    -Headers @{Authorization="Bearer $token"}
```

---

## Documentation Files

- **Complete Guide**: `c:\goTap\examples\auth\README.md` (15 pages)
- **Quick Start**: `c:\goTap\examples\auth\QUICK_START.md` (5 minute setup)
- **Working Example**: `c:\goTap\examples\auth\complete_example.go` (450 lines)

---

## Security Features

✅ HMAC-SHA256 signature  
✅ Token expiration checking  
✅ Constant-time password comparison (bcrypt)  
✅ Timing attack prevention  
✅ Custom error handlers  
✅ Flexible token extraction (Header/Query/Cookie)  
✅ Context isolation (thread-safe)  
✅ Role-based access control  
✅ Fine-grained permissions  
✅ Password hashing (bcrypt cost 14)  

---

## Best Practices

1. **Use Environment Variables** for JWT_SECRET
2. **HTTPS Only** in production
3. **Short Token Expiration** (1-24 hours)
4. **Implement Refresh Tokens** for long sessions
5. **Hash Passwords** with bcrypt (cost 14)
6. **Validate Input** with binding tags
7. **Log Authentication Events** for auditing
8. **Rate Limit** login endpoints
9. **Implement 2FA** for admin accounts
10. **Regular Security Audits**

---

## Summary

**YES, goTap has full authentication support with:**

- ✅ JWT authentication (built-in)
- ✅ Role-based access control (RequireRole, RequireAnyRole)
- ✅ Permissions system (many-to-many with GORM)
- ✅ Basic authentication (username/password)
- ✅ Token refresh support
- ✅ Custom error handling
- ✅ Multiple token extraction methods
- ✅ Context integration (GetJWTClaims)
- ✅ Production-ready security

**To use in your new project:**

1. Copy `complete_example.go` OR
2. Follow `QUICK_START.md` (5 minutes) OR
3. Read full guide in `README.md`

**Default admin credentials in example:**
- Email: `admin@example.com`
- Password: `admin123`

**Need help?** Check the examples or ask for specific authentication scenarios!
