# Quick Start: Adding Authentication to Your goTap Project

This guide shows you how to add JWT authentication with roles and permissions to a project created with `new-project.ps1`.

## Step 1: Install bcrypt Package

```powershell
cd C:\Users\verve\Music\vervepos
go get golang.org/x/crypto/bcrypt
```

## Step 2: Update Models

Add these models to your `models/models.go`:

```go
package models

import (
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

// User model for authentication
type User struct {
    gorm.Model
    Username     string       `gorm:"uniqueIndex;not null" json:"username"`
    Email        string       `gorm:"uniqueIndex;not null" json:"email"`
    PasswordHash string       `gorm:"not null" json:"-"`
    Role         string       `gorm:"default:'user'" json:"role"` // "admin", "manager", "user"
    IsActive     bool         `gorm:"default:true" json:"is_active"`
    Permissions  []Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}

type Permission struct {
    gorm.Model
    Name        string `gorm:"uniqueIndex;not null" json:"name"`
    Description string `json:"description"`
    Users       []User `gorm:"many2many:user_permissions;" json:"-"`
}

// HashPassword hashes a plain text password
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

// VerifyPassword checks if password matches hash
func (u *User) VerifyPassword(password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
    return err == nil
}

// HasPermission checks if user has a specific permission
func (u *User) HasPermission(permissionName string) bool {
    for _, perm := range u.Permissions {
        if perm.Name == permissionName {
            return true
        }
    }
    return false
}
```

## Step 3: Add Auth Handlers

Add these to your `handlers/handlers.go`:

```go
package handlers

import (
    "fmt"
    "time"
    "yourapp/models"
    "github.com/yourusername/goTap"
)

var JWTSecret = "change-this-to-a-long-random-secret-minimum-32-characters"

// Register creates a new user account
func Register(c *goTap.Context) {
    var req struct {
        Username string `json:"username" binding:"required,min=3"`
        Email    string `json:"email" binding:"required,email"`
        Password string `json:"password" binding:"required,min=8"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    passwordHash, err := models.HashPassword(req.Password)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to hash password"})
        return
    }

    user := models.User{
        Username:     req.Username,
        Email:        req.Email,
        PasswordHash: passwordHash,
        Role:         "user",
    }

    db := goTap.MustGetGorm(c)
    if err := db.Create(&user).Error; err != nil {
        c.JSON(400, goTap.H{"error": "Username or email already exists"})
        return
    }

    c.JSON(201, goTap.H{
        "message": "User created successfully",
        "user": goTap.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
        },
    })
}

// Login authenticates user and returns JWT token
func Login(c *goTap.Context) {
    var req struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }

    db := goTap.MustGetGorm(c)
    var user models.User
    if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }

    if !user.IsActive {
        c.JSON(403, goTap.H{"error": "Account is deactivated"})
        return
    }

    if !user.VerifyPassword(req.Password) {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }

    claims := goTap.JWTClaims{
        UserID:    fmt.Sprint(user.ID),
        Username:  user.Username,
        Email:     user.Email,
        Role:      user.Role,
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    }

    token, err := goTap.GenerateJWT(JWTSecret, claims)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to generate token"})
        return
    }

    c.JSON(200, goTap.H{
        "token": token,
        "user": goTap.H{
            "id":       user.ID,
            "username": user.Username,
            "email":    user.Email,
            "role":     user.Role,
        },
    })
}

// GetProfile returns current user's profile
func GetProfile(c *goTap.Context) {
    claims, _ := goTap.GetJWTClaims(c)

    db := goTap.MustGetGorm(c)
    var user models.User
    if err := db.First(&user, claims.UserID).Error; err != nil {
        c.JSON(404, goTap.H{"error": "User not found"})
        return
    }

    c.JSON(200, goTap.H{
        "id":         user.ID,
        "username":   user.Username,
        "email":      user.Email,
        "role":       user.Role,
        "created_at": user.CreatedAt,
    })
}
```

## Step 4: Update main.go

Update your routes in `main.go`:

```go
package main

import (
    "log"
    "os"
    "yourapp/handlers"
    "yourapp/models"
    "github.com/yourusername/goTap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // Database connection
    dsn := os.Getenv("DB_DSN")
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // Auto-migrate (add User and Permission models)
    db.AutoMigrate(
        &models.User{},
        &models.Permission{},
        &models.Product{},
        &models.Customer{},
    )

    // Create default admin user
    createDefaultAdmin(db)

    // Initialize goTap
    r := goTap.Default()
    r.Use(goTap.GormInject(db))

    // JWT secret from environment
    if secret := os.Getenv("JWT_SECRET"); secret != "" {
        handlers.JWTSecret = secret
    }

    // Public routes (no authentication)
    public := r.Group("/api/v1")
    {
        public.POST("/register", handlers.Register)
        public.POST("/login", handlers.Login)
    }

    // Protected routes (require authentication)
    auth := r.Group("/api/v1")
    auth.Use(goTap.JWTAuth(handlers.JWTSecret))
    {
        auth.GET("/profile", handlers.GetProfile)
        
        // Products (authenticated users)
        auth.GET("/products", handlers.GetProducts)
        auth.GET("/products/:id", handlers.GetProduct)
    }

    // Admin-only routes
    admin := r.Group("/api/v1")
    admin.Use(goTap.JWTAuth(handlers.JWTSecret))
    admin.Use(goTap.RequireRole("admin"))
    {
        admin.POST("/products", handlers.CreateProduct)
        admin.PUT("/products/:id", handlers.UpdateProduct)
        admin.DELETE("/products/:id", handlers.DeleteProduct)
    }

    // Manager or Admin routes
    manage := r.Group("/api/v1")
    manage.Use(goTap.JWTAuth(handlers.JWTSecret))
    manage.Use(goTap.RequireAnyRole("admin", "manager"))
    {
        manage.GET("/customers", handlers.GetCustomers)
        manage.POST("/customers", handlers.CreateCustomer)
    }

    log.Println("🚀 Server starting on :8080")
    log.Println("📝 Default admin: admin@example.com / admin123")
    r.Run(":8080")
}

