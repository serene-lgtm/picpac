package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/dto/response"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles auth HTTP requests.
type AuthHandler struct {
	svc       service.AuthService
	urlSigner service.ObjectURLSigner
}

// NewAuthHandler creates an auth handler.
func NewAuthHandler(svc service.AuthService, urlSigner service.ObjectURLSigner) *AuthHandler {
	return &AuthHandler{svc: svc, urlSigner: urlSigner}
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

	authResponse, err := h.buildAuthResponse(c.Request.Context(), result)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, authResponse)
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

	userResponse, err := h.buildUserResponse(c.Request.Context(), user)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, userResponse)
}

// UpdateMyProfile handles current user profile update requests.
func (h *AuthHandler) UpdateMyProfile(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	username := strings.TrimSpace(c.PostForm("username"))
	gender := strings.TrimSpace(c.PostForm("gender"))
	birthday := strings.TrimSpace(c.PostForm("birthday"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username is required"})
		return
	}
	if gender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gender is required"})
		return
	}

	input := request.UpdateMyProfileInput{
		Username: username,
		Gender:   gender,
		Birthday: birthday,
	}
	if file, header, err := c.Request.FormFile("avatar"); err == nil {
		defer file.Close()
		input.File = file
		input.FileName = header.Filename
	}

	user, err := h.svc.UpdateMyProfile(c.Request.Context(), userID, input)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	userResponse, err := h.buildUserResponse(c.Request.Context(), user)
	if err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, userResponse)
}

// DeleteMe handles current account deletion requests.
func (h *AuthHandler) DeleteMe(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.svc.DeleteMe(c.Request.Context(), userID); err != nil {
		respondAuthError(c, err)
		return
	}

	c.JSON(http.StatusOK, response.DeleteMeResponse{Deleted: true})
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
		strings.Contains(message, "refresh token is required"),
		strings.Contains(message, "username is required"),
		strings.Contains(message, "gender is required"),
		strings.Contains(message, "avatar is required"),
		strings.Contains(message, "username is too long"),
		strings.Contains(message, "gender is invalid"),
		strings.Contains(message, "birthday is invalid"):
		status = http.StatusBadRequest
	case strings.Contains(message, "access token is invalid"),
		strings.Contains(message, "access token is expired"),
		strings.Contains(message, "refresh token is invalid"),
		strings.Contains(message, "refresh token is expired"),
		strings.Contains(message, "refresh token is revoked"):
		status = http.StatusUnauthorized
	case strings.Contains(message, "user is disabled"):
		status = http.StatusForbidden
	case strings.Contains(message, "phone code send too frequently"):
		status = http.StatusTooManyRequests
	case strings.Contains(message, "user not found"):
		status = http.StatusNotFound
	case strings.Contains(message, "create auth identity failed"):
		status = http.StatusConflict
	case strings.Contains(message, "upload user avatar failed"):
		status = http.StatusBadGateway
	case strings.Contains(message, "sign user avatar url failed"):
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{"error": message})
}

func (h *AuthHandler) buildAuthResponse(ctx context.Context, result *service.AuthResult) (response.AuthResponse, error) {
	userResponse, err := h.buildUserResponse(ctx, result.User)
	if err != nil {
		return response.AuthResponse{}, err
	}

	return response.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User:         userResponse,
	}, nil
}

func (h *AuthHandler) buildUserResponse(ctx context.Context, user *domain.User) (response.UserResponse, error) {
	avatarURL, err := h.signAvatarURL(ctx, user.Profile.AvatarObjectKey)
	if err != nil {
		return response.UserResponse{}, err
	}

	return response.UserResponse{
		ID: user.ID.Hex(),
		Profile: response.UserProfileResponse{
			Username:  user.Profile.Username,
			Gender:    string(user.Profile.Gender),
			Birthday:  formatUserBirthday(user.Profile.Birthday),
			AvatarURL: avatarURL,
		},
		Status: string(user.Status),
	}, nil
}

func (h *AuthHandler) signAvatarURL(ctx context.Context, objectKey string) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", nil
	}
	if h.urlSigner == nil {
		return "", fmt.Errorf("sign user avatar url failed: url signer is not configured")
	}

	signedURL, err := h.urlSigner.SignGetURL(ctx, objectKey)
	if err != nil {
		return "", fmt.Errorf("sign user avatar url failed: %w", err)
	}

	return signedURL, nil
}

func formatUserBirthday(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}
