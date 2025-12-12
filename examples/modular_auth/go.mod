module gotap_modular_auth

go 1.23

replace github.com/yourusername/goTap => ../..

require (
	github.com/yourusername/goTap v0.0.0
	golang.org/x/crypto v0.31.0
	gorm.io/driver/postgres v1.5.11
	gorm.io/gorm v1.25.12
)
