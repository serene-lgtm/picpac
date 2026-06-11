package mongodb

import (
	"context"
	"regexp"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const checklistCollectionName = "checklists"

type checklistDocument struct {
	ID          bson.ObjectID           `json:"id" bson:"_id,omitempty"`
	UserID      bson.ObjectID           `json:"user_id" bson:"uid"`
	Name        string                  `json:"name" bson:"nm"`
	Description string                  `json:"description" bson:"desc"`
	TargetDate  time.Time               `json:"target_date" bson:"tdt"`
	Items       []checklistItemDocument `json:"items" bson:"itm"`
	Status      domain.ChecklistStatus  `json:"status" bson:"st"`
	CreatedAt   time.Time               `json:"created_at" bson:"cat"`
	UpdatedAt   time.Time               `json:"updated_at" bson:"uat"`
}

type checklistItemDocument struct {
	ID            bson.ObjectID         `json:"id" bson:"_id"`
	ReferenceID   bson.ObjectID         `json:"reference_id" bson:"rid"`
	ReferenceType domain.LineItemType   `json:"reference_type" bson:"rt"`
	Snapshot      *itemSnapshotDocument `json:"snapshot" bson:"snap"`
	Status        domain.LineItemStatus `json:"status" bson:"st"`
}

type itemSnapshotDocument struct {
	Name string `json:"name" bson:"nm"`
}

// ChecklistRepository stores checklist domain models in MongoDB.
type ChecklistRepository struct {
	collection *mongo.Collection
}

// NewChecklistRepository creates a MongoDB-backed checklist repository.
func NewChecklistRepository(db *mongo.Database) *ChecklistRepository {
	return &ChecklistRepository{
		collection: db.Collection(checklistCollectionName),
	}
}

// Create inserts a checklist document into MongoDB.
func (r *ChecklistRepository) Create(ctx context.Context, checklist *domain.Checklist) error {
	_, err := r.collection.InsertOne(ctx, newChecklistDocument(checklist))
	return err
}

// ListAll returns all non-deleted checklists ordered by creation time descending.
func (r *ChecklistRepository) ListAll(ctx context.Context) ([]domain.Checklist, error) {
	docs, err := r.find(ctx, bson.M{"st": bson.M{"$ne": domain.ChecklistStatusDeleted}})
	if err != nil {
		return nil, err
	}

	return newDomainChecklists(docs), nil
}

// ListByUserID returns all non-deleted checklists owned by the given user ordered by creation time descending.
func (r *ChecklistRepository) ListByUserID(ctx context.Context, userID bson.ObjectID) ([]domain.Checklist, error) {
	docs, err := r.find(ctx, bson.M{
		"uid": userID,
		"st":  bson.M{"$ne": domain.ChecklistStatusDeleted},
	})
	if err != nil {
		return nil, err
	}

	return newDomainChecklists(docs), nil
}

// SearchByKeyword returns non-deleted checklists whose names or descriptions contain the keyword.
func (r *ChecklistRepository) SearchByKeyword(ctx context.Context, keyword string) ([]domain.Checklist, error) {
	docs, err := r.find(ctx, checklistKeywordSearchFilter(keyword))
	if err != nil {
		return nil, err
	}

	return newDomainChecklists(docs), nil
}

// SearchByKeywordAndUserID returns non-deleted user checklists whose names or descriptions contain the keyword.
func (r *ChecklistRepository) SearchByKeywordAndUserID(ctx context.Context, userID bson.ObjectID, keyword string) ([]domain.Checklist, error) {
	filter := checklistKeywordSearchFilter(keyword)
	filter["uid"] = userID

	docs, err := r.find(ctx, filter)
	if err != nil {
		return nil, err
	}

	return newDomainChecklists(docs), nil
}

// GetByID returns the non-deleted checklist with the given ID.
func (r *ChecklistRepository) GetByID(ctx context.Context, checklistID bson.ObjectID) (*domain.Checklist, error) {
	var doc checklistDocument
	if err := r.collection.FindOne(ctx, bson.M{
		"_id": checklistID,
		"st":  bson.M{"$ne": domain.ChecklistStatusDeleted},
	}).Decode(&doc); err != nil {
		return nil, err
	}

	checklist := newDomainChecklist(doc)
	return &checklist, nil
}

// Update replaces the non-deleted checklist document with the given ID.
func (r *ChecklistRepository) Update(ctx context.Context, checklist *domain.Checklist) error {
	doc := newChecklistDocument(checklist)
	result, err := r.collection.ReplaceOne(ctx, bson.M{
		"_id": doc.ID,
		"st":  bson.M{"$ne": domain.ChecklistStatusDeleted},
	}, doc)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// UpdateLineItemStatus updates a checklist line item status.
func (r *ChecklistRepository) UpdateLineItemStatus(ctx context.Context, checklistID bson.ObjectID, lineItemID bson.ObjectID, status domain.LineItemStatus, updatedAt time.Time) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id":     checklistID,
		"st":      bson.M{"$ne": domain.ChecklistStatusDeleted},
		"itm._id": lineItemID,
	}, bson.M{
		"$set": bson.M{
			"itm.$.st": status,
			"uat":      updatedAt,
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// DeleteByID logically deletes the checklist with the given ID.
func (r *ChecklistRepository) DeleteByID(ctx context.Context, checklistID bson.ObjectID) error {
	result, err := r.collection.UpdateOne(ctx, bson.M{
		"_id": checklistID,
		"st":  bson.M{"$ne": domain.ChecklistStatusDeleted},
	}, bson.M{
		"$set": bson.M{
			"st":  domain.ChecklistStatusDeleted,
			"uat": time.Now().UTC(),
		},
	})
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func checklistKeywordSearchFilter(keyword string) bson.M {
	keywordPattern := bson.M{
		"$regex":   regexp.QuoteMeta(keyword),
		"$options": "i",
	}

	return bson.M{
		"$or": bson.A{
			bson.M{"nm": keywordPattern},
			bson.M{"desc": keywordPattern},
		},
		"st": bson.M{"$ne": domain.ChecklistStatusDeleted},
	}
}

func (r *ChecklistRepository) find(ctx context.Context, filter bson.M) ([]checklistDocument, error) {
	opts := options.Find().SetSort(bson.D{{Key: "cat", Value: -1}})
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []checklistDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	if docs == nil {
		return []checklistDocument{}, nil
	}

	return docs, nil
}

func newChecklistDocument(checklist *domain.Checklist) checklistDocument {
	return checklistDocument{
		ID:          checklist.ID,
		UserID:      checklist.UserID,
		Name:        checklist.Name,
		Description: checklist.Description,
		TargetDate:  checklist.TargetDate,
		Items:       newChecklistItemDocuments(checklist.Items),
		Status:      checklist.Status,
		CreatedAt:   checklist.CreatedAt,
		UpdatedAt:   checklist.UpdatedAt,
	}
}

func newChecklistItemDocuments(items []domain.LineItem) []checklistItemDocument {
	docs := make([]checklistItemDocument, 0, len(items))
	for _, item := range items {
		docs = append(docs, checklistItemDocument{
			ID:            item.ID,
			ReferenceID:   item.ReferenceID,
			ReferenceType: item.ReferenceType,
			Snapshot:      newItemSnapshotDocument(item.Snapshot),
			Status:        item.Status,
		})
	}

	return docs
}

func newItemSnapshotDocument(snapshot *domain.ItemSnapshot) *itemSnapshotDocument {
	if snapshot == nil {
		return nil
	}
	return &itemSnapshotDocument{Name: snapshot.Name}
}

func newDomainChecklist(doc checklistDocument) domain.Checklist {
	return domain.Checklist{
		ID:          doc.ID,
		UserID:      doc.UserID,
		Name:        doc.Name,
		Description: doc.Description,
		TargetDate:  doc.TargetDate,
		Items:       newDomainLineItems(doc.Items),
		Status:      doc.Status,
		CreatedAt:   doc.CreatedAt,
		UpdatedAt:   doc.UpdatedAt,
	}
}

func newDomainLineItems(docs []checklistItemDocument) []domain.LineItem {
	items := make([]domain.LineItem, 0, len(docs))
	for _, doc := range docs {
		items = append(items, domain.LineItem{
			ID:            doc.ID,
			ReferenceID:   doc.ReferenceID,
			ReferenceType: doc.ReferenceType,
			Snapshot:      newDomainItemSnapshot(doc.Snapshot),
			Status:        doc.Status,
		})
	}

	return items
}

func newDomainItemSnapshot(doc *itemSnapshotDocument) *domain.ItemSnapshot {
	if doc == nil {
		return nil
	}
	return &domain.ItemSnapshot{Name: doc.Name}
}

func newDomainChecklists(docs []checklistDocument) []domain.Checklist {
	checklists := make([]domain.Checklist, 0, len(docs))
	for _, doc := range docs {
		checklists = append(checklists, newDomainChecklist(doc))
	}

	return checklists
}
