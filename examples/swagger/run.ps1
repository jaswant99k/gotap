# goTap Swagger Example - Quick Start

Write-Host "🚀 goTap Swagger Example Setup" -ForegroundColor Cyan
Write-Host ""

# Check if swag is installed
$swagPath = Get-Command swag -ErrorAction SilentlyContinue
if (-not $swagPath) {
    Write-Host "📦 Installing swag CLI tool..." -ForegroundColor Yellow
    go install github.com/swaggo/swag/cmd/swag@latest
    
    # Add Go bin to PATH for current session
    $goBin = Join-Path $env:GOPATH "bin"
    if (-not $env:GOPATH) {
        $goBin = Join-Path $env:USERPROFILE "go\bin"
    }
    $env:PATH = "$goBin;$env:PATH"
}

Write-Host "✅ swag CLI tool ready" -ForegroundColor Green

# Install dependencies
Write-Host ""
Write-Host "📦 Installing dependencies..." -ForegroundColor Yellow
go mod tidy

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to install dependencies" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Dependencies installed" -ForegroundColor Green

# Generate Swagger documentation
Write-Host ""
Write-Host "📝 Generating Swagger documentation..." -ForegroundColor Yellow
swag init -g main.go --output docs

if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Failed to generate Swagger docs" -ForegroundColor Red
    exit 1
}

Write-Host "✅ Swagger documentation generated" -ForegroundColor Green

# Run the application
Write-Host ""
Write-Host "╔════════════════════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "║           🎉 Setup Complete!                               ║" -ForegroundColor Cyan
Write-Host "╚════════════════════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "📋 Generated files:" -ForegroundColor White
Write-Host "   - docs/docs.go" -ForegroundColor Gray
Write-Host "   - docs/swagger.json" -ForegroundColor Gray
Write-Host "   - docs/swagger.yaml" -ForegroundColor Gray
Write-Host ""
Write-Host "🚀 Starting server..." -ForegroundColor Yellow
Write-Host ""
Write-Host "Server will be available at:" -ForegroundColor White
Write-Host "   🌐 API: http://localhost:8080" -ForegroundColor Cyan
Write-Host "   📚 Swagger UI: http://localhost:8080/swagger/index.html" -ForegroundColor Cyan
Write-Host "   🔑 Default admin: admin@example.com / admin123" -ForegroundColor Green
Write-Host ""
Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
Write-Host ""

# Run the application
go run main.go
