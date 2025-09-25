### Running the server

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
