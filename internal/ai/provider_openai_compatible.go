package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"ai_chat/internal/domain"
)

type OpenAICompatibleClient struct {
	httpClient    *http.Client
	retryAttempts int
	retryBackoff  time.Duration
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model           string        `json:"model"`
	Messages        []ChatMessage `json:"messages"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type modelsResponse struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func NewOpenAICompatibleClient() *OpenAICompatibleClient {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = envDurationSeconds("MODEL_API_TLS_HANDSHAKE_TIMEOUT_SECONDS", 30*time.Second)
	timeout := envDurationSeconds("MODEL_API_TIMEOUT_SECONDS", 90*time.Second)
	return &OpenAICompatibleClient{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		retryAttempts: envInt("MODEL_API_RETRY_ATTEMPTS", 2),
		retryBackoff:  envDurationMillis("MODEL_API_RETRY_BACKOFF_MS", 800*time.Millisecond),
	}
}

func (c *OpenAICompatibleClient) GenerateReply(ctx context.Context, input ReplyInput) (Reply, error) {
	systemPrompt := AppendFileContext(BuildSystemPrompt(input.Chat, input.Role), input.Files)
	return c.generate(ctx, input.ModelConfig, input.Role, systemPrompt, BuildConversation(input.Messages, input.UserMessage))
}

func (c *OpenAICompatibleClient) GenerateReview(ctx context.Context, input ReviewInput) (Reply, error) {
	systemPrompt := AppendFileContext(BuildReviewSystemPrompt(input.Chat, input.Role), input.Files)
	return c.generate(ctx, input.ModelConfig, input.Role, systemPrompt, BuildReviewConversation(input.Messages, input.UserMessage, input.FirstRoundReplies))
}

func (c *OpenAICompatibleClient) generate(ctx context.Context, config domain.ModelConfig, role domain.Role, systemPrompt string, conversation []ChatMessage) (Reply, error) {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		return Reply{}, errors.New("model base URL is required")
	}
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return Reply{}, errors.New("model API key is required")
	}
	model := strings.TrimSpace(role.Model)
	if model == "" {
		model = strings.TrimSpace(config.DefaultModel)
	}
	if model == "" {
		return Reply{}, errors.New("model is required")
	}

	messages := []ChatMessage{{Role: "system", Content: systemPrompt}}
	messages = append(messages, conversation...)
	requestBody := chatCompletionRequest{Model: model, Messages: messages}
	if role.ReasoningEffort != "" {
		requestBody.ReasoningEffort = role.ReasoningEffort
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return Reply{}, err
	}

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		return req, nil
	})
	if err != nil {
		return Reply{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return Reply{}, err
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return Reply{}, fmt.Errorf("decode model response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return Reply{}, errors.New(parsed.Error.Message)
		}
		return Reply{}, fmt.Errorf("model API returned status %d", resp.StatusCode)
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return Reply{}, errors.New("model response did not include content")
	}
	reply := Reply{Content: strings.TrimSpace(parsed.Choices[0].Message.Content)}
	if parsed.Usage != nil {
		reply.Usage = Usage{
			PromptTokens:     parsed.Usage.PromptTokens,
			CompletionTokens: parsed.Usage.CompletionTokens,
			TotalTokens:      parsed.Usage.TotalTokens,
		}
	}
	return reply, nil
}

func (c *OpenAICompatibleClient) ListModels(ctx context.Context, config domain.ModelConfig) ([]string, error) {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	if baseURL == "" {
		return nil, errors.New("model base URL is required")
	}
	apiKey := strings.TrimSpace(config.APIKey)
	if apiKey == "" {
		return nil, errors.New("model API key is required")
	}

	resp, err := c.doWithRetry(func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		return req, nil
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	var parsed modelsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decode models response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, errors.New(parsed.Error.Message)
		}
		return nil, fmt.Errorf("model API returned status %d", resp.StatusCode)
	}
	models := make([]string, 0, len(parsed.Data))
	seen := map[string]bool{}
	for _, item := range parsed.Data {
		id := strings.TrimSpace(item.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		models = append(models, id)
	}
	if len(models) == 0 {
		return nil, errors.New("model API returned no models")
	}
	return models, nil
}

func (c *OpenAICompatibleClient) doWithRetry(newRequest func() (*http.Request, error)) (*http.Response, error) {
	attempts := c.maxAttempts()
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		req, err := newRequest()
		if err != nil {
			return nil, err
		}
		resp, err := c.httpClient.Do(req)
		if err == nil {
			if attempt < attempts && shouldRetryStatus(resp.StatusCode) {
				lastErr = fmt.Errorf("model API returned status %d", resp.StatusCode)
				_ = resp.Body.Close()
				c.sleepBeforeRetry(req.Context(), attempt)
				continue
			}
			return resp, nil
		}
		lastErr = err
		if attempt >= attempts || !isTransientModelAPIError(err) {
			return nil, modelAPITransportError(err, attempt)
		}
		c.sleepBeforeRetry(req.Context(), attempt)
	}
	return nil, modelAPITransportError(lastErr, attempts)
}

func (c *OpenAICompatibleClient) maxAttempts() int {
	if c.retryAttempts <= 0 {
		return 1
	}
	return c.retryAttempts + 1
}

func (c *OpenAICompatibleClient) sleepBeforeRetry(ctx context.Context, attempt int) {
	backoff := c.retryBackoff
	if backoff == 0 {
		backoff = 800 * time.Millisecond
	}
	if backoff < 0 {
		backoff = 0
	}
	if backoff > 0 {
		backoff *= time.Duration(attempt)
	}
	timer := time.NewTimer(backoff)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusRequestTimeout ||
		status == http.StatusTooManyRequests ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout ||
		status >= 500
}

func isTransientModelAPIError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "tls handshake timeout") ||
		strings.Contains(text, "timeout") ||
		strings.Contains(text, "connection reset") ||
		strings.Contains(text, "temporary")
}

func modelAPITransportError(err error, attempts int) error {
	if err == nil {
		return errors.New("model API request failed")
	}
	text := strings.ToLower(err.Error())
	if strings.Contains(text, "tls handshake timeout") {
		return fmt.Errorf("模型 API TLS 握手超时，已尝试 %d 次；请稍后重试，或检查模型供应商网络和 base_url 配置", attempts)
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("模型 API 连接超时，已尝试 %d 次；请稍后重试，或检查模型供应商网络和 base_url 配置", attempts)
	}
	if isTransientModelAPIError(err) {
		return fmt.Errorf("模型 API 临时网络错误，已尝试 %d 次：%v", attempts, err)
	}
	return err
}

func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}

func envDurationMillis(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	millis, err := strconv.Atoi(value)
	if err != nil || millis < 0 {
		return fallback
	}
	return time.Duration(millis) * time.Millisecond
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
