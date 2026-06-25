package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakeChecklistRepository struct {
	created        *domain.Checklist
	listed         []domain.Checklist
	searched       []domain.Checklist
	listUserID     bson.ObjectID
	searchUserID   bson.ObjectID
	searchKeyword  string
	gotChecklistID bson.ObjectID
	got            *domain.Checklist
	updated        *domain.Checklist
	statusUpdated  bool
	status         domain.LineItemStatus
	statusLineID   bson.ObjectID
	deleted        bson.ObjectID
	err            error
	updateErr      error
	deleteErr      error
}

func (r *fakeChecklistRepository) Create(_ context.Context, checklist *domain.Checklist) error {
	if r.err != nil {
		return r.err
	}
	r.created = checklist
	return nil
}

func (r *fakeChecklistRepository) ListAll(_ context.Context) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) ListByUserID(_ context.Context, userID bson.ObjectID) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.listUserID = userID
	return r.listed, nil
}

func (r *fakeChecklistRepository) SearchByKeyword(_ context.Context, keyword string) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) SearchByKeywordAndUserID(_ context.Context, userID bson.ObjectID, keyword string) ([]domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchUserID = userID
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakeChecklistRepository) GetByID(_ context.Context, checklistID bson.ObjectID) (*domain.Checklist, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.gotChecklistID = checklistID
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakeChecklistRepository) Update(_ context.Context, checklist *domain.Checklist) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.updated = checklist
	return nil
}

func (r *fakeChecklistRepository) UpdateLineItemStatus(_ context.Context, _ bson.ObjectID, lineItemID bson.ObjectID, status domain.LineItemStatus, updatedAt time.Time) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.statusUpdated = true
	r.statusLineID = lineItemID
	r.status = status
	if r.got != nil {
		r.got.UpdatedAt = updatedAt
		for i := range r.got.Items {
			if r.got.Items[i].ID == lineItemID {
				r.got.Items[i].Status = status
				return nil
			}
		}
	}
	return mongo.ErrNoDocuments
}

func (r *fakeChecklistRepository) DeleteByID(_ context.Context, checklistID bson.ObjectID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if r.err != nil {
		return r.err
	}
	r.deleted = checklistID
	return nil
}

type fakeChecklistItemRepository struct {
	items     map[bson.ObjectID]*domain.Item
	getErr    error
	gotItemID bson.ObjectID
}

func (r *fakeChecklistItemRepository) Create(_ context.Context, _ *domain.Item) error { return nil }
func (r *fakeChecklistItemRepository) ListAll(_ context.Context) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakeChecklistItemRepository) ListByUserID(_ context.Context, _ bson.ObjectID) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakeChecklistItemRepository) SearchByKeyword(_ context.Context, _ string) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakeChecklistItemRepository) SearchByKeywordAndUserID(_ context.Context, _ bson.ObjectID, _ string) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakeChecklistItemRepository) GetByID(_ context.Context, itemID bson.ObjectID) (*domain.Item, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	r.gotItemID = itemID
	item, ok := r.items[itemID]
	if !ok {
		return nil, mongo.ErrNoDocuments
	}
	return item, nil
}
func (r *fakeChecklistItemRepository) Update(_ context.Context, _ *domain.Item) error { return nil }
func (r *fakeChecklistItemRepository) DeleteByID(_ context.Context, _ bson.ObjectID) error {
	return nil
}

func validChecklistItemRepository(userID bson.ObjectID) *fakeChecklistItemRepository {
	itemID := bson.NewObjectID()
	return &fakeChecklistItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: userID, Status: domain.ItemStatusCreated},
		},
	}
}

