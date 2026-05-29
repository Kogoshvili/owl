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
│  │  ┌──────────┐  ┌──────────────────┐    │  │
│  │  │ HTTP API │  │ LLM subprocess   │    │  │
│  │  │ serve    │  │ manager          │    │  │
│  │  │ frontend │  │ (start/health/   │    │  │
│  │  │ (dev)    │  │  kill)           │    │  │
│  │  └──────────┘  └──────┬───────────┘    │  │
│  └───────────────────────┼─────────────────┘  │
│                          │ HTTP :1234          │
│  ┌───────────────────────▼──────────────────┐  │
│  │  llama-server (bundled, ~50MB binary)    │  │
│  │  + model.gguf (downloaded on first run)  │  │
│  └──────────────────────────────────────────┘  │
│                                                │
│  Installer: ~100MB (Go + llama-server binary)  │
│  First-run download: ~1GB (model file)          │
└────────────────────────────────────────────────┘
```

## LLM Strategy Decision

Two competing approaches for the organization backend. Need to settle before shipping.

### Content TF-IDF (current default)

| Pro | Con |
|-----|-----|
| Fast (~30s for 12K files) | Keyword-based, misses semantic meaning |
| No model dependency | "report.pdf" and "report.txt" don't share signal |
| Works offline immediately | Can't group files that share no common words |
| Simple, well-tested | |

### Embeddings (LM Studio / other API)

| Pro | Con |
|-----|-----|
| Understands semantics | Requires an embedding model (~400MB) |
| "Q3 report.xlsx" and "financials.pdf" group together | Slower first run (~20-40min to embed 12K) |
| Better clusters | Another download, another model management |
| | embeddings cached to DB, so only slow once |

### Recommendation

Ship with **content_tfidf** as the only strategy. It works immediately, no model download needed for organization logic. The LLM (for guard + refine) is separate and already needed. If users need better grouping, embeddings can be added as an optional power-up in a later release.

If embeddings strategy is kept, the embedding model would be downloaded alongside the chat model on first run (~400MB extra).

## First-Run Flow

```
User installs (100MB) → Launches app
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

## Go Sidecar

### Tauri sidecar config

- Go binary registered as a Tauri sidecar in `tauri.conf.json`
- Passed flags: `--port 3721 --data-dir "$APPDATA/owl/data" --models-dir "$APPDATA/owl/models"`
- Tauri spawns on app start, kills on app close
- Tauri waits for health check (`GET /health` → 200) before showing window

### Go backend changes needed

- Add subprocess manager for llama-server:
  - `cmd/llm/start` — locate + spawn llama-server
  - `cmd/llm/stop` — graceful kill
  - `cmd/llm/health` — poll `/v1/models`
  - `cmd/llm/download` — download model from Hugging Face with progress
- Add `--models-dir` flag to main CLI
- Embed frontend dist (for dev mode without Tauri)
- On startup: if llama-server not running, start it; if model missing, show "downloading" state via API

### Graceful degradation

| LLM State | App State |
|-----------|-----------|
| Not downloaded yet | "Downloading AI model…" splash |
| Downloading | Progress bar, block guard + refine |
| Starting | Show spinner on guard/refine buttons |
| Running | Full functionality |
| Crashed mid-session | Disable guard/refine, show error, offer restart |

The suggestion pipeline (TF-IDF) works without the LLM. Only guard classification and refinement are affected.

## Build & CI

### Build environment

Current blocker: WSL lacks `libwebkit2gtk-4.1-dev` and other Tauri dependencies.

Options:

1. **Native Linux VM / dual-boot** — simplest, full Linux Tauri build
2. **Docker build container** — reproducible, works from WSL
3. **cross-compile from Windows** — Tauri cross-compile is complex, not recommended
4. **GitHub Actions CI** — Ubuntu runner has all deps, auto-builds on tag push

### Recommended CI flow (GitHub Actions)

```
Tag push v0.1.0
  → Build Go binary (linux/amd64, windows/amd64, darwin/arm64)
  → Download llama-server binary for each platform
  → Build frontend (pnpm build)
  → Build Tauri (.deb, .AppImage, .msi, .dmg)
  → Attach model download URL
  → Upload artifacts
```

### Windows signing

- `msi` installer needs a code signing certificate ($200-500/year)
- Without signing: SmartScreen warning on first install
- Mitigation: publish on GitHub with enough downloads to build reputation, or skip signing for v0.1

## Installer size breakdown

| Component | Size |
|-----------|------|
| Go binary | ~20MB |
| llama-server binary | ~50MB (single platform) |
| Frontend assets | ~200KB |
| Tauri runtime | ~10MB |
| **Total installer** | **~80-100MB** (per platform) |

Model downloaded on first run: ~950MB extra.

## Unresolved Questions (Decide Before Shipping)

1. **content_tfidf vs embeddings**: Ship with only content_tfidf? Add embeddings model download?
2. **Model choice**: Stick with DeepSeek 1.5B or go smaller (0.5B)?
3. **Windows signing**: Pay for a cert or ship unsigned for v0.1?
4. **First platform**: Ship Windows only first, or all three simultaneously?
5. **Model download source**: Hugging Face direct, or our own mirror?
