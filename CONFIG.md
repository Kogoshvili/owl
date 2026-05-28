# Configuration

Owl can be configured via a JSON file or environment variables.

## Configuration File

The config file is located at:
- **Linux/macOS**: `~/.config/owl/config.json`
- **Windows**: `%APPDATA%\owl\config.json`

You can override the path with the `OWL_CONFIG` environment variable.

### Example Configuration

```json
{
  "llm": {
    "enabled": true,
    "base_url": "http://localhost:1234/v1",
    "model": ""
  }
}
```

## Environment Variables

Environment variables override the config file settings:

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_ENABLED` | Enable LLM refinement | `false` |
| `LLM_BASE_URL` | OpenAI-compatible API URL | `http://localhost:1234/v1` |
| `LLM_MODEL` | LLM model to use (supplied by server) | `""` (auto) |

## LLM Configuration

To use LLM refinement for folder suggestions and auto-tagging:

### Using LM Studio (recommended)

1. Download and install [LM Studio](https://lmstudio.ai/)
2. Open LM Studio and load a model (e.g., `gemma-3-4b-it`, `llama-3.1-8b`)
3. Start the local inference server (click "Start Server" button)
   - By default it runs at `http://localhost:1234`
4. Enable in config: Set `llm.enabled: true` or `LLM_ENABLED=true`

**Note:** The `model` field can be left empty (`""`) — LM Studio uses whichever model you have loaded in the server. You only need to set it if you're using a multi-model proxy.

### Using any OpenAI-compatible API

This works with any OpenAI-compatible API endpoint. Simply set:
- `base_url`: The API base URL (e.g., `http://localhost:1234/v1`)
- `model`: The model name (if required by your API)

Recommended models:
- `gemma-3-4b-it` — Small, fast (~2-3s per cluster), good enough for this task
- `llama-3.1-8b` — Higher quality, but slower (~5-10s per cluster)

## Log Level

Set `LOG_LEVEL` to control logging verbosity:
- `debug` — Detailed logs including progress
- `info` — General information (default)
- `warn` — Warnings only
- `error` — Errors only
