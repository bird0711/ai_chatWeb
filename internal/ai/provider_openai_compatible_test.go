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
