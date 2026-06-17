package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

const currentUserIDKey = "current_user_id"

// AuthMiddleware handles access token authentication.
type AuthMiddleware struct {
	tokens service.TokenService
}

// NewAuthMiddleware creates an auth middleware.
func NewAuthMiddleware(tokens service.TokenService) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens}
}

// RequireAuth requires a valid bearer access token.
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization is required"})
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(header, bearerPrefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization is invalid"})
			return
		}

		userID, err := m.tokens.ParseAccessToken(strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix)))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set(currentUserIDKey, userID.Hex())
		c.Next()
	}
}

// CurrentUserID returns the authenticated user id from Gin context.
func CurrentUserID(c *gin.Context) (string, bool) {
	value, ok := c.Get(currentUserIDKey)
	if !ok {
		return "", false
	}
	userID, ok := value.(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}
