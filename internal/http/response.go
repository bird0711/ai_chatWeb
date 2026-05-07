package http

import (
	"errors"
	"log"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"ai_chat/internal/app"
	"ai_chat/internal/domain"
	"ai_chat/internal/store"

	"github.com/gin-gonic/gin"
)

func parseModelChoice(choice string) (int64, string) {
	parts := strings.SplitN(choice, "::", 2)
	if len(parts) != 2 {
		return 0, ""
	}
	configID, _ := strconv.ParseInt(parts[0], 10, 64)
	return configID, parts[1]
}

func parseChatID(c *gin.Context) (int64, bool) {
	chatID, err := strconv.ParseInt(c.Param("chatID"), 10, 64)
	if err != nil || chatID <= 0 {
		renderError(c, nethttp.StatusBadRequest, errors.New("invalid chat id"))
		return 0, false
	}
	return chatID, true
}

func parseRoleID(c *gin.Context) (int64, bool) {
	roleID, err := strconv.ParseInt(c.Param("roleID"), 10, 64)
	if err != nil || roleID <= 0 {
		renderError(c, nethttp.StatusBadRequest, errors.New("invalid role id"))
		return 0, false
	}
	return roleID, true
}

func renderError(c *gin.Context, status int, err error) {
	log.Printf("http_error method=%s path=%s status=%d error=%q", c.Request.Method, c.Request.URL.Path, status, userFacingError(err))
	c.HTML(status, "error.html", gin.H{
		"Title": "Error",
		"Error": userFacingError(err),
	})
}

func logChatActionError(c *gin.Context, chatID int64, action string, err error) {
	if err == nil {
		return
	}
	log.Printf("chat_action_error action=%s method=%s path=%s chat_id=%d status=%d error=%q", action, c.Request.Method, c.Request.URL.Path, chatID, statusForAppError(err), userFacingError(err))
}

func statusForError(err error) int {
	if errors.Is(err, store.ErrNotFound) {
		return nethttp.StatusNotFound
	}
	return nethttp.StatusInternalServerError
}

func statusForAppError(err error) int {
	if errors.Is(err, store.ErrNotFound) {
		return nethttp.StatusNotFound
	}
	if errors.Is(err, app.ErrValidation) || errors.Is(err, app.ErrMVPBlocked) {
		return nethttp.StatusBadRequest
	}
	return nethttp.StatusInternalServerError
}

func userFacingError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type messageResponse struct {
	ID           int64  `json:"id"`
	ChatID       int64  `json:"chat_id"`
	SenderType   string `json:"sender_type"`
	SenderName   string `json:"sender_name"`
	SenderAvatar string `json:"sender_avatar"`
	RoleID       *int64 `json:"role_id,omitempty"`
	Content      string `json:"content"`
	CreatedAt    string `json:"created_at"`
}

func toMessageResponses(messages []domain.Message) []messageResponse {
	out := make([]messageResponse, 0, len(messages))
	for _, msg := range messages {
		out = append(out, toMessageResponse(msg))
	}
	return out
}

func toMessageResponse(msg domain.Message) messageResponse {
	return messageResponse{
		ID:           msg.ID,
		ChatID:       msg.ChatID,
		SenderType:   string(msg.SenderType),
		SenderName:   msg.SenderName,
		SenderAvatar: msg.SenderAvatar,
		RoleID:       msg.RoleID,
		Content:      msg.Content,
		CreatedAt:    msg.CreatedAt.Format(time.RFC3339),
	}
}
