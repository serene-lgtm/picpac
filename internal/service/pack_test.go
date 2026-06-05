package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pack_mate/internal/domain"
	"pack_mate/internal/dto/request"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type fakePackRepository struct {
	created       *domain.Pack
	listed        []domain.Pack
	searched      []domain.Pack
	searchKeyword string
	got           *domain.Pack
	updated       *domain.Pack
	deleted       bson.ObjectID
	err           error
	updateErr     error
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

func (r *fakePackRepository) ListByUserID(_ context.Context, _ bson.ObjectID) ([]domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
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

func (r *fakePackRepository) SearchByKeywordAndUserID(_ context.Context, _ bson.ObjectID, keyword string) ([]domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
	r.searchKeyword = keyword
	if r.searched != nil {
		return r.searched, nil
	}
	return r.listed, nil
}

func (r *fakePackRepository) GetByID(_ context.Context, _ bson.ObjectID) (*domain.Pack, error) {
	if r.err != nil {
		return nil, r.err
	}
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
	if r.err != nil {
		return r.err
	}
	r.deleted = packID
	return nil
}

func TestCreatePackStoresPack(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	itemID := bson.NewObjectID()
	repo := &fakePackRepository{}
	svc := NewPackService(repo)

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
	if pack.Name != "日本出差" || pack.Description != "东京 5 天商务行程" {
		t.Fatalf("unexpected pack: %+v", pack)
	}
	if pack.Status != domain.PackStatusCreated {
		t.Fatalf("expected status=created, got %s", pack.Status)
	}
	if len(pack.Items) != 1 || pack.Items[0] != itemID {
		t.Fatalf("unexpected items: %+v", pack.Items)
	}
}

func TestCreatePackAllowsOptionalFields(t *testing.T) {
	t.Parallel()

	repo := &fakePackRepository{}
	svc := NewPackService(repo)

	pack, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name: "周末短途",
	})
	if err != nil {
		t.Fatalf("CreatePack returned error: %v", err)
	}
	if !pack.UserID.IsZero() {
		t.Fatalf("expected zero user id, got %s", pack.UserID.Hex())
	}
	if len(pack.Items) != 0 {
		t.Fatalf("expected empty items, got %+v", pack.Items)
	}
}

func TestCreatePackRejectsMissingName(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{})
	if err == nil || !strings.Contains(err.Error(), "pack name is required") {
		t.Fatalf("expected pack name error, got %v", err)
	}
}

func TestCreatePackRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name:   "日本出差",
		UserID: "bad-user-id",
	})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreatePackRejectsInvalidItemID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{
		Name:  "日本出差",
		Items: []string{"bad-item-id"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreatePackMapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: errors.New("database down")})

	_, err := svc.CreatePack(context.Background(), request.CreatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "create pack failed") {
		t.Fatalf("expected create pack failure, got %v", err)
	}
}

func TestListPacksReturnsRepositoryResults(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Pack{{UserID: userID, Name: "日本出差", Status: domain.PackStatusCreated}}
	svc := NewPackService(&fakePackRepository{listed: expected})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{UserID: userID.Hex()})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Name != "日本出差" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
}

func TestListPacksAllowsMissingUserID(t *testing.T) {
	t.Parallel()

	expected := []domain.Pack{{Name: "日本出差", Status: domain.PackStatusCreated}}
	svc := NewPackService(&fakePackRepository{listed: expected})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Name != "日本出差" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
}

func TestListPacksRejectsInvalidUserID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{UserID: "bad-user-id"})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListPacksMapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: errors.New("database down")})

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{})
	if err == nil || !strings.Contains(err.Error(), "list packs failed") {
		t.Fatalf("expected list packs failure, got %v", err)
	}
}

func TestListPacksSearchesByNameKeyword(t *testing.T) {
	t.Parallel()

	expected := []domain.Pack{{Name: "日本出差", Description: "东京 5 天商务行程", Status: domain.PackStatusCreated}}
	repo := &fakePackRepository{searched: expected}
	svc := NewPackService(repo)

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{
		Q:    "  出差  ",
		HasQ: true,
	})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Name != "日本出差" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
	if repo.searchKeyword != "出差" {
		t.Fatalf("expected trimmed keyword, got %q", repo.searchKeyword)
	}
}

