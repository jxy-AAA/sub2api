#!/bin/bash
# =============================================================================
# Sub2API Docker Deployment Preparation Script
# =============================================================================
# This script prepares deployment files from the local release/repo bundle:
#   - Copies docker-compose.local.yml and .env.example into the current directory
#   - Pins SUB2API_IMAGE_REF to the bundled release version when available
#   - Generates secure secrets into a 0600 .env file
#   - Creates the application data directory
# =============================================================================

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMPOSE_TEMPLATE="${SCRIPT_DIR}/docker-compose.local.yml"
ENV_TEMPLATE="${SCRIPT_DIR}/.env.example"
BUNDLE_BINARY="${SCRIPT_DIR}/../sub2api"

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

generate_secret() {
    openssl rand -hex 32
}

require_template() {
    if [ ! -f "$1" ]; then
        print_error "Required template not found: $1"
        exit 1
    fi
}

set_env_value() {
    local key="$1"
    local value="$2"

    if sed --version >/dev/null 2>&1; then
        sed -i "s#^${key}=.*#${key}=${value}#" .env
    else
        sed -i '' "s#^${key}=.*#${key}=${value}#" .env
    fi
}

detect_image_ref() {
    if [ -x "$BUNDLE_BINARY" ]; then
        local detected_version
        detected_version=$("$BUNDLE_BINARY" --version 2>/dev/null | grep -oE 'v?[0-9]+\.[0-9]+\.[0-9]+' | head -1 || true)
        if [ -n "$detected_version" ]; then
            if [[ "$detected_version" != v* ]]; then
                detected_version="v${detected_version}"
            fi
            echo "weishaw/sub2api:${detected_version}"
            return 0
        fi
    fi

    if [ -n "${SUB2API_IMAGE_REF:-}" ]; then
        echo "${SUB2API_IMAGE_REF}"
        return 0
    fi

    echo "weishaw/sub2api:vX.Y.Z"
}

main() {
    echo ""
    echo "=========================================="
    echo "  Sub2API Deployment Preparation"
    echo "=========================================="
    echo ""

    require_template "$COMPOSE_TEMPLATE"
    require_template "$ENV_TEMPLATE"

    if ! command_exists openssl; then
        print_error "openssl is not installed. Please install openssl first."
        exit 1
    fi

    if [ -e "docker-compose.yml" ] || [ -e ".env" ] || [ -e ".env.example" ]; then
        print_warning "Deployment files already exist in current directory."
        read -p "Overwrite existing files? (y/N): " -r
        echo
        if [[ ! ${REPLY:-} =~ ^[Yy]$ ]]; then
            print_info "Cancelled."
            exit 0
        fi
    fi

    print_info "Copying bundled templates..."
    cp "$COMPOSE_TEMPLATE" docker-compose.yml
    cp "$ENV_TEMPLATE" .env.example
    cp "$ENV_TEMPLATE" .env
    chmod 600 .env
    print_success "Copied docker-compose.yml, .env.example, and .env"

    print_info "Generating deployment secrets..."
    local postgres_password
    local redis_password
    local admin_password
    local jwt_secret
    local totp_encryption_key
    local image_ref

    postgres_password="$(generate_secret)"
    redis_password="$(generate_secret)"
    admin_password="$(generate_secret)"
    jwt_secret="$(generate_secret)"
    totp_encryption_key="$(generate_secret)"
    image_ref="$(detect_image_ref)"

    set_env_value "SUB2API_IMAGE_REF" "$image_ref"
    set_env_value "POSTGRES_PASSWORD" "$postgres_password"
    set_env_value "DATABASE_PASSWORD" "$postgres_password"
    set_env_value "REDIS_PASSWORD" "$redis_password"
    set_env_value "ADMIN_PASSWORD" "$admin_password"
    set_env_value "JWT_SECRET" "$jwt_secret"
    set_env_value "TOTP_ENCRYPTION_KEY" "$totp_encryption_key"
    print_success "Wrote generated secrets to .env (mode 0600)"

    print_info "Creating application data directory..."
    mkdir -p data
    print_success "Created data/"

    echo ""
    echo "=========================================="
    echo "  Preparation Complete"
    echo "=========================================="
    echo ""
    echo "Generated and stored in .env (not printed):"
    echo "  - SUB2API_IMAGE_REF"
    echo "  - POSTGRES_PASSWORD / DATABASE_PASSWORD"
    echo "  - REDIS_PASSWORD"
    echo "  - ADMIN_PASSWORD"
    echo "  - JWT_SECRET"
    echo "  - TOTP_ENCRYPTION_KEY"
    echo ""
    print_info "Default bind address remains 127.0.0.1. Put Caddy/Nginx/TLS in front before exposing the service publicly."
    print_info "PostgreSQL and Redis now live in named volumes, so they stay out of release and migration bundles."
    echo ""
    echo "Next steps:"
    echo "  1. Review .env and adjust non-secret settings if needed"
    echo "  2. Start services: docker compose up -d"
    echo "  3. View logs:      docker compose logs -f sub2api"
    echo "  4. Access UI:      http://127.0.0.1:${SERVER_PORT:-8080}"
    echo ""
}

main "$@"
