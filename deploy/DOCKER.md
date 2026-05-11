# Sub2API Docker Image

Sub2API is an AI API Gateway Platform for distributing and managing AI product subscription API quotas.

## Quick Start

Use a pinned image reference plus a private env file instead of inline secrets:

```bash
install -m 600 /dev/null sub2api.env
cat > sub2api.env <<'EOF'
AUTO_SETUP=true
DATABASE_HOST=host
DATABASE_PORT=5432
DATABASE_USER=user
DATABASE_PASSWORD=change_this_secure_password
DATABASE_DBNAME=sub2api
DATABASE_SSLMODE=disable
REDIS_HOST=host
REDIS_PORT=6379
REDIS_PASSWORD=change_this_secure_redis_password
REDIS_DB=0
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=change_this_secure_admin_password
JWT_SECRET=change_this_jwt_secret
TOTP_ENCRYPTION_KEY=change_this_totp_key
EOF

chmod 600 sub2api.env

docker run -d \
  --name sub2api \
  --env-file ./sub2api.env \
  -p 127.0.0.1:8080:8080 \
  -v sub2api_data:/app/data \
  weishaw/sub2api:vX.Y.Z
```

## Docker Compose

```yaml
services:
  sub2api:
    image: ${SUB2API_IMAGE_REF:?set a pinned tag or digest}
    ports:
      - "${BIND_HOST:-127.0.0.1}:${SERVER_PORT:-8080}:8080"
    environment:
      - AUTO_SETUP=true
      - DATABASE_HOST=postgres
      - DATABASE_PORT=5432
      - DATABASE_USER=${POSTGRES_USER:-sub2api}
      - DATABASE_PASSWORD=${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      - DATABASE_DBNAME=${POSTGRES_DB:-sub2api}
      - DATABASE_SSLMODE=disable
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - REDIS_PASSWORD=${REDIS_PASSWORD:?REDIS_PASSWORD is required}
      - REDIS_DB=${REDIS_DB:-0}
      - ADMIN_EMAIL=${ADMIN_EMAIL:-admin@sub2api.local}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:?ADMIN_PASSWORD is required}
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:18-alpine
    environment:
      - POSTGRES_USER=${POSTGRES_USER:-sub2api}
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}
      - POSTGRES_DB=${POSTGRES_DB:-sub2api}
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:8-alpine
    command: ['sh', '-c', 'exec redis-server --save 60 1 --appendonly yes --appendfsync everysec --requirepass "$$REDIS_PASSWORD"']
    environment:
      - REDIS_PASSWORD=${REDIS_PASSWORD:?REDIS_PASSWORD is required}
      - REDISCLI_AUTH=${REDIS_PASSWORD:?REDIS_PASSWORD is required}
    volumes:
      - redis_data:/data

volumes:
  sub2api_data:
  postgres_data:
  redis_data:
```

## Security Defaults

- Keep `BIND_HOST=127.0.0.1` unless a reverse proxy or firewall policy is already in place.
- Use `SUB2API_IMAGE_REF` with a fixed tag or digest; avoid `:latest` in production.
- Keep secrets in a `0600` env file; do not put passwords directly in shell history or systemd units.
- Keep PostgreSQL and Redis in named volumes or external storage, not inside release or migration archives.

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `SUB2API_IMAGE_REF` | Pinned image tag or digest | Yes | - |
| `POSTGRES_PASSWORD` | PostgreSQL password | Yes | - |
| `REDIS_PASSWORD` | Redis password | Yes | - |
| `ADMIN_PASSWORD` | Initial admin password for `AUTO_SETUP=true` | Yes | - |
| `JWT_SECRET` | Stable JWT secret | Recommended | unset |
| `TOTP_ENCRYPTION_KEY` | Stable TOTP encryption key | Recommended | unset |
| `BIND_HOST` | Host-side bind address | No | `127.0.0.1` |
| `SERVER_PORT` | Host-side port | No | `8080` |

## Supported Architectures

- `linux/amd64`
- `linux/arm64`

## Tags

Use a fixed `x.y.z` tag or an image digest. Example:

- `weishaw/sub2api:v1.2.3`
- `weishaw/sub2api@sha256:...`

## Links

- [GitHub Repository](https://github.com/weishaw/sub2api)
- [Deployment Guide](https://github.com/Wei-Shaw/sub2api/blob/main/deploy/README.md)
