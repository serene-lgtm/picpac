package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeAuthService struct {
	err  error
	user *domain.User
}

type fakeObjectURLSigner struct {
	err error
}

func (s fakeObjectURLSigner) SignGetURL(_ context.Context, objectKey string) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return "https://signed.example/" + strings.TrimLeft(objectKey, "/") + "?Expires=3600&Signature=test", nil
}

func (s *fakeAuthService) SendPhoneCode(_ context.Context, _ request.SendPhoneCodeInput) error {
	return s.err
}

func (s *fakeAuthService) LoginWithPhone(_ context.Context, _ request.PhoneLoginInput) (*service.AuthResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &service.AuthResult{AccessToken: "access", RefreshToken: "refresh", User: s.defaultUser()}, nil
}

func (s *fakeAuthService) Refresh(_ context.Context, _ request.RefreshTokenInput) (*service.RefreshResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &service.RefreshResult{AccessToken: "access"}, nil
}

func (s *fakeAuthService) Logout(_ context.Context, _ request.LogoutInput) error {
	return s.err
}

func (s *fakeAuthService) Me(_ context.Context, _ string) (*domain.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultUser(), nil
}

func (s *fakeAuthService) UpdateMyProfile(_ context.Context, _ string, input request.UpdateMyProfileInput) (*domain.User, error) {
	if s.err != nil {
		return nil, s.err
	}

	user := s.defaultUser()
	user.Profile.Username = strings.TrimSpace(input.Username)
	user.Profile.Gender = domain.UserGender(strings.TrimSpace(input.Gender))
	if strings.TrimSpace(input.Birthday) != "" {
		birthday, err := time.Parse("2006-01-02", input.Birthday)
		if err != nil {
			return nil, err
		}
		user.Profile.Birthday = &birthday
	}
	user.Profile.AvatarObjectKey = "user-avatar/user_1.png"

	return user, nil
}

func (s *fakeAuthService) defaultUser() *domain.User {
	if s.user != nil {
		return s.user
	}
	return &domain.User{
		ID: bson.NewObjectID(),
		Profile: domain.UserProfile{
			Username:        "用户8000",
			Gender:          "",
			AvatarObjectKey: "user-avatar/default.jpg",
		},
		Status: domain.UserStatusCreated,
	}
}

func TestSendPhoneCodeHandlerReturnsSent(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{}, fakeObjectURLSigner{})
	router.POST("/api/v1/auth/phone/code", authHandler.SendPhoneCode)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/code", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"sent":true`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestLoginWithPhoneHandlerReturnsTokens(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{}, fakeObjectURLSigner{})
	router.POST("/api/v1/auth/phone/login", authHandler.LoginWithPhone)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/login", bytes.NewBufferString(`{"phone":"13800138000","code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			Profile struct {
				Username  string `json:"username"`
				AvatarURL string `json:"avatar_url"`
			} `json:"profile"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.User.Profile.Username != "用户8000" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.User.Profile.AvatarURL != "https://signed.example/user-avatar/default.jpg?Expires=3600&Signature=test" {
		t.Fatalf("unexpected avatar_url: %s", resp.User.Profile.AvatarURL)
	}
}

func TestRefreshHandlerReturnsAccessTokenOnly(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{}, fakeObjectURLSigner{})
	router.POST("/api/v1/auth/refresh", authHandler.Refresh)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{"refresh_token":"refresh"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["access_token"] == "" {
		t.Fatalf("expected access token, got %+v", resp)
	}
	if _, ok := resp["refresh_token"]; ok {
		t.Fatalf("did not expect refresh_token in response: %+v", resp)
	}
}

func TestLoginWithPhoneHandlerRejectsMissingCode(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{}, fakeObjectURLSigner{})
	router.POST("/api/v1/auth/phone/login", authHandler.LoginWithPhone)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/login", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAuthHandlerMapsTooFrequent(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{err: errors.New("phone code send too frequently")}, fakeObjectURLSigner{})
	router.POST("/api/v1/auth/phone/code", authHandler.SendPhoneCode)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/phone/code", bytes.NewBufferString(`{"phone":"13800138000"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func TestMeHandlerReturnsCurrentUser(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authHandler := NewAuthHandler(&fakeAuthService{user: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: "user-avatar/default.jpg"}, Status: domain.UserStatusCreated}}, fakeObjectURLSigner{})
	authMiddleware := NewAuthMiddleware(tokenService, authHandler.svc)
	router.GET("/api/v1/me", authMiddleware.RequireAuth(), authHandler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), userID.Hex()) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestUpdateMyProfileHandlerReturnsUpdatedUser(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authHandler := NewAuthHandler(&fakeAuthService{user: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "旧用户名", AvatarObjectKey: "user-avatar/default.jpg"}, Status: domain.UserStatusCreated}}, fakeObjectURLSigner{})
	authMiddleware := NewAuthMiddleware(tokenService, authHandler.svc)
	router.PUT("/api/v1/me/profile", authMiddleware.RequireAuth(), authHandler.UpdateMyProfile)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("username", "新用户"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	if err := writer.WriteField("gender", "female"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	if err := writer.WriteField("birthday", "1998-08-20"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	part, err := writer.CreateFormFile("avatar", "avatar.png")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := io.Copy(part, bytes.NewReader([]byte("fake image"))); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/profile", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"username":"新用户"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{}, fakeObjectURLSigner{})
	authMiddleware := NewAuthMiddleware(service.NewTokenService("test-secret", time.Hour), authHandler.svc)
	router.GET("/api/v1/me", authMiddleware.RequireAuth(), authHandler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAuthMiddlewareRejectsDeletedUser(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authHandler := NewAuthHandler(&fakeAuthService{err: errors.New("user not found")}, fakeObjectURLSigner{})
	authMiddleware := NewAuthMiddleware(tokenService, authHandler.svc)
	router.GET("/api/v1/me", authMiddleware.RequireAuth(), authHandler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}
