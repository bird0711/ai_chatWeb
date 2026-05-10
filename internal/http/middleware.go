package http

import (
	"crypto/subtle"
	"errors"
	"log"
	nethttp "net/http"
	"runtime/debug"
	"strings"
	"time"

	"ai_chat/internal/domain"

	"github.com/gin-gonic/gin"
)

const csrfCookieName = "csrf_token"

func (s *Server) assignRequestID(c *gin.Context) {
	id := normalizeRequestID(c.GetHeader(requestIDHeaderName))
	if id == "" {
		id = newRequestID()
	}
	c.Set(requestIDContextKey, id)
	c.Writer.Header().Set(requestIDHeaderName, id)
	c.Next()
}

func (s *Server) logRequests(c *gin.Context) {
	start := time.Now()
	c.Next()

	log.Printf(
		"http_request request_id=%s method=%s path=%s status=%d duration_ms=%d client_ip=%q",
		requestID(c),
		c.Request.Method,
		c.Request.URL.RequestURI(),
		c.Writer.Status(),
		time.Since(start).Milliseconds(),
		c.ClientIP(),
	)
}

func (s *Server) recoverPanics(c *gin.Context) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf(
				"panic_recovered request_id=%s method=%s path=%s panic=%v stack=%q",
				requestID(c),
				c.Request.Method,
				c.Request.URL.RequestURI(),
				recovered,
				string(debug.Stack()),
			)
			if c.Writer.Written() {
				c.Abort()
				return
			}
			renderInternalServerError(c)
			c.Abort()
		}
	}()
	c.Next()
}

func (s *Server) ensureCSRFCookie(c *gin.Context) {
	token, err := c.Cookie(csrfCookieName)
	if err != nil || strings.TrimSpace(token) == "" {
		token, err = randomTokenHex(32)
		if err != nil {
			log.Printf("csrf_token_generation_failed request_id=%s method=%s path=%s error=%q", requestID(c), c.Request.Method, c.Request.URL.RequestURI(), err.Error())
			renderInternalServerError(c)
			c.Abort()
			return
		}
		setCSRFCookie(c, token)
	}
	c.Set("CSRFToken", token)
	c.Next()
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
		renderJSONError(c, nethttp.StatusUnauthorized, errors.New("login required"))
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
		Secure:   getenvBool("COOKIE_SECURE", false),
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
		Secure:   getenvBool("COOKIE_SECURE", false),
		SameSite: nethttp.SameSiteLaxMode,
	})
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
