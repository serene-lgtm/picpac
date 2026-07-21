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

type fakePackService struct {
	pack         *domain.Pack
	packs        []domain.Pack
	err          error
	createInput  request.CreatePackInput
	listInput    request.ListPacksInput
	gotPackID    string
	gotUserID    string
	updatePackID string
	updateUserID string
	profileInput request.UpdatePackProfileInput
	addInput     request.AddPackItemsInput
	removeInput  request.RemovePackItemsInput
	deletePackID string
	deleteUserID string
}

func (s *fakePackService) CreatePack(_ context.Context, input request.CreatePackInput) (*domain.Pack, error) {
	s.createInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) ListPacks(_ context.Context, input request.ListPacksInput) ([]domain.Pack, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	if s.packs != nil {
		return s.packs, nil
	}
	return []domain.Pack{*s.defaultPack()}, nil
}

func (s *fakePackService) GetPack(_ context.Context, packID string, userID string) (*domain.Pack, error) {
	s.gotPackID = packID
	s.gotUserID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) UpdatePackProfile(_ context.Context, packID string, userID string, input request.UpdatePackProfileInput) (*domain.Pack, error) {
	s.updatePackID = packID
	s.updateUserID = userID
	s.profileInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) AddPackItems(_ context.Context, packID string, userID string, input request.AddPackItemsInput) (*domain.Pack, error) {
	s.updatePackID = packID
	s.updateUserID = userID
	s.addInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) RemovePackItems(_ context.Context, packID string, userID string, input request.RemovePackItemsInput) (*domain.Pack, error) {
	s.updatePackID = packID
	s.updateUserID = userID
	s.removeInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultPack(), nil
}

func (s *fakePackService) DeletePack(_ context.Context, packID string, userID string) error {
	s.deletePackID = packID
	s.deleteUserID = userID
	return s.err
}

func (s *fakePackService) defaultPack() *domain.Pack {
	if s.pack != nil {
		return s.pack
	}
	return &domain.Pack{
		ID:          bson.NewObjectID(),
		UserID:      bson.NewObjectID(),
		Name:        "日本出差",
		Description: "东京 5 天商务行程",
		Items:       []bson.ObjectID{bson.NewObjectID()},
		Status:      domain.PackStatusCreated,
	}
}

func newAuthenticatedPackRouter(t *testing.T, packService *fakePackService) (*gin.Engine, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	packHandler := NewPackHandler(packService)
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authMiddleware := NewAuthMiddleware(tokenService, &fakeAuthService{user: &domain.User{ID: userID, Profile: domain.UserProfile{Username: "用户8000", AvatarObjectKey: "user-avatar/default.jpg"}, Status: domain.UserStatusCreated}})
	packRoutes := router.Group("/api/v1/pack")
	packRoutes.Use(authMiddleware.RequireAuth())
	packRoutes.POST("", packHandler.CreatePack)
	packRoutes.GET("", packHandler.ListPacks)
	packRoutes.GET("/:pack_id", packHandler.GetPack)
	packRoutes.PATCH("/:pack_id/profile", packHandler.UpdatePackProfile)
	packRoutes.POST("/:pack_id/items", packHandler.AddPackItems)
	packRoutes.DELETE("/:pack_id/items", packHandler.RemovePackItems)
	packRoutes.DELETE("/:pack_id", packHandler.DeletePack)

	return router, token, userID.Hex()
}

func TestCreatePackHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)
	itemID := bson.NewObjectID().Hex()

	body := bytes.NewBufferString(`{"name":"日本出差","items":["` + itemID + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.createInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, packService.createInput.UserID)
	}
}

func TestCreatePackHandlerRequiresName(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", bytes.NewBufferString(`{"description":"东京"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestCreatePackHandlerRejectsInvalidItemID(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack", bytes.NewBufferString(`{"name":"日本出差","items":["bad-item-id"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestPackRoutesRequireAuthorization(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, _, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListPacksHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.listInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, packService.listInput.UserID)
	}
}

func TestListPacksHandlerSearchesByQ(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{
		packs: []domain.Pack{{ID: bson.NewObjectID(), UserID: bson.NewObjectID(), Name: "日本出差", Description: "东京 5 天"}},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack?q=东京", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"description":"东京 5 天"`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestGetPackHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{err: errors.New("pack not found")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/pack/"+bson.NewObjectID().Hex(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdatePackProfileHandlerRequiresName(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pack/"+bson.NewObjectID().Hex()+"/profile", bytes.NewBufferString(`{"description":"东京"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestUpdatePackProfileHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)
	packID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/pack/"+packID+"/profile", bytes.NewBufferString(`{"name":"日本出差","description":"东京"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.updateUserID != userID || packService.updatePackID != packID {
		t.Fatalf("unexpected update call: user=%s pack=%s", packService.updateUserID, packService.updatePackID)
	}
	if packService.profileInput.Name != "日本出差" || packService.profileInput.Description != "东京" {
		t.Fatalf("unexpected profile input: %+v", packService.profileInput)
	}
}

func TestAddPackItemsHandlerRequiresItems(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack/"+bson.NewObjectID().Hex()+"/items", bytes.NewBufferString(`{"items":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAddPackItemsHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)
	packID := bson.NewObjectID().Hex()
	itemID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pack/"+packID+"/items", bytes.NewBufferString(`{"items":["`+itemID+`"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.updateUserID != userID || packService.updatePackID != packID {
		t.Fatalf("unexpected add call: user=%s pack=%s", packService.updateUserID, packService.updatePackID)
	}
	if len(packService.addInput.Items) != 1 || packService.addInput.Items[0] != itemID {
		t.Fatalf("unexpected add input: %+v", packService.addInput)
	}
}

func TestRemovePackItemsHandlerRequiresItems(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedPackRouter(t, &fakePackService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/"+bson.NewObjectID().Hex()+"/items", bytes.NewBufferString(`{"items":[]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestRemovePackItemsHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)
	packID := bson.NewObjectID().Hex()
	itemID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/"+packID+"/items", bytes.NewBufferString(`{"items":["`+itemID+`"]}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.updateUserID != userID || packService.updatePackID != packID {
		t.Fatalf("unexpected remove call: user=%s pack=%s", packService.updateUserID, packService.updatePackID)
	}
	if len(packService.removeInput.Items) != 1 || packService.removeInput.Items[0] != itemID {
		t.Fatalf("unexpected remove input: %+v", packService.removeInput)
	}
}

func TestDeletePackHandlerReturnsDeletedTrue(t *testing.T) {
	t.Parallel()

	packService := &fakePackService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedPackRouter(t, packService)
	packID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/pack/"+packID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if packService.deleteUserID != userID || packService.deletePackID != packID {
		t.Fatalf("unexpected delete call: user=%s pack=%s", packService.deleteUserID, packService.deletePackID)
	}
	var resp struct {
		Deleted bool `json:"deleted"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.Deleted {
		t.Fatalf("expected deleted=true")
	}
}
