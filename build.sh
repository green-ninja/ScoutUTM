#!/bin/sh
set -e

GO="/usr/local/go/bin/go"
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$PROJECT_DIR/build"

echo "=== ScoutUTM Build ==="
echo ""

# 1. Build scoutd binary.
echo "[1/4] Building scoutd..."
cd "$PROJECT_DIR/scoutd"
CGO_ENABLED=0 "$GO" build -ldflags "-X main.version=0.1.0-dev" -o "$BUILD_DIR/scoutd" ./cmd/scoutd/
echo "  OK: build/scoutd"

# 2. Copy templates into build directory.
echo "[2/4] Copying templates..."
mkdir -p "$BUILD_DIR/templates"
cp "$PROJECT_DIR/scoutd/templates/"*.tmpl "$BUILD_DIR/templates/"
echo "  OK: build/templates/ ($(ls "$BUILD_DIR/templates/"*.tmpl | wc -l) files)"

# 3. Destroy existing VM.
echo "[3/4] Destroying existing VM..."
cd "$PROJECT_DIR"
vagrant destroy -f 2>/dev/null || true

# 4. Create and provision VM (integration tests run at the end).
echo "[4/4] Starting VM (provision + tests)..."
vagrant up

echo ""
echo "=== Done ==="
echo "SSH:     vagrant ssh"
echo "Web UI:  https://localhost:8443"
echo "Login:   admin / admin"
