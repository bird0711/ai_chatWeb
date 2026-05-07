package http

import (
	"context"
	"html/template"

	"ai_chat/internal/app"

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

	router.Use(server.ensureCSRFCookie)
	router.GET("/login", server.showLogin)
	router.POST("/login", server.login)
	router.GET("/register", server.showRegister)
	router.POST("/register", server.register)

    router.Use(server.requireCSRF)
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
