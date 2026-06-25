package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"
	"pack_mate/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type fakeChecklistService struct {
	checklist         *domain.Checklist
	checklists        []domain.Checklist
	err               error
	createInput       request.CreateChecklistInput
	listInput         request.ListChecklistsInput
	getChecklistID    string
	getUserID         string
	updateChecklistID string
	updateUserID      string
	updateInput       request.UpdateChecklistInput
	addChecklistID    string
	addUserID         string
	addInput          request.AddChecklistLineItemsInput
	removeChecklistID string
	removeUserID      string
	removeInput       request.RemoveChecklistLineItemsInput
	statusChecklistID string
	statusLineItemID  string
	statusUserID      string
	statusInput       request.UpdateChecklistLineItemStatusInput
	deleteChecklistID string
	deleteUserID      string
}

func (s *fakeChecklistService) CreateChecklist(_ context.Context, input request.CreateChecklistInput) (*domain.Checklist, error) {
	s.createInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) ListChecklists(_ context.Context, input request.ListChecklistsInput) ([]domain.Checklist, error) {
	s.listInput = input
	if s.err != nil {
		return nil, s.err
	}
	if s.checklists != nil {
		return s.checklists, nil
	}
	return []domain.Checklist{*s.defaultChecklist()}, nil
}

func (s *fakeChecklistService) GetChecklist(_ context.Context, checklistID string, userID string) (*domain.Checklist, error) {
	s.getChecklistID = checklistID
	s.getUserID = userID
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) UpdateChecklist(_ context.Context, checklistID string, userID string, input request.UpdateChecklistInput) (*domain.Checklist, error) {
	s.updateChecklistID = checklistID
	s.updateUserID = userID
	s.updateInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) AddChecklistLineItems(_ context.Context, checklistID string, userID string, input request.AddChecklistLineItemsInput) (*domain.Checklist, error) {
	s.addChecklistID = checklistID
	s.addUserID = userID
	s.addInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) RemoveChecklistLineItems(_ context.Context, checklistID string, userID string, input request.RemoveChecklistLineItemsInput) (*domain.Checklist, error) {
	s.removeChecklistID = checklistID
	s.removeUserID = userID
	s.removeInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) UpdateChecklistLineItemStatus(_ context.Context, checklistID string, lineItemID string, userID string, input request.UpdateChecklistLineItemStatusInput) (*domain.Checklist, error) {
	s.statusChecklistID = checklistID
	s.statusLineItemID = lineItemID
	s.statusUserID = userID
	s.statusInput = input
	if s.err != nil {
		return nil, s.err
	}
	return s.defaultChecklist(), nil
}

func (s *fakeChecklistService) DeleteChecklist(_ context.Context, checklistID string, userID string) error {
	s.deleteChecklistID = checklistID
	s.deleteUserID = userID
	return s.err
}

func (s *fakeChecklistService) defaultChecklist() *domain.Checklist {
	if s.checklist != nil {
		return s.checklist
	}
	return &domain.Checklist{
		ID:          bson.NewObjectID(),
		UserID:      bson.NewObjectID(),
		Name:        "日本出差 checklist",
		Description: "东京 5 天",
		TargetDate:  time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Items: []domain.LineItem{{
			ID:            bson.NewObjectID(),
			ReferenceID:   bson.NewObjectID(),
			ReferenceType: domain.LineItemTypeItem,
			Status:        domain.LineItemStatusUnchecked,
		}, {
			ID:            bson.NewObjectID(),
			ReferenceType: domain.LineItemTypeSnapshot,
			Snapshot:      &domain.ItemSnapshot{Name: "临时雨伞"},
			Status:        domain.LineItemStatusUnchecked,
		}},
		Status: domain.ChecklistStatusCreated,
	}
}

