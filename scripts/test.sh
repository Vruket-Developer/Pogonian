#!/usr/bin/env bash
set -e

# Run all Pogonian unit and integration tests
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "🧪 Running all Pogonian tests..."
cd "$ROOT_DIR"
go test -v -race ./...
