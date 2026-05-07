package http

import (
	"strings"
	"time"
    "crypto/rand"
	"encoding/hex"
	"log"
	"crypto/subtle"
	"errors"

	"ai_chat/internal/domain"

	"github.com/gin-gonic/gin"
	nethttp "net/http"
)
const csrfCookieName = "csrf_token"

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
		Secure: getenvBool("COOKIE_SECURE", false),
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
		Secure: getenvBool("COOKIE_SECURE", false),
		SameSite: nethttp.SameSiteLaxMode,
	})
}

func randomTokenHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (s *Server) ensureCSRFCookie(c *gin.Context) {
	token, err := c.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(token) == "" {
		token, err = randomTokenHex(32)
		if err != nil {
			log.Printf("csrf token generation failed: %v", err)
			c.AbortWithStatus(nethttp.StatusInternalServerError)
			return
		}
		setCSRFCookie(c, token)
	}
	c.Set("CSRFToken", token)
	c.Next()
}

func setCSRFCookie(c *gin.Context, token string) {
	nethttp.SetCookie(c.Writer, &nethttp.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   getenvBool("COOKIE_SECURE", false),
		SameSite: nethttp.SameSiteLaxMode,
	})
}

func (s *Server) requireCSRF(c *gin.Context) {
	switch c.Request.Method {
	case nethttp.MethodPost, nethttp.MethodPut, nethttp.MethodPatch, nethttp.MethodDelete:
	default:
		c.Next()
		return
	}

	cookieToken, err := c.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(cookieToken) == "" {
		renderError(c, nethttp.StatusForbidden, errors.New("csrf token missing"))
		c.Abort()
		return
	}

	requestToken := strings.TrimSpace(c.GetHeader("X-CSRF-Token"))
	if requestToken == "" {
		requestToken = strings.TrimSpace(c.PostForm("csrf_token"))
	}
	if requestToken == "" {
		renderError(c, nethttp.StatusForbidden, errors.New("csrf token missing"))
		c.Abort()
		return
	}

	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(requestToken)) != 1 {
		renderError(c, nethttp.StatusForbidden, errors.New("csrf token invalid"))
		c.Abort()
		return
	}

	c.Next()
}
