package auth

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User represents a user account
type User struct {
	gorm.Model
	Username     string       `gorm:"uniqueIndex;not null" json:"username"`
	Email        string       `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string       `gorm:"not null" json:"-"`
	Role         string       `gorm:"default:'user'" json:"role"`
	IsActive     bool         `gorm:"default:true" json:"is_active"`
	Permissions  []Permission `gorm:"many2many:user_permissions;" json:"permissions,omitempty"`
}

// Permission represents a permission
type Permission struct {
	gorm.Model
	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Description string `json:"description"`
	Users       []User `gorm:"many2many:user_permissions;" json:"-"`
}

// LoginRequest represents login credentials
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents registration data
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// ChangePasswordRequest represents password change data
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdateProfileRequest represents profile update data
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"omitempty,min=3"`
	Email    string `json:"email" binding:"omitempty,email"`
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
