package http

import (
	"os"
	"strconv"
	"strings"

	"ai_chat/internal/domain"

	"github.com/gin-gonic/gin"
)

func roleIDValue(id *int64) int64 {
	if id == nil {
		return 0
	}
	return *id
}

func joinModels(models []string) string {
	out := ""
	for i, model := range models {
		if i > 0 {
			out += "\n"
		}
		out += model
	}
	return out
}

func modelEquals(a, b string) bool {
	return a == b
}

func stringEquals(a, b string) bool {
	return a == b
}

func formatInt(value int) string {
	return strconv.Itoa(value)
}

func configByID(configs []domain.ModelConfig, id int64) domain.ModelConfig {
	for _, config := range configs {
		if config.ID == id {
			return config
		}
	}
	return domain.ModelConfig{}
}

func canSpeakCount(roles []domain.Role) int {
	count := 0
	for _, role := range roles {
		if role.CanSpeak {
			count++
		}
	}
	return count
}

func avatarText(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = strings.TrimSpace(fallback)
	}
	if value == "" {
		return "AI"
	}
	runes := []rune(value)
	if len(runes) <= 2 {
		return value
	}
	return string(runes[:2])
}

func isImageAvatar(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(value, "/uploads/") || strings.HasPrefix(value, "/static/")
}

func currentUser(c *gin.Context) domain.User {
	user, _ := c.Get("CurrentUser")
	if typed, ok := user.(domain.User); ok {
		return typed
	}
	return domain.User{}
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
