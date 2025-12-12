package auth

import "github.com/jaswant99k/gotap"

// RegisterRoutes registers all authentication routes
func RegisterRoutes(r *goTap.Engine, handler *Handler, jwtSecret string) {
	// Public routes
	public := r.Group("/api")
	{
		public.POST("/register", handler.Register)
		public.POST("/login", handler.Login)
	}

	// Protected routes (require authentication)
	auth := r.Group("/api")
	auth.Use(goTap.JWTAuth(jwtSecret))
	{
		auth.GET("/profile", handler.GetProfile)
		auth.PUT("/profile", handler.UpdateProfile)
		auth.POST("/change-password", handler.ChangePassword)
	}

	// Admin-only routes
	admin := r.Group("/api/admin")
	admin.Use(goTap.JWTAuth(jwtSecret))
	admin.Use(goTap.RequireRole("admin"))
	{
		admin.GET("/users", handler.ListUsers)
		admin.DELETE("/users/:id", handler.DeleteUser)
		admin.POST("/users/:id/permissions", handler.AssignPermissions)
	}
}
