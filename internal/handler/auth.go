package handler

import (
	"net/http"
	"strings"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles auth HTTP requests.
type AuthHandler struct {
	svc service.AuthService
}

// NewAuthHandler creates an auth handler.
func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// SendPhoneCode handles phone code sending requests.
func (h *AuthHandler) SendPhoneCode(c *gin.Context) {
	var input request.SendPhoneCodeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if strings.TrimSpace(input.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}

	if err := h.svc.SendPhoneCode(c.Request.Context(), input); err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.SendPhoneCodeResponse{Sent: true})
}

// LoginWithPhone handles phone login requests.
func (h *AuthHandler) LoginWithPhone(c *gin.Context) {
	var input request.PhoneLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if strings.TrimSpace(input.Phone) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone is required"})
		return
	}
	if strings.TrimSpace(input.Code) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	result, err := h.svc.LoginWithPhone(c.Request.Context(), input)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildAuthResponse(result))
}

// Refresh handles refresh token requests.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var input request.RefreshTokenInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if strings.TrimSpace(input.RefreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	result, err := h.svc.Refresh(c.Request.Context(), input)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.RefreshAccessTokenResponse{
		AccessToken: result.AccessToken,
	})
}

// Logout handles logout requests.
func (h *AuthHandler) Logout(c *gin.Context) {
	var input request.LogoutInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
		return
	}
	if strings.TrimSpace(input.RefreshToken) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refresh_token is required"})
		return
	}

	if err := h.svc.Logout(c.Request.Context(), input); err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.LogoutResponse{LoggedOut: true})
}

// Me handles current user requests.
func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.svc.Me(c.Request.Context(), userID)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, buildUserResponse(user))
}

func respondAuthError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := err.Error()
	switch {
	case strings.Contains(message, "invalid input"),
		strings.Contains(message, "phone is required"),
		strings.Contains(message, "phone is invalid"),
		strings.Contains(message, "phone code is required"),
		strings.Contains(message, "phone code is invalid"),
		strings.Contains(message, "code is required"),
		strings.Contains(message, "refresh token is required"):
		status = http.StatusBadRequest
	case strings.Contains(message, "access token is invalid"),
		strings.Contains(message, "access token is expired"),
		strings.Contains(message, "refresh token is invalid"),
		strings.Contains(message, "refresh token is expired"),
		strings.Contains(message, "refresh token is revoked"):
		status = http.StatusUnauthorized
	case strings.Contains(message, "phone code send too frequently"):
		status = http.StatusTooManyRequests
	case strings.Contains(message, "user not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "create auth identity failed"):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": message})
}

func buildAuthResponse(result *service.AuthResult) response.AuthResponse {
	return response.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         buildUserResponse(result.User),
	}
}

func buildUserResponse(user *domain.User) response.UserResponse {
	return response.UserResponse{
		ID:          user.ID.Hex(),
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Status:      string(user.Status),
	}
}
