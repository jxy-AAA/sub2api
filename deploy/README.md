# Sub2API Deployment Files

This directory contains files for deploying Sub2API on Linux servers.

## Deployment Methods

| Method | Best For | Setup Wizard |
|--------|----------|--------------|
| **Docker Compose** | Quick setup, all-in-one | Not needed (auto-setup) |
| **Binary Install** | Production servers, systemd | Web-based wizard |

## Files

| File | Description |
|------|-------------|
| `docker-compose.yml` | Docker Compose configuration (named volumes) |
| `docker-compose.local.yml` | Docker Compose configuration (local directories, easy migration) |
| `docker-deploy.sh` | **One-click Docker deployment script (recommended)** |
| `.env.example` | Docker environment variables template |
| `DOCKER.md` | Docker Hub documentation |
| `install.sh` | One-click binary installation script |
| `install-datamanagementd.sh` | datamanagementd 一键安装脚本 |
| `sub2api.service` | Systemd service unit file |
| `sub2api-datamanagementd.service` | datamanagementd systemd service unit file |
| `DATAMANAGEMENTD_CN.md` | datamanagementd 部署与联动说明（中文） |
| `config.example.yaml` | Example configuration file |

---

## Docker Deployment (Recommended)

### Method 1: Verified Bundle + `docker-deploy.sh`

Use a fixed release bundle, verify it, then run the bundled preparation script locally.

```bash
VERSION=vX.Y.Z
ARCH=amd64   # or arm64
ARCHIVE="sub2api_${VERSION#v}_linux_${ARCH}.tar.gz"

curl -fsSLO "https://github.com/Wei-Shaw/sub2api/releases/download/${VERSION}/${ARCHIVE}"
curl -fsSLO "https://github.com/Wei-Shaw/sub2api/releases/download/${VERSION}/checksums.txt"
grep " ${ARCHIVE}$" checksums.txt > "${ARCHIVE}.sha256"
test -s "${ARCHIVE}.sha256"
sha256sum -c "${ARCHIVE}.sha256"
tar -xzf "${ARCHIVE}"

mkdir -p sub2api-deploy
cd sub2api-deploy
bash ../deploy/docker-deploy.sh
docker compose up -d
docker compose logs -f sub2api
```

