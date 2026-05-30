package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Client struct {
	cfg   ClientConfig
	http  *http.Client
	ready bool
	mu    sync.Mutex
}

func NewClient(cfg ClientConfig) *Client {
	if !cfg.Enabled {
		return &Client{cfg: cfg, ready: false}
	}

	return &Client{
		cfg: cfg,
		http: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		},
		ready: false,
	}
}

// --- OpenAI-compatible API types ---

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ollamaChatResponse struct {
	Model     string      `json:"model"`
	Message   chatMessage `json:"message"`
	Done      bool        `json:"done"`
	DoneReason string     `json:"done_reason,omitempty"`
}

type ollamaModel struct {
	Name string `json:"name"`
}

type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

const (
	clusterRefineTimeout = 120 * time.Second
	heartbeatTimeout     = 5 * time.Second
	maxRetries           = 3
)

// chatCompletion sends a prompt to the OpenAI-compatible chat completion endpoint
// and returns the raw response text.
func (c *Client) chatCompletion(ctx context.Context, prompt string) (string, error) {
	body := chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshalling request: %w", err)
	}

	url := c.cfg.BaseURL + "/api/chat"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaChatResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if ollamaResp.Message.Content == "" {
		return "", fmt.Errorf("empty response from Ollama")
	}

	return ollamaResp.Message.Content, nil
}

func (c *Client) IsAvailable(ctx context.Context) bool {
	if !c.cfg.Enabled || c.http == nil {
		return false
	}

	if c.ready {
		return true
	}

	checkCtx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
	defer cancel()

	// Use GET /api/tags as a lightweight health check
	url := c.cfg.BaseURL + "/api/tags"
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, url, nil)
	if err != nil {
		slog.Debug("llm: heartbeat request failed", "error", err)
		return false
	}

	resp, err := c.http.Do(req)
	if err != nil {
		slog.Debug("llm: heartbeat failed, will retry", "error", err)
		return false
	}

	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("llm: heartbeat returned non-OK status", "status", resp.StatusCode)
		return false
	}

	var tags ollamaTagsResponse
	if err := json.Unmarshal(respBody, &tags); err == nil {
		found := false
		for _, m := range tags.Models {
			if m.Name == c.cfg.Model {
				found = true
				break
			}
		}
		if !found {
			slog.Error("llm: model not found in Ollama", "model", c.cfg.Model, "available_models", modelNames(tags.Models))
			return false
		}
	}

	c.ready = true
	slog.Info("llm: connected", "url", c.cfg.BaseURL, "model", c.cfg.Model)
	return true
}

func (c *Client) RefineCluster(ctx context.Context, files []ClusterFile, fileIDs []int64, currentName string) (*RefinementResult, error) {
	if !c.cfg.Enabled || c.http == nil {
		return nil, fmt.Errorf("LLM not available")
	}

	if !c.ready {
		if !c.IsAvailable(ctx) {
			return nil, fmt.Errorf("LLM not available")
		}
	}

	slog.Info("llm: refining cluster", "name", currentName, "files", len(files))

	var result *RefinementResult
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			slog.Info("llm: refine cluster retry", "attempt", attempt, "last_error", lastErr)
		}

		prompt := buildClusterPrompt(files, currentName)

		chatCtx, cancel := context.WithTimeout(context.Background(), clusterRefineTimeout)
		var rawResponse string
		var err error

		c.mu.Lock()
		rawResponse, err = c.chatCompletion(chatCtx, prompt)
		c.mu.Unlock()
		cancel()

		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: refine cluster failed after retries", "name", currentName, "files", len(files), "error", err)
				return nil, err
			}
			continue
		}

		result, err = parseClusterResponse(rawResponse, fileIDs)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: parse cluster response failed after retries", "name", currentName, "files", len(files), "error", err, "raw", rawResponse)
				return nil, err
			}
			slog.Warn("llm: parse cluster response failed, retrying", "name", currentName, "attempt", attempt, "error", err)
			continue
		}

		break
	}

	slog.Info("llm: refined cluster", "name", result.Name, "related", result.Related, "removed", len(result.RemovedIDs), "description", result.Description)
	return result, nil
}


func (c *Client) ClassifyFolder(ctx context.Context, folderName string, subfolders []string, fileNames []string, parentName string, parentGuarded bool) (*FolderClassification, error) {
	if !c.cfg.Enabled || c.http == nil {
		return nil, fmt.Errorf("LLM not available")
	}

	if !c.ready {
		if !c.IsAvailable(ctx) {
			return nil, fmt.Errorf("LLM not available")
		}
	}

	slog.Info("llm: classifying folder", "folder", folderName, "parent", parentName, "parent_guarded", parentGuarded, "subfolders", len(subfolders), "files", len(fileNames))

	var result *FolderClassification
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			slog.Info("llm: classify folder retry", "attempt", attempt, "last_error", lastErr)
		}

		prompt := buildFolderGuardPrompt(folderName, subfolders, fileNames, parentName, parentGuarded)

		chatCtx, cancel := context.WithTimeout(context.Background(), clusterRefineTimeout)
		var rawResponse string
		var err error

		c.mu.Lock()
		rawResponse, err = c.chatCompletion(chatCtx, prompt)
		c.mu.Unlock()
		cancel()

		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: classify folder failed after retries", "folder", folderName, "error", err)
				return nil, err
			}
			continue
		}

		result, err = parseFolderGuardResponse(rawResponse)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: parse folder guard response failed after retries", "folder", folderName, "error", err, "raw", rawResponse)
				return nil, err
			}
			slog.Warn("llm: parse folder guard response failed, retrying", "folder", folderName, "attempt", attempt, "error", err)
			continue
		}

		break
	}

		slog.Info("llm: classified folder", "folder", folderName, "related", result.Related, "reason", result.Reason)
		return result, nil
}

func modelNames(models []ollamaModel) string {
	names := make([]string, len(models))
	for i, m := range models {
		names[i] = m.Name
	}
	return strings.Join(names, ", ")
}
