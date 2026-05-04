package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"ai_chat/internal/domain"
)

func TestOpenAICompatibleClientSendsReasoningEffort(t *testing.T) {
	var requestBody map[string]any
	client := &OpenAICompatibleClient{httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`), nil
	})}}

	_, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:            "Architect",
			Persona:         "persona",
			ReplyStyle:      "direct",
			Model:           "test-model",
			ReasoningEffort: "high",
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestBody["reasoning_effort"] != "high" {
		t.Fatalf("expected reasoning_effort high, got %#v", requestBody)
	}
}

func TestOpenAICompatibleClientOmitsDefaultReasoningEffort(t *testing.T) {
	var requestBody map[string]any
	client := &OpenAICompatibleClient{httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`), nil
	})}}

	_, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:       "Architect",
			Persona:    "persona",
			ReplyStyle: "direct",
			Model:      "test-model",
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := requestBody["reasoning_effort"]; ok {
		t.Fatalf("expected default reasoning_effort to be omitted, got %#v", requestBody)
	}
}

func TestOpenAICompatibleClientParsesTokenUsage(t *testing.T) {
	client := &OpenAICompatibleClient{httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResponse(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`), nil
	})}}

	reply, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:       "Architect",
			Persona:    "persona",
			ReplyStyle: "direct",
			Model:      "test-model",
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Usage.PromptTokens != 11 || reply.Usage.CompletionTokens != 7 || reply.Usage.TotalTokens != 18 {
		t.Fatalf("unexpected usage: %#v", reply.Usage)
	}
}

func TestOpenAICompatibleClientInjectsFileContext(t *testing.T) {
	var requestBody struct {
		Messages []ChatMessage `json:"messages"`
	}
	client := &OpenAICompatibleClient{httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatal(err)
		}
		return jsonResponse(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`), nil
	})}}

	_, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:       "Architect",
			Persona:    "persona",
			ReplyStyle: "direct",
			Model:      "test-model",
		},
		Files: []domain.ChatFile{
			{OriginalName: "brief.md", ExtractedText: "项目目标：降低延迟"},
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(requestBody.Messages) == 0 || !strings.Contains(requestBody.Messages[0].Content, "项目目标：降低延迟") {
		t.Fatalf("expected system prompt to include file context, messages: %#v", requestBody.Messages)
	}
}

func TestOpenAICompatibleClientRetriesTransientTimeout(t *testing.T) {
	attempts := 0
	client := &OpenAICompatibleClient{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			attempts++
			if attempts == 1 {
				return nil, timeoutError{"net/http: TLS handshake timeout"}
			}
			return jsonResponse(`{"choices":[{"message":{"role":"assistant","content":"ok after retry"}}]}`), nil
		})},
		retryAttempts: 1,
		retryBackoff:  -1,
	}

	reply, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:       "Architect",
			Persona:    "persona",
			ReplyStyle: "direct",
			Model:      "test-model",
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || reply.Content != "ok after retry" {
		t.Fatalf("expected retry success, attempts=%d reply=%#v", attempts, reply)
	}
}

func TestOpenAICompatibleClientReturnsReadableTimeoutError(t *testing.T) {
	client := &OpenAICompatibleClient{
		httpClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return nil, timeoutError{"net/http: TLS handshake timeout"}
		})},
		retryAttempts: 1,
		retryBackoff:  -1,
	}

	_, err := client.GenerateReply(context.Background(), ReplyInput{
		Chat: domain.Chat{Name: "Chat"},
		Role: domain.Role{
			Name:       "Architect",
			Persona:    "persona",
			ReplyStyle: "direct",
			Model:      "test-model",
		},
		ModelConfig: domain.ModelConfig{
			BaseURL:      "https://example.test/v1",
			APIKey:       "test-key",
			DefaultModel: "test-model",
		},
		UserMessage: domain.Message{Content: "hello"},
	})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	for _, expected := range []string{"模型 API TLS 握手超时", "已尝试 2 次", "base_url"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected timeout error to contain %q, got %v", expected, err)
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}

func jsonResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type timeoutError struct {
	message string
}

func (e timeoutError) Error() string {
	return e.message
}

func (e timeoutError) Timeout() bool {
	return true
}

func (e timeoutError) Temporary() bool {
	return true
}
