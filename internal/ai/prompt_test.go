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
	for _, expected := range []string{
		"本群聊当前讨论主题是：成本控制",
		"你的接话必须服务于该主题",
		"选择一个最值得回应的观点自然接话",
		"可以认同、补充、追问、指出风险或温和反驳",
		"不要逐条总结全场",
		"1 到 3 段短段落",
		"少用“首先”“其次”“综上”这类报告式表达",
		"不要重复自己的第一轮回答",
		"Reviewer",
	} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("expected review prompt to contain %q, prompt: %s", expected, prompt)
		}
	}
}

func TestBuildReviewConversationAsksForNaturalReply(t *testing.T) {
	userMessage := domain.Message{ID: 1, SenderType: domain.SenderUser, SenderName: "用户", Content: "怎么控制预算？"}
	replies := []domain.Message{
		{SenderName: "Planner", Content: "先拆分固定成本和可变成本。"},
		{SenderName: "Risk", Content: "别忽略供应商涨价风险。"},
	}
	conversation := BuildReviewConversation([]domain.Message{userMessage}, userMessage, replies)
	if len(conversation) == 0 {
		t.Fatal("expected review conversation to contain messages")
	}
	content := conversation[len(conversation)-1].Content
	for _, expected := range []string{
		"请选一个最值得回应的观点",
		"用群聊里的自然语气接话",
		"Planner：先拆分固定成本和可变成本。",
		"Risk：别忽略供应商涨价风险。",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("expected review conversation to contain %q, content: %s", expected, content)
		}
	}
}

func TestBuildFileContextIncludesUploadedText(t *testing.T) {
	context := BuildFileContext([]domain.ChatFile{
		{OriginalName: "brief.md", ExtractedText: "项目目标：降低延迟"},
	})
	for _, expected := range []string{"用户上传到本群聊", "文件：brief.md", "项目目标：降低延迟"} {
		if !strings.Contains(context, expected) {
			t.Fatalf("expected file context to contain %q, context: %s", expected, context)
		}
	}
}
