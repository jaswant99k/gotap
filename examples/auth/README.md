# Authentication & Authorization in goTap

goTap provides built-in JWT and BasicAuth middleware for securing your APIs.

## Table of Contents
1. [JWT Authentication](#jwt-authentication)
2. [Role-Based Access Control (RBAC)](#role-based-access-control)
3. [Complete User Authentication Example](#complete-example)
4. [Password Hashing](#password-hashing)
5. [Refresh Tokens](#refresh-tokens)
6. [BasicAuth](#basic-authentication)
7. [Custom Permissions](#custom-permissions)

---

## JWT Authentication

### Quick Start

```go
import "github.com/yourusername/goTap"

func main() {
    r := goTap.Default()
    
    // JWT secret key (use environment variable in production!)
    secret := "your-secret-key-minimum-32-characters"
    
    // Public routes (no authentication)
    r.POST("/login", loginHandler)
    r.POST("/register", registerHandler)
    
    // Protected routes (require authentication)
    authorized := r.Group("/api")
    authorized.Use(goTap.JWTAuth(secret))
    {
        authorized.GET("/profile", getProfile)
        authorized.PUT("/profile", updateProfile)
    }
    
    r.Run(":8080")
}
```

### Generating JWT Tokens

```go
func loginHandler(c *goTap.Context) {
    var loginReq struct {
        Email    string `json:"email" binding:"required"`
        Password string `json:"password" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&loginReq); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    
    // Find user in database
    db := goTap.MustGetGorm(c)
    var user User
    if err := db.Where("email = ?", loginReq.Email).First(&user).Error; err != nil {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }
    
    // Verify password (using bcrypt)
    if !verifyPassword(user.PasswordHash, loginReq.Password) {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }
    
    // Generate JWT token
    claims := goTap.JWTClaims{
        UserID:    user.ID,
        Username:  user.Username,
        Email:     user.Email,
        Role:      user.Role, // "admin", "manager", "user", etc.
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    }
    
    token, err := goTap.GenerateJWT(secret, claims)
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
```

### Accessing JWT Claims in Handlers

```go
func getProfile(c *goTap.Context) {
    // Get JWT claims from context
    claims, exists := goTap.GetJWTClaims(c)
    if !exists {
        c.JSON(401, goTap.H{"error": "Unauthorized"})
        return
    }
    
    // Access user information
    userID := claims.UserID
    username := claims.Username
    email := claims.Email
    role := claims.Role
    
    // Fetch full user profile from database
    db := goTap.MustGetGorm(c)
    var user User
    if err := db.First(&user, "id = ?", userID).Error; err != nil {
        c.JSON(404, goTap.H{"error": "User not found"})
        return
    }
    
    c.JSON(200, goTap.H{
        "id":       user.ID,
        "username": user.Username,
        "email":    user.Email,
        "role":     user.Role,
        "created":  user.CreatedAt,
    })
}
```

---

## Role-Based Access Control (RBAC)

### Single Role Requirement

```go
// Only admins can access
adminRoutes := r.Group("/admin")
adminRoutes.Use(goTap.JWTAuth(secret))
adminRoutes.Use(goTap.RequireRole("admin"))
{
    adminRoutes.GET("/users", listAllUsers)
    adminRoutes.DELETE("/users/:id", deleteUser)
    adminRoutes.POST("/products", createProduct)
}
```

### Multiple Role Options

```go
// Admins OR managers can access
managerRoutes := r.Group("/manage")
managerRoutes.Use(goTap.JWTAuth(secret))
managerRoutes.Use(goTap.RequireAnyRole("admin", "manager"))
{
    managerRoutes.GET("/orders", listOrders)
    managerRoutes.PUT("/orders/:id", updateOrder)
}
```

### User Role Hierarchy

```go
const (
    RoleAdmin    = "admin"      // Full access
    RoleManager  = "manager"    // Manage resources
    RoleEmployee = "employee"   // Limited access
    RoleCustomer = "customer"   // Read-only
)

// In your User model
type User struct {
    gorm.Model
    Username     string `gorm:"uniqueIndex;not null"`
    Email        string `gorm:"uniqueIndex;not null"`
    PasswordHash string `gorm:"not null"`
    Role         string `gorm:"default:'customer'"`
    IsActive     bool   `gorm:"default:true"`
}
```

---

## Complete Example: User Authentication System

### 1. User Model with Permissions

```go
package models

import (
    "gorm.io/gorm"
    "golang.org/x/crypto/bcrypt"
)

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

// HasAnyPermission checks if user has any of the given permissions
func (u *User) HasAnyPermission(permissions ...string) bool {
    for _, perm := range permissions {
        if u.HasPermission(perm) {
            return true
        }
    }
    return false
}
```

### 2. Authentication Handlers

```go
package handlers

import (
    "time"
    "yourapp/models"
    "github.com/yourusername/goTap"
)

var jwtSecret = "your-secret-key-minimum-32-characters" // Use env variable!

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
    
    // Hash password
    passwordHash, err := models.HashPassword(req.Password)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to hash password"})
        return
    }
    
    // Create user
    user := models.User{
        Username:     req.Username,
        Email:        req.Email,
        PasswordHash: passwordHash,
        Role:         "user", // Default role
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
    
    // Find user
    db := goTap.MustGetGorm(c)
    var user models.User
    if err := db.Where("email = ?", req.Email).
               Preload("Permissions").
               First(&user).Error; err != nil {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }
    
    // Check if user is active
    if !user.IsActive {
        c.JSON(403, goTap.H{"error": "Account is deactivated"})
        return
    }
    
    // Verify password
    if !user.VerifyPassword(req.Password) {
        c.JSON(401, goTap.H{"error": "Invalid credentials"})
        return
    }
    
    // Generate JWT token
    claims := goTap.JWTClaims{
        UserID:    fmt.Sprint(user.ID),
        Username:  user.Username,
        Email:     user.Email,
        Role:      user.Role,
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
        Custom: map[string]interface{}{
            "is_active": user.IsActive,
        },
    }
    
    token, err := goTap.GenerateJWT(jwtSecret, claims)
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
    if err := db.Preload("Permissions").First(&user, claims.UserID).Error; err != nil {
        c.JSON(404, goTap.H{"error": "User not found"})
        return
    }
    
    c.JSON(200, goTap.H{
        "id":          user.ID,
        "username":    user.Username,
        "email":       user.Email,
        "role":        user.Role,
        "permissions": user.Permissions,
        "created_at":  user.CreatedAt,
    })
}

// UpdateProfile updates user information
func UpdateProfile(c *goTap.Context) {
    claims, _ := goTap.GetJWTClaims(c)
    
    var req struct {
        Username string `json:"username" binding:"omitempty,min=3"`
        Email    string `json:"email" binding:"omitempty,email"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    
    db := goTap.MustGetGorm(c)
    updates := make(map[string]interface{})
    
    if req.Username != "" {
        updates["username"] = req.Username
    }
    if req.Email != "" {
        updates["email"] = req.Email
    }
    
    if err := db.Model(&models.User{}).Where("id = ?", claims.UserID).
               Updates(updates).Error; err != nil {
        c.JSON(400, goTap.H{"error": "Update failed"})
        return
    }
    
    c.JSON(200, goTap.H{"message": "Profile updated successfully"})
}

// ChangePassword changes user password
func ChangePassword(c *goTap.Context) {
    claims, _ := goTap.GetJWTClaims(c)
    
    var req struct {
        CurrentPassword string `json:"current_password" binding:"required"`
        NewPassword     string `json:"new_password" binding:"required,min=8"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    
    db := goTap.MustGetGorm(c)
    var user models.User
    if err := db.First(&user, claims.UserID).Error; err != nil {
        c.JSON(404, goTap.H{"error": "User not found"})
        return
    }
    
    // Verify current password
    if !user.VerifyPassword(req.CurrentPassword) {
        c.JSON(401, goTap.H{"error": "Current password is incorrect"})
        return
    }
    
    // Hash new password
    newHash, err := models.HashPassword(req.NewPassword)
    if err != nil {
        c.JSON(500, goTap.H{"error": "Failed to hash password"})
        return
    }
    
    // Update password
    if err := db.Model(&user).Update("password_hash", newHash).Error; err != nil {
        c.JSON(500, goTap.H{"error": "Failed to update password"})
        return
    }
    
    c.JSON(200, goTap.H{"message": "Password changed successfully"})
}
```

### 3. Main Application Setup

```go
package main

import (
    "log"
    "os"
    "time"
    "yourapp/models"
    "yourapp/handlers"
    "github.com/yourusername/goTap"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

func main() {
    // Database connection
    dsn := os.Getenv("DATABASE_URL")
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }
    
    // Auto-migrate models
    db.AutoMigrate(&models.User{}, &models.Permission{})
    
    // Create default permissions
    seedPermissions(db)
    
    // Initialize goTap
    r := goTap.Default()
    r.Use(goTap.GormInject(db))
    
    // JWT secret from environment
    jwtSecret := os.Getenv("JWT_SECRET")
    if jwtSecret == "" {
        log.Fatal("JWT_SECRET environment variable is required")
    }
    
    // Public routes
    public := r.Group("/api")
    {
        public.POST("/register", handlers.Register)
        public.POST("/login", handlers.Login)
    }
    
    // Protected routes (require authentication)
    auth := r.Group("/api")
    auth.Use(goTap.JWTAuth(jwtSecret))
    {
        auth.GET("/profile", handlers.GetProfile)
        auth.PUT("/profile", handlers.UpdateProfile)
        auth.POST("/change-password", handlers.ChangePassword)
    }
    
    // Admin-only routes
    admin := r.Group("/api/admin")
    admin.Use(goTap.JWTAuth(jwtSecret))
    admin.Use(goTap.RequireRole("admin"))
    {
        admin.GET("/users", handlers.ListUsers)
        admin.DELETE("/users/:id", handlers.DeleteUser)
        admin.POST("/users/:id/permissions", handlers.AssignPermissions)
    }
    
    // Manager or Admin routes
    manage := r.Group("/api/manage")
    manage.Use(goTap.JWTAuth(jwtSecret))
    manage.Use(goTap.RequireAnyRole("admin", "manager"))
    {
        manage.GET("/orders", handlers.ListOrders)
        manage.PUT("/orders/:id/status", handlers.UpdateOrderStatus)
    }
    
    log.Println("Server starting on :8080")
    r.Run(":8080")
}

func seedPermissions(db *gorm.DB) {
    permissions := []models.Permission{
        {Name: "create_product", Description: "Create new products"},
        {Name: "edit_product", Description: "Edit existing products"},
        {Name: "delete_product", Description: "Delete products"},
        {Name: "view_orders", Description: "View all orders"},
        {Name: "manage_users", Description: "Manage user accounts"},
        {Name: "view_reports", Description: "View analytics reports"},
    }
    
    for _, perm := range permissions {
        db.FirstOrCreate(&perm, models.Permission{Name: perm.Name})
    }
}
```

---

## Password Hashing

Always use bcrypt for password hashing:

```bash
go get golang.org/x/crypto/bcrypt
```

```go
import "golang.org/x/crypto/bcrypt"

// Hash password
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    return string(bytes), err
}

// Verify password
func VerifyPassword(hash, password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

---

## Refresh Tokens

```go
func RefreshTokenHandler(c *goTap.Context) {
    var req struct {
        Token string `json:"token" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, goTap.H{"error": err.Error()})
        return
    }
    
    // Refresh token with 7-day extension
    newToken, err := goTap.RefreshToken(req.Token, jwtSecret, 7*24*time.Hour)
    if err != nil {
        c.JSON(401, goTap.H{"error": "Invalid or expired token"})
        return
    }
    
    c.JSON(200, goTap.H{
        "token": newToken,
    })
}
```

---

## Basic Authentication

For simple username/password protection:

```go
func main() {
    r := goTap.Default()
    
    // Basic Auth accounts
    accounts := goTap.Accounts{
        "admin":    "secret123",
        "manager":  "pass456",
    }
    
    // Protected routes
    authorized := r.Group("/admin")
    authorized.Use(goTap.BasicAuth(accounts))
    {
        authorized.GET("/dashboard", dashboardHandler)
    }
    
    r.Run(":8080")
}

func dashboardHandler(c *goTap.Context) {
    // Get authenticated username
    user := c.MustGet("user").(string)
    c.JSON(200, goTap.H{
        "message": "Welcome " + user,
    })
}
```

---

## Custom Permissions Middleware

Create fine-grained permission checking:

```go
// RequirePermission checks if user has a specific permission
func RequirePermission(permission string) goTap.HandlerFunc {
    return func(c *goTap.Context) {
        claims, exists := goTap.GetJWTClaims(c)
        if !exists {
            c.JSON(401, goTap.H{"error": "Unauthorized"})
            c.Abort()
            return
        }
        
        // Load user with permissions
        db := goTap.MustGetGorm(c)
        var user models.User
        if err := db.Preload("Permissions").First(&user, claims.UserID).Error; err != nil {
            c.JSON(404, goTap.H{"error": "User not found"})
            c.Abort()
            return
        }
        
        // Check permission
        if !user.HasPermission(permission) {
            c.JSON(403, goTap.H{
                "error": "Insufficient permissions",
                "required_permission": permission,
            })
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// Usage
r.DELETE("/products/:id", RequirePermission("delete_product"), deleteProductHandler)
```

---

## Testing Authentication

### Example curl commands:

```bash
# Register
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{"username":"john","email":"john@example.com","password":"secret123"}'

# Login
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"secret123"}'

# Get profile (with token)
curl http://localhost:8080/api/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Update profile
curl -X PUT http://localhost:8080/api/profile \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"username":"johndoe"}'
```

### PowerShell examples:

```powershell
# Login
$response = Invoke-RestMethod -Uri "http://localhost:8080/api/login" `
    -Method POST `
    -ContentType "application/json" `
    -Body '{"email":"john@example.com","password":"secret123"}'

$token = $response.token

# Get profile
Invoke-RestMethod -Uri "http://localhost:8080/api/profile" `
    -Headers @{Authorization="Bearer $token"}
```

---

## Best Practices

1. **Environment Variables**: Store JWT secret in environment variables, never hardcode
2. **Password Requirements**: Enforce minimum 8 characters, complexity rules
3. **Token Expiration**: Use short expiration times (1-24 hours) with refresh tokens
4. **HTTPS Only**: Always use HTTPS in production
5. **Rate Limiting**: Protect login endpoints from brute force attacks
6. **Audit Logging**: Log authentication attempts and permission checks
7. **Password Reset**: Implement secure password reset flow with email tokens
8. **Two-Factor Authentication**: Consider adding 2FA for sensitive operations
9. **Session Management**: Implement token revocation/blacklist for logout
10. **Regular Updates**: Keep dependencies updated for security patches

---

## Common Patterns

### Role Hierarchy Check

```go
func hasHigherRole(userRole, requiredRole string) bool {
    hierarchy := map[string]int{
        "customer": 1,
        "employee": 2,
        "manager": 3,
        "admin": 4,
    }
    return hierarchy[userRole] >= hierarchy[requiredRole]
}
```

### Permission-Based Routes

```go
type RoutePermission struct {
    Method     string
    Path       string
    Permission string
}

var routePermissions = []RoutePermission{
    {"POST", "/products", "create_product"},
    {"PUT", "/products/:id", "edit_product"},
    {"DELETE", "/products/:id", "delete_product"},
}
```

### Ownership Check

```go
func RequireOwnership(resourceType string) goTap.HandlerFunc {
    return func(c *goTap.Context) {
        claims, _ := goTap.GetJWTClaims(c)
        resourceID := c.Param("id")
        
        db := goTap.MustGetGorm(c)
        var count int64
        
        // Check if resource belongs to user
        db.Model(&Resource{}).Where("id = ? AND user_id = ?", resourceID, claims.UserID).Count(&count)
        
        if count == 0 {
            c.JSON(403, goTap.H{"error": "You don't own this resource"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}
```

---

## Summary

goTap provides everything you need for authentication:

- ✅ **JWT Authentication** with claims and roles
- ✅ **Role-Based Access Control** (single or multiple roles)
- ✅ **Basic Authentication** for simple use cases
- ✅ **Token Refresh** support
- ✅ **Custom Permissions** system
- ✅ **Context Integration** for easy access to user data

Start with JWT authentication and roles, then add permissions as your requirements grow!
