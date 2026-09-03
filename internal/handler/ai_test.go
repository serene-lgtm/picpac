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
	draftItems       []service.ItemDraft
	err              error
	recommendInput   request.RecommendPackItemsInput
	draftInput       request.GenerateItemDraftsInput
}

func (s *fakeAISuggestionHandlerService) RecommendPackItems(_ context.Context, input request.RecommendPackItemsInput) ([]service.RecommendedItem, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.recommendInput = input
	return s.recommendedItems, nil
}

func (s *fakeAISuggestionHandlerService) GenerateItemDrafts(_ context.Context, input request.GenerateItemDraftsInput) ([]service.ItemDraft, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.draftInput = input
	return s.draftItems, nil
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
	aiRoutes.POST("/item-drafts", aiHandler.GenerateItemDrafts)
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
	if aiService.recommendInput.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID, aiService.recommendInput.UserID)
	}
	if aiService.recommendInput.PackName != "日本出差" || aiService.recommendInput.Description != "东京5天" {
		t.Fatalf("unexpected input: %+v", aiService.recommendInput)
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

func TestGenerateItemDraftsHandlerReturnsDrafts(t *testing.T) {
	t.Parallel()

	categoryID := bson.NewObjectID().Hex()
	aiService := &fakeAISuggestionHandlerService{draftItems: []service.ItemDraft{{
		Name:         "手机",
		CategoryID:   categoryID,
		CategoryKey:  "electronics",
		CategoryName: "电子设备",
	}}}
	router, token, userID := newAuthenticatedAIRouter(t, aiService)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/item-drafts", bytes.NewBufferString(`{"text":" 请帮我添加手机 "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "手机") || !strings.Contains(recorder.Body.String(), categoryID) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
	if aiService.draftInput.UserID != userID || aiService.draftInput.Text != "请帮我添加手机" {
		t.Fatalf("unexpected input: %+v", aiService.draftInput)
	}
}

func TestGenerateItemDraftsHandlerRequiresText(t *testing.T) {
	t.Parallel()

	router, token, _ := newAuthenticatedAIRouter(t, &fakeAISuggestionHandlerService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai/item-drafts", bytes.NewBufferString(`{"text":" "}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "text is required") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
