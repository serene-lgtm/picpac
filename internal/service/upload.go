package service

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// UploadService defines object upload behavior.
type UploadService interface {
	Upload(ctx context.Context, objectKey string, contentType string, body io.Reader) error
}

// ObjectURLSigner defines private object URL signing behavior.
type ObjectURLSigner interface {
	SignGetURL(ctx context.Context, objectKey string) (string, error)
}

// OSSUploadService uploads objects to Alibaba Cloud OSS.
type OSSUploadService struct {
	bucket              *oss.Bucket
	signedURLTTLSeconds int64
}

// NewOSSUploadService creates an OSS-backed upload service.
func NewOSSUploadService(bucket *oss.Bucket, signedURLTTLSeconds int64) *OSSUploadService {
	return &OSSUploadService{
		bucket:              bucket,
		signedURLTTLSeconds: signedURLTTLSeconds,
	}
}

// Upload uploads an object to OSS.
func (s *OSSUploadService) Upload(ctx context.Context, objectKey string, contentType string, body io.Reader) error {
	if s == nil || s.bucket == nil {
		return fmt.Errorf("upload service is not configured")
	}

	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if objectKey == "" {
		return fmt.Errorf("object key is required")
	}

	if err := s.bucket.PutObject(objectKey, body, oss.ContentType(strings.TrimSpace(contentType))); err != nil {
		return err
	}

	return nil
}

// SignGetURL signs an OSS object URL for temporary GET access.
func (s *OSSUploadService) SignGetURL(ctx context.Context, objectKey string) (string, error) {
	if s == nil || s.bucket == nil {
		return "", fmt.Errorf("upload service is not configured")
	}

	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if objectKey == "" {
		return "", fmt.Errorf("object key is required")
	}

	ttl := s.signedURLTTLSeconds
	if ttl <= 0 {
		ttl = 3600
	}

	signedURL, err := s.bucket.SignURL(objectKey, oss.HTTPGet, ttl)
	if err != nil {
		return "", err
	}

	return normalizeSignedURLPath(signedURL)
}

func normalizeSignedURLPath(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	parsedURL.Path = strings.ReplaceAll(parsedURL.EscapedPath(), "%2F", "/")
	parsedURL.Path = strings.ReplaceAll(parsedURL.Path, "%2f", "/")
	parsedURL.RawPath = ""

	return parsedURL.String(), nil
}
