package mongodb

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNewDomainItemMapsLegacyImageToStablePhoto(t *testing.T) {
	t.Parallel()

	itemID := bson.NewObjectID()
	createdAt := time.Now().UTC()
	item := newDomainItem(itemDocument{
		ID:                   itemID,
		SourceImageObjectKey: "items/item_1/source.jpg",
		CreatedAt:            createdAt,
	})

	if len(item.Photos) != 1 {
		t.Fatalf("expected one legacy photo, got %+v", item.Photos)
	}
	if item.Photos[0].ID != itemID {
		t.Fatalf("expected legacy photo id to be stable item id, got %s", item.Photos[0].ID.Hex())
	}
	if item.Photos[0].SourceObjectKey != "items/item_1/source.jpg" || item.Photos[0].DisplayObjectKey != "items/item_1/source.jpg" {
		t.Fatalf("unexpected legacy photo: %+v", item.Photos[0])
	}
	if !item.Photos[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("expected legacy photo created_at %s, got %s", createdAt, item.Photos[0].CreatedAt)
	}
}
