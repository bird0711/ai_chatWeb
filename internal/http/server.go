package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ai_chat/internal/app"
	"ai_chat/internal/domain"
	"ai_chat/internal/store"

	"github.com/gin-gonic/gin"
)

type Server struct {
	services *app.Services
	redis    Pinger
}

type Pinger interface {
	Ping(ctx context.Context) error
}

func NewRouter(services *app.Services, redis Pinger) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.SetFuncMap(template.FuncMap{
		"roleIDValue":   roleIDValue,
		"joinModels":    joinModels,
		"modelEquals":   modelEquals,
		"stringEquals":  stringEquals,
		"canSpeakCount": canSpeakCount,
		"formatInt":     formatInt,
		"configByID":    configByID,
		"avatarText":    avatarText,
		"isImageAvatar": isImageAvatar,
	})
	router.LoadHTMLGlob(getenv("TEMPLATE_GLOB", "web/templates/*.html"))
	router.Static("/static", getenv("STATIC_DIR", "web/static"))
	router.Static("/uploads", getenv("UPLOAD_DIR", "uploads"))

	server := &Server{services: services, redis: redis}

	router.GET("/login", server.showLogin)
	router.POST("/login", server.login)
	router.GET("/register", server.showRegister)
	router.POST("/register", server.register)

	router.Use(server.requireAuth)
	router.GET("/", server.redirectRoot)
	router.POST("/logout", server.logout)
	router.GET("/chats", server.listChats)
	router.POST("/chats", server.createChat)
	router.GET("/chats/:chatID", server.showChat)
	router.POST("/chats/:chatID/delete", server.deleteChat)
	router.POST("/chats/:chatID/ai-review", server.setChatAIReview)
	router.POST("/chats/:chatID/topic", server.setChatTopic)
	router.POST("/chats/:chatID/files", server.uploadChatFile)
	router.POST("/chats/:chatID/tools", server.runTool)
	router.POST("/chats/:chatID/roles", server.addRole)
	router.POST("/chats/:chatID/roles/:roleID", server.updateRole)
	router.POST("/chats/:chatID/roles/:roleID/toggle-speaking", server.toggleRoleSpeaking)
	router.POST("/chats/:chatID/roles/:roleID/delete", server.deleteRole)
	router.POST("/chats/:chatID/messages", server.sendMessage)
	router.POST("/chats/:chatID/messages/async", server.sendMessageAsync)
	router.GET("/chats/:chatID/messages/updates", server.listMessageUpdates)
	router.GET("/settings/model", server.showSettings)
	router.POST("/settings/model", server.saveSettings)
	router.POST("/settings/model/check", server.checkSettings)
	router.POST("/settings/model/:configID/delete", server.deleteSettings)
	router.GET("/usage", server.showUsage)
	router.GET("/health", server.showHealth)
	return router
}

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

func (s *Server) showLogin(c *gin.Context) {
	c.HTML(nethttp.StatusOK, "auth.html", gin.H{
		"Title":  "登录",
		"Mode":   "login",
		"Action": "/login",
	})
}

func (s *Server) showRegister(c *gin.Context) {
	c.HTML(nethttp.StatusOK, "auth.html", gin.H{
		"Title":  "注册",
		"Mode":   "register",
		"Action": "/register",
	})
}

func (s *Server) login(c *gin.Context) {
	user, token, expiresAt, err := s.services.Login(c.Request.Context(), c.PostForm("email"), c.PostForm("password"))
	if err != nil {
		c.HTML(nethttp.StatusBadRequest, "auth.html", gin.H{
			"Title": "登录",
			"Mode":  "login",
			"Email": c.PostForm("email"),
			"Error": userFacingError(err),
		})
		return
	}
	setSessionCookie(c, token, expiresAt)
	c.Set("CurrentUser", user)
	c.Redirect(nethttp.StatusFound, "/chats")
}

