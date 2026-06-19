# DSR Automation

Go API for automating Daily Status Reports (DSRs). It aggregates developer activity from remote Git (GitHub/GitLab), local Git repositories, and planning tools (Jira, Trello), then uses AI to generate structured DSR drafts.

## Features

- **Authentication** — user registration and JWT-based login
- **Remote Git integrations** — connect GitHub or GitLab, sync projects, and pull commits from tracked repos
- **Local Git tracking** — register local repositories and scan commits on disk
- **Planning tool integrations** — connect Jira or Trello to capture task activity (assignments, status changes, comments)
- **Activity aggregation** — combine remote commits, local commits, planning events, and active repository context
- **AI DSR generation** — generate yesterday/today/blockers summaries from activity using OpenAI
- **Background sync** — periodic commit sync and activity sync workers

## Prerequisites

- Go 1.26+
- PostgreSQL

## Configuration

Create a `.env` file in the project root (or export the variables directly):

```bash
# Required
DATABASE_URL=postgres://postgres:postgres@localhost:5432/dsr_automation?sslmode=disable
JWT_SECRET=your-super-secret-key
PORT=3000

# Optional — required for AI DSR generation
OPENAI_API_KEY=sk-...
OPENAI_MODEL=gpt-4o-mini

# Optional — background sync (defaults shown)
COMMIT_SYNC_INTERVAL_MINUTES=15
COMMIT_SYNC_LOOKBACK_DAYS=30
ACTIVITY_SYNC_CRON_SPEC=*/15 * * * *
ACTIVITY_SYNC_LOOKBACK_DAYS=30
```

## Run

```bash
go run ./cmd/server
```

Health check:

```bash
curl http://localhost:3000/api/health
```

## Project structure

```
cmd/server/main.go          # Application entry point
internal/
  database/                 # PostgreSQL connection and auto-migrations
  dto/                      # Request/response types
  handlers/                 # HTTP handlers
  middleware/               # Fiber middleware (logger, recover, JWT)
  models/                   # GORM models
  repository/               # Data access layer
  routes/                   # Route registration
  services/                 # Business logic
  worker/                   # Background sync workers
pkg/
  activity/                 # Activity summary types
  ai/                       # OpenAI summarizer
  config/                   # Environment configuration
  github/                   # GitHub API client
  gitlab/                   # GitLab API client
  gitactivity/              # Commit helpers
  jira/                     # Jira API client
  localgit/                 # Local repository scanner
  planning/                 # Planning tool provider registry (Jira, Trello)
  jwt/                      # JWT token service
  utils/                    # Shared utilities
```

## API overview

All endpoints except `/api/health`, `/api/auth/register`, and `/api/auth/login` require a JWT bearer token:

```
Authorization: Bearer <access_token>
```

### Auth

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/register` | Register a new user |
| POST | `/api/auth/login` | Login and receive access/refresh tokens |

**Register**

```bash
curl -X POST http://localhost:3000/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"name":"John Doe","email":"john@example.com","password":"secret12"}'
```

**Login**

```bash
curl -X POST http://localhost:3000/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"john@example.com","password":"secret12"}'
```

### Git integrations

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/integrations/git` | Connect GitHub or GitLab |
| GET | `/api/integrations/git` | List connected integrations |
| POST | `/api/integrations/git/sync` | Sync projects from a provider |
| POST | `/api/integrations/git/commits/sync` | Sync commits for tracked projects |
| GET | `/api/integrations/git/projects` | List synced projects |
| PATCH | `/api/integrations/git/projects/tracking` | Update which projects are tracked |

**Connect GitLab**

```bash
curl -X POST http://localhost:3000/api/integrations/git \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"gitlab","base_url":"https://gitlab.com","access_token":"glpat-..."}'
```

Supported `provider` values: `github`, `gitlab`.

### Activity

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/activity` | Get activity summary for a date (`?report_date=YYYY-MM-DD`) |
| POST | `/api/activity/sync` | Manually trigger activity sync |
| PUT | `/api/activity/active-repository` | Set the currently active repository |
| GET | `/api/activity/active-repository` | Get the active repository |
| POST | `/api/activity/local-git/repositories` | Register a local Git repo |
| GET | `/api/activity/local-git/repositories` | List local repos (`?tracked_only=true`) |
| PATCH | `/api/activity/local-git/repositories/tracking` | Update tracked local repos |
| DELETE | `/api/activity/local-git/repositories/:id` | Remove a local repo |
| GET | `/api/activity/planning/providers` | List supported planning providers |
| POST | `/api/activity/planning/connect` | Connect a planning tool |
| GET | `/api/activity/planning` | List planning integrations |
| GET | `/api/activity/planning/integration` | Get planning integration (`?provider=jira`) |
| DELETE | `/api/activity/planning` | Disconnect a planning tool (`?provider=jira`) |

**Connect Jira**

```bash
curl -X POST http://localhost:3000/api/activity/planning/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider":"jira","base_url":"https://your-org.atlassian.net","email":"you@example.com","api_token":"..."}'
```

Supported planning `provider` values: `jira`, `trello`.

### DSR

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/dsr/activity` | Preview activity for DSR generation |
| POST | `/api/dsr/generate` | Generate an AI DSR report |
| GET | `/api/dsr` | List saved reports (`?from=&to=&limit=&offset=`) |
| GET | `/api/dsr/:id` | Get a single report |

**Preview activity**

```bash
curl "http://localhost:3000/api/dsr/activity?report_date=2026-06-19" \
  -H "Authorization: Bearer $TOKEN"
```

**Generate DSR**

```bash
curl -X POST http://localhost:3000/api/dsr/generate \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"report_date":"2026-06-19","git_project_ids":["optional-project-uuid"]}'
```

If `git_project_ids` is omitted, activity from all tracked remote Git projects is included.

## Typical workflow

1. Register and log in to obtain a JWT.
2. Connect GitHub or GitLab and mark projects as tracked.
3. Register local Git repositories and connect Jira/Trello if needed.
4. Let background workers sync commits and planning activity, or call `/api/activity/sync` manually.
5. Preview activity with `GET /api/dsr/activity?report_date=...`.
6. Generate a DSR with `POST /api/dsr/generate`.

## Background workers

On startup, the server runs two background workers:

- **Commit sync** — pulls commits from tracked remote Git projects on a fixed interval (`COMMIT_SYNC_INTERVAL_MINUTES`, default 15 min).
- **Activity sync** — scans local Git repos and syncs planning tool activity on a cron schedule (`ACTIVITY_SYNC_CRON_SPEC`, default every 15 min).

Both workers respect configurable lookback windows (`COMMIT_SYNC_LOOKBACK_DAYS`, `ACTIVITY_SYNC_LOOKBACK_DAYS`).
