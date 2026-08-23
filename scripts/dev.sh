#!/usr/bin/env bash
set -e

# Pogonian Development Loop Script
# 1. Build CLI from source
# 2. Run test app creation in disposable directory
# 3. Verify files and sanity
# 4. Clean up disposable workspace

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="$ROOT_DIR/bin"
TEST_WORK_DIR="$ROOT_DIR/.test-work"
SMOKE_APP_NAME="smoke-app"
SMOKE_APP_DIR="$TEST_WORK_DIR/$SMOKE_APP_NAME"

echo "🔨 Building Pogonian CLI binary..."
mkdir -p "$BIN_DIR"
go build -o "$BIN_DIR/pogn" "$ROOT_DIR/cmd/pogn"
echo "✓ Binary built: $BIN_DIR/pogn"

echo "🧪 Creating disposable test project at $SMOKE_APP_DIR..."
rm -rf "$SMOKE_APP_DIR"
mkdir -p "$TEST_WORK_DIR"

"$BIN_DIR/pogn" create "$SMOKE_APP_NAME" "$SMOKE_APP_DIR"

echo "🔍 Validating project files..."
test -f "$SMOKE_APP_DIR/pogonian.yaml" || (echo "❌ pogonian.yaml missing" && exit 1)
test -f "$SMOKE_APP_DIR/pogonian.lock" || (echo "❌ pogonian.lock missing" && exit 1)
test -f "$SMOKE_APP_DIR/app/page.html" || (echo "❌ app/page.html missing" && exit 1)
test -f "$SMOKE_APP_DIR/app/layout.html" || (echo "❌ app/layout.html missing" && exit 1)
test -f "$SMOKE_APP_DIR/.pogonian/architecture/app.ai" || (echo "❌ app.ai missing" && exit 1)
test -f "$SMOKE_APP_DIR/globals.css" || (echo "❌ globals.css missing" && exit 1)
test -f "$SMOKE_APP_DIR/server/routes.go" || (echo "❌ server/routes.go missing" && exit 1)

echo "✓ All files verified!"

echo "🧹 Cleaning up disposable test project..."
rm -rf "$SMOKE_APP_DIR"
echo "✨ Dev cycle passed with 0 pollution!"
