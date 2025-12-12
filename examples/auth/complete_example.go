package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jaswant99k/gotap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// MODELS
// ============================================================================

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

// ============================================================================
// HANDLERS
// ============================================================================

var jwtSecret = "my-super-secret-jwt-key-change-this-in-production-minimum-32-chars"

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
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		c.JSON(500, goTap.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	user := User{
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
			"role":     user.Role,
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
	var user User
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
	var user User
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

	if err := db.Model(&User{}).Where("id = ?", claims.UserID).
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
	var user User
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
	newHash, err := HashPassword(req.NewPassword)
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

// ListUsers returns all users (admin only)
func ListUsers(c *goTap.Context) {
	db := goTap.MustGetGorm(c)
	var users []User

	if err := db.Preload("Permissions").Find(&users).Error; err != nil {
		c.JSON(500, goTap.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(200, goTap.H{
		"users": users,
		"count": len(users),
	})
}

// DeleteUser deletes a user (admin only)
func DeleteUser(c *goTap.Context) {
	userID := c.Param("id")

	db := goTap.MustGetGorm(c)
	if err := db.Delete(&User{}, userID).Error; err != nil {
		c.JSON(500, goTap.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(200, goTap.H{"message": "User deleted successfully"})
}

// AssignPermissions assigns permissions to a user (admin only)
func AssignPermissions(c *goTap.Context) {
	userID := c.Param("id")

	var req struct {
		PermissionIDs []uint `json:"permission_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	db := goTap.MustGetGorm(c)

	// Find user
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(404, goTap.H{"error": "User not found"})
		return
	}

	// Find permissions
	var permissions []Permission
	if err := db.Find(&permissions, req.PermissionIDs).Error; err != nil {
		c.JSON(400, goTap.H{"error": "Invalid permission IDs"})
		return
	}

	// Replace permissions
	if err := db.Model(&user).Association("Permissions").Replace(permissions); err != nil {
		c.JSON(500, goTap.H{"error": "Failed to assign permissions"})
		return
	}

	c.JSON(200, goTap.H{
		"message":     "Permissions assigned successfully",
		"permissions": permissions,
	})
}

// ============================================================================
// MAIN
// ============================================================================

func main() {
	// Database connection
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=gotap_auth port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&User{}, &Permission{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed default permissions
	seedPermissions(db)

	// Create default admin user if not exists
	createDefaultAdmin(db)

	// Initialize goTap
	r := goTap.Default()
	r.Use(goTap.GormInject(db))

	// JWT secret from environment or use default (change in production!)
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		jwtSecret = secret
	}

	// Public routes
	public := r.Group("/api")
	{
		public.POST("/register", Register)
		public.POST("/login", Login)
	}

	// Protected routes (require authentication)
	auth := r.Group("/api")
	auth.Use(goTap.JWTAuth(jwtSecret))
	{
		auth.GET("/profile", GetProfile)
		auth.PUT("/profile", UpdateProfile)
		auth.POST("/change-password", ChangePassword)
	}

	// Admin-only routes
	admin := r.Group("/api/admin")
	admin.Use(goTap.JWTAuth(jwtSecret))
	admin.Use(goTap.RequireRole("admin"))
	{
		admin.GET("/users", ListUsers)
		admin.DELETE("/users/:id", DeleteUser)
		admin.POST("/users/:id/permissions", AssignPermissions)
	}

	log.Println("🚀 Server starting on http://localhost:8080")
	log.Println("📝 Default admin: admin@example.com / admin123")
	log.Println()
	log.Println("Try these commands:")
	log.Println("  Login: curl -X POST http://localhost:8080/api/login -H \"Content-Type: application/json\" -d '{\"email\":\"admin@example.com\",\"password\":\"admin123\"}'")
	log.Println()

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func seedPermissions(db *gorm.DB) {
	permissions := []Permission{
		{Name: "create_product", Description: "Create new products"},
		{Name: "edit_product", Description: "Edit existing products"},
		{Name: "delete_product", Description: "Delete products"},
		{Name: "view_orders", Description: "View all orders"},
		{Name: "manage_users", Description: "Manage user accounts"},
		{Name: "view_reports", Description: "View analytics reports"},
	}

	for _, perm := range permissions {
		db.FirstOrCreate(&perm, Permission{Name: perm.Name})
	}

	log.Println("✅ Permissions seeded")
}

func createDefaultAdmin(db *gorm.DB) {
	var count int64
	db.Model(&User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		hash, _ := HashPassword("admin123")
		admin := User{
			Username:     "admin",
			Email:        "admin@example.com",
			PasswordHash: hash,
			Role:         "admin",
			IsActive:     true,
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Println("Failed to create default admin:", err)
			return
		}

		// Assign all permissions to admin
		var permissions []Permission
		db.Find(&permissions)
		db.Model(&admin).Association("Permissions").Append(permissions)

		log.Println("✅ Default admin created: admin@example.com / admin123")
	}
}
