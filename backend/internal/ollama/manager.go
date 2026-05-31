package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"owl/internal/config"
	"owl/internal/llm"
	"path/filepath"
	"sync"
	"time"
)

type State string

const (
	StateNotStarted   State = "not_started"
	StateDownloading  State = "downloading"
	StateStarting     State = "starting"
	StatePullingModel State = "pulling_model"
	StateReady        State = "ready"
	StateError        State = "error"
)

type Status struct {
	State    State  `json:"state"`
	Message  string `json:"message"`
	Progress int64  `json:"progress,omitempty"`
	Total    int64  `json:"total,omitempty"`
}

type Manager struct {
	dataDir   string
	binPath   string
	host      string
	model     string
	modelsDir string
	cmd       *exec.Cmd
	httpCli   *http.Client
	statusMu  sync.Mutex
	status    Status
	setupMu   sync.Mutex
	setupDone chan struct{}
}

func New(config *config.Config) *Manager {
	parsed, err := url.Parse(config.LLM.BaseURL)
	host := config.LLM.BaseURL
	if err == nil && parsed.Host != "" {
		host = parsed.Host
	}

	return &Manager{
		dataDir:   config.DataDir,
		host:      host,
		model:     config.LLM.Model,
		modelsDir: filepath.Join(config.DataDir, "ollama", "models"),
		httpCli:   &http.Client{Timeout: 10 * time.Second},
		status:    Status{State: StateNotStarted, Message: "Not started"},
	}
}

func Init(config *config.Config) *Manager {
	ollamaMgr := New(config)

	llmCfg := llm.ConfigFromEnv(config)

	if !llmCfg.Enabled {
		slog.Info("LLM refinement disabled")
		return ollamaMgr
	}

	if ollamaMgr.IsAlreadyRunning(context.Background()) {
		if ollamaMgr.ModelExists(context.Background()) {
			slog.Info("LLM refinement enabled", "url", llmCfg.BaseURL, "model", llmCfg.Model)
			return ollamaMgr
		}
		slog.Info("Ollama running but model missing, will pull on demand", "model", llmCfg.Model)
		return ollamaMgr
	}

	slog.Info("LLM refinement enabled", "url", llmCfg.BaseURL, "model", llmCfg.Model)
	return ollamaMgr
}

func (m *Manager) setStatus(s State, msg string, args ...int64) {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	m.status.State = s
	m.status.Message = msg
	if len(args) > 0 {
		m.status.Progress = args[0]
	}
	if len(args) > 1 {
		m.status.Total = args[1]
	}
}

func (m *Manager) Status() Status {
	m.statusMu.Lock()
	defer m.statusMu.Unlock()
	return m.status
}

func (m *Manager) BaseURL() string { return "http://" + m.host }

// IsAlreadyRunning checks if an Ollama instance is already available.
func (m *Manager) IsAlreadyRunning(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.BaseURL()+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := m.httpCli.Do(req)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ModelExists checks if the configured model is available in the running Ollama.
func (m *Manager) ModelExists(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.BaseURL()+"/api/tags", nil)
	if err != nil {
		return false
	}
	resp, err := m.httpCli.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false
	}
	for _, mod := range tags.Models {
		if mod.Name == m.model {
			return true
		}
	}
	return false
}

// RunSetup starts the full setup flow: download, start, pull model.
// It runs in a goroutine and updates status for frontend polling.
func (m *Manager) RunSetup(ctx context.Context) {
	m.setupMu.Lock()
	if m.setupDone != nil {
		m.setupMu.Unlock()
		return
	}
	m.setupDone = make(chan struct{})
	m.setupMu.Unlock()

	defer func() {
		m.setupMu.Lock()
		close(m.setupDone)
		m.setupMu.Unlock()
	}()

	// Step 1: check if already running (race condition: user started ollama manually)
	if m.IsAlreadyRunning(ctx) {
		slog.Info("ollama: already running, skipping download")
		if m.ModelExists(ctx) {
			m.setStatus(StateReady, "Ollama ready")
			return
		}
		m.pullModel(ctx)
		return
	}

	// Step 2: download if needed
	binDir := filepath.Join(m.dataDir, "ollama", "bin")
	m.binPath = filepath.Join(binDir, "ollama.exe")

	if _, err := os.Stat(m.binPath); os.IsNotExist(err) {
		slog.Info("ollama: downloading binary")
		m.setStatus(StateDownloading, "Downloading Ollama engine...")
		if err := m.download(ctx, binDir); err != nil {
			slog.Error("ollama: download failed", "error", err)
			m.setStatus(StateError, "Download failed: "+err.Error())
			return
		}
	} else {
		slog.Info("ollama: binary found locally", "path", m.binPath)
	}

	// Step 3: start ollama serve
	slog.Info("ollama: starting server")
	m.setStatus(StateStarting, "Starting Ollama engine...")
	if err := m.startServer(ctx); err != nil {
		slog.Error("ollama: failed to start", "error", err)
		m.setStatus(StateError, "Failed to start: "+err.Error())
		return
	}

	// Step 4: pull model
	m.pullModel(ctx)
}