func (s *Server) register(c *gin.Context) {
	user, token, expiresAt, err := s.services.Register(c.Request.Context(), c.PostForm("email"), c.PostForm("password"))
	if err != nil {
		c.HTML(nethttp.StatusBadRequest, "auth.html", gin.H{
			"Title": "注册",
			"Mode":  "register",
			"Email": c.PostForm("email"),
			"Error": userFacingError(err),
		})
		return
	}
	setSessionCookie(c, token, expiresAt)
	c.Set("CurrentUser", user)
	c.Redirect(nethttp.StatusFound, "/chats")
}

func (s *Server) logout(c *gin.Context) {
	token, _ := c.Cookie("session_token")
	if err := s.services.Logout(c.Request.Context(), token); err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	clearSessionCookie(c)
	c.Redirect(nethttp.StatusFound, "/login")
}

func (s *Server) requireAuth(c *gin.Context) {
	token, err := c.Cookie("session_token")
	if err != nil || strings.TrimSpace(token) == "" {
		s.rejectUnauthenticated(c)
		return
	}
	user, err := s.services.UserBySession(c.Request.Context(), token)
	if err != nil {
		clearSessionCookie(c)
		s.rejectUnauthenticated(c)
		return
	}
	c.Set("CurrentUser", user)
	ctx := domain.WithUserID(c.Request.Context(), user.ID)
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

func (s *Server) rejectUnauthenticated(c *gin.Context) {
	if wantsJSON(c) {
		c.JSON(nethttp.StatusUnauthorized, gin.H{"error": "login required"})
		c.Abort()
		return
	}
	c.Redirect(nethttp.StatusFound, "/login")
	c.Abort()
}

func wantsJSON(c *gin.Context) bool {
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		return true
	}
	path := c.Request.URL.Path
	return strings.Contains(path, "/messages/async") || strings.Contains(path, "/messages/updates")
}

func setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	nethttp.SetCookie(c.Writer, &nethttp.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		SameSite: nethttp.SameSiteLaxMode,
	})
}

func clearSessionCookie(c *gin.Context) {
	nethttp.SetCookie(c.Writer, &nethttp.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: nethttp.SameSiteLaxMode,
	})
}

func (s *Server) redirectRoot(c *gin.Context) {
	c.Redirect(nethttp.StatusFound, "/chats")
}

