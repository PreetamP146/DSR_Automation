# DSR Automation

Go boilerplate with `User` and `DSRReport` models.

## Run

```bash
export DATABASE_URL=postgres://postgres:postgres@localhost:5432/dsr_automation?sslmode=disable
go run ./cmd/server
```

## Structure

```
cmd/server/main.go
internal/config/
internal/models/
```