func (m *Manager) pullModel(ctx context.Context) {
	m.setStatus(StatePullingModel, "Downloading AI model...", 0, 0)
	slog.Info("ollama: pulling model", "model", m.model)

	body := map[string]any{"model": m.model, "stream": true}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.BaseURL()+"/api/pull", bytes.NewReader(bodyJSON))
	if err != nil {
		m.setStatus(StateError, "Pull request failed: "+err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		m.setStatus(StateError, "Pull connection failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for {
		var line struct {
			Status    string `json:"status"`
			Digest    string `json:"digest,omitempty"`
			Total     int64  `json:"total,omitempty"`
			Completed int64  `json:"completed,omitempty"`
		}
		if err := dec.Decode(&line); err == io.EOF {
			break
		} else if err != nil {
			m.setStatus(StateError, "Pull failed: "+err.Error())
			return
		}

		if line.Status == "success" {
			m.setStatus(StateReady, "AI model ready")
			slog.Info("ollama: model pull complete", "model", m.model)
			return
		}

		if line.Status == "downloading" && line.Total > 0 {
			m.setStatus(StatePullingModel, "Downloading AI model...", line.Completed, line.Total)
		}
	}
}

func (m *Manager) download(ctx context.Context, destDir string) error {
	os.MkdirAll(destDir, 0755)

	// For now, download from Ollama's official GitHub release
	// In production, we'll host this ourselves
	url := "https://github.com/ollama/ollama/releases/latest/download/OllamaSetup.exe"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	tmpPath := filepath.Join(destDir, "OllamaSetup.exe")
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			wn, writeErr := f.Write(buf[:n])
			if writeErr != nil {
				f.Close()
				return fmt.Errorf("writing temp file: %w", writeErr)
			}
			written += int64(wn)
			if total > 0 {
				m.setStatus(StateDownloading, "Downloading Ollama engine...", written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			return fmt.Errorf("reading download: %w", readErr)
		}
	}
	f.Close()

	// Extract with /S /D
	extractDir := filepath.Join(destDir, "extracted")
	os.RemoveAll(extractDir)
	os.MkdirAll(extractDir, 0755)

	slog.Info("ollama: extracting installer", "installer", tmpPath, "dest", extractDir)
	cmd := exec.CommandContext(ctx, tmpPath, "/S", "/D="+extractDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("extracting Ollama: %w, output: %s", err, string(output))
	}

	// Verify ollama.exe exists
	exePath := filepath.Join(extractDir, "ollama.exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		// Try with version subdirectory
		entries, _ := os.ReadDir(extractDir)
		for _, e := range entries {
			if e.IsDir() {
				altPath := filepath.Join(extractDir, e.Name(), "ollama.exe")
				if _, err := os.Stat(altPath); err == nil {
					exePath = altPath
					break
				}
			}
		}
	}

	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		os.Remove(tmpPath)
		return fmt.Errorf("ollama.exe not found after extraction at %s", extractDir)
	}

	// Move extracted ollama.exe to bin dir
	destExe := filepath.Join(destDir, "ollama.exe")
	if err := os.Rename(exePath, destExe); err != nil {
		// Fallback: copy
		srcFile, _ := os.Open(exePath)
		dstFile, _ := os.Create(destExe)
		if srcFile != nil && dstFile != nil {
			io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
		}
	}

	// Cleanup
	os.Remove(tmpPath)
	os.RemoveAll(extractDir)

	m.binPath = destExe
	slog.Info("ollama: binary ready", "path", destExe)
	return nil
}

func (m *Manager) startServer(ctx context.Context) error {
	port := "11434"
	host := "127.0.0.1:" + port

	modelsDir := m.modelsDir
	os.MkdirAll(modelsDir, 0755)

	cmd := exec.CommandContext(ctx, m.binPath, "serve")
	cmd.Env = append(os.Environ(),
		"OLLAMA_HOST="+host,
		"OLLAMA_MODELS="+modelsDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting ollama serve: %w", err)
	}
	m.cmd = cmd

	// Wait for server to be ready
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			cmd.Process.Kill()
			return fmt.Errorf("ollama server did not start within 30s")
		case <-ticker.C:
			if m.IsAlreadyRunning(ctx) {
				slog.Info("ollama: server is ready")
				return nil
			}
		}
	}
}

func (m *Manager) Shutdown() error {
	if m.cmd != nil && m.cmd.Process != nil {
		slog.Info("ollama: stopping server")
		return m.cmd.Process.Kill()
	}
	return nil
}