func TestListPacksSearchesByDescriptionKeyword(t *testing.T) {
	t.Parallel()

	expected := []domain.Pack{{Name: "商务行程", Description: "东京 5 天", Status: domain.PackStatusCreated}}
	svc := NewPackService(&fakePackRepository{searched: expected})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{
		Q:    "东京",
		HasQ: true,
	})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Description != "东京 5 天" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
}

func TestListPacksSearchesByUserID(t *testing.T) {
	t.Parallel()

	userID := bson.NewObjectID()
	expected := []domain.Pack{{UserID: userID, Name: "日本出差", Status: domain.PackStatusCreated}}
	svc := NewPackService(&fakePackRepository{searched: expected})

	packs, err := svc.ListPacks(context.Background(), request.ListPacksInput{
		UserID: userID.Hex(),
		Q:      "出差",
		HasQ:   true,
	})
	if err != nil {
		t.Fatalf("ListPacks returned error: %v", err)
	}
	if len(packs) != 1 || packs[0].Name != "日本出差" {
		t.Fatalf("unexpected packs: %+v", packs)
	}
}

func TestListPacksRejectsEmptySearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "pack search keyword is required") {
		t.Fatalf("expected keyword required error, got %v", err)
	}
}

func TestListPacksRejectsTooLongSearchKeyword(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})
	keyword := strings.Repeat("行", maxPackSearchKeywordRunes+1)

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{Q: keyword, HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "pack search keyword is too long") {
		t.Fatalf("expected keyword too long error, got %v", err)
	}
}

func TestListPacksWrapsSearchRepositoryError(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: errors.New("database down")})

	_, err := svc.ListPacks(context.Background(), request.ListPacksInput{Q: "出差", HasQ: true})
	if err == nil || !strings.Contains(err.Error(), "list packs failed") {
		t.Fatalf("expected list packs failure, got %v", err)
	}
}

func TestGetPackReturnsRepositoryResult(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	expected := &domain.Pack{ID: packID, Name: "日本出差", Status: domain.PackStatusCreated}
	svc := NewPackService(&fakePackRepository{got: expected})

	pack, err := svc.GetPack(context.Background(), packID.Hex())
	if err != nil {
		t.Fatalf("GetPack returned error: %v", err)
	}
	if pack.Name != "日本出差" {
		t.Fatalf("unexpected pack: %+v", pack)
	}
}

func TestGetPackRejectsInvalidID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.GetPack(context.Background(), "bad-pack-id")
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestGetPackMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.GetPack(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestGetPackHidesDeletedPack(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{
			ID:     packID,
			Status: domain.PackStatusDeleted,
		},
	})

	_, err := svc.GetPack(context.Background(), packID.Hex())
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestGetPackMapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: errors.New("database down")})

	_, err := svc.GetPack(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "get pack failed") {
		t.Fatalf("expected get pack failure, got %v", err)
	}
}

func TestUpdatePackReplacesEditableFields(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	originalUpdatedAt := bson.NewObjectID().Timestamp()
	itemID := bson.NewObjectID()
	repo := &fakePackRepository{
		got: &domain.Pack{
			ID:          packID,
			UserID:      bson.NewObjectID(),
			Name:        "旧行程",
			Description: "old",
			Items:       []bson.ObjectID{bson.NewObjectID()},
			Status:      domain.PackStatusCreated,
			CreatedAt:   originalUpdatedAt,
			UpdatedAt:   originalUpdatedAt,
		},
	}
	svc := NewPackService(repo)

	pack, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{
		Name:        "日本出差",
		Description: "东京 6 天商务行程",
		Items:       []string{itemID.Hex()},
	})
	if err != nil {
		t.Fatalf("UpdatePack returned error: %v", err)
	}
	if repo.updated == nil {
		t.Fatalf("expected repository update to be called")
	}
	if pack.Name != "日本出差" || pack.Description != "东京 6 天商务行程" {
		t.Fatalf("unexpected pack: %+v", pack)
	}
	if pack.Status != domain.PackStatusCreated {
		t.Fatalf("expected status to be preserved, got %s", pack.Status)
	}
	if len(pack.Items) != 1 || pack.Items[0] != itemID {
		t.Fatalf("unexpected items: %+v", pack.Items)
	}
	if !pack.UpdatedAt.After(originalUpdatedAt) {
		t.Fatalf("expected updated_at to advance, got %s", pack.UpdatedAt)
	}
}

