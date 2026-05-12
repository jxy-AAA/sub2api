# Developer Guide

This guide replaces the old fork-specific notes and documents the current
`Wei-Shaw/sub2api` repository layout, local workflow, and validation targets.

## Project Facts

| Item | Value |
|------|-------|
| Primary repository | `Wei-Shaw/sub2api` |
| Backend toolchain | Go `1.26.3` |
| Frontend toolchain | Node.js `20+`, `pnpm` |
| Datastores | PostgreSQL `15+`, Redis `7+` |
| Embedded frontend build | `backend/internal/web/dist/` |

## Repository Map

```text
sub2api/
├── backend/
│   ├── cmd/
│   │   ├── server/
│   │   └── jwtgen/
│   ├── ent/
│   │   └── schema/
│   ├── internal/
│   │   ├── config/
│   │   ├── handler/
│   │   ├── middleware/
│   │   ├── model/
│   │   ├── repository/
│   │   ├── server/
│   │   ├── service/
│   │   └── web/
│   ├── migrations/
│   └── Makefile
├── frontend/
│   ├── src/
│   ├── package.json
│   └── pnpm-lock.yaml
├── deploy/
├── docs/
├── CLA.md
└── .claude/settings.local.json
```

## Local Setup

### Backend

```bash
cd backend
go test -tags=unit ./...
go test -tags=integration ./...
golangci-lint run ./...
```

### Frontend

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run lint:check
pnpm run typecheck
pnpm exec vitest run
```

### Root Shortcuts

```bash
make test-backend
make test-frontend
make test-frontend-critical
make test-frontend-affiliate-auth-admin
make test-frontend-quality
```

`make test-frontend` now runs lint, typecheck, and the full Vitest suite.
Use the narrower frontend targets only for faster local iteration.

## Common Tasks

### Rebuild embedded frontend

```bash
cd frontend
pnpm install
pnpm run build

cd ../backend
go build -tags embed -o sub2api ./cmd/server
```

### Regenerate Ent code after schema changes

```bash
cd backend
go generate ./ent
```

### Run the server locally

```bash
cd backend
go run ./cmd/server
```

## Documentation Pointers

- General docs index: `docs/README.md`
- Deployment examples: `deploy/config.example.yaml`
- Payment system: `docs/PAYMENT.md`
- Changelog: `CHANGELOG.md`

## Guardrails

- Treat `deploy/.env` and `/etc/sub2api/sub2api.env` as secret stores.
- Keep `SUB2API_IMAGE_REF` pinned; do not switch deployment docs back to `:latest`.
- Keep Docker bind defaults on `127.0.0.1`; publish externally through Caddy or another TLS reverse proxy.
- Do not archive PostgreSQL / Redis state together with the control-plane files; `docker-compose.local.yml` keeps them in named volumes on purpose.
- Prefer `pnpm`, never `npm`, for frontend dependency and lockfile operations.
