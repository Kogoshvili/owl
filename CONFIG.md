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
    "base_url": "http://localhost:11434",
    "model": "gemma3:4b"
  }
}
```

## Environment Variables

Environment variables override the config file settings:

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_ENABLED` | Enable LLM refinement | `false` |
| `LLM_BASE_URL` | Ollama API URL | `http://localhost:11434` |
| `LLM_MODEL` | LLM model to use | `gemma3:4b` |

## LLM Configuration

To use LLM refinement for folder suggestions and auto-tagging:

1. Install and run Ollama: `ollama serve`
2. Pull a model: `ollama pull gemma3:4b`
3. Enable in config: Set `llm.enabled: true` or `LLM_ENABLED=true`

Recommended models:
- `gemma3:4b` — Small, fast (~2-3s per cluster), good enough for this task
- `llama3.1:8b` — Higher quality, but slower (~5-10s per cluster)

## Log Level

Set `LOG_LEVEL` to control logging verbosity:
- `debug` — Detailed logs including progress
- `info` — General information (default)
- `warn` — Warnings only
- `error` — Errors only