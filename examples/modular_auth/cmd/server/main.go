package main

import (
	"log"
	"os"

	"gotap_modular_auth/modules/auth"
	"gotap_modular_auth/shared/database"

	"github.com/yourusername/goTap"
	"gorm.io/gorm"
)

func main() {
	// Load JWT secret
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "my-super-secret-jwt-key-change-this-in-production-minimum-32-chars"
	}

	// Initialize database
	db, err := database.Connect()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate models
	if err := db.AutoMigrate(&auth.User{}, &auth.Permission{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Seed default data
	seedDefaultData(db)

	// Initialize goTap
	r := goTap.Default()
	r.Use(goTap.GormInject(db))

	// Health check
	r.GET("/health", func(c *goTap.Context) {
		c.JSON(200, goTap.H{
			"status": "ok",
			"app":    "goTap Modular Auth Example",
		})
	})

	// Initialize auth module
	initAuthModule(r, db, jwtSecret)

	// Start server
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("🚀 Server starting on http://localhost:" + port)
	log.Println("📝 Default admin: admin@example.com / admin123")
	log.Println()
	log.Println("📚 API Endpoints:")
	log.Println("  POST /api/register - Register new user")
	log.Println("  POST /api/login - Login and get JWT token")
	log.Println("  GET  /api/profile - Get current user profile")
	log.Println("  PUT  /api/profile - Update profile")
	log.Println("  POST /api/change-password - Change password")
	log.Println("  GET  /api/admin/users - List all users (admin only)")
	log.Println()

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initAuthModule(r *goTap.Engine, db *gorm.DB, jwtSecret string) {
	repo := auth.NewRepository(db)
	service := auth.NewService(repo, jwtSecret)
	handler := auth.NewHandler(service)
	auth.RegisterRoutes(r, handler, jwtSecret)
	log.Println("✅ Auth module initialized")
}

func seedDefaultData(db *gorm.DB) {
	// Seed permissions
	permissions := []auth.Permission{
		{Name: "create_product", Description: "Create new products"},
		{Name: "edit_product", Description: "Edit existing products"},
		{Name: "delete_product", Description: "Delete products"},
		{Name: "view_orders", Description: "View all orders"},
		{Name: "manage_users", Description: "Manage user accounts"},
		{Name: "view_reports", Description: "View analytics reports"},
	}

	for _, perm := range permissions {
		db.FirstOrCreate(&perm, auth.Permission{Name: perm.Name})
	}

	// Create default admin user
	var count int64
	db.Model(&auth.User{}).Where("role = ?", "admin").Count(&count)

	if count == 0 {
		hash, _ := auth.HashPassword("admin123")
		admin := auth.User{
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
		var perms []auth.Permission
		db.Find(&perms)
		db.Model(&admin).Association("Permissions").Append(perms)

		log.Println("✅ Default admin created: admin@example.com / admin123")
	}
}
