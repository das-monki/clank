# Clank Development Guide

## Project Structure

```
clank/
├── cmd/clank/
│   ├── main.go          # CLI + server entry point
│   └── web/             # Frontend (embedded in binary)
│       ├── index.html
│       ├── style.css
│       └── app.js
├── internal/
│   ├── api/api.go       # REST API handlers
│   ├── db/db.go         # SQLite database setup
│   ├── models/models.go # Data structures
│   └── store/store.go   # Database operations
├── flake.nix            # Nix build configuration
└── module.nix           # NixOS service module
```

## Development

### Quick Start

```bash
# Enter dev shell (provides go, gopls, sqlite)
nix develop

# Build and run in dev mode (hot reload for frontend)
go build -o clank ./cmd/clank
./clank serve --dev --port 8080
```

### Dev Mode

Use `--dev` flag to serve frontend files from disk instead of embedded:
- Frontend changes (CSS/JS/HTML) are instant - just refresh browser
- Go changes require rebuild: `go build -o clank ./cmd/clank`

### Production Build

```bash
nix build
./result/bin/clank serve --port 8080
```

## Code Conventions

### Go

- Use `chi` router for HTTP routing
- Use `modernc.org/sqlite` (pure Go, no CGO)
- API handlers in `internal/api/`
- Database operations in `internal/store/`
- Models are simple structs with JSON tags

### Frontend

- Vanilla JS, no framework
- CSS variables for theming (defined in `:root`)
- SortableJS for drag-and-drop
- marked.js for markdown preview

### API Routes

All API routes are prefixed with `/api`:
- `GET/POST /api/projects`
- `GET/DELETE /api/projects/:id`
- `GET/POST /api/tasks`
- `GET/PATCH/DELETE /api/tasks/:id`
- `POST /api/tasks/:id/move`

## Database

SQLite database stored in `clank.db` (configurable with `--db` flag).

Tables:
- `projects`: id, name, description, created_at
- `tasks`: id, project_id, parent_id, title, description, spec, status, position, created_at

## Testing

### CLI Testing

```bash
export CLANK_API=http://localhost:8080
./clank project list
./clank add "Task" -p <project-id>
./clank list
```

### Verify in SQLite

```bash
sqlite3 clank.db "SELECT * FROM tasks;"
```

## Formatting & Linting

Run before committing:
```bash
go fmt ./...
go vet ./...
```
