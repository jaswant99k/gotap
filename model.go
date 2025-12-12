package goTap

import (
	"time"

	"gorm.io/gorm"
)

// Model is a base model struct that includes common fields for database records.
// It provides the same fields as gorm.Model but under the goTap namespace.
// This allows for better framework integration and potential future enhancements.
//
// Usage:
//
//	type User struct {
//	    gotap.Model
//	    Username string `gorm:"uniqueIndex;not null" json:"username"`
//	    Email    string `gorm:"uniqueIndex;not null" json:"email"`
//	}
type Model struct {
	ID        uint           `gorm:"primarykey" json:"id" example:"1"`
	CreatedAt time.Time      `json:"created_at" example:"2024-01-01T00:00:00Z"`
	UpdatedAt time.Time      `json:"updated_at" example:"2024-01-01T00:00:00Z"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty" swaggertype:"string" example:"2024-01-01T00:00:00Z"`
}

// BaseModel is an alias for Model for backward compatibility
type BaseModel = Model
