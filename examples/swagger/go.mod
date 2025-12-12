module github.com/yourusername/goTap/examples/swagger

go 1.23

replace github.com/yourusername/goTap => ../..

require (
	github.com/swaggo/files v1.0.1
	github.com/swaggo/gin-swagger v1.6.1
	github.com/yourusername/goTap v0.0.0
	golang.org/x/crypto v0.31.0
	gorm.io/driver/sqlite v1.5.7
	gorm.io/gorm v1.25.12
)
