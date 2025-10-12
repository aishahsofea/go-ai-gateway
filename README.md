## Running the server

1. Ensure you have Docker and Docker Compose installed.
2. Start the services using Docker Compose:

   ```bash
   docker-compose up -d
   ```

3. Do health checks:

   ```bash
   curl http://localhost:8080/health
   ```

### How to access the database

```bash
docker compose exec postgres psql -h localhost -p 5432 -U gateway_user -d gateway
```

## Running Tests

### Gateway Tests

Make the test script executable:

```bash
chmod +x scripts/run-tests.sh
```

Run the tests:

```bash
./scripts/run-tests.sh
```

## Troubleshooting

### Docker Issues

#### Port Conflicts

If you see "port is already allocated":

```bash
# Stop existing PostgreSQL services
sudo service postgresql stop

# Or kill existing containers
docker compose down
docker container prune

Database Volume Issues

If you see "directory exists but is not empty":
# Complete reset (WARNING: deletes all data)
docker compose down -v
docker system prune -f
rm -rf ./database/postgres-data
rm -rf ./database/postgres-test-data
docker compose up --build

Test Database Connection

# Start only test database
docker compose up test_db -d

# Verify connection
psql "postgres://gateway_user:gateway_password@localhost:5433/gateway"

Running Tests

# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/api -v
```