**What the script does now:**
- Copies the bundled `docker-compose.local.yml` and `.env.example` into the current directory
- Pins `SUB2API_IMAGE_REF` to the bundle version when possible
- Generates `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `ADMIN_PASSWORD`, `JWT_SECRET`, and `TOTP_ENCRYPTION_KEY`
- Writes them only to `.env` with mode `0600`
- Creates only `./data`; PostgreSQL and Redis use named volumes

### Method 2: Manual Deployment

```bash
git clone https://github.com/Wei-Shaw/sub2api.git
cd sub2api/deploy
cp .env.example .env
chmod 600 .env
nano .env
mkdir -p data
docker compose -f docker-compose.local.yml up -d
```

**Required `.env` values:**

```bash
SUB2API_IMAGE_REF=weishaw/sub2api:vX.Y.Z
POSTGRES_PASSWORD=your_secure_postgres_password
REDIS_PASSWORD=your_secure_redis_password
ADMIN_EMAIL=admin@example.com
ADMIN_PASSWORD=your_secure_admin_password
JWT_SECRET=your_jwt_secret_here
TOTP_ENCRYPTION_KEY=your_totp_key_here
BIND_HOST=127.0.0.1
SERVER_PORT=8080
```

### Deployment Version Comparison

| Version | Data Storage | Migration Posture | Best For |
|---------|-------------|-------------------|----------|
| **docker-compose.local.yml** | `./data` + named PostgreSQL/Redis volumes | Archive only allowlisted files; export DB/Redis separately | Production, repeatable upgrades |
| **docker-compose.yml** | Named volumes for app + DB + Redis | Docker-managed volume workflow | Simple setup |

**Recommendation:** Use `docker-compose.local.yml` for production. It keeps the editable app config/logs local while preventing `postgres_data` / `redis_data` from landing in release or migration bundles.

### How Auto-Setup Works

When using Docker Compose with `AUTO_SETUP=true`:

1. On first run, the system automatically:
   - Connects to PostgreSQL and Redis
   - Applies database migrations (SQL files in `backend/migrations/*.sql`) and records them in `schema_migrations`
   - Uses the explicit `ADMIN_PASSWORD` / `JWT_SECRET` / `TOTP_ENCRYPTION_KEY` values you stored in `.env`
   - Writes config.yaml

2. No manual Setup Wizard is needed once `.env` is complete.

3. Keep `BIND_HOST=127.0.0.1` and publish the service through Caddy/Nginx/TLS before exposing it publicly.

### Database Migration Notes (PostgreSQL)

- Migrations are applied in lexicographic order (e.g. `001_...sql`, `002_...sql`).
- `schema_migrations` tracks applied migrations (filename + checksum).
- Migrations are forward-only; rollback requires a DB backup restore or a manual compensating SQL script.

**Verify `users.allowed_groups` ? `user_allowed_groups` backfill**

During the incremental GORM?Ent migration, `users.allowed_groups` (legacy `BIGINT[]`) is being replaced by a normalized join table `user_allowed_groups(user_id, group_id)`.

Run this query to compare the legacy data vs the join table:

```sql
WITH old_pairs AS (
  SELECT DISTINCT u.id AS user_id, x.group_id
  FROM users u
  CROSS JOIN LATERAL unnest(u.allowed_groups) AS x(group_id)
  WHERE u.allowed_groups IS NOT NULL
)
SELECT
  (SELECT COUNT(*) FROM old_pairs)           AS old_pair_count,
  (SELECT COUNT(*) FROM user_allowed_groups) AS new_pair_count;
```

### datamanagementd????????

????????????????????????? `datamanagementd`?
- ??????? `/tmp/sub2api-datamanagement.sock`
- Docker ???????? Socket ?????????
- ??????`deploy/DATAMANAGEMENTD_CN.md`

### Commands

For **docker-compose.local.yml**:

```bash
# Start services
docker compose -f docker-compose.local.yml up -d

# Stop services
docker compose -f docker-compose.local.yml down

# View logs
docker compose -f docker-compose.local.yml logs -f sub2api

# Restart Sub2API only
docker compose -f docker-compose.local.yml restart sub2api

# Update after editing SUB2API_IMAGE_REF in .env
docker compose -f docker-compose.local.yml pull
docker compose -f docker-compose.local.yml up -d

# Inspect named volumes before backup or cleanup
docker volume ls | grep sub2api
```

For **docker-compose.yml**:

```bash
# Start services
docker compose up -d

# Stop services
docker compose down

# View logs
docker compose logs -f sub2api

# Restart Sub2API only
docker compose restart sub2api
```

### Environment Variables

> Note: Sub2API runtime reads `DATABASE_*` and `REDIS_*` variables (not `DATABASE_URL` / `REDIS_URL`).
> In `docker-compose*.yml`, these are already wired from `.env` values.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `SUB2API_IMAGE_REF` | **Yes** | - | Pinned image tag or digest |
| `POSTGRES_PASSWORD` | **Yes** | - | PostgreSQL password |
| `REDIS_PASSWORD` | **Yes** | - | Redis password |
| `ADMIN_PASSWORD` | **Yes** | - | Initial admin password for `AUTO_SETUP=true` |
| `JWT_SECRET` | **Recommended** | unset | JWT secret (fixed for persistent sessions) |
| `TOTP_ENCRYPTION_KEY` | **Recommended** | unset | TOTP encryption key (fixed for persistent 2FA) |
| `BIND_HOST` | No | `127.0.0.1` | Host-side bind address |
| `SERVER_PORT` | No | `8080` | Host-side port |
| `TZ` | No | `Asia/Shanghai` | Timezone |

See `.env.example` for all available options.

### Migration Notes

Do **not** tar the whole deployment directory together with runtime state anymore. Archive only control-plane files such as:

- `.env`
- `docker-compose.yml` / `docker-compose.local.yml`
- `data/`

Then export PostgreSQL and Redis separately using your normal backup tooling (for example `pg_dump` and an RDB/AOF snapshot) before restoring them on the target host.

---

## Gemini OAuth Configuration

Sub2API supports three methods to connect to Gemini:

### Method 1: Code Assist OAuth (Recommended for GCP Users)

**No configuration needed** - always uses the built-in Gemini CLI OAuth client (public).

1. Leave `GEMINI_OAUTH_CLIENT_ID` and `GEMINI_OAUTH_CLIENT_SECRET` empty
2. In the Admin UI, create a Gemini OAuth account and select **"Code Assist"** type
3. Complete the OAuth flow in your browser

> Note: Even if you configure `GEMINI_OAUTH_CLIENT_ID` / `GEMINI_OAUTH_CLIENT_SECRET` for AI Studio OAuth,
> Code Assist OAuth will still use the built-in Gemini CLI client.

**Requirements:**
- Google account with access to Google Cloud Platform
- A GCP project (auto-detected or manually specified)

**How to get Project ID (if auto-detection fails):**
1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Click the project dropdown at the top of the page
3. Copy the Project ID (not the project name) from the list
4. Common formats: `my-project-123456` or `cloud-ai-companion-xxxxx`

### Method 2: AI Studio OAuth (For Regular Google Accounts)

Requires your own OAuth client credentials.

**Step 1: Create OAuth Client in Google Cloud Console**

1. Go to [Google Cloud Console - Credentials](https://console.cloud.google.com/apis/credentials)
2. Create a new project or select an existing one
3. **Enable the Generative Language API:**
   - Go to "APIs & Services" → "Library"
   - Search for "Generative Language API"
   - Click "Enable"
4. **Configure OAuth Consent Screen** (if not done):
   - Go to "APIs & Services" → "OAuth consent screen"
   - Choose "External" user type
   - Fill in app name, user support email, developer contact
   - Add scopes: `https://www.googleapis.com/auth/generative-language.retriever` (and optionally `https://www.googleapis.com/auth/cloud-platform`)
   - Add test users (your Google account email)
5. **Create OAuth 2.0 credentials:**
   - Go to "APIs & Services" → "Credentials"
   - Click "Create Credentials" → "OAuth client ID"
   - Application type: **Web application** (or **Desktop app**)
   - Name: e.g., "Sub2API Gemini"
   - Authorized redirect URIs: Add `http://localhost:1455/auth/callback`
6. Copy the **Client ID** and **Client Secret**
7. **⚠️ Publish to Production (IMPORTANT):**
   - Go to "APIs & Services" → "OAuth consent screen"
   - Click "PUBLISH APP" to move from Testing to Production
   - **Testing mode limitations:**
     - Only manually added test users can authenticate (max 100 users)
     - Refresh tokens expire after 7 days
     - Users must be re-added periodically
   - **Production mode:** Any Google user can authenticate, tokens don't expire
   - Note: For sensitive scopes, Google may require verification (demo video, privacy policy)

**Step 2: Configure Environment Variables**

```bash
GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret

# 可选：如需使用 Gemini CLI 内置 OAuth Client（Code Assist / Google One）
# 安全说明：本仓库不会内置该 client_secret，请在运行环境通过环境变量注入。
# GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
```

**Step 3: Create Account in Admin UI**

1. Create a Gemini OAuth account and select **"AI Studio"** type
2. Complete the OAuth flow
   - After consent, your browser will be redirected to `http://localhost:1455/auth/callback?code=...&state=...`
   - Copy the full callback URL (recommended) or just the `code` and paste it back into the Admin UI

### Method 3: API Key (Simplest)

1. Go to [Google AI Studio](https://aistudio.google.com/app/apikey)
2. Click "Create API key"
3. In Admin UI, create a Gemini **API Key** account
4. Paste your API key (starts with `AIza...`)

### Comparison Table

| Feature | Code Assist OAuth | AI Studio OAuth | API Key |
|---------|-------------------|-----------------|---------|
| Setup Complexity | Easy (no config) | Medium (OAuth client) | Easy |
| GCP Project Required | Yes | No | No |
| Custom OAuth Client | No (built-in) | Yes (required) | N/A |
| Rate Limits | GCP quota | Standard | Standard |
| Best For | GCP developers | Regular users needing OAuth | Quick testing |

---

## Binary Installation

For production servers using systemd.

### Verified Bundle Installation

```bash
VERSION=vX.Y.Z
ARCH=amd64   # or arm64
ARCHIVE="sub2api_${VERSION#v}_linux_${ARCH}.tar.gz"

curl -fsSLO "https://github.com/Wei-Shaw/sub2api/releases/download/${VERSION}/${ARCHIVE}"
curl -fsSLO "https://github.com/Wei-Shaw/sub2api/releases/download/${VERSION}/checksums.txt"
grep " ${ARCHIVE}$" checksums.txt > "${ARCHIVE}.sha256"
test -s "${ARCHIVE}.sha256"
sha256sum -c "${ARCHIVE}.sha256"
tar -xzf "${ARCHIVE}"
sudo ./deploy/install.sh
```

The installer copies the binary into `/opt/sub2api`, installs a hardened systemd unit, and writes server defaults to `/etc/sub2api/sub2api.env` with mode `0600`.

### Commands

```bash
# Install
sudo ./install.sh

# Upgrade
sudo ./install.sh upgrade

# Uninstall
sudo ./install.sh uninstall
```

### Service Management

```bash
# Start the service
sudo systemctl start sub2api

# Stop the service
sudo systemctl stop sub2api

# Restart the service
sudo systemctl restart sub2api

# Check status
sudo systemctl status sub2api

# View logs
sudo journalctl -u sub2api -f

# Enable auto-start on boot
sudo systemctl enable sub2api
```

### Configuration

#### Server Address and Port

During installation, you will be prompted to configure the server listen address and port. These settings are stored in `/etc/sub2api/sub2api.env`.

To change after installation:

1. Edit the environment file:
   ```bash
   sudoeditor /etc/sub2api/sub2api.env
   ```

2. Add or modify:
   ```bash
   SERVER_HOST=127.0.0.1
   SERVER_PORT=3000
   ```

3. Restart the service:
   ```bash
   sudo systemctl restart sub2api
   ```

#### Gemini OAuth Configuration

If you need to use AI Studio OAuth for Gemini accounts, add the OAuth client credentials to the systemd service file:

1. Edit the service file:
   ```bash
   sudo nano /etc/systemd/system/sub2api.service
   ```

2. Add your OAuth credentials in the `[Service]` section (after the existing `Environment=` lines):
   ```ini
   Environment=GEMINI_OAUTH_CLIENT_ID=your-client-id.apps.googleusercontent.com
   Environment=GEMINI_OAUTH_CLIENT_SECRET=GOCSPX-your-client-secret
   ```

   如需使用“内置 Gemini CLI OAuth Client”（Code Assist / Google One），还需要注入：
   ```ini
   Environment=GEMINI_CLI_OAUTH_CLIENT_SECRET=GOCSPX-your-built-in-secret
   ```

3. Reload and restart:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl restart sub2api
   ```

> **Note:** Code Assist OAuth does not require any configuration - it uses the built-in Gemini CLI client.
> See the [Gemini OAuth Configuration](#gemini-oauth-configuration) section above for detailed setup instructions.

#### Application Configuration

The main config file is at `/etc/sub2api/config.yaml` (created by Setup Wizard).

### Prerequisites

- Linux server (Ubuntu 20.04+, Debian 11+, CentOS 8+, etc.)
- PostgreSQL 14+
- Redis 6+
- systemd

### Directory Structure

```
/opt/sub2api/
├── sub2api              # Main binary
├── sub2api.backup       # Backup (after upgrade)
└── data/                # Runtime data

/etc/sub2api/
└── config.yaml          # Configuration file
```

---

## Troubleshooting

### Docker

For **local directory version**:

```bash
# Check container status
docker compose -f docker-compose.local.yml ps

# View detailed logs
docker compose -f docker-compose.local.yml logs --tail=100 sub2api

# Check database connection
docker compose -f docker-compose.local.yml exec postgres pg_isready

# Check Redis connection
docker compose -f docker-compose.local.yml exec redis redis-cli --no-auth-warning ping

# Restart all services
docker compose -f docker-compose.local.yml restart

# Check data directories
docker volume ls | grep sub2api
```

For **named volumes version**:

```bash
# Check container status
docker compose ps

# View detailed logs
docker compose logs --tail=100 sub2api

# Check database connection
docker compose exec postgres pg_isready

# Check Redis connection
docker compose exec redis redis-cli ping

# Restart all services
docker compose restart
```

### Binary Install

```bash
# Check service status
sudo systemctl status sub2api

# View recent logs
sudo journalctl -u sub2api -n 50

# Check config file
sudo cat /etc/sub2api/config.yaml

# Check PostgreSQL
sudo systemctl status postgresql

# Check Redis
sudo systemctl status redis
```

### Common Issues

1. **Port already in use**: Change `SERVER_PORT` in `.env` or systemd config
2. **Database connection failed**: Check PostgreSQL is running and credentials are correct
3. **Redis connection failed**: Check Redis is running and password is correct
4. **Permission denied**: Ensure proper file ownership for binary install

---

## TLS Fingerprint Configuration

Sub2API supports TLS fingerprint simulation to make requests appear as if they come from the official Claude CLI (Node.js client).

> **💡 Tip:** Visit **[tls.sub2api.org](https://tls.sub2api.org/)** to get TLS fingerprint information for different devices and browsers.

### Default Behavior

- Built-in `claude_cli_v2` profile simulates Node.js 20.x + OpenSSL 3.x
- JA3 Hash: `1a28e69016765d92e3b381168d68922c`
- JA4: `t13d5911h1_a33745022dd6_1f22a2ca17c4`
- Profile selection: `accountID % profileCount`

### Configuration

```yaml
gateway:
  tls_fingerprint:
    enabled: true  # Global switch
    profiles:
      # Simple profile (uses default cipher suites)
      profile_1:
        name: "Profile 1"

      # Profile with custom cipher suites (use compact array format)
      profile_2:
        name: "Profile 2"
        cipher_suites: [4866, 4867, 4865, 49199, 49195, 49200, 49196]
        curves: [29, 23, 24]
        point_formats: 0

      # Another custom profile
      profile_3:
        name: "Profile 3"
        cipher_suites: [4865, 4866, 4867, 49199, 49200]
        curves: [29, 23, 24, 25]
```

### Profile Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name (required) |
| `cipher_suites` | []uint16 | Cipher suites in decimal. Empty = default |
| `curves` | []uint16 | Elliptic curves in decimal. Empty = default |
| `point_formats` | []uint8 | EC point formats. Empty = default |

### Common Values Reference

**Cipher Suites (TLS 1.3):** `4865` (AES_128_GCM), `4866` (AES_256_GCM), `4867` (CHACHA20)

**Cipher Suites (TLS 1.2):** `49195`, `49196`, `49199`, `49200` (ECDHE variants)

**Curves:** `29` (X25519), `23` (P-256), `24` (P-384), `25` (P-521)