func TestCreateChecklistStoresChecklist(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{}
	itemRepo := &fakeChecklistItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: userID, Status: domain.ItemStatusCreated},
		},
	}
	svc := NewChecklistService(repo, itemRepo)

	checklist, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		UserID:      userID.Hex(),
		Name:        "日本出差 checklist",
		Description: "东京 5 天",
		TargetDate:  "2026-07-01",
		Items: []request.ChecklistLineItemInput{
			{ReferenceType: string(domain.LineItemTypeItem), ReferenceID: itemID.Hex()},
			{ReferenceType: string(domain.LineItemTypeSnapshot), Snapshot: &request.ItemSnapshotInput{Name: "临时雨伞"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateChecklist returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatalf("expected repository create to be called")
	}
	if checklist.UserID != userID || checklist.Status != domain.ChecklistStatusCreated {
		t.Fatalf("unexpected checklist: %+v", checklist)
	}
}

func TestCreateChecklistRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, &fakeChecklistItemRepository{})

	_, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		Name:       "日本出差 checklist",
		TargetDate: "2026-07-01",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreateChecklistRejectsForeignReferenceItem(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	itemRepo := &fakeChecklistItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: bson.NewObjectID(), Status: domain.ItemStatusCreated},
		},
	}
	svc := NewChecklistService(&fakeChecklistRepository{}, itemRepo)

	_, err := svc.CreateChecklist(context.Background(), request.CreateChecklistInput{
		UserID:     userID.Hex(),
		Name:       "清单",
		TargetDate: "2026-07-01",
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeItem),
			ReferenceID:   itemID.Hex(),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item reference item not found") {
		t.Fatalf("expected reference item not found, got %v", err)
	}
}

func TestListChecklistsReturnsCurrentUserResults(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakeChecklistRepository{listed: []domain.Checklist{{UserID: userID, Name: "日本出差 checklist", Status: domain.ChecklistStatusCreated}}}
	svc := NewChecklistService(repo, validChecklistItemRepository(userID))

	checklists, err := svc.ListChecklists(context.Background(), request.ListChecklistsInput{UserID: userID.Hex()})
	if err != nil {
		t.Fatalf("ListChecklists returned error: %v", err)
	}
	if len(checklists) != 1 || checklists[0].Name != "日本出差 checklist" {
		t.Fatalf("unexpected checklists: %+v", checklists)
	}
	if repo.listUserID != userID {
		t.Fatalf("expected list by user id %s, got %s", userID.Hex(), repo.listUserID.Hex())
	}
}

func TestListChecklistsRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewChecklistService(&fakeChecklistRepository{}, &fakeChecklistItemRepository{})

	_, err := svc.ListChecklists(context.Background(), request.ListChecklistsInput{})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListChecklistsSearchesByKeyword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakeChecklistRepository{searched: []domain.Checklist{{Name: "日本出差 checklist", Description: "东京 5 天", Status: domain.ChecklistStatusCreated}}}
	svc := NewChecklistService(repo, validChecklistItemRepository(userID))

	checklists, err := svc.ListChecklists(context.Background(), request.ListChecklistsInput{
		UserID: userID.Hex(),
		Q:      "  东京  ",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListChecklists returned error: %v", err)
	}
	if len(checklists) != 1 || checklists[0].Description != "东京 5 天" {
		t.Fatalf("unexpected checklists: %+v", checklists)
	}
	if repo.searchUserID != userID || repo.searchKeyword != "东京" {
		t.Fatalf("unexpected search args: user=%s keyword=%q", repo.searchUserID.Hex(), repo.searchKeyword)
	}
}

func TestGetChecklistRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	svc := NewChecklistService(&fakeChecklistRepository{
		got: &domain.Checklist{ID: checklistID, UserID: bson.NewObjectID(), Status: domain.ChecklistStatusCreated},
	}, &fakeChecklistItemRepository{})

	_, err := svc.GetChecklist(context.Background(), checklistID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "checklist not found") {
		t.Fatalf("expected checklist not found, got %v", err)
	}
}