func (s *Server) listChats(c *gin.Context) {
	chats, err := s.services.ListChats(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	c.HTML(nethttp.StatusOK, "chats_index.html", gin.H{
		"Title":       "AI 群聊",
		"Chats":       chats,
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) createChat(c *gin.Context) {
	chat, err := s.services.CreateChat(c.Request.Context(), c.PostForm("name"))
	if err != nil {
		chats, _ := s.services.ListChats(c.Request.Context())
		c.HTML(nethttp.StatusBadRequest, "chats_index.html", gin.H{
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
			c.JSON(statusForAppError(err), gin.H{"error": userFacingError(err)})
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
	c.HTML(nethttp.StatusOK, "chat_detail.html", gin.H{
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
		c.JSON(statusForAppError(err), gin.H{"error": userFacingError(err)})
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
		c.JSON(nethttp.StatusBadRequest, gin.H{"error": "invalid after_id"})
		return
	}
	messages, err := s.services.ListMessagesAfter(c.Request.Context(), chatID, afterID)
	if err != nil {
		logChatActionError(c, chatID, "list_message_updates", err)
		c.JSON(statusForAppError(err), gin.H{"error": userFacingError(err)})
		return
	}
	c.JSON(nethttp.StatusOK, gin.H{
		"messages": toMessageResponses(messages),
	})
}

func (s *Server) showSettings(c *gin.Context) {
	configs, err := s.services.ListModelConfigs(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	c.HTML(nethttp.StatusOK, "settings.html", gin.H{
		"Title":       "模型 API 设置",
		"Configs":     configs,
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) saveSettings(c *gin.Context) {
	config, err := s.services.SaveModelConfig(
		c.Request.Context(),
		c.PostForm("name"),
		c.PostForm("provider"),
		c.PostForm("base_url"),
		c.PostForm("api_key"),
		c.PostForm("default_model"),
		c.PostForm("models"),
	)
	if err != nil {
		configs, _ := s.services.ListModelConfigs(c.Request.Context())
		c.HTML(nethttp.StatusBadRequest, "settings.html", gin.H{
			"Title":       "模型 API 设置",
			"Config":      config,
			"Configs":     configs,
			"Error":       userFacingError(err),
			"CurrentUser": currentUser(c),
		})
		return
	}
	configs, _ := s.services.ListModelConfigs(c.Request.Context())
	c.HTML(nethttp.StatusOK, "settings.html", gin.H{
		"Title":       "模型 API 设置",
		"Config":      config,
		"Configs":     configs,
		"Notice":      "模型 API 设置已保存。",
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) checkSettings(c *gin.Context) {
	config, err := s.services.CheckModelConfig(
		c.Request.Context(),
		c.PostForm("provider"),
		c.PostForm("base_url"),
		c.PostForm("api_key"),
	)
	if err != nil {
		configs, _ := s.services.ListModelConfigs(c.Request.Context())
		c.HTML(nethttp.StatusBadRequest, "settings.html", gin.H{
			"Title":       "模型 API 设置",
			"Config":      config,
			"Configs":     configs,
			"Error":       userFacingError(err),
			"CurrentUser": currentUser(c),
		})
		return
	}
	config.Name = c.PostForm("name")
	configs, _ := s.services.ListModelConfigs(c.Request.Context())
	c.HTML(nethttp.StatusOK, "settings.html", gin.H{
		"Title":       "模型 API 设置",
		"Config":      config,
		"Configs":     configs,
		"Notice":      "连接成功，已获取支持的模型列表。请选择默认模型并保存设置。",
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) deleteSettings(c *gin.Context) {
	configID, err := strconv.ParseInt(c.Param("configID"), 10, 64)
	if err != nil || configID <= 0 {
		renderError(c, nethttp.StatusBadRequest, errors.New("invalid model config id"))
		return
	}
	if err := s.services.DeleteModelConfig(c.Request.Context(), configID); err != nil {
		configs, _ := s.services.ListModelConfigs(c.Request.Context())
		c.HTML(nethttp.StatusBadRequest, "settings.html", gin.H{
			"Title":       "模型 API 设置",
			"Configs":     configs,
			"Error":       userFacingError(err),
			"CurrentUser": currentUser(c),
		})
		return
	}
	c.Redirect(nethttp.StatusFound, "/settings/model")
}

func (s *Server) showHealth(c *gin.Context) {
	mysqlStatus := "正常"
	mysqlError := ""
	if err := s.services.Health(c.Request.Context()); err != nil {
		mysqlStatus = "异常"
		mysqlError = err.Error()
	}
	redisStatus := "未配置"
	redisError := ""
	if s.redis != nil {
		redisStatus = "正常"
		if err := s.redis.Ping(c.Request.Context()); err != nil {
			redisStatus = "异常"
			redisError = err.Error()
		}
	}
	c.HTML(nethttp.StatusOK, "health.html", gin.H{
		"Title":       "系统状态",
		"MySQLStatus": mysqlStatus,
		"MySQLError":  mysqlError,
		"RedisStatus": redisStatus,
		"RedisError":  redisError,
		"CurrentUser": currentUser(c),
	})
}

func (s *Server) showUsage(c *gin.Context) {
	stats, err := s.services.TokenUsageStats(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	c.HTML(nethttp.StatusOK, "usage.html", gin.H{
		"Title":       "Token 统计",
		"Stats":       stats,
		"CurrentUser": currentUser(c),
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
	c.HTML(nethttp.StatusBadRequest, "chat_detail.html", gin.H{
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

func parseModelChoice(choice string) (int64, string) {
	parts := strings.SplitN(choice, "::", 2)
	if len(parts) != 2 {
		return 0, ""
	}
	configID, _ := strconv.ParseInt(parts[0], 10, 64)
	return configID, parts[1]
}

func currentUser(c *gin.Context) domain.User {
	user, _ := c.Get("CurrentUser")
	if typed, ok := user.(domain.User); ok {
		return typed
	}
	return domain.User{}
}

type chatFileUpload struct {
	originalName  string
	storagePath   string
	contentType   string
	sizeBytes     int64
	extractedText string
}

func saveChatFileUpload(c *gin.Context) (chatFileUpload, error) {
	file, err := c.FormFile("chat_file")
	if err != nil {
		if errors.Is(err, nethttp.ErrMissingFile) {
			return chatFileUpload{}, errors.New("file is required")
		}
		return chatFileUpload{}, err
	}
	if file.Size <= 0 {
		return chatFileUpload{}, errors.New("file is empty")
	}
	if file.Size > maxChatFileBytes {
		return chatFileUpload{}, errors.New("chat file must be 10MB or smaller")
	}
	ext, err := chatFileExtension(file)
	if err != nil {
		return chatFileUpload{}, err
	}
	src, err := file.Open()
	if err != nil {
		return chatFileUpload{}, err
	}
	defer src.Close()
	raw, err := io.ReadAll(io.LimitReader(src, maxChatFileBytes+1))
	if err != nil {
		return chatFileUpload{}, err
	}
	if int64(len(raw)) > maxChatFileBytes {
		return chatFileUpload{}, errors.New("chat file must be 10MB or smaller")
	}
	text, err := extractChatFileText(ext, raw)
	if err != nil {
		return chatFileUpload{}, err
	}
	name, err := randomHex(16)
	if err != nil {
		return chatFileUpload{}, err
	}
	uploadRoot := getenv("CHAT_FILE_DIR", filepath.Join("data", "chat-files"))
	dir := uploadRoot
	if err := os.MkdirAll(dir, 0755); err != nil {
		return chatFileUpload{}, err
	}
	filename := name + ext
	dstPath := filepath.Join(dir, filename)
	if err := os.WriteFile(dstPath, raw, 0644); err != nil {
		return chatFileUpload{}, err
	}
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain"
	}
	return chatFileUpload{
		originalName:  filepath.Base(file.Filename),
		storagePath:   dstPath,
		contentType:   contentType,
		sizeBytes:     int64(len(raw)),
		extractedText: text,
	}, nil
}

func chatFileExtension(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".txt", ".md", ".json", ".csv", ".log", ".docx", ".pdf":
		return ext, nil
	default:
		return "", errors.New("chat file must be txt, md, json, csv, log, docx, or pdf")
	}
}

func saveAvatarUpload(c *gin.Context, existing string) (string, error) {
	if !strings.HasPrefix(strings.ToLower(c.GetHeader("Content-Type")), "multipart/form-data") {
		return existing, nil
	}
	file, err := c.FormFile("avatar_file")
	if err != nil {
		if errors.Is(err, nethttp.ErrMissingFile) {
			return existing, nil
		}
		return "", err
	}
	if file.Size > 2*1024*1024 {
		return "", errors.New("avatar image must be 2MB or smaller")
	}
	ext, err := avatarExtension(file)
	if err != nil {
		return "", err
	}
	name, err := randomHex(16)
	if err != nil {
		return "", err
	}
	uploadRoot := getenv("UPLOAD_DIR", "uploads")
	dir := filepath.Join(uploadRoot, "avatars")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	filename := name + ext
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()
	dstPath := filepath.Join(dir, filename)
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return "/uploads/avatars/" + filename, nil
}

func avatarExtension(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext, nil
	default:
		return "", errors.New("avatar image must be jpg, png, gif, or webp")
	}
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
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

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
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
