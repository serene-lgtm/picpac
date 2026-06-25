package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakePackRepository struct {
	created       *domain.Pack
	listed        []domain.Pack
	searched      []domain.Pack
	listUserID    bson.ObjectID
	searchUserID  bson.ObjectID
	searchKeyword string
	gotPackID     bson.ObjectID
	got           *domain.Pack
	updated       *domain.Pack
	deleted       bson.ObjectID
	err           error
	updateErr     error
	deleteErr     error
}

func (r *fakePackRepository) Create(_ context.Context, pack *domain.Pack) error {
	if r.err != nil {
		return r.err
	}
	r.created = pack
	return nil
}

func (r *fakePackRepository) ListAll(_ context.Context) ([]domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.listed, nil
}

func (r *fakePackRepository) ListByUserID(_ context.Context, userID bson.ObjectID) ([]domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.listUserID = userID
	return r.listed, nil
}

func (r *fakePackRepository) SearchByKeyword(_ context.Context, keyword string) ([]domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakePackRepository) SearchByKeywordAndUserID(_ context.Context, userID bson.ObjectID, keyword string) ([]domain.Pack, error) {
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

func (r *fakePackRepository) GetByID(_ context.Context, packID bson.ObjectID) (*domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.gotPackID = packID
	if r.got == nil {
		return nil, mongo.ErrNoDocuments
	}
	return r.got, nil
}

func (r *fakePackRepository) Update(_ context.Context, pack *domain.Pack) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}
	r.updated = pack
	return nil
}

func (r *fakePackRepository) DeleteByID(_ context.Context, packID bson.ObjectID) error {
	if r.deleteErr != nil {
		return r.deleteErr
	}
	if r.err != nil {
		return r.err
	}
	r.deleted = packID
	return nil
}

type fakePackItemRepository struct {
	items     map[bson.ObjectID]*domain.Item
	getErr    error
	gotItemID bson.ObjectID
}

func (r *fakePackItemRepository) Create(_ context.Context, _ *domain.Item) error { return nil }
func (r *fakePackItemRepository) ListAll(_ context.Context) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakePackItemRepository) ListByUserID(_ context.Context, _ bson.ObjectID) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakePackItemRepository) SearchByKeyword(_ context.Context, _ string) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakePackItemRepository) SearchByKeywordAndUserID(_ context.Context, _ bson.ObjectID, _ string) ([]domain.Item, error) {
	return nil, nil
}
func (r *fakePackItemRepository) GetByID(_ context.Context, itemID bson.ObjectID) (*domain.Item, error) {
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
func (r *fakePackItemRepository) Update(_ context.Context, _ *domain.Item) error { return nil }
func (r *fakePackItemRepository) DeleteByID(_ context.Context, _ bson.ObjectID) error {
	return nil
}

func TestCreatePackStoresPack(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	repo := &fakePackRepository{}
	items := &fakePackItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: userID, Status: domain.ItemStatusCreated},
		},
	}
	svc := NewPackService(repo, items)

	pack, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name:        "日本出差",
		UserID:      userID.Hex(),
		Description: "东京 5 天商务行程",
		Items:       []string{itemID.Hex()},
	})
	if err != nil {
		t.Fatalf("CreatePack returned error: %v", err)
	}
	if repo.created == nil {
		t.Fatalf("expected repository create to be called")
	}
	if pack.UserID != userID {
		t.Fatalf("expected user id %s, got %s", userID.Hex(), pack.UserID.Hex())
	}
	if len(pack.Items) != 1 || pack.Items[0] != itemID {
		t.Fatalf("unexpected items: %+v", pack.Items)
	}
}

func TestCreatePackRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{}, &fakePackItemRepository{})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreatePackRejectsForeignItem(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{}, &fakePackItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: bson.NewObjectID(), Status: domain.ItemStatusCreated},
		},
	})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name:   "日本出差",
		UserID: userID.Hex(),
		Items:  []string{itemID.Hex()},
	})
	if err == nil || !strings.Contains(err.Error(), "pack item not found") {
		t.Fatalf("expected pack item not found, got %v", err)
	}
}

func TestCreatePackRejectsDeletedItem(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{}, &fakePackItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: userID, Status: domain.ItemStatusDeleted},
		},
	})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name:   "日本出差",
		UserID: userID.Hex(),
		Items:  []string{itemID.Hex()},
	})
	if err == nil || !strings.Contains(err.Error(), "pack item not found") {
		t.Fatalf("expected pack item not found, got %v", err)
	}
}

