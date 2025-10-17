package llm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AnthropicProvider struct {
	apiKey   string
	cacheDir string
	force    bool
	client   *http.Client
	limiter  *rateLimiter
}

type rateLimiter struct {
	lastRequest time.Time
	minDelay    time.Duration
}

func NewAnthropicProvider(config AnthropicConfig) (Provider, error) {
	apiKey := config.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	if config.CacheDir == "" {
		config.CacheDir = ".codedoc-cache"
	}

	if err := os.MkdirAll(config.CacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	maxQPS := config.MaxQPS
	if maxQPS == 0 {
		maxQPS = 2.0
	}

	return &AnthropicProvider{
		apiKey:   apiKey,
		cacheDir: config.CacheDir,
		force:    config.Force,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		limiter: &rateLimiter{
			minDelay: time.Duration(1000/maxQPS) * time.Millisecond,
		},
	}, nil
}

func (p *AnthropicProvider) Summarize(ctx context.Context, request SummarizeRequest) (SummarizeResponse, error) {
	cacheKey := p.getCacheKey(request)
	cacheFile := filepath.Join(p.cacheDir, cacheKey+".json")

	if !p.force {
		if cached, err := p.loadFromCache(cacheFile); err == nil {
			return cached, nil
		}
	}

	prompt := p.buildPrompt(request)

	p.limiter.wait()

	response, err := p.callAPI(ctx, prompt)
	if err != nil {
		return SummarizeResponse{}, err
	}

	result := SummarizeResponse{
		Summary: response,
		Cached:  false,
		Tokens:  p.estimateTokens(prompt + response),
	}

	if err := p.saveToCache(cacheFile, result); err != nil {
	}

	return result, nil
}

func (p *AnthropicProvider) getCacheKey(request SummarizeRequest) string {
	if request.CacheKey != "" {
		return request.CacheKey
	}

	data := fmt.Sprintf("%s-%s-%d-%d",
		request.Type,
		request.Context,
		request.Constraints.MaxWords,
		request.Constraints.MaxBullets,
	)

	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (p *AnthropicProvider) loadFromCache(cacheFile string) (SummarizeResponse, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return SummarizeResponse{}, err
	}

	var result SummarizeResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return SummarizeResponse{}, err
	}

	result.Cached = true
	return result, nil
}

func (p *AnthropicProvider) saveToCache(cacheFile string, response SummarizeResponse) error {
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0o644)
}

func (p *AnthropicProvider) buildPrompt(request SummarizeRequest) string {
	var systemPrompt string
	var userPrompt string

	switch request.Type {
	case SummaryTypeArchitecture:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf(
			"Provide an architecture overview of this codebase in no more than %d words. "+
				"Focus on: what the project does, main components, data flow, and key dependencies/frameworks.\n\n"+
				"Context:\n%s\n\n"+
				"Write a clear, concise overview:",
			request.Constraints.MaxWords, request.Context)

	case SummaryTypeModule:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf(
			"Summarize this module/directory in no more than %d words. "+
				"Focus on: purpose, noteworthy submodules, and cross-dependencies.\n\n"+
				"Context:\n%s\n\n"+
				"Write a clear, concise summary:",
			request.Constraints.MaxWords, request.Context)

	case SummaryTypeFile:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf(
			"Summarize this file in no more than %d words. "+
				"Focus on: role, key responsibilities, important imports, and side-effects.\n\n"+
				"Context:\n%s\n\n"+
				"Write a clear, concise summary:",
			request.Constraints.MaxWords, request.Context)

	case SummaryTypeFunction:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf(
			"List the key functions/classes in bullet points (maximum %d bullets). "+
				"Format: '- Name() — purpose; inputs → outputs; side effects (if any)'\n\n"+
				"Context:\n%s\n\n"+
				"List the key functions/classes:",
			request.Constraints.MaxBullets, request.Context)

	case SummaryTypeQuickstart:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf(
			"Provide quickstart instructions in no more than %d bullet points. "+
				"Focus on: how to run, test, and build the project.\n\n"+
				"Context:\n%s\n\n"+
				"List the quickstart steps:",
			request.Constraints.MaxBullets, request.Context)

	default:
		systemPrompt = "You are a senior software engineer writing concise internal documentation."
		userPrompt = fmt.Sprintf("Summarize the following:\n\n%s", request.Context)
	}

	return systemPrompt + "\n\n" + userPrompt
}

func (p *AnthropicProvider) callAPI(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model": "claude-3-haiku-20240307",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  1000,
		"temperature": 0.2,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", fmt.Errorf("rate limited, please retry")
		}
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if len(response.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return strings.TrimSpace(response.Content[0].Text), nil
}

func (p *AnthropicProvider) estimateTokens(text string) int {
	return len(text) / 4
}

func (l *rateLimiter) wait() {
	elapsed := time.Since(l.lastRequest)
	if elapsed < l.minDelay {
		time.Sleep(l.minDelay - elapsed)
	}
	l.lastRequest = time.Now()
}
