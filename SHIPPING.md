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
│  │  └─────────────────────────────────┘   │  │
│  └──────────────────┬──────────────────────┘  │
│                     │                         │
│  (optional)         │ HTTP :1234               │
│  ┌──────────────────▼──────────────────────┐  │
│  │  llama-server (bundled, ~50MB binary)    │  │
│  │  + model.gguf (downloaded on first run) │  │
│  └──────────────────────────────────────────┘  │
│                                                │
│  Data stored at <install-dir>/data/            │
│  (next to the installed .exe)                 │
└────────────────────────────────────────────────┘
```

## Current Implementation Status

### ✅ Implemented

- **Go backend** with `--port` and `--data-dir` CLI flags
- **API routes** mounted under `/api` prefix via `http.StripPrefix`
- **Tauri sidecar** — Go binary registered as `externalBin` in `tauri.conf.json`
- **Sidecar lifecycle** — Tauri spawns Go on startup, kills on window close
- **Frontend BASE** auto-detects Tauri vs browser (`window.__TAURI_INTERNALS__`)
- **Browse button** — native folder picker via `@tauri-apps/plugin-dialog`, with `<input webkitdirectory>` fallback
- **Build scripts** — `build.sh` (Linux/macOS) and `build.ps1` (Windows)
- **GitHub Actions release workflow** — `.github/workflows/release.yml` builds all 3 platforms on tag push
- **Data directory** — stored at `<install-dir>/data/` (next to the exe in the installed app, or `backend/data/` in dev)

### 🚧 Future (llama-server bundling)

LLM integration is optional and not bundled in the installer. Users currently need their own Ollama endpoint. The sections below describe the planned first-run flow for bundled model support.

## LLM Strategy Decision

Two competing approaches for the organization backend.

### Content TF-IDF (current default)

| Pro | Con |
|-----|-----|
| Fast (~30s for 12K files) | Keyword-based, misses semantic meaning |
| No model dependency | "report.pdf" and "report.txt" don't share signal |
| Works offline immediately | Can't group files that share no common words |
| Simple, well-tested | |

### Embeddings (Ollama / other API)

| Pro | Con |
|-----|-----|
| Understands semantics | Requires an embedding model (~400MB) |
| "Q3 report.xlsx" and "financials.pdf" group together | Slower first run (~20-40min to embed 12K) |
| Better clusters | Another download, another model management |
| | embeddings cached to DB, so only slow once |

### Recommendation

Ship with **content_tfidf** as the only strategy. It works immediately, no model download needed for organization logic. The LLM (for guard + refine) is separate and already needed. If users need better grouping, embeddings can be added as an optional power-up in a later release.

If embeddings strategy is kept, the embedding model would be downloaded alongside the chat model on first run (~400MB extra).

## First-Run Flow (with bundled llama-server)

```
User installs (30MB without llama-server / 100MB with) → Launches app
  → Tauri starts Go sidecar
  → Go checks: is llama-server running on :1234?
  → No? Look for bundled llama-server
  → Check: does model.gguf exist in data/models/?
  → No? Show "Downloading AI model (1GB)…" in UI
  → Download from Hugging Face / GitHub releases
  → Start llama-server with model
  → Health-check /v1/models
  → Set LLM_BASE_URL to http://127.0.0.1:1234/v1
  → Show main UI
```

### Model download UX

- Progress bar with ETA shown in Tauri splash/loading window
- Resume support if interrupted (range requests)
- Store model in `data/models/deepseek-r1-distill-qwen-1.5b-q4_k_m.gguf`
- Model source: Hugging Face (e.g., `lmstudio-community/DeepSeek-R1-Distill-Qwen-1.5B-GGUF`)
- When download is cached, subsequent launches start immediately

### Model selection

- **Chat model**: `DeepSeek-R1-Distill-Qwen-1.5B-Q4_K_M` (~950MB GGUF)
  - Already tested, prompts tuned for it
  - Alternatives: Qwen2.5-Coder-0.5B (~380MB, smaller but needs prompt rework)
- **Embedding model** (if added later): e.g., `all-MiniLM-L6-v2-Q4_K_M` (~70MB)

## llama-server Bundling

### Sourcing

Pre-built `llama-server` binaries from llama.cpp releases (MIT license):
- `https://github.com/ggml-org/llama.cpp/releases`

