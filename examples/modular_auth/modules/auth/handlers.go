package auth

import (
	"strconv"

	"github.com/yourusername/goTap"
)

// Handler contains HTTP handlers for authentication
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register handles user registration
func (h *Handler) Register(c *goTap.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	user, err := h.service.Register(req)
	if err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
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

// Login handles user login
func (h *Handler) Login(c *goTap.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	token, user, err := h.service.Login(req)
	if err != nil {
		c.JSON(401, goTap.H{"error": err.Error()})
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
func (h *Handler) GetProfile(c *goTap.Context) {
	claims, exists := goTap.GetJWTClaims(c)
	if !exists {
		c.JSON(401, goTap.H{"error": "Unauthorized"})
		return
	}

	userID, _ := strconv.ParseUint(claims.UserID, 10, 32)
	user, err := h.service.GetUserByID(uint(userID))
	if err != nil {
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

// UpdateProfile updates user profile
func (h *Handler) UpdateProfile(c *goTap.Context) {
	claims, _ := goTap.GetJWTClaims(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	userID, _ := strconv.ParseUint(claims.UserID, 10, 32)
	if err := h.service.UpdateProfile(uint(userID), req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, goTap.H{"message": "Profile updated successfully"})
}

// ChangePassword handles password change
func (h *Handler) ChangePassword(c *goTap.Context) {
	claims, _ := goTap.GetJWTClaims(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	userID, _ := strconv.ParseUint(claims.UserID, 10, 32)
	if err := h.service.ChangePassword(uint(userID), req); err != nil {
		c.JSON(401, goTap.H{"error": err.Error()})
		return
	}

	c.JSON(200, goTap.H{"message": "Password changed successfully"})
}

// ListUsers returns all users (admin only)
func (h *Handler) ListUsers(c *goTap.Context) {
	users, err := h.service.GetAllUsers()
	if err != nil {
		c.JSON(500, goTap.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(200, goTap.H{
		"users": users,
		"count": len(users),
	})
}

// DeleteUser deletes a user (admin only)
func (h *Handler) DeleteUser(c *goTap.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	if err := h.service.DeleteUser(uint(userID)); err != nil {
		c.JSON(500, goTap.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(200, goTap.H{"message": "User deleted successfully"})
}

// AssignPermissions assigns permissions to a user (admin only)
func (h *Handler) AssignPermissions(c *goTap.Context) {
	userID, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var req struct {
		PermissionIDs []uint `json:"permission_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, goTap.H{"error": err.Error()})
		return
	}

	if err := h.service.AssignPermissions(uint(userID), req.PermissionIDs); err != nil {
		c.JSON(500, goTap.H{"error": "Failed to assign permissions"})
		return
	}

	c.JSON(200, goTap.H{"message": "Permissions assigned successfully"})
}
