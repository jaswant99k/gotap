# goTap Modular Authentication Example - Quick Start

This is the modular version of the complete authentication example.

## Prerequisites

- Go 1.23+
- PostgreSQL database

## Setup

1. **Install dependencies:**
```powershell
go mod tidy
```

2. **Configure database:**

Edit connection string in `shared/database/connection.go` or set environment variable:
```powershell
$env:DATABASE_URL = "host=localhost user=postgres password=postgres dbname=gotap_auth port=5432 sslmode=disable"
```

3. **Run the application:**
```powershell
go run cmd/server/main.go
```

The server will start on http://localhost:8080

## Default Credentials

- **Email:** admin@example.com
- **Password:** admin123

## API Endpoints

### Public Endpoints

**Register User:**
```powershell
$headers = @{ "Content-Type" = "application/json" }
$body = @{
    username = "john"
    email = "john@example.com"
    password = "password123"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/register" `
    -Method POST -Headers $headers -Body $body
```

**Login:**
```powershell
$headers = @{ "Content-Type" = "application/json" }
$body = @{
    email = "admin@example.com"
    password = "admin123"
} | ConvertTo-Json

$response = Invoke-RestMethod -Uri "http://localhost:8080/api/login" `
    -Method POST -Headers $headers -Body $body

$token = $response.token
Write-Host "Token: $token"
```

### Protected Endpoints (Require Authentication)

**Get Profile:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
}

Invoke-RestMethod -Uri "http://localhost:8080/api/profile" `
    -Method GET -Headers $headers
```

**Update Profile:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}
$body = @{
    username = "john_updated"
    email = "john.new@example.com"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/profile" `
    -Method PUT -Headers $headers -Body $body
```

**Change Password:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}
$body = @{
    current_password = "password123"
    new_password = "newpassword123"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/change-password" `
    -Method POST -Headers $headers -Body $body
```

### Admin Endpoints (Require Admin Role)

**List All Users:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
}

Invoke-RestMethod -Uri "http://localhost:8080/api/admin/users" `
    -Method GET -Headers $headers
```

**Delete User:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
}

Invoke-RestMethod -Uri "http://localhost:8080/api/admin/users/2" `
    -Method DELETE -Headers $headers
```

**Assign Permissions:**
```powershell
$headers = @{ 
    "Authorization" = "Bearer $token"
    "Content-Type" = "application/json"
}
$body = @{
    permission_ids = @(1, 2, 3)
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://localhost:8080/api/admin/users/2/permissions" `
    -Method POST -Headers $headers -Body $body
```

## Project Structure

```
modular_auth/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── modules/
│   └── auth/                    # Auth module
│       ├── models.go            # User & Permission models
│       ├── repository.go        # Database operations
│       ├── service.go           # Business logic
│       ├── handlers.go          # HTTP handlers
│       └── routes.go            # Route registration
├── shared/
│   └── database/
│       └── connection.go        # Database connection
└── go.mod
```

## Benefits of Modular Structure

✅ **Clear Separation:** Each module is self-contained  
✅ **Easy Testing:** Test modules independently  
✅ **Scalability:** Add new modules without touching existing ones  
✅ **Team Collaboration:** Different teams can work on different modules  
✅ **Reusability:** Modules can be reused across projects  

## Comparison with Original

**Original (complete_example.go):**
- Single file with 450+ lines
- All code mixed together
- Hard to maintain as project grows

**Modular Structure:**
- 7 files, each focused on specific responsibility
- Clear separation of concerns
- Easy to navigate and maintain
- Repository pattern for database operations
- Service pattern for business logic
- Handler pattern for HTTP layer

## Next Steps

1. **Add More Modules:**
   - Products module
   - Orders module
   - Customers module

2. **Enhance Auth Module:**
   - Email verification
   - Password reset
   - Two-factor authentication
   - OAuth integration

3. **Add Middleware:**
   - Rate limiting
   - Request logging
   - CORS handling

4. **Testing:**
   - Unit tests for service layer
   - Integration tests for handlers
   - Repository tests with test database

## Learn More

- See `README.md` for detailed documentation
- Compare with `examples/auth/complete_example.go` (original version)
- Read `examples/modular/COMPARISON.md` for architecture comparison
