package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
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

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type modelList struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

const (
	clusterRefineTimeout = 120 * time.Second
	tagRefineTimeout     = 120 * time.Second
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

	url := c.cfg.BaseURL + "/chat/completions"
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

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
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

	// Use GET /v1/models as a lightweight health check
	url := c.cfg.BaseURL + "/models"
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
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Debug("llm: heartbeat returned non-OK status", "status", resp.StatusCode)
		return false
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

func (c *Client) RefineTag(ctx context.Context, tagName string, fileNames []string, keywords []string) (*TagRefinementResult, error) {
	if !c.cfg.Enabled || c.http == nil {
		return nil, fmt.Errorf("LLM not available")
	}

	if !c.ready {
		if !c.IsAvailable(ctx) {
			return nil, fmt.Errorf("LLM not available")
		}
	}

	slog.Info("llm: refining tag", "name", tagName, "files", len(fileNames))

	var result *TagRefinementResult
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			slog.Info("llm: refine tag retry", "attempt", attempt, "last_error", lastErr)
		}

		prompt := buildTagPrompt(tagName, fileNames, keywords)

		chatCtx, cancel := context.WithTimeout(context.Background(), tagRefineTimeout)
		var rawResponse string
		var err error

		c.mu.Lock()
		rawResponse, err = c.chatCompletion(chatCtx, prompt)
		c.mu.Unlock()
		cancel()

		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: refine tag failed after retries", "name", tagName, "files", len(fileNames), "error", err)
				return nil, err
			}
			continue
		}

		result, err = parseTagResponse(rawResponse)
		if err != nil {
			lastErr = err
			if attempt == maxRetries {
				slog.Error("llm: parse tag response failed after retries", "name", tagName, "files", len(fileNames), "error", err, "raw", rawResponse)
				return nil, err
			}
			slog.Warn("llm: parse tag response failed, retrying", "name", tagName, "attempt", attempt, "error", err)
			continue
		}

		break
	}

	slog.Info("llm: refined tag", "name", tagName, "meaningful", result.Meaningful, "better_name", result.BetterName, "description", result.Description)
	return result, nil
}

func (c *Client) ExtractKeywords(ctx context.Context, files []struct{ ID int64; Name string; Content string }) ([]KeywordExtraction, error) {
	if !c.IsAvailable(ctx) {
		return nil, fmt.Errorf("llm: not available")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	prompt := buildKeywordExtractionPrompt(files)

	var result []KeywordExtraction
	for attempt := 1; attempt <= maxRetries; attempt++ {
		rawResponse, err := c.chatCompletion(ctx, prompt)
		if err != nil {
			if attempt == maxRetries {
				return nil, fmt.Errorf("llm: extract keywords failed: %w", err)
			}
			slog.Warn("llm: extract keywords failed, retrying", "attempt", attempt, "error", err)
			continue
		}

		parsed, err := parseKeywordExtractionResponse(rawResponse)
		if err != nil {
			if attempt == maxRetries {
				slog.Error("llm: parse keyword response failed", "files", len(files), "error", err)
				return nil, err
			}
			slog.Warn("llm: parse keyword response failed, retrying", "attempt", attempt, "error", err)
			continue
		}
		result = parsed
		break
	}

	slog.Info("llm: extracted keywords", "files", len(result))
	return result, nil
}
