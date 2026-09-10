package mongodb

import (
	"testing"
	"time"

	"pack_mate/internal/domain"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestUserPasswordCredentialDocumentRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	lockedUntil := now.Add(15 * time.Minute)
	lastUsedAt := now.Add(-time.Hour)
	credential := &domain.UserPasswordCredential{
		ID:                 bson.NewObjectID(),
		UserID:             bson.NewObjectID(),
		PasswordHash:       "$2a$10$hash",
		PasswordAlgo:       "bcrypt",
		FailedAttemptCount: 3,
		LockedUntil:        &lockedUntil,
		LastUsedAt:         &lastUsedAt,
		CreatedAt:          now.Add(-24 * time.Hour),
		UpdatedAt:          now,
	}

	doc := newUserPasswordCredentialDocument(credential)
	roundTripped := newDomainUserPasswordCredential(doc)

	if roundTripped.ID != credential.ID || roundTripped.UserID != credential.UserID ||
		roundTripped.PasswordHash != credential.PasswordHash || roundTripped.PasswordAlgo != credential.PasswordAlgo ||
		roundTripped.FailedAttemptCount != credential.FailedAttemptCount {
		t.Fatalf("unexpected credential round trip: %+v", roundTripped)
	}
	if roundTripped.LockedUntil == nil || !roundTripped.LockedUntil.Equal(lockedUntil) {
		t.Fatalf("unexpected locked until: %+v", roundTripped.LockedUntil)
	}
	if roundTripped.LastUsedAt == nil || !roundTripped.LastUsedAt.Equal(lastUsedAt) {
		t.Fatalf("unexpected last used at: %+v", roundTripped.LastUsedAt)
	}
}
