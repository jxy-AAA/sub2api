#!/usr/bin/env bash

set -euo pipefail

BIN_PATH=""
SOURCE_PATH=""
INSTALL_DIR="/opt/sub2api"
DATA_DIR="/var/lib/sub2api/datamanagement"
SERVICE_FILE_NAME="sub2api-datamanagementd.service"

print_help() {
  cat <<'EOF'
Usage:
  install-datamanagementd.sh --binary <path-to-datamanagementd>

Options:
  --binary  Path to an existing datamanagementd binary.
  --source  Deprecated in this repository (kept for backward compatibility).
  -h, --help  Show this help.

Examples:
  sudo ./install-datamanagementd.sh --binary /tmp/datamanagementd
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary)
      BIN_PATH="${2:-}"
      shift 2
      ;;
    --source)
      SOURCE_PATH="${2:-}"
      shift 2
      ;;
    -h|--help)
      print_help
      exit 0
      ;;
    *)
      echo "Unknown argument: $1"
      print_help
      exit 1
      ;;
  esac
done

if [[ -n "$SOURCE_PATH" ]]; then
  echo "Error: --source mode is no longer supported in this repository."
  echo "Please provide a prebuilt binary via --binary."
  exit 1
fi

if [[ -z "$BIN_PATH" ]]; then
  echo "Error: --binary is required."
  print_help
  exit 1
fi

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Error: please run as root (e.g. with sudo)."
  exit 1
fi

if [[ ! -f "$BIN_PATH" ]]; then
  echo "Error: binary file not found: $BIN_PATH"
  exit 1
fi

if ! id sub2api >/dev/null 2>&1; then
  echo "[1/5] Creating system user sub2api..."
  useradd --system --no-create-home --shell /usr/sbin/nologin sub2api
else
  echo "[1/5] System user sub2api already exists, skipping."
fi

echo "[2/5] Installing datamanagementd binary..."
mkdir -p "$INSTALL_DIR"
install -m 0755 "$BIN_PATH" "$INSTALL_DIR/datamanagementd"

echo "[3/5] Preparing data directory..."
mkdir -p "$DATA_DIR"
chown -R sub2api:sub2api /var/lib/sub2api
chmod 0750 "$DATA_DIR"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_TEMPLATE="$SCRIPT_DIR/$SERVICE_FILE_NAME"
if [[ ! -f "$SERVICE_TEMPLATE" ]]; then
  echo "Error: service template not found: $SERVICE_TEMPLATE"
  exit 1
fi

echo "[4/5] Installing systemd service..."
cp "$SERVICE_TEMPLATE" "/etc/systemd/system/$SERVICE_FILE_NAME"
systemctl daemon-reload
systemctl enable --now sub2api-datamanagementd

echo "[5/5] Done. Current service status:"
systemctl --no-pager --full status sub2api-datamanagementd || true

cat <<'EOF'

Next steps:
1. Check logs:
   sudo journalctl -u sub2api-datamanagementd -f
2. For Docker deployments, mount the socket into sub2api:
   /tmp/sub2api-datamanagement.sock:/tmp/sub2api-datamanagement.sock
3. Verify Data Management page shows agent=enabled.

EOF
