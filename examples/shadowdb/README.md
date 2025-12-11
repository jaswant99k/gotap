# Shadow DB Example

This example demonstrates the Shadow Database (Dual-DB) system in goTap, which provides high-availability database access with automatic failover.

## Features Demonstrated

- **Dual Database Configuration**: Primary and Shadow databases
- **Auto Failover**: Automatic switch to shadow when primary fails
- **Auto Failback**: Automatic return to primary when it recovers
- **Health Monitoring**: Periodic health checks on both databases
- **Read Strategies**: Multiple read distribution strategies
- **Write Strategies**: Flexible write handling (primary-only, shadow-only, both)
- **Manual Control**: Manual failover/failback endpoints
- **Transaction Support**: ACID transactions with dual-write capability

## Prerequisites

```bash
go get github.com/mattn/go-sqlite3
```

## Running the Example

```bash
cd examples/shadowdb
go run main.go
```

## API Endpoints

### Health & Status

```bash
# Get system health
curl http://localhost:5066/health

# Get detailed database status
curl http://localhost:5066/db/status
```

### Manual Failover Control

```bash
# Trigger manual failover to shadow
curl -X POST http://localhost:5066/db/failover

# Trigger manual failback to primary
curl -X POST http://localhost:5066/db/failback
```

### CRUD Operations

```bash
# Create transaction (write operation)
curl -X POST http://localhost:5066/transactions \
  -d "amount=99.99" \
  -d "description=Test transaction"

# List transactions (read operation)
curl http://localhost:5066/transactions

# Get single transaction
curl http://localhost:5066/transactions/1

# Create batch transactions
curl -X POST http://localhost:5066/transactions/batch
```

## Configuration Options

### Read Strategies

- `primary-only`: Always read from primary
- `shadow-only`: Always read from shadow
- `round-robin`: Alternate between primary and shadow
- `primary-first`: Try primary, fallback to shadow
- `shadow-first`: Try shadow, fallback to primary

### Write Strategies

- `primary-only`: Write only to primary (with failover to shadow)
- `shadow-only`: Write only to shadow
- `both`: Write to both databases (dual-write)

## Testing Failover

1. Start the server
2. Create some transactions
3. Stop the primary database (delete/rename `primary.db`)
4. Watch the logs for automatic failover
5. Try creating more transactions (should use shadow DB)
6. Restore the primary database
7. Watch the logs for automatic failback

## Health Check Callbacks

The example includes callbacks for:

- **OnFailover**: Called when failover occurs
- **OnFailback**: Called when failback occurs
- **OnHealthChange**: Called when database health status changes

## Production Considerations

1. Use proper database drivers (MySQL, PostgreSQL, etc.)
2. Configure connection pooling appropriately
3. Set up database replication between primary and shadow
4. Monitor health metrics
5. Implement alerting for failover events
6. Test failover scenarios regularly
7. Consider geographic distribution for shadow DB
