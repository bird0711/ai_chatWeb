package http

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	nethttp "net/http"
)

func (s *Server) showSettings(c *gin.Context) {
	configs, err := s.services.ListModelConfigs(c.Request.Context())
	if err != nil {
		renderError(c, nethttp.StatusInternalServerError, err)
		return
	}
	renderHTML(c, nethttp.StatusOK, "settings.html", gin.H{
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
		renderHTML(c, nethttp.StatusBadRequest, "settings.html", gin.H{
			"Title":       "模型 API 设置",
			"Config":      config,
			"Configs":     configs,
			"Error":       userFacingError(err),
			"CurrentUser": currentUser(c),
		})
		return
	}
	configs, _ := s.services.ListModelConfigs(c.Request.Context())
	renderHTML(c, nethttp.StatusOK, "settings.html", gin.H{
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
		renderHTML(c, nethttp.StatusBadRequest, "settings.html", gin.H{
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
	renderHTML(c, nethttp.StatusOK, "settings.html", gin.H{
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
		renderHTML(c, nethttp.StatusBadRequest, "settings.html", gin.H{
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
	renderHTML(c, nethttp.StatusOK, "health.html", gin.H{
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
	renderHTML(c, nethttp.StatusOK, "usage.html", gin.H{
		"Title":       "Token 统计",
		"Stats":       stats,
		"CurrentUser": currentUser(c),
	})
}