func newAuthenticatedChecklistRouter(t *testing.T, checklistService *fakeChecklistService) (*gin.Engine, string, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	checklistHandler := NewChecklistHandler(checklistService)
	tokenService := service.NewTokenService("test-secret", time.Hour)
	userID := bson.NewObjectID()
	token, err := tokenService.CreateAccessToken(userID)
	if err != nil {
		t.Fatalf("CreateAccessToken returned error: %v", err)
	}
	authMiddleware := NewAuthMiddleware(tokenService, &fakeAuthService{user: &domain.User{ID: userID, DisplayName: "用户8000", Status: domain.UserStatusCreated}})
	checklistRoutes := router.Group("/api/v1/checklist")
	checklistRoutes.Use(authMiddleware.RequireAuth())
	checklistRoutes.POST("", checklistHandler.CreateChecklist)
	checklistRoutes.GET("", checklistHandler.ListChecklists)
	checklistRoutes.GET("/:checklist_id", checklistHandler.GetChecklist)
	checklistRoutes.PUT("/:checklist_id", checklistHandler.UpdateChecklist)
	checklistRoutes.POST("/:checklist_id/items", checklistHandler.AddChecklistLineItems)
	checklistRoutes.DELETE("/:checklist_id/items", checklistHandler.RemoveChecklistLineItems)
	checklistRoutes.PATCH("/:checklist_id/items/:line_item_id/status", checklistHandler.UpdateChecklistLineItemStatus)
	checklistRoutes.DELETE("/:checklist_id", checklistHandler.DeleteChecklist)

	return router, token, userID.Hex()
}

func TestCreateChecklistHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	checklistService := &fakeChecklistService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedChecklistRouter(t, checklistService)

	body := bytes.NewBufferString(`{"name":"日本出差 checklist","target_date":"2026-07-01","items":[{"reference_type":"snapshot","snapshot":{"name":"临时雨伞"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if checklistService.createInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, checklistService.createInput.UserID)
	}
}

func TestChecklistRoutesRequireAuthorization(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, _, _ := newAuthenticatedChecklistRouter(t, &fakeChecklistService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist", nil)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListChecklistsHandlerUsesCurrentUserID(t *testing.T) {
	t.Parallel()

	checklistService := &fakeChecklistService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedChecklistRouter(t, checklistService)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if checklistService.listInput.UserID != userID {
		t.Fatalf("expected current user id %s, got %s", userID, checklistService.listInput.UserID)
	}
}

func TestListChecklistsHandlerRejectsEmptyQ(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedChecklistRouter(t, &fakeChecklistService{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist?q=+%20", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestGetChecklistHandlerMapsNotFound(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedChecklistRouter(t, &fakeChecklistService{err: errors.New("checklist not found")})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/checklist/"+bson.NewObjectID().Hex(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateChecklistHandlerRejectsItemsField(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	router, token, _ := newAuthenticatedChecklistRouter(t, &fakeChecklistService{})

	body := bytes.NewBufferString(`{"name":"清单","target_date":"2026-07-01","items":[]}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/checklist/"+bson.NewObjectID().Hex(), body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAddChecklistLineItemsHandlerPassesCurrentUserID(t *testing.T) {
	t.Parallel()

	checklistService := &fakeChecklistService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedChecklistRouter(t, checklistService)
	checklistID := bson.NewObjectID().Hex()

	body := bytes.NewBufferString(`{"items":[{"reference_type":"snapshot","snapshot":{"name":"临时雨伞"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/checklist/"+checklistID+"/items", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if checklistService.addUserID != userID || checklistService.addChecklistID != checklistID {
		t.Fatalf("unexpected add call: user=%s checklist=%s", checklistService.addUserID, checklistService.addChecklistID)
	}
}

func TestUpdateChecklistLineItemStatusHandlerPassesCurrentUserID(t *testing.T) {
	t.Parallel()

	checklistService := &fakeChecklistService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedChecklistRouter(t, checklistService)
	checklistID := bson.NewObjectID().Hex()
	lineItemID := bson.NewObjectID().Hex()

	body := bytes.NewBufferString(`{"status":"checked"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/checklist/"+checklistID+"/items/"+lineItemID+"/status", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if checklistService.statusUserID != userID || checklistService.statusChecklistID != checklistID || checklistService.statusLineItemID != lineItemID {
		t.Fatalf("unexpected status call: user=%s checklist=%s line=%s", checklistService.statusUserID, checklistService.statusChecklistID, checklistService.statusLineItemID)
	}
}

func TestDeleteChecklistHandlerReturnsDeletedTrue(t *testing.T) {
	t.Parallel()

	checklistService := &fakeChecklistService{}
	recorder := httptest.NewRecorder()
	router, token, userID := newAuthenticatedChecklistRouter(t, checklistService)
	checklistID := bson.NewObjectID().Hex()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/checklist/"+checklistID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if checklistService.deleteUserID != userID || checklistService.deleteChecklistID != checklistID {
		t.Fatalf("unexpected delete call: user=%s checklist=%s", checklistService.deleteUserID, checklistService.deleteChecklistID)
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