func TestUpdatePackClearsItemsWhenEmpty(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	repo := &fakePackRepository{
		got: &domain.Pack{
			ID:     packID,
			Name:   "旧行程",
			Items:  []bson.ObjectID{bson.NewObjectID()},
			Status: domain.PackStatusCreated,
		},
	}
	svc := NewPackService(repo)

	pack, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{
		Name:  "日本出差",
		Items: []string{},
	})
	if err != nil {
		t.Fatalf("UpdatePack returned error: %v", err)
	}
	if len(pack.Items) != 0 {
		t.Fatalf("expected items to be cleared, got %+v", pack.Items)
	}
}

func TestUpdatePackRejectsMissingName(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, Status: domain.PackStatusCreated},
	})

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{})
	if err == nil || !strings.Contains(err.Error(), "pack name is required") {
		t.Fatalf("expected pack name error, got %v", err)
	}
}

func TestUpdatePackRejectsInvalidItemID(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, Status: domain.PackStatusCreated},
	})

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{
		Name:  "日本出差",
		Items: []string{"bad-item-id"},
	})
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestUpdatePackMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	_, err := svc.UpdatePack(context.Background(), bson.NewObjectID().Hex(), request.UpdatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestUpdatePackHidesDeletedPack(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	svc := NewPackService(&fakePackRepository{
		got: &domain.Pack{ID: packID, Status: domain.PackStatusDeleted},
	})

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestUpdatePackMapsGetRepositoryFailure(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	repo := &fakePackRepository{
		got: &domain.Pack{ID: packID, Status: domain.PackStatusCreated},
		err: errors.New("database down"),
	}
	svc := NewPackService(repo)

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "get pack failed") {
		t.Fatalf("expected get pack failure, got %v", err)
	}
}

func TestUpdatePackMapsUpdateRepositoryFailure(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	repo := &fakePackRepository{
		got:       &domain.Pack{ID: packID, Status: domain.PackStatusCreated},
		updateErr: errors.New("database down"),
	}
	svc := NewPackService(repo)

	_, err := svc.UpdatePack(context.Background(), packID.Hex(), request.UpdatePackInput{Name: "日本出差"})
	if err == nil || !strings.Contains(err.Error(), "update pack failed") {
		t.Fatalf("expected update pack failure, got %v", err)
	}
}

func TestDeletePackDeletesByID(t *testing.T) {
	t.Parallel()

	packID := bson.NewObjectID()
	repo := &fakePackRepository{}
	svc := NewPackService(repo)

	if err := svc.DeletePack(context.Background(), packID.Hex()); err != nil {
		t.Fatalf("DeletePack returned error: %v", err)
	}
	if repo.deleted != packID {
		t.Fatalf("expected deleted pack id %s, got %s", packID.Hex(), repo.deleted.Hex())
	}
}

func TestDeletePackRejectsInvalidID(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{})

	err := svc.DeletePack(context.Background(), "bad-pack-id")
	if err == nil || !strings.Contains(err.Error(), "invalid input") {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestDeletePackMapsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: mongo.ErrNoDocuments})

	err := svc.DeletePack(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "pack not found") {
		t.Fatalf("expected pack not found, got %v", err)
	}
}

func TestDeletePackMapsRepositoryFailure(t *testing.T) {
	t.Parallel()

	svc := NewPackService(&fakePackRepository{err: errors.New("database down")})

	err := svc.DeletePack(context.Background(), bson.NewObjectID().Hex())
	if err == nil || !strings.Contains(err.Error(), "delete pack failed") {
		t.Fatalf("expected delete pack failure, got %v", err)
	}
}
