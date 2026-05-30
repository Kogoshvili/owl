# Shipping Plan

## Architecture

```
┌───────────────────────────────────────────────┐
│              Tauri Window                      │
│  ┌─────────────────────────────────────────┐  │
│  │        Preact WebView (frontend)        │  │
│  └──────────────────┬──────────────────────┘  │
│                     │ HTTP :3721               │
│  ┌──────────────────▼──────────────────────┐  │
│  │      Go Backend (Tauri sidecar)         │  │
│  │  ┌─────────────────────────────────┐   │  │
│  │  │ HTTP API (127.0.0.1:3721)       │   │  │
│  │  │ Routes under /api prefix        │   │  │
│  │  │ SQLite DB + file extraction     │   │  │
│  │  └──────────────┬──────────────────┘   │  │
│  │                 │ auto-detect/spawn     │  │
│  │  ┌──────────────▼──────────────────┐   │  │
│  │  │ Ollama serve (127.0.0.1:11434)  │   │  │
│  │  │ Downloaded on first "Setup AI"  │   │  │
│  │  │ click (~200MB binary)           │   │  │
│  │  │ Model pulled on first run       │   │  │
│  │  │ (~1.1GB)                        │   │  │
│  │  └─────────────────────────────────┘   │  │
│  │                                         │  │
│  │  Data stored at <install-dir>/data/     │  │
│  │  (next to the installed .exe)           │  │
│  └─────────────────────────────────────────┘  │
└────────────────────────────────────────────────┘
```

## Status

### ✅ Implemented

- **Go backend** with `--port` and `--data-dir` CLI flags
- **API routes** mounted under `/api` prefix via `http.StripPrefix`
- **Tauri sidecar** — Go binary registered as `externalBin` in `tauri.conf.json`
- **Sidecar lifecycle** — Tauri spawns Go on startup, kills on window close
- **Frontend BASE** auto-detects Tauri vs browser (`window.__TAURI_INTERNALS__`)
- **Browse button** — native folder picker via `@tauri-apps/plugin-dialog`, with `<input webkitdirectory>` fallback
- **Build scripts** — `build.sh` (Linux/macOS) and `build.ps1` (Windows)
- **GitHub Actions release workflow** — `.github/workflows/release.yml`
- **Data directory** — stored at `<install-dir>/data/` next to the exe
- **Bundled Ollama management** — `internal/ollama/` manager handles: detect existing Ollama, download binary (~200MB), start `ollama serve`, pull model (~1.1GB), report progress via API
- **Setup AI button** — LLM banner shows "Setup AI" when Ollama is missing or model not found. One click downloads binary, starts server, and pulls model. Stays installed for subsequent launches.

### Graceful degradation (LLM unavailable)

| LLM State | App State |
|-----------|-----------|
| Not running / no model | Yellow "LLM not available" banner with "Setup AI" button |
| Running | Full functionality |
| Crashed mid-session | Disable guard/refine, show error, offer restart |

The suggestion pipeline (TF-IDF) works without the LLM. Only guard classification and refinement are affected.

## First-Run Flow

```
User installs (30MB MSI) → Launches app
  → Go checks: is Ollama running on 127.0.0.1:11434?
  → Yes, with model → LLM ready, full functionality
  → Yes, model missing → auto-pull model (1.1GB)
  → No Ollama → yellow banner with "Setup AI" button
  → User clicks: download Ollama binary (~200MB)
  → Start ollama serve
  → Pull deepseek-r1:1.5b model (~1.1GB)
  → LLM ready
```

- Ollama binary cached at `<data-dir>/ollama/bin/ollama.exe`
- Model stored by Ollama in `<data-dir>/ollama/models/`
- Subsequent launches: detect existing Ollama → start cached binary → quick start

## Build & CI

### Build scripts

- **`build.sh`** (Linux/macOS) and **`build.ps1`** (Windows) automate the full pipeline:
  1. Build Go backend sidecar → output to `src-tauri/binaries/`
  2. Run `pnpm tauri build` (which triggers `beforeBuildCommand` → `pnpm build`)

### GitHub Actions (`.github/workflows/release.yml`)

Triggered on `v*` tag push or manual dispatch. Windows-only currently:

| Platform | Runner | Target triple | Installer |
|----------|--------|---------------|-----------|
| Windows | `windows-latest` | `x86_64-pc-windows-msvc` | `.msi` |

Steps:
1. Checkout + Setup Go/Rust/Node/pnpm with caching
2. `pnpm install` in `frontend/`
3. Build Go sidecar into `binaries/`
4. `pnpm tauri build`
5. Upload `.msi` artifact
6. Create GitHub Release with artifact

### Windows signing

- MSI needs a code signing certificate ($200-500/year)
- Without signing: SmartScreen warning on first install
- Mitigation: publish on GitHub with enough downloads to build reputation, or skip for v0.1

## Installer size

| Component | Size |
|-----------|------|
| Go binary | ~20MB |
| Frontend assets | ~200KB |
| Tauri runtime | ~10MB |
| **Installer** | **~30MB** |

Ollama binary (~200MB) and model (~1.1GB) downloaded on first "Setup AI" click.

## Unresolved Questions

1. **Windows signing**: Pay for a cert or ship unsigned for v0.1?
2. **Platform expansion**: Windows only for now; Linux/macOS when demand grows.
3. **Model choice**: DeepSeek 1.5B works; consider smaller (0.5B) for faster downloads.