### Targets

| Platform | Binary | Size |
|----------|--------|------|
| Windows x86_64 | `llama-server.exe` | ~45MB |
| Linux x86_64 | `llama-server` | ~50MB |
| macOS arm64 | `llama-server` | ~45MB |

Bundled inside the Tauri installer alongside the Go binary. Placed in the app's resources directory at runtime.

## Go Sidecar (implemented)

### Tauri sidecar config

- Go binary registered as a Tauri sidecar in `tauri.conf.json`: `"externalBin": ["binaries/owl-backend"]`
- Binary placed at `src-tauri/binaries/owl-backend-<target-triple>`.exe (built by `build.sh`/`build.ps1`)
- Passed flags: `--port 3721 --data-dir <app-dir>/data`
  - In dev: `--data-dir` resolves to `backend/data/` relative to project root
  - In production: resolves to `<install-dir>/data/` (next to the exe)
- Tauri spawns on app start via `tauri_plugin_shell`
- Tauri kills on close via `CommandChild::kill()` in `on_window_event`

### Graceful degradation (when LLM is unavailable)

| LLM State | App State |
|-----------|-----------|
| Not configured | Yellow "LLM not available" banner, guard/refine disabled |
| Running | Full functionality |
| Crashed mid-session | Disable guard/refine, show error, offer restart |

The suggestion pipeline (TF-IDF) works without the LLM. Only guard classification and refinement are affected.

## Build & CI

### Build scripts

- **`build.sh`** (Linux/macOS) and **`build.ps1`** (Windows) automate the full pipeline:
  1. Build Go backend sidecar → output to `src-tauri/binaries/`
  2. Run `pnpm tauri build` (which triggers `beforeBuildCommand` → `pnpm build`)

### GitHub Actions (`.github/workflows/release.yml`)

Triggered on `v*` tag push or manual dispatch. Matrix build across 3 platforms:

| Platform | Runner | Target triple | Installer |
|----------|--------|---------------|-----------|
| Windows | `windows-latest` | `x86_64-pc-windows-msvc` | `.msi` |
| Linux | `ubuntu-latest` | `x86_64-unknown-linux-gnu` | `.deb` + `.AppImage` |
| macOS | `macos-latest` | `aarch64-apple-darwin` | `.dmg` |

Steps per platform:
1. Install system deps (Linux: webkit2gtk, etc.)
2. Setup Go + Rust + Node + pnpm with caching
3. Build Go sidecar into `binaries/`
4. `pnpm tauri build` (frontend built automatically via `beforeBuildCommand`)
5. Upload installer artifact

A final `release` job collects all artifacts and creates a GitHub Release with `softprops/action-gh-release`.

### Windows signing

- `msi` installer needs a code signing certificate ($200-500/year)
- Without signing: SmartScreen warning on first install
- Mitigation: publish on GitHub with enough downloads to build reputation, or skip signing for v0.1

## Installer size breakdown

| Component | Size |
|-----------|------|
| Go binary | ~20MB |
| (future) llama-server binary | ~50MB (single platform) |
| Frontend assets | ~200KB |
| Tauri runtime | ~10MB |
| **Current installer** (no llama-server) | **~30MB** |
| **Future installer** (with llama-server) | **~80-100MB** |

Model downloaded on first run: ~950MB extra (for bundled LLM).

## Unresolved Questions (Decide Before Shipping)

1. **content_tfidf vs embeddings**: Ship with only content_tfidf? Add embeddings model download?
2. **Model choice**: Stick with DeepSeek 1.5B or go smaller (0.5B)?
3. **Windows signing**: Pay for a cert or ship unsigned for v0.1?
4. **First platform**: Ship Windows only first, or all three simultaneously?
5. **Model download source**: Hugging Face direct, or our own mirror?
