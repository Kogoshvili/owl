package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ollama/ollama/api"
)

type Client struct {
	cfg   ClientConfig
	cli   *api.Client
	ready bool
	mu    sync.Mutex
}

func NewClient(cfg ClientConfig) *Client {
	if !cfg.Enabled {
		return &Client{cfg: cfg, ready: false}
	}

	cli, err := api.ClientFromEnvironment()
	if err != nil {
		slog.Warn("llm: failed to create client", "error", err)
		return &Client{cfg: cfg, ready: false}
	}

	return &Client{
		cfg:   cfg,
		cli:   cli,
		ready: false,
	}
}

const (
	clusterRefineTimeout = 120 * time.Second
	tagRefineTimeout     = 120 * time.Second
	heartbeatTimeout    = 2 * time.Second
	maxRetries           = 3
)

func (c *Client) IsAvailable(ctx context.Context) bool {
	if !c.cfg.Enabled || c.cli == nil {
		return false
	}

	if c.ready {
		return true
	}

	checkCtx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
	defer cancel()

	err := c.cli.Heartbeat(checkCtx)
	if err != nil {
		slog.Debug("llm: heartbeat failed, will retry", "error", err)
		return false
	}

	c.ready = true
	slog.Info("llm: connected", "url", c.cfg.BaseURL, "model", c.cfg.Model)
	return true
}

func (c *Client) RefineCluster(ctx context.Context, files []ClusterFile, fileIDs []int64, currentName string) (*RefinementResult, error) {
	if !c.cfg.Enabled || c.cli == nil {
		return nil, fmt.Errorf("LLM not available")
	}

	if !c.ready {
		checkCtx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
		defer cancel()
		if err := c.cli.Heartbeat(checkCtx); err != nil {
			slog.Debug("llm: heartbeat failed", "error", err)
			return nil, fmt.Errorf("LLM not available")
		}
		c.ready = true
	}

	slog.Info("llm: refining cluster", "name", currentName, "files", len(files))

	var result *RefinementResult
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			slog.Info("llm: refine cluster retry", "attempt", attempt, "last_error", lastErr)
		}

		prompt := buildClusterPrompt(files, currentName)

		messages := []api.Message{
			{Role: "user", Content: prompt},
		}

		req := &api.ChatRequest{
			Model:    c.cfg.Model,
			Messages: messages,
			Stream:   new(bool),
		}

		chatCtx, cancel := context.WithTimeout(context.Background(), clusterRefineTimeout)
		var rawResponse string
		var err error

		c.mu.Lock()
		err = c.cli.Chat(chatCtx, req, func(resp api.ChatResponse) error {
			rawResponse += resp.Message.Content
			return nil
		})
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

	slog.Info("llm: refined cluster", "name", result.Name, "related", result.Related, "removed", len(result.RemovedIDs), "description", result.Description, "reason", result.Reason)
	return result, nil
}

func (c *Client) RefineTag(ctx context.Context, tagName string, fileNames []string, keywords []string) (*TagRefinementResult, error) {
	if !c.cfg.Enabled || c.cli == nil {
		return nil, fmt.Errorf("LLM not available")
	}

	if !c.ready {
		checkCtx, cancel := context.WithTimeout(context.Background(), heartbeatTimeout)
		defer cancel()
		if err := c.cli.Heartbeat(checkCtx); err != nil {
			slog.Debug("llm: heartbeat failed", "error", err)
			return nil, fmt.Errorf("LLM not available")
		}
		c.ready = true
	}

	slog.Info("llm: refining tag", "name", tagName, "files", len(fileNames))

	var result *TagRefinementResult
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			slog.Info("llm: refine tag retry", "attempt", attempt, "last_error", lastErr)
		}

		prompt := buildTagPrompt(tagName, fileNames, keywords)

		messages := []api.Message{
			{Role: "user", Content: prompt},
		}

		req := &api.ChatRequest{
			Model:    c.cfg.Model,
			Messages: messages,
			Stream:   new(bool),
		}

		chatCtx, cancel := context.WithTimeout(context.Background(), tagRefineTimeout)
		var rawResponse string
		var err error

		c.mu.Lock()
		err = c.cli.Chat(chatCtx, req, func(resp api.ChatResponse) error {
			rawResponse += resp.Message.Content
			return nil
		})
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

	slog.Info("llm: refined tag", "name", tagName, "meaningful", result.Meaningful, "better_name", result.BetterName, "description", result.Description, "reason", result.Reason)
	return result, nil
}