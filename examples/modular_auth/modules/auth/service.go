package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/yourusername/goTap"
)

// Service contains business logic for authentication
type Service struct {
	repo      *Repository
	jwtSecret string
}

// NewService creates a new auth service
func NewService(repo *Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: jwtSecret,
	}
}

// Register creates a new user account
func (s *Service) Register(req RegisterRequest) (*User, error) {
	// Hash password
	hash, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		Role:         "user",
		IsActive:     true,
	}

	return s.repo.Create(user)
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(req LoginRequest) (string, *User, error) {
	// Find user
	user, err := s.repo.FindByEmail(req.Email)
	if err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Check if active
	if !user.IsActive {
		return "", nil, errors.New("account is deactivated")
	}

	// Verify password
	if !user.VerifyPassword(req.Password) {
		return "", nil, errors.New("invalid credentials")
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

	token, err := goTap.GenerateJWT(s.jwtSecret, claims)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return token, user, nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(id uint) (*User, error) {
	return s.repo.FindByID(id)
}

// UpdateProfile updates user profile information
func (s *Service) UpdateProfile(userID uint, req UpdateProfileRequest) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}

	return s.repo.Update(user)
}

// ChangePassword changes user password
func (s *Service) ChangePassword(userID uint, req ChangePasswordRequest) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	// Verify current password
	if !user.VerifyPassword(req.CurrentPassword) {
		return errors.New("current password is incorrect")
	}

	// Hash new password
	newHash, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	user.PasswordHash = newHash
	return s.repo.Update(user)
}

// GetAllUsers returns all users
func (s *Service) GetAllUsers() ([]User, error) {
	return s.repo.FindAll()
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(id uint) error {
	return s.repo.Delete(id)
}

// AssignPermissions assigns permissions to a user
func (s *Service) AssignPermissions(userID uint, permissionIDs []uint) error {
	return s.repo.AssignPermissions(userID, permissionIDs)
}
