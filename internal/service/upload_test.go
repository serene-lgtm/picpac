package service

import (
	"strings"
	"testing"
)

func TestNormalizeSignedURLPathRestoresObjectKeySlashes(t *testing.T) {
	t.Parallel()

	rawURL := "https://picpac.oss-cn-shanghai.aliyuncs.com/items%2Fuser_1%2Fitem_1%2Fphotos%2Fphoto_1%2Fsource.jpg?Expires=1784014733&OSSAccessKeyId=test&Signature=c2X80qr6yZOpK7yrLPvtEXdSFWM%3D"

	normalizedURL, err := normalizeSignedURLPath(rawURL)
	if err != nil {
		t.Fatalf("normalizeSignedURLPath returned error: %v", err)
	}

	if strings.Contains(normalizedURL, "%2F") || strings.Contains(normalizedURL, "%2f") {
		t.Fatalf("expected object key slashes to be restored, got %s", normalizedURL)
	}
	if !strings.Contains(normalizedURL, "/items/user_1/item_1/photos/photo_1/source.jpg?") {
		t.Fatalf("expected normalized object path, got %s", normalizedURL)
	}
	if !strings.Contains(normalizedURL, "Signature=c2X80qr6yZOpK7yrLPvtEXdSFWM%3D") {
		t.Fatalf("expected query string encoding to be preserved, got %s", normalizedURL)
	}
}
