package handler

import (
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func validateOptionalObjectID(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return true
	}
	_, err := bson.ObjectIDFromHex(trimmed)
	return err == nil
}