func TestUpdateChecklistReplacesEditableFields(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	userID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	originalUpdatedAt := time.Now().UTC().Add(-time.Hour)
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			UserID: userID,
			Name:   "旧清单",
			Items: []domain.LineItem{{
				ID:            lineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "旧快照"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status:    domain.ChecklistStatusCreated,
			UpdatedAt: originalUpdatedAt,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository(userID))

	checklist, err := svc.UpdateChecklist(context.Background(), checklistID.Hex(), userID.Hex(), request.UpdateChecklistInput{
		Name:       "新清单",
		TargetDate: "2026-07-02",
	})
	if err != nil {
		t.Fatalf("UpdateChecklist returned error: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "新清单" {
		t.Fatalf("expected repository update to be called")
	}
	if !checklist.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance")
	}
}

func TestAddChecklistLineItemsRejectsForeignReferenceItem(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{ID: checklistID, UserID: userID, Status: domain.ChecklistStatusCreated},
	}
	itemRepo := &fakeChecklistItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: bson.NewObjectID(), Status: domain.ItemStatusCreated},
		},
	}
	svc := NewChecklistService(repo, itemRepo)

	_, err := svc.AddChecklistLineItems(context.Background(), checklistID.Hex(), userID.Hex(), request.AddChecklistLineItemsInput{
		Items: []request.ChecklistLineItemInput{{
			ReferenceType: string(domain.LineItemTypeItem),
			ReferenceID:   itemID.Hex(),
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist line item reference item not found") {
		t.Fatalf("expected reference item not found, got %v", err)
	}
}

func TestRemoveChecklistLineItemsRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	svc := NewChecklistService(&fakeChecklistRepository{
		got: &domain.Checklist{ID: checklistID, UserID: bson.NewObjectID(), Status: domain.ChecklistStatusCreated},
	}, &fakeChecklistItemRepository{})

	_, err := svc.RemoveChecklistLineItems(context.Background(), checklistID.Hex(), bson.NewObjectID().Hex(), request.RemoveChecklistLineItemsInput{
		LineItemIDs: []string{bson.NewObjectID().Hex()},
	})
	if err == nil || !strings.Contains(err.Error(), "checklist not found") {
		t.Fatalf("expected checklist not found, got %v", err)
	}
}

func TestUpdateChecklistLineItemStatusChecksOwner(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	userID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			UserID: userID,
			Items: []domain.LineItem{{
				ID:            lineItemID,
				ReferenceType: domain.LineItemTypeSnapshot,
				Snapshot:      &domain.ItemSnapshot{Name: "护照"},
				Status:        domain.LineItemStatusUnchecked,
			}},
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository(userID))

	checklist, err := svc.UpdateChecklistLineItemStatus(context.Background(), checklistID.Hex(), lineItemID.Hex(), userID.Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusChecked),
	})
	if err != nil {
		t.Fatalf("UpdateChecklistLineItemStatus returned error: %v", err)
	}
	if !repo.statusUpdated || repo.statusLineID != lineItemID || repo.status != domain.LineItemStatusChecked {
		t.Fatalf("expected repository status update")
	}
	if checklist.Items[0].Status != domain.LineItemStatusChecked {
		t.Fatalf("expected checked status, got %+v", checklist.Items[0])
	}
}

func TestUpdateChecklistLineItemStatusRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	lineItemID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{
			ID:     checklistID,
			UserID: bson.NewObjectID(),
			Items:  []domain.LineItem{{ID: lineItemID, ReferenceType: domain.LineItemTypeSnapshot, Snapshot: &domain.ItemSnapshot{Name: "护照"}}},
			Status: domain.ChecklistStatusCreated,
		},
	}
	svc := NewChecklistService(repo, &fakeChecklistItemRepository{})

	_, err := svc.UpdateChecklistLineItemStatus(context.Background(), checklistID.Hex(), lineItemID.Hex(), bson.NewObjectID().Hex(), request.UpdateChecklistLineItemStatusInput{
		Status: string(domain.LineItemStatusChecked),
	})
	if err == nil || !strings.Contains(err.Error(), "checklist not found") {
		t.Fatalf("expected checklist not found, got %v", err)
	}
}

func TestDeleteChecklistMarksStatusDeleted(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	userID := bson.NewObjectID()
	repo := &fakeChecklistRepository{
		got: &domain.Checklist{ID: checklistID, UserID: userID, Status: domain.ChecklistStatusCreated},
	}
	svc := NewChecklistService(repo, validChecklistItemRepository(userID))

	err := svc.DeleteChecklist(context.Background(), checklistID.Hex(), userID.Hex())
	if err != nil {
		t.Fatalf("DeleteChecklist returned error: %v", err)
	}
	if repo.deleted != checklistID {
		t.Fatalf("expected repository delete to be called")
	}
}

func TestDeleteChecklistRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	checklistID := bson.NewObjectID()
	svc := NewChecklistService(&fakeChecklistRepository{
		got: &domain.Checklist{ID: checklistID, UserID: bson.NewObjectID(), Status: domain.ChecklistStatusCreated},
	}, &fakeChecklistItemRepository{})

	err := svc.DeleteChecklist(context.Background(), checklistID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "checklist not found") {
		t.Fatalf("expected checklist not found, got %v", err)
	}
}
