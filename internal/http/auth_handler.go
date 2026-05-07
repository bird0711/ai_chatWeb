package http

import (
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) showLogin(c *gin.Context) {
	c.HTML(nethttp.StatusOK, "auth.html", gin.H{
		"Title":  "鐧诲綍",
		"Mode":   "login",
		"Action": "/login",
	})
}

func (s *Server) showRegister(c *gin.Context) {
	c.HTML(nethttp.StatusOK, "auth.html", gin.H{
		"Title":  "娉ㄥ唽",
		"Mode":   "register",
		"Action": "/register",
	})
}

func (s *Server) login(c *gin.Context) {
	user, token, expiresAt, err := s.services.Login(c.Request.Context(), c.PostForm("email"), c.PostForm("password"))
	if err != nil {
		c.HTML(nethttp.StatusBadRequest, "auth.html", gin.H{
			"Title": "鐧诲綍",
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
			"Title": "娉ㄥ唽",
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

func (s *Server) redirectRoot(c *gin.Context) {
	c.Redirect(nethttp.StatusFound, "/chats")
}
