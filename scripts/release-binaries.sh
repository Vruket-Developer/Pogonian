#!/usr/bin/env bash
set -euo pipefail

# Pogonian Multi-Platform Cross-Compilation Script
# Generates stripped release binaries for all major OS and CPU architectures:
# - Linux (amd64, arm64)
# - macOS (Intel amd64, Apple Silicon arm64)
# - Windows (amd64, arm64)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist/binaries"
VERSION="0.2.0"

echo "🚀 Building Pogonian v${VERSION} release binaries across all platforms..."
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

TARGETS=(
    "linux/amd64/pogn-linux-amd64"
    "linux/arm64/pogn-linux-arm64"
    "darwin/amd64/pogn-darwin-amd64"
    "darwin/arm64/pogn-darwin-arm64"
    "windows/amd64/pogn-windows-amd64.exe"
    "windows/arm64/pogn-windows-arm64.exe"
)

for target in "${TARGETS[@]}"; do
    IFS="/" read -r GOOS GOARCH OUT_FILE <<< "${target}"
    OUTPUT_PATH="${DIST_DIR}/${OUT_FILE}"

    echo "  🔨 Compiling for ${GOOS}/${GOARCH} -> ${OUT_FILE}..."
    CGO_ENABLED=0 GOOS="${GOOS}" GOARCH="${GOARCH}" go build \
        -ldflags="-s -w" \
        -o "${OUTPUT_PATH}" \
        "${ROOT_DIR}/cmd/pogn"

    chmod +x "${OUTPUT_PATH}"
done

echo ""
echo "🔐 Generating SHA256 Checksums..."
cd "${DIST_DIR}"
sha256sum pogn-* > SHA256SUMS.txt
cat SHA256SUMS.txt

echo ""
echo "✨ All binaries successfully compiled to ${DIST_DIR}:"
ls -lh "${DIST_DIR}"