func createDefaultAdmin(db *gorm.DB) {
    var count int64
    db.Model(&models.User{}).Where("role = ?", "admin").Count(&count)

    if count == 0 {
        hash, _ := models.HashPassword("admin123")
        admin := models.User{
            Username:     "admin",
            Email:        "admin@example.com",
            PasswordHash: hash,
            Role:         "admin",
            IsActive:     true,
        }

        if err := db.Create(&admin).Error; err == nil {
            log.Println("✅ Default admin created: admin@example.com / admin123")
        }
    }
}
```

## Step 5: Update Environment Variables

Add to your `config/.env`:

```env
# JWT Secret (use a long random string in production!)
JWT_SECRET=your-super-secret-jwt-key-minimum-32-characters-long
```

## Step 6: Run and Test

```powershell
# Start server
go run main.go

# In another terminal, test the API:

# Register a new user
curl -X POST http://localhost:8080/api/v1/register `
  -H "Content-Type: application/json" `
  -d '{\"username\":\"john\",\"email\":\"john@example.com\",\"password\":\"password123\"}'

# Login (get token)
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/login" `
    -Method POST `
    -ContentType "application/json" `
    -Body '{"email":"john@example.com","password":"password123"}'

$token = $response.token
Write-Host "Token: $token"

# Get profile (requires authentication)
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/profile" `
    -Headers @{Authorization="Bearer $token"}

# Try to create product (will fail for regular user)
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/products" `
    -Method POST `
    -Headers @{Authorization="Bearer $token"} `
    -ContentType "application/json" `
    -Body '{"name":"Test","sku":"TEST01","price":99.99}'

# Login as admin
$adminResponse = Invoke-RestMethod -Uri "http://localhost:8080/api/v1/login" `
    -Method POST `
    -ContentType "application/json" `
    -Body '{"email":"admin@example.com","password":"admin123"}'

$adminToken = $adminResponse.token

# Create product as admin (will succeed)
Invoke-RestMethod -Uri "http://localhost:8080/api/v1/products" `
    -Method POST `
    -Headers @{Authorization="Bearer $adminToken"} `
    -ContentType "application/json" `
    -Body '{"name":"Test","sku":"TEST01","price":99.99,"stock":100}'
```

## Authentication Flow

1. **User Registration** → POST `/api/v1/register`
2. **User Login** → POST `/api/v1/login` → Returns JWT token
3. **Access Protected Routes** → Include header: `Authorization: Bearer <token>`
4. **JWT Middleware** → Validates token → Sets user info in context
5. **Role Middleware** → Checks user role → Allows/denies access

## Available Roles

- **admin**: Full access to all routes
- **manager**: Can manage products and customers
- **user**: Can view products, access own profile

## Adding Custom Permissions

```go
// In handlers/handlers.go
func RequirePermission(permission string) goTap.HandlerFunc {
    return func(c *goTap.Context) {
        claims, _ := goTap.GetJWTClaims(c)
        
        db := goTap.MustGetGorm(c)
        var user models.User
        if err := db.Preload("Permissions").First(&user, claims.UserID).Error; err != nil {
            c.JSON(404, goTap.H{"error": "User not found"})
            c.Abort()
            return
        }
        
        if !user.HasPermission(permission) {
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

## Security Best Practices

1. **Use Environment Variables** for JWT_SECRET
2. **Use HTTPS** in production
3. **Set Short Token Expiration** (1-24 hours)
4. **Implement Refresh Tokens** for long sessions
5. **Hash Passwords** with bcrypt (cost 14)
6. **Validate Input** with binding tags
7. **Log Authentication Events** for auditing
8. **Rate Limit** login attempts
9. **Implement 2FA** for admin accounts
10. **Regular Security Audits**

## Complete Example

See `c:\goTap\examples\auth\complete_example.go` for a full working example with:
- User registration and login
- JWT token generation
- Role-based access control
- Permissions system
- Admin panel
- Password change
- Profile management

## Next Steps

- [ ] Add password reset via email
- [ ] Implement refresh tokens
- [ ] Add rate limiting
- [ ] Add 2FA (two-factor authentication)
- [ ] Implement session management
- [ ] Add audit logging
- [ ] Create user management UI

---

**Ready to secure your goTap application!** 🔒