func TestListPacksReturnsCurrentUserResults(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakePackRepository{listed: []domain.Pack{{UserID: userID, Name: "日本出差", Status: domain.PackStatusCreated}}}
	svc := NewPackService(repo, &fakePackItemRepository{})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{UserID: userID.Hex()})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Name != "日本出差" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
	if repo.listUserID != userID {
		t.Fatalf("expected list by user id %s, got %s", userID.Hex(), repo.listUserID.Hex())
	}
}

func TestListPacksRejectsMissingUserID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{}, &fakePackItemRepository{})

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListPacksSearchesByKeyword(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	repo := &fakePackRepository{searched: []domain.Pack{{Name: "日本出差", Description: "东京 5 天", Status: domain.PackStatusCreated}}}
	svc := NewPackService(repo, &fakePackItemRepository{})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{
		UserID: userID.Hex(),
		Q:      "  东京  ",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Description != "东京 5 天" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
	if repo.searchUserID != userID || repo.searchKeyword != "东京" {
		t.Fatalf("unexpected search args: user=%s keyword=%q", repo.searchUserID.Hex(), repo.searchKeyword)
	}
}

func TestGetPackRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, UserID: bson.NewObjectID(), Status: domain.PackStatusCreated},
	}, &fakePackItemRepository{})

	_, err := svc.GetPack(context.Background(), packID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestUpdatePackReplacesEditableFields(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	originalUpdatedAt := time.Now().UTC().Add(-time.Hour)
	repo := &fakePackRepository{
		got: &domain.Pack{
			ID:          packID,
			UserID:      userID,
			Name:        "旧行程",
			Description: "old",
			Items:       []bson.ObjectID{bson.NewObjectID()},
			Status:      domain.PackStatusCreated,
			CreatedAt:   originalUpdatedAt,
			UpdatedAt:   originalUpdatedAt,
		},
	}
	items := &fakePackItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: userID, Status: domain.ItemStatusCreated},
		},
	}
	svc := NewPackService(repo, items)

	pack, err := svc.UpdatePack(context.Background(), packID.Hex(), userID.Hex(), request.UpdatePackInput{
		Name:        "日本出差",
		Description: "东京 6 天商务行程",
		Items:       []string{itemID.Hex()},
	})
	if err != nil {
		t.Fatalf("UpdatePack returned error: %v", err)
	}
	if repo.updated == nil || repo.updated.Name != "日本出差" {
		t.Fatalf("expected repository update to be called")
	}
	if !pack.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance, got %s", pack.UpdatedAt)
	}
}

func TestUpdatePackRejectsForeignItem(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, UserID: userID, Status: domain.PackStatusCreated},
	}, &fakePackItemRepository{
		items: map[bson.ObjectID]*domain.Item{
			itemID: {ID: itemID, UserID: bson.NewObjectID(), Status: domain.ItemStatusCreated},
		},
	})

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), userID.Hex(), request.UpdatePackInput{
		Name:  "日本出差",
		Items: []string{itemID.Hex()},
	})
	if err == nil || !strings.Contains(err.Error(), "pack item not found") {
		t.Fatalf("expected pack item not found, got %v", err)
	}
}

func TestDeletePackMarksStatusDeleted(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	userID := bson.NewObjectID()
	repo := &fakePackRepository{
		got: &domain.Pack{ID: packID, UserID: userID, Status: domain.PackStatusCreated},
	}
	svc := NewPackService(repo, &fakePackItemRepository{})

	err := svc.DeletePack(context.Background(), packID.Hex(), userID.Hex())
	if err != nil {
		t.Fatalf("DeletePack returned error: %v", err)
	}
	if repo.deleted != packID {
		t.Fatalf("expected repository delete to be called")
	}
}

func TestDeletePackRejectsForeignOwner(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, UserID: bson.NewObjectID(), Status: domain.PackStatusCreated},
	}, &fakePackItemRepository{})

	err := svc.DeletePack(context.Background(), packID.Hex(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestDeletePackWrapsRepositoryError(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	userID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got:       &domain.Pack{ID: packID, UserID: userID, Status: domain.PackStatusCreated},
		deleteErr: errors.New("database down"),
	}, &fakePackItemRepository{})

	err := svc.DeletePack(context.Background(), packID.Hex(), userID.Hex())
	if err == nil || !strings.Contains(err.Error(), "delete pack failed") {
		t.Fatalf("expected delete pack failure, got %v", err)
	}
}
