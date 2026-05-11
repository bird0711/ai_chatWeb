package http

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	nethttp "net/http"
)

func (s *Server) listChats(c *gin.Context) {
	chats, err := s.services.ListChats(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	renderHTML(c, nethttp.StatusOK, "chats_index.html", gin.H{
		"Title":       "AI 群聊",
		"Chats":       chats,
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) createChat(c *gin.Context) {
	chat, err := s.services.CreateChat(c.Request.Context(), c.PostForm("name"))
	if err != nil {
		chats, _ := s.services.ListChats(c.Request.Context())
		renderHTML(c, nethttp.StatusBadRequest, "chats_index.html", gin.H{
			"Title":       "AI 群聊",
			"Chats":       chats,
			"Error":       userFacingError(err),
			"CurrentUser": currentUser(c),
		})
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chat.ID, 10))
}

func (s *Server) deleteChat(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	if err := s.services.DeleteChat(c.Request.Context(), chatID); err != nil {
		renderError(c, statusForError(err), err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats")
}

func (s *Server) setChatAIReview(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	chat, err := s.services.SetChatAIReview(c.Request.Context(), chatID, c.PostForm("enabled") == "1")
	if err != nil {
		logChatActionError(c, chatID, "set_ai_review", err)
		if wantsJSON(c) {
			renderJSONError(c, statusForAppError(err), err)
			return
		}
		s.renderChatWithError(c, chatID, err)
		return
	}
	if wantsJSON(c) {
		c.JSON(nethttp.StatusOK, gin.H{
			"chat_id":            chat.ID,
			"ai_review_enabled":  chat.AIReviewEnabled,
			"ai_review_status":   map[bool]string{true: "已开启", false: "已关闭"}[chat.AIReviewEnabled],
			"ai_review_headline": map[bool]string{true: "AI 互评已开启", false: "AI 互评已关闭"}[chat.AIReviewEnabled],
		})
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) setChatTopic(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	if _, err := s.services.SetChatTopic(c.Request.Context(), chatID, c.PostForm("topic")); err != nil {
		logChatActionError(c, chatID, "set_topic", err)
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) showChat(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	detail, err := s.services.GetChat(c.Request.Context(), chatID)
	if err != nil {
		renderError(c, statusForError(err), err)
		return
	}
	configs, err := s.services.ListModelConfigs(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	renderHTML(c, nethttp.StatusOK, "chat_detail.html", gin.H{
		"Title":       detail.Chat.Name,
		"Chat":        detail.Chat,
		"Roles":       detail.Roles,
		"Messages":    detail.Messages,
		"Files":       detail.Files,
		"Tools":       detail.Tools,
		"ToolDefs":    s.services.ListTools(c.Request.Context()),
		"HasConfig":   len(configs) > 0,
		"Configs":     configs,
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) uploadChatFile(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	if _, err := s.services.GetChat(c.Request.Context(), chatID); err != nil {
		logChatActionError(c, chatID, "upload_file_get_chat", err)
		renderError(c, statusForError(err), err)
		return
	}
	upload, err := saveChatFileUpload(c)
	if err != nil {
		logChatActionError(c, chatID, "save_file_upload", err)
		s.renderChatWithError(c, chatID, err)
		return
	}
	if _, err := s.services.AddChatFile(c.Request.Context(), chatID, upload.originalName, upload.storagePath, upload.contentType, upload.sizeBytes, upload.extractedText); err != nil {
		logChatActionError(c, chatID, "add_chat_file", err)
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) runTool(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	if _, err := s.services.RunTool(c.Request.Context(), chatID, c.PostForm("tool_name"), c.PostForm("tool_input")); err != nil {
		logChatActionError(c, chatID, "run_tool", err)
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) addRole(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	modelConfigID, model := parseModelChoice(c.PostForm("model_choice"))
	avatar, err := saveAvatarUpload(c, "")
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	_, err = s.services.AddRole(
		c.Request.Context(),
		chatID,
		modelConfigID,
		c.PostForm("name"),
		avatar,
		c.PostForm("persona"),
		c.PostForm("reply_style"),
		model,
		c.PostForm("reasoning_effort"),
	)
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) updateRole(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}
	modelConfigID, model := parseModelChoice(c.PostForm("model_choice"))
	current, err := s.services.GetRole(c.Request.Context(), chatID, roleID)
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	avatar, err := saveAvatarUpload(c, current.Avatar)
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	_, err = s.services.UpdateRole(
		c.Request.Context(),
		chatID,
		roleID,
		modelConfigID,
		c.PostForm("name"),
		avatar,
		c.PostForm("persona"),
		c.PostForm("reply_style"),
		model,
		c.PostForm("reasoning_effort"),
		c.PostForm("can_speak") == "1",
	)
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) toggleRoleSpeaking(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}
	if _, err := s.services.ToggleRoleSpeaking(c.Request.Context(), chatID, roleID); err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) deleteRole(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	roleID, ok := parseRoleID(c)
	if !ok {
		return
	}
	if err := s.services.DeleteRole(c.Request.Context(), chatID, roleID); err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) sendMessage(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	_, err := s.services.SendUserMessage(c.Request.Context(), chatID, c.PostForm("content"))
	if err != nil {
		s.renderChatWithError(c, chatID, err)
		return
	}
	c.Redirect(nethttp.StatusFound, "/chats/"+strconv.FormatInt(chatID, 10))
}

func (s *Server) sendMessageAsync(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	msg, err := s.services.SendUserMessageAsync(c.Request.Context(), chatID, c.PostForm("content"))
	if err != nil {
		logChatActionError(c, chatID, "send_message_async", err)
		renderJSONError(c, statusForAppError(err), err)
		return
	}
	c.JSON(nethttp.StatusAccepted, gin.H{
		"message": toMessageResponse(msg),
	})
}

func (s *Server) listMessageUpdates(c *gin.Context) {
	chatID, ok := parseChatID(c)
	if !ok {
		return
	}
	afterID, err := strconv.ParseInt(c.DefaultQuery("after_id", "0"), 10, 64)
	if err != nil || afterID < 0 {
		renderJSONError(c, nethttp.StatusBadRequest, errors.New("invalid after_id"))
		return
	}
	messages, err := s.services.ListMessagesAfter(c.Request.Context(), chatID, afterID)
	if err != nil {
		logChatActionError(c, chatID, "list_message_updates", err)
		renderJSONError(c, statusForAppError(err), err)
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{
		"messages": toMessageResponses(messages),
	})
}

func (s *Server) renderChatWithError(c *gin.Context, chatID int64, err error) {
	detail, detailErr := s.services.GetChat(c.Request.Context(), chatID)
	if detailErr != nil {
		logChatActionError(c, chatID, "render_chat_error_detail", detailErr)
		renderError(c, statusForError(detailErr), detailErr)
		return
	}
	configs, configErr := s.services.ListModelConfigs(c.Request.Context())
	hasConfig := configErr == nil && len(configs) > 0
	renderHTML(c, nethttp.StatusBadRequest, "chat_detail.html", gin.H{
		"Title":       detail.Chat.Name,
		"Chat":        detail.Chat,
		"Roles":       detail.Roles,
		"Messages":    detail.Messages,
		"Files":       detail.Files,
		"Tools":       detail.Tools,
		"ToolDefs":    s.services.ListTools(c.Request.Context()),
		"HasConfig":   hasConfig,
		"Configs":     configs,
		"Error":       userFacingError(err),
		"CurrentUser": currentUser(c),
	})
}
