package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

func (s *fakeAuthService) defaultUser() *domain.User {
	if s.user != nil {
		return s.user
	}
	return &domain.User{ID: bson.NewObjectID(), DisplayName: "用户8000", Status: domain.UserStatusCreated}
}

func TestSendPhoneCodeHandlerReturnsSent(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{})
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
	authHandler := NewAuthHandler(&fakeAuthService{})
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
			DisplayName string `json:"display_name"`
		} `json:"user"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.AccessToken == "" || resp.RefreshToken == "" || resp.User.DisplayName != "用户8000" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRefreshHandlerReturnsAccessTokenOnly(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{})
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
	authHandler := NewAuthHandler(&fakeAuthService{})
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
	authHandler := NewAuthHandler(&fakeAuthService{err: errors.New("phone code send too frequently")})
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
	authHandler := NewAuthHandler(&fakeAuthService{user: &domain.User{ID: userID, DisplayName: "用户8000", Status: domain.UserStatusCreated}})
	authMiddleware := NewAuthMiddleware(tokenService)
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

func TestAuthMiddlewareRejectsMissingToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	router := gin.New()
	authHandler := NewAuthHandler(&fakeAuthService{})
	authMiddleware := NewAuthMiddleware(service.NewTokenService("test-secret", time.Hour))
	router.GET("/api/v1/me", authMiddleware.RequireAuth(), authHandler.Me)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}
