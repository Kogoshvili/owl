#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# Owl Folder Suggester — Build Pipeline
# ──────────────────────────────────────────────
# Builds Go backend + frontend + Tauri installer
# Hardcoded project path for now.
# ──────────────────────────────────────────────

PROJECT_DIR="/mnt/c/projects/owl"

FRONTEND_DIR="$PROJECT_DIR/frontend"
TAURI_BINARIES="$FRONTEND_DIR/src-tauri/binaries"

echo "=== 1. Build Go backend sidecar ==="
mkdir -p "$TAURI_BINARIES"

# Determine target triple
if command -v rustc &>/dev/null; then
  TARGET=$(rustc -vV | grep host | cut -d' ' -f2)
else
  case "$(uname -s)" in
    Linux)  TARGET="x86_64-unknown-linux-gnu" ;;
    Darwin) TARGET="aarch64-apple-darwin" ;;
    *)      echo "Unknown platform"; exit 1 ;;
  esac
fi

cd "$PROJECT_DIR/backend"
go build -o "$TAURI_BINARIES/owl-backend-$TARGET" ./cmd/owl

echo "   Binary: $TAURI_BINARIES/owl-backend-$TARGET"

echo ""
echo "=== 2. Build Tauri installer (frontend built automatically) ==="
cd "$FRONTEND_DIR"
pnpm tauri build

echo ""
echo "=== Done ==="
echo "Installer available in: $FRONTEND_DIR/src-tauri/target/release/bundle/"
