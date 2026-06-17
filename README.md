# DSR Automation

Go API with layered architecture, user registration, and login.

## Run

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/dsr_automation?sslmode=disable
export JWT_SECRET=your-super-secret-key
export PORT=3000
go run ./cmd/server
```

## Project structure

```
cmd/server/main.go
internal/
  database/       # DB connect + migrations + FK
  dto/            # Request/response types
  handlers/       # HTTP handlers
  middleware/     # Fiber middleware
  models/         # User, DSRReport (FK linked)
  repository/     # Data access
  routes/         # Route registration
  services/       # Business logic
pkg/
  config/         # Env config
  jwt/            # JWT token service
  utils/passwordhashing/
```

## Register API

```bash
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","password":"secret12"}'
```

Response `201`:

```json
{
  "id": "uuid",
  "name": "John Doe",
  "email": "john@example.com"
}
```

## Login API

```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"secret12"}'
```

Response `200`:

```json
{
  "id": "uuid",
  "name": "John Doe",
  "email": "john@example.com",
  "access_token": "jwt_access_token",
  "refresh_token": "jwt_refresh_token"
}
```
