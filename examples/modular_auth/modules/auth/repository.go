package auth

import "gorm.io/gorm"

// Repository handles database operations for auth
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new auth repository
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new user
func (r *Repository) Create(user *User) (*User, error) {
	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

// FindByEmail finds a user by email
func (r *Repository) FindByEmail(email string) (*User, error) {
	var user User
	if err := r.db.Where("email = ?", email).
		Preload("Permissions").
		First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by ID
func (r *Repository) FindByID(id uint) (*User, error) {
	var user User
	if err := r.db.Preload("Permissions").First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *Repository) Update(user *User) error {
	return r.db.Save(user).Error
}

// FindAll returns all users
func (r *Repository) FindAll() ([]User, error) {
	var users []User
	if err := r.db.Preload("Permissions").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Delete deletes a user
func (r *Repository) Delete(id uint) error {
	return r.db.Delete(&User{}, id).Error
}

// AssignPermissions assigns permissions to a user
func (r *Repository) AssignPermissions(userID uint, permissionIDs []uint) error {
	var user User
	if err := r.db.First(&user, userID).Error; err != nil {
		return err
	}

	var permissions []Permission
	if err := r.db.Find(&permissions, permissionIDs).Error; err != nil {
		return err
	}

	return r.db.Model(&user).Association("Permissions").Replace(permissions)
}
