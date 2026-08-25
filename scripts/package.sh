#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION="0.2.0"
DIST_DIR="${ROOT_DIR}/dist"
RELEASE_NAME="pogonian-v${VERSION}"
ARCHIVE_PATH="${DIST_DIR}/${RELEASE_NAME}.tar.gz"

echo "📦 Packaging Pogonian Release v${VERSION}..."
mkdir -p "${DIST_DIR}"

# Build clean release archive excluding git history, build caches, and test artifacts
tar \
    --exclude='.git' \
    --exclude='.git/*' \
    --exclude='bin/pogn' \
    --exclude='bin/pogn.exe' \
    --exclude='.test-work' \
    --exclude='.test-work/*' \
    --exclude='.pogonian/build' \
    --exclude='.pogonian/build/*' \
    --exclude='dist' \
    --exclude='dist/*' \
    --exclude='*.db' \
    --exclude='*.db-wal' \
    --exclude='*.db-shm' \
    --exclude='.DS_Store' \
    --exclude='tmp' \
    --exclude='/tmp' \
    -czf "${ARCHIVE_PATH}" \
    -C "$(dirname "${ROOT_DIR}")" "$(basename "${ROOT_DIR}")"

ARCHIVE_SIZE=$(du -h "${ARCHIVE_PATH}" | cut -f1)

echo "✨ Release archive successfully created!"
echo "   • File: ${ARCHIVE_PATH}"
echo "   • Size: ${ARCHIVE_SIZE}"
echo "   • Excluded: .git, bin/, .test-work/, .pogonian/build/, dist/"
