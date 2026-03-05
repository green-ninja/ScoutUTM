#!/bin/sh
set -e

GO="/usr/local/go/bin/go"
PROJECT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$PROJECT_DIR/build"
NO_TESTS=false

for arg in "$@"; do
  case "$arg" in
    --no-tests) NO_TESTS=true ;;
  esac
done

echo "=== ScoutUTM Build ==="
echo ""

# 1. Build frontend.
echo "[1/5] Building frontend..."
cd "$PROJECT_DIR/web"
npm run build --silent
echo "  OK: scoutd/web/dist/"

# 2. Build scoutd binary.
echo "[2/5] Building scoutd..."
cd "$PROJECT_DIR/scoutd"
CGO_ENABLED=0 "$GO" build -ldflags "-X main.version=0.1.0-dev" -o "$BUILD_DIR/scoutd" ./cmd/scoutd/
echo "  OK: build/scoutd"

# 3. Copy templates into build directory.
echo "[3/5] Copying templates..."
mkdir -p "$BUILD_DIR/templates"
cp "$PROJECT_DIR/scoutd/templates/"*.tmpl "$BUILD_DIR/templates/"
echo "  OK: build/templates/ ($(ls "$BUILD_DIR/templates/"*.tmpl | wc -l) files)"

# 4. Destroy existing VM.
echo "[4/5] Destroying existing VM..."
cd "$PROJECT_DIR"
vagrant destroy -f 2>/dev/null || true

# 5. Create and provision VM.
if [ "$NO_TESTS" = true ]; then
  echo "[5/5] Starting VM (--no-tests: skipping integration tests)..."
  SCOUT_NO_TESTS=1 vagrant up
else
  echo "[5/5] Starting VM (provision + tests)..."
  vagrant up
fi

echo ""
echo "=== Done ==="
echo "SSH:     vagrant ssh utm      (UTM appliance)"
echo "SSH:     vagrant ssh client   (LAN client)"
echo "Web UI:  https://localhost:8443"
echo "Login:   admin / admin"
