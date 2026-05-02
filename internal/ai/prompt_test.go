package ai

import (
	"strings"
	"testing"

	"ai_chat/internal/domain"
)

func TestBuildSystemPromptIncludesChatTopic(t *testing.T) {
	chat := domain.Chat{Topic: "v0.3 用户体验优化"}
	role := domain.Role{Name: "Architect", Persona: "persona", ReplyStyle: "concise"}
	prompt := BuildSystemPrompt(chat, role)
	for _, expected := range []string{"本群聊当前讨论主题是：v0.3 用户体验优化", "请优先围绕该主题回答", "Architect"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected prompt to contain %q, prompt: %s", expected, prompt)
		}
	}
}

func TestBuildReviewSystemPromptIncludesChatTopic(t *testing.T) {
	chat := domain.Chat{Topic: "成本控制"}
	role := domain.Role{Name: "Reviewer", Persona: "persona", ReplyStyle: "direct"}
	prompt := BuildReviewSystemPrompt(chat, role)
	for _, expected := range []string{"本群聊当前讨论主题是：成本控制", "必须服务于该主题", "Reviewer"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected review prompt to contain %q, prompt: %s", expected, prompt)
		}
	}
}
