package handler

import (
	"bytes"
	"context"
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

type fakeAISuggestionHandlerService struct {
	recommendedItems []service.RecommendedItem
	err              error
	input            request.RecommendPackItemsInput
}

func (s *fakeAISuggestionHandlerService) RecommendPackItems(_ context.Context, input request.RecommendPackItemsInput) ([]service.RecommendedItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.input = input
	return s.recommendedItems, nil
}

func newAuthenticatedAIRouter(t *testing.T, aiService *fakeAISuggestionHandlerService) (*gin.Engine, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	aiHandler := NewAIHandler(aiService)
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authMiddleware := NewAuthMiddleware(tokenService, &fakeAuthService{user: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: "user-avatar/default.jpg"}, Status: domain.UserStatusCreated}})
	aiRoutes := router.Group("/api/v1/ai")
	aiRoutes.Use(authMiddleware.RequireAuth())
	aiRoutes.POST("/pack/item-recommendations", aiHandler.RecommendPackItems)

	return router, token, userID.Hex()
}

func TestRecommendPackItemsHandlerReturnsItems(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID().Hex()
	aiService := &fakeAISuggestionHandlerService{recommendedItems: []service.RecommendedItem{{ID: itemID, Name: "护照"}}}
	router, token, userID := newAuthenticatedAIRouter(t, aiService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/pack/item-recommendations", bytes.NewBufferString(`{"pack_name":" 日本出差 ","description":" 东京5天 "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), itemID) || !strings.Contains(recorder.Body.String(), "护照") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if aiService.input.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, aiService.input.UserID)
	}
	if aiService.input.PackName != "日本出差" || aiService.input.Description != "东京5天" {
		t.Fatalf("unexpected input: %+v", aiService.input)
	}
}

func TestRecommendPackItemsHandlerRequiresPackName(t *testing.T) {
	t.Parallel()

	router, token, _ := newAuthenticatedAIRouter(t, &fakeAISuggestionHandlerService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/pack/item-recommendations", bytes.NewBufferString(`{"description":"东京5天"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "pack_name is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestRecommendPackItemsHandlerMapsServiceValidationError(t *testing.T) {
	t.Parallel()

	router, token, _ := newAuthenticatedAIRouter(t, &fakeAISuggestionHandlerService{err: errors.New("description is too long")})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/pack/item-recommendations", bytes.NewBufferString(`{"pack_name":"日本出差","description":"东京5天"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}
