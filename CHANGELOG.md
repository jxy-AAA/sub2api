# Changelog

All notable changes to this repository are documented in this file.

The format is based on Keep a Changelog, and this project follows a rolling
`Unreleased` section between tagged releases.

## Unreleased

### Changed

- Hardened Redis container startup in Compose by rendering `requirepass` into a
  temporary `redis.conf` instead of exposing it in the `redis-server` command line.
- Rebuilt `deploy/config.example.yaml` so it is valid YAML again and covers the
  current `setDefaults()` configuration surface.
- Switched root `make test-frontend` and CI to run lint, typecheck, and the full
  frontend Vitest suite.
- Refreshed `DEV_GUIDE.md` to match the main repository, current tree layout, and
  actual validation commands.

### Added

- Added baseline unit tests for `backend/internal/model`.
- Added focused feature docs for channel monitoring, moderation/risk control,
  subscriptions, auth providers, TOTP, TLS fingerprints, gateway modes, and
  debug environment variables.
