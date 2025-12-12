# goTap Modular Authentication Example

This is the complete authentication example converted to modular structure.

## Structure

```
modular_auth_example/
├── cmd/
│   └── server/
│       └── main.go           # Application entry point
│
├── modules/
│   └── auth/
│       ├── models.go         # User, Permission models
│       ├── repository.go     # Database operations
│       ├── service.go        # Business logic
│       ├── handlers.go       # HTTP handlers
│       └── routes.go         # Route registration
│
└── shared/
    └── database/
        └── connection.go     # Database connection
```

## Files

See individual files in this directory:
- `cmd/server/main.go` - Main application
- `modules/auth/*.go` - Auth module files
- `shared/database/connection.go` - Database setup

## Usage

```bash
# Create database
createdb gotap_auth

# Run application
export DATABASE_URL="host=localhost user=postgres password=postgres dbname=gotap_auth port=5432 sslmode=disable"
export JWT_SECRET="your-super-secret-jwt-key-minimum-32-characters"
go run cmd/server/main.go
```

## Default Credentials

- Email: admin@example.com
- Password: admin123

## Benefits of Modular Structure

1. **All auth code in one place** - Easy to find and maintain
2. **Clear boundaries** - Module dependencies are explicit
3. **Reusable** - Extract as separate package easily
4. **Testable** - Test entire feature independently
5. **Scalable** - Add new modules without touching existing code

## Comparison

**Before (Single File):**
```
complete_example.go (450 lines) - Everything mixed together
```

**After (Modular):**
```
modules/auth/
├── models.go (60 lines)
├── repository.go (50 lines)
├── service.go (80 lines)
├── handlers.go (100 lines)
└── routes.go (30 lines)

cmd/server/main.go (100 lines)
```

**Result:** Better organization, easier to maintain, scales better!
