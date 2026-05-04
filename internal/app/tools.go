package app

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"ai_chat/internal/domain"
)

type ToolDefinition struct {
	Name        string
	DisplayName string
	Description string
	InputHint   string
}

func (s *Services) ListTools(ctx context.Context) []ToolDefinition {
	return []ToolDefinition{
		{Name: "current_time", DisplayName: "当前时间", Description: "返回服务器当前时间。", InputHint: "无需输入。"},
		{Name: "text_stats", DisplayName: "文本统计", Description: "统计输入文本的字符数、词数和行数。", InputHint: "输入要统计的文本。"},
		{Name: "calculator", DisplayName: "四则计算", Description: "计算只包含数字、括号和 + - * / 的表达式。", InputHint: "例如：12 * (3 + 4)。"},
	}
}

func (s *Services) RunTool(ctx context.Context, chatID int64, toolName, input string) (domain.ToolExecution, error) {
	if _, err := s.store.GetChat(ctx, chatID); err != nil {
		return domain.ToolExecution{}, err
	}
	toolName = strings.TrimSpace(toolName)
	input = strings.TrimSpace(input)
	result, runErr := runBuiltinTool(toolName, input)
	status := domain.ToolExecutionSuccess
	errorText := ""
	content := fmt.Sprintf("工具 %s 执行成功。\n输入：%s\n结果：%s", toolName, input, result)
	if runErr != nil {
		status = domain.ToolExecutionFailed
		errorText = runErr.Error()
		content = fmt.Sprintf("工具 %s 执行失败。\n输入：%s\n错误：%s", toolName, input, errorText)
	}
	message, err := s.store.CreateMessage(ctx, domain.Message{
		ChatID:     chatID,
		SenderType: domain.SenderSystem,
		SenderName: "系统",
		Content:    content,
	})
	if err != nil {
		return domain.ToolExecution{}, err
	}
	execution, err := s.store.CreateToolExecution(ctx, domain.ToolExecution{
		ChatID:    chatID,
		MessageID: message.ID,
		ToolName:  toolName,
		Input:     input,
		Status:    status,
		Result:    result,
		Error:     errorText,
	})
	if err != nil {
		return domain.ToolExecution{}, err
	}
	return execution, nil
}

func runBuiltinTool(toolName, input string) (string, error) {
	switch toolName {
	case "current_time":
		return time.Now().Format(time.RFC3339), nil
	case "text_stats":
		return textStats(input), nil
	case "calculator":
		value, err := evalArithmetic(input)
		if err != nil {
			return "", err
		}
		return strconv.FormatFloat(value, 'f', -1, 64), nil
	default:
		return "", fmt.Errorf("%w: unsupported tool", ErrValidation)
	}
}

func textStats(input string) string {
	lines := 0
	if input != "" {
		lines = strings.Count(input, "\n") + 1
	}
	return fmt.Sprintf("字符数：%d，词数：%d，行数：%d", len([]rune(input)), len(strings.Fields(input)), lines)
}

type arithmeticParser struct {
	input string
	pos   int
}

func evalArithmetic(input string) (float64, error) {
	if strings.TrimSpace(input) == "" {
		return 0, fmt.Errorf("%w: expression is required", ErrValidation)
	}
	parser := arithmeticParser{input: input}
	value, err := parser.parseExpression()
	if err != nil {
		return 0, err
	}
	parser.skipSpaces()
	if parser.pos != len(parser.input) {
		return 0, fmt.Errorf("%w: invalid expression", ErrValidation)
	}
	return value, nil
}

func (p *arithmeticParser) parseExpression() (float64, error) {
	value, err := p.parseTerm()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('+') {
			next, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			value += next
			continue
		}
		if p.match('-') {
			next, err := p.parseTerm()
			if err != nil {
				return 0, err
			}
			value -= next
			continue
		}
		return value, nil
	}
}

func (p *arithmeticParser) parseTerm() (float64, error) {
	value, err := p.parseFactor()
	if err != nil {
		return 0, err
	}
	for {
		p.skipSpaces()
		if p.match('*') {
			next, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			value *= next
			continue
		}
		if p.match('/') {
			next, err := p.parseFactor()
			if err != nil {
				return 0, err
			}
			if next == 0 {
				return 0, fmt.Errorf("%w: division by zero", ErrValidation)
			}
			value /= next
			continue
		}
		return value, nil
	}
}

func (p *arithmeticParser) parseFactor() (float64, error) {
	p.skipSpaces()
	if p.match('+') {
		return p.parseFactor()
	}
	if p.match('-') {
		value, err := p.parseFactor()
		return -value, err
	}
	if p.match('(') {
		value, err := p.parseExpression()
		if err != nil {
			return 0, err
		}
		p.skipSpaces()
		if !p.match(')') {
			return 0, fmt.Errorf("%w: missing closing parenthesis", ErrValidation)
		}
		return value, nil
	}
	return p.parseNumber()
}

func (p *arithmeticParser) parseNumber() (float64, error) {
	p.skipSpaces()
	start := p.pos
	dotSeen := false
	for p.pos < len(p.input) {
		r := rune(p.input[p.pos])
		if unicode.IsDigit(r) {
			p.pos++
			continue
		}
		if r == '.' && !dotSeen {
			dotSeen = true
			p.pos++
			continue
		}
		break
	}
	if start == p.pos {
		return 0, fmt.Errorf("%w: number is required", ErrValidation)
	}
	value, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid number", ErrValidation)
	}
	return value, nil
}

func (p *arithmeticParser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *arithmeticParser) match(ch byte) bool {
	if p.pos >= len(p.input) || p.input[p.pos] != ch {
		return false
	}
	p.pos++
	return true
}
