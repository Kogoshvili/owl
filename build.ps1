# ──────────────────────────────────────────────
# Owl Folder Suggester — Build Pipeline
# ──────────────────────────────────────────────
# Builds Go backend + frontend + Tauri installer
# Hardcoded project path for now.
# ──────────────────────────────────────────────

$PROJECT_DIR = "C:\projects\owl"

$FRONTEND_DIR = "$PROJECT_DIR\frontend"
$TAURI_BINARIES = "$FRONTEND_DIR\src-tauri\binaries"

Write-Host "=== 1. Build Go backend sidecar ===" -ForegroundColor Cyan
New-Item -ItemType Directory -Force -Path $TAURI_BINARIES | Out-Null

# Tauri target triple for Windows x86_64
$TARGET = "x86_64-pc-windows-msvc"

Set-Location "$PROJECT_DIR\backend"
go build -o "$TAURI_BINARIES\owl-backend-$TARGET.exe" .\cmd\owl

Write-Host "   Binary: $TAURI_BINARIES\owl-backend-$TARGET.exe"

Write-Host "`n=== 2. Build Tauri installer (frontend built automatically) ===" -ForegroundColor Cyan
Set-Location $FRONTEND_DIR
pnpm tauri build

Write-Host "`n=== Done ===" -ForegroundColor Green
Write-Host "Installer available in: $FRONTEND_DIR\src-tauri\target\release\bundle\"
