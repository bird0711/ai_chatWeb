package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ai_chat/internal/domain"
)

type OpenAICompatibleClient struct {
	httpClient *http.Client
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
	return &OpenAICompatibleClient{
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAICompatibleClient) GenerateReply(ctx context.Context, input ReplyInput) (Reply, error) {
	return c.generate(ctx, input.ModelConfig, input.Role, BuildSystemPrompt(input.Chat, input.Role), BuildConversation(input.Messages, input.UserMessage))
}

func (c *OpenAICompatibleClient) GenerateReview(ctx context.Context, input ReviewInput) (Reply, error) {
	return c.generate(ctx, input.ModelConfig, input.Role, BuildReviewSystemPrompt(input.Chat, input.Role), BuildReviewConversation(input.Messages, input.UserMessage, input.FirstRoundReplies))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return Reply{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Reply{}, err
	}
	defer resp.Body.Close()

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
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
